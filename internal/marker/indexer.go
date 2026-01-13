package marker

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/n0roo/pal-kit/internal/db"
)

// IndexResult contains results from marker indexing
type IndexResult struct {
	Added   int
	Updated int
	Removed int
	Errors  []string
}

// Indexer handles marker indexing to database
type Indexer struct {
	db          *db.DB
	service     *Service
	projectRoot string
}

// NewIndexer creates a new marker indexer
func NewIndexer(database *db.DB, projectRoot string) *Indexer {
	return &Indexer{
		db:          database,
		service:     NewService(projectRoot),
		projectRoot: projectRoot,
	}
}

// Index scans and indexes all markers in the project
func (i *Indexer) Index() (*IndexResult, error) {
	result := &IndexResult{}

	// 1. 프로젝트 내 모든 마커 스캔
	markers, err := i.service.Scan()
	if err != nil {
		return nil, fmt.Errorf("마커 스캔 실패: %w", err)
	}

	// 2. 기존 마커 목록 조회
	existingMarkers := make(map[string]bool)
	rows, err := i.db.Query(`SELECT file_path, line FROM code_markers WHERE project_root = ?`, i.projectRoot)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var path string
			var line int
			if rows.Scan(&path, &line) == nil {
				key := fmt.Sprintf("%s:%d", path, line)
				existingMarkers[key] = true
			}
		}
	}

	// 3. 마커 저장/업데이트
	newMarkers := make(map[string]bool)
	for _, m := range markers {
		key := fmt.Sprintf("%s:%d", m.FilePath, m.Line)
		newMarkers[key] = true

		generated := 0
		if m.Generated {
			generated = 1
		}

		// UPSERT
		res, err := i.db.Exec(`
			INSERT INTO code_markers (port, layer, domain, adapter, generated, file_path, line, project_root, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(file_path, line) DO UPDATE SET
				port = excluded.port,
				layer = excluded.layer,
				domain = excluded.domain,
				adapter = excluded.adapter,
				generated = excluded.generated,
				updated_at = CURRENT_TIMESTAMP
		`, m.Port, m.Layer, m.Domain, m.Adapter, generated, m.FilePath, m.Line, i.projectRoot)

		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s:%d: %v", m.FilePath, m.Line, err))
			continue
		}

		affected, _ := res.RowsAffected()
		if existingMarkers[key] {
			result.Updated++
		} else if affected > 0 {
			result.Added++
		}

		// 4. 의존성 저장
		if len(m.Depends) > 0 {
			// 기존 의존성 삭제
			i.db.Exec(`DELETE FROM code_marker_deps WHERE from_port = ?`, m.Port)

			// 새 의존성 추가
			for _, dep := range m.Depends {
				i.db.Exec(`INSERT OR IGNORE INTO code_marker_deps (from_port, to_port) VALUES (?, ?)`, m.Port, dep)
			}
		}
	}

	// 5. 삭제된 마커 정리
	for key := range existingMarkers {
		if !newMarkers[key] {
			parts := strings.SplitN(key, ":", 2)
			if len(parts) == 2 {
				i.db.Exec(`DELETE FROM code_markers WHERE file_path = ? AND line = ?`, parts[0], parts[1])
				result.Removed++
			}
		}
	}

	return result, nil
}

// LinkToDocuments links markers to port specification documents
func (i *Indexer) LinkToDocuments() (int, error) {
	linked := 0

	// 마커 포트 목록 조회
	rows, err := i.db.Query(`SELECT DISTINCT port FROM code_markers WHERE project_root = ?`, i.projectRoot)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	var ports []string
	for rows.Next() {
		var port string
		if rows.Scan(&port) == nil {
			ports = append(ports, port)
		}
	}

	// 각 포트에 대해 문서 매칭
	for _, port := range ports {
		// 포트 이름에서 문서 ID 추론
		// L1-SessionService -> ports-session, ports-sessions
		// L2-InventoryCompositeService -> ports-inventory

		docPatterns := generateDocPatterns(port)

		for _, pattern := range docPatterns {
			var docID string
			err := i.db.QueryRow(`SELECT id FROM documents WHERE id LIKE ? OR path LIKE ?`, pattern+"%", "%"+pattern+"%").Scan(&docID)
			if err == nil && docID != "" {
				_, err = i.db.Exec(`INSERT OR IGNORE INTO marker_port_links (marker_port, document_id) VALUES (?, ?)`, port, docID)
				if err == nil {
					linked++
				}
				break
			}
		}
	}

	return linked, nil
}

// generateDocPatterns generates document search patterns from port name
func generateDocPatterns(port string) []string {
	patterns := []string{}

	// 원본 포트명
	patterns = append(patterns, port)

	// 레이어 접두사 제거 (L1-, L2-, LM-)
	name := port
	if strings.HasPrefix(port, "L1-") || strings.HasPrefix(port, "L2-") || strings.HasPrefix(port, "LM-") {
		name = port[3:]
	}

	// Service 접미사 제거
	name = strings.TrimSuffix(name, "Service")
	name = strings.TrimSuffix(name, "Composite")
	name = strings.TrimSuffix(name, "Command")
	name = strings.TrimSuffix(name, "Query")
	name = strings.TrimSuffix(name, "Coordinator")

	// CamelCase -> kebab-case
	kebab := camelToKebab(name)
	patterns = append(patterns, kebab)
	patterns = append(patterns, "ports-"+kebab)

	// 복수형 시도
	patterns = append(patterns, kebab+"s")
	patterns = append(patterns, "ports-"+kebab+"s")

	return patterns
}

// camelToKebab converts CamelCase to kebab-case
func camelToKebab(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune('-')
		}
		result.WriteRune(r)
	}
	return strings.ToLower(result.String())
}

// GetMarkerStats returns marker statistics
type MarkerStats struct {
	TotalMarkers int            `json:"total_markers"`
	ByLayer      map[string]int `json:"by_layer"`
	ByDomain     map[string]int `json:"by_domain"`
	Generated    int            `json:"generated"`
	WithDeps     int            `json:"with_deps"`
}

// GetStats returns marker statistics
func (i *Indexer) GetStats() (*MarkerStats, error) {
	stats := &MarkerStats{
		ByLayer:  make(map[string]int),
		ByDomain: make(map[string]int),
	}

	// 총 개수
	i.db.QueryRow(`SELECT COUNT(*) FROM code_markers WHERE project_root = ?`, i.projectRoot).Scan(&stats.TotalMarkers)

	// 레이어별
	rows, err := i.db.Query(`SELECT COALESCE(layer, 'unknown'), COUNT(*) FROM code_markers WHERE project_root = ? GROUP BY layer`, i.projectRoot)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var layer string
			var count int
			if rows.Scan(&layer, &count) == nil {
				stats.ByLayer[layer] = count
			}
		}
	}

	// 도메인별
	rows, err = i.db.Query(`SELECT COALESCE(domain, 'unknown'), COUNT(*) FROM code_markers WHERE project_root = ? AND domain IS NOT NULL AND domain != '' GROUP BY domain`, i.projectRoot)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var domain string
			var count int
			if rows.Scan(&domain, &count) == nil {
				stats.ByDomain[domain] = count
			}
		}
	}

	// Generated
	i.db.QueryRow(`SELECT COUNT(*) FROM code_markers WHERE project_root = ? AND generated = 1`, i.projectRoot).Scan(&stats.Generated)

	// 의존성 있는 마커
	i.db.QueryRow(`SELECT COUNT(DISTINCT from_port) FROM code_marker_deps`, i.projectRoot).Scan(&stats.WithDeps)

	return stats, nil
}

// FindMarkersByDomain returns markers for a specific domain
func (i *Indexer) FindMarkersByDomain(domain string) ([]Marker, error) {
	rows, err := i.db.Query(`
		SELECT port, layer, domain, adapter, generated, file_path, line
		FROM code_markers
		WHERE project_root = ? AND domain = ?
		ORDER BY layer, port
	`, i.projectRoot, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMarkersFromRows(rows)
}

// FindMarkersByLayer returns markers for a specific layer
func (i *Indexer) FindMarkersByLayer(layer string) ([]Marker, error) {
	rows, err := i.db.Query(`
		SELECT port, layer, domain, adapter, generated, file_path, line
		FROM code_markers
		WHERE project_root = ? AND layer = ?
		ORDER BY domain, port
	`, i.projectRoot, layer)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMarkersFromRows(rows)
}

// GetDependencyGraph returns the full dependency graph
func (i *Indexer) GetDependencyGraph() (map[string][]string, error) {
	rows, err := i.db.Query(`SELECT from_port, to_port FROM code_marker_deps`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	graph := make(map[string][]string)
	for rows.Next() {
		var from, to string
		if rows.Scan(&from, &to) == nil {
			graph[from] = append(graph[from], to)
		}
	}

	return graph, nil
}

// GetReverseDependencies returns ports that depend on the given port
func (i *Indexer) GetReverseDependencies(port string) ([]string, error) {
	rows, err := i.db.Query(`SELECT from_port FROM code_marker_deps WHERE to_port = ?`, port)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var deps []string
	for rows.Next() {
		var from string
		if rows.Scan(&from) == nil {
			deps = append(deps, from)
		}
	}

	return deps, nil
}

// GetLinkedDocument returns the linked document ID for a marker port
func (i *Indexer) GetLinkedDocument(markerPort string) (string, error) {
	var docID string
	err := i.db.QueryRow(`SELECT document_id FROM marker_port_links WHERE marker_port = ?`, markerPort).Scan(&docID)
	if err != nil {
		return "", err
	}
	return docID, nil
}

// GetMarkersForDocument returns all markers linked to a document
func (i *Indexer) GetMarkersForDocument(docID string) ([]Marker, error) {
	rows, err := i.db.Query(`
		SELECT cm.port, cm.layer, cm.domain, cm.adapter, cm.generated, cm.file_path, cm.line
		FROM code_markers cm
		JOIN marker_port_links mpl ON cm.port = mpl.marker_port
		WHERE mpl.document_id = ?
	`, docID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return scanMarkersFromRows(rows)
}

// scanMarkersFromRows is a helper to scan markers from DB rows
func scanMarkersFromRows(rows interface {
	Next() bool
	Scan(...interface{}) error
}) ([]Marker, error) {
	var markers []Marker
	for rows.Next() {
		var m Marker
		var generated int
		if err := rows.Scan(&m.Port, &m.Layer, &m.Domain, &m.Adapter, &generated, &m.FilePath, &m.Line); err != nil {
			return nil, err
		}
		m.Generated = generated == 1

		// 상대 경로 처리
		if filepath.IsAbs(m.FilePath) {
			// 그대로 유지
		}

		markers = append(markers, m)
	}
	return markers, nil
}
