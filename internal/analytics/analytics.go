package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/marcboeker/go-duckdb/v2"
)

// AnalyticsDB wraps DuckDB for analytics queries
type AnalyticsDB struct {
	conn *duckdb.Conn
	path string
}

// Config holds analytics configuration
type Config struct {
	DBPath    string // DuckDB 파일 경로
	CachePath string // JSON/Parquet 캐시 경로
	DocsPath  string // 문서 베이스 경로 (Obsidian vault)
}

// DocMeta represents document metadata
type DocMeta struct {
	Path          string   `json:"path"`
	Title         string   `json:"title"`
	Type          string   `json:"type"`
	Tags          []string `json:"tags"`
	TokenEstimate int      `json:"token_estimate"`
	UpdatedAt     string   `json:"updated_at"`
}

// Convention represents a convention definition
type Convention struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	TokenCount       int      `json:"token_count"`
	Priority         string   `json:"priority"`
	AutoLoadPatterns []string `json:"auto_load_patterns"`
	ContentFile      string   `json:"content_file"`
}

// TokenStats represents token usage statistics
type TokenStats struct {
	TotalTokens  int64   `json:"total_tokens"`
	AvgAttention float64 `json:"avg_attention"`
	MessageCount int     `json:"message_count"`
	CompactCount int     `json:"compact_count"`
}

// TokenTrendPoint represents a single point in token usage trend
type TokenTrendPoint struct {
	Day          string  `json:"day"`
	TotalTokens  int64   `json:"total_tokens"`
	AvgAttention float64 `json:"avg_attention"`
	EventCount   int     `json:"event_count"`
}

// AgentVersionStat represents statistics for a specific agent version
type AgentVersionStat struct {
	Version      int     `json:"version"`
	UsageCount   int     `json:"usage_count"`
	AvgAttention float64 `json:"avg_attention"`
	AvgQuality   float64 `json:"avg_quality"`
	SuccessRate  float64 `json:"success_rate"`
}

// New creates a new AnalyticsDB instance
func New(cfg Config) (*AnalyticsDB, error) {
	// 디렉토리 생성
	dir := filepath.Dir(cfg.DBPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	// 캐시 디렉토리 생성
	if cfg.CachePath != "" {
		if err := os.MkdirAll(cfg.CachePath, 0755); err != nil {
			return nil, fmt.Errorf("캐시 디렉토리 생성 실패: %w", err)
		}
	}

	// DuckDB 연결
	conn, err := duckdb.NewConn(cfg.DBPath, nil)
	if err != nil {
		return nil, fmt.Errorf("DuckDB 열기 실패: %w", err)
	}

	return &AnalyticsDB{
		conn: conn,
		path: cfg.DBPath,
	}, nil
}

// Close closes the database connection
func (a *AnalyticsDB) Close() error {
	return a.conn.Close()
}

// SearchDocs searches documents using DuckDB's JSON query capabilities
func (a *AnalyticsDB) SearchDocs(ctx context.Context, indexPath, query string) ([]DocMeta, error) {
	sqlQuery := fmt.Sprintf(`
		SELECT path, title, type, tags, token_estimate, updated_at
		FROM read_json_auto('%s')
		WHERE lower(title) LIKE lower('%%%s%%')
		   OR lower(path) LIKE lower('%%%s%%')
		ORDER BY updated_at DESC
		LIMIT 50
	`, indexPath, query, query)

	rows, err := a.conn.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("문서 검색 실패: %w", err)
	}
	defer rows.Close()

	var results []DocMeta
	for rows.Next() {
		var doc DocMeta
		var tagsJSON string
		if err := rows.Scan(&doc.Path, &doc.Title, &doc.Type, &tagsJSON, &doc.TokenEstimate, &doc.UpdatedAt); err != nil {
			continue
		}
		if tagsJSON != "" {
			json.Unmarshal([]byte(tagsJSON), &doc.Tags)
		}
		results = append(results, doc)
	}

	return results, nil
}

// GetDocsIndex returns all documents from the index
func (a *AnalyticsDB) GetDocsIndex(ctx context.Context, indexPath string) ([]DocMeta, error) {
	sqlQuery := fmt.Sprintf(`
		SELECT path, title, type, tags, token_estimate, updated_at
		FROM read_json_auto('%s')
		ORDER BY path
	`, indexPath)

	rows, err := a.conn.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("문서 인덱스 로드 실패: %w", err)
	}
	defer rows.Close()

	var results []DocMeta
	for rows.Next() {
		var doc DocMeta
		var tagsJSON string
		if err := rows.Scan(&doc.Path, &doc.Title, &doc.Type, &tagsJSON, &doc.TokenEstimate, &doc.UpdatedAt); err != nil {
			continue
		}
		if tagsJSON != "" {
			json.Unmarshal([]byte(tagsJSON), &doc.Tags)
		}
		results = append(results, doc)
	}

	return results, nil
}

// FindConventions finds conventions matching a pattern
func (a *AnalyticsDB) FindConventions(ctx context.Context, indexPath, pattern string) ([]Convention, error) {
	sqlQuery := fmt.Sprintf(`
		SELECT id, name, token_count, priority, auto_load_patterns, content_file
		FROM read_json_auto('%s')
		WHERE lower(name) LIKE lower('%%%s%%')
		ORDER BY priority, name
	`, indexPath, pattern)

	rows, err := a.conn.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("컨벤션 검색 실패: %w", err)
	}
	defer rows.Close()

	var results []Convention
	for rows.Next() {
		var conv Convention
		var patternsJSON string
		if err := rows.Scan(&conv.ID, &conv.Name, &conv.TokenCount, &conv.Priority, &patternsJSON, &conv.ContentFile); err != nil {
			continue
		}
		if patternsJSON != "" {
			json.Unmarshal([]byte(patternsJSON), &conv.AutoLoadPatterns)
		}
		results = append(results, conv)
	}

	return results, nil
}

// GetTokenStats calculates token statistics from parquet history
func (a *AnalyticsDB) GetTokenStats(ctx context.Context, historyPath, sessionID string) (*TokenStats, error) {
	sqlQuery := fmt.Sprintf(`
		SELECT 
			COALESCE(SUM(token_count), 0) as total_tokens,
			COALESCE(AVG(attention_score), 0) as avg_attention,
			COUNT(*) as message_count,
			COALESCE(SUM(CASE WHEN event_type = 'compact' THEN 1 ELSE 0 END), 0) as compact_count
		FROM read_parquet('%s')
		WHERE session_id = '%s'
	`, historyPath, sessionID)

	rows, err := a.conn.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("토큰 통계 조회 실패: %w", err)
	}
	defer rows.Close()

	var stats TokenStats
	if rows.Next() {
		if err := rows.Scan(&stats.TotalTokens, &stats.AvgAttention, &stats.MessageCount, &stats.CompactCount); err != nil {
			return nil, fmt.Errorf("토큰 통계 스캔 실패: %w", err)
		}
	}

	return &stats, nil
}

// GetTokenTrend returns token usage trend over time
func (a *AnalyticsDB) GetTokenTrend(ctx context.Context, historyPath string, days int) ([]TokenTrendPoint, error) {
	sqlQuery := fmt.Sprintf(`
		SELECT 
			strftime(created_at, '%%Y-%%m-%%d') as day,
			SUM(token_count) as total_tokens,
			AVG(attention_score) as avg_attention,
			COUNT(*) as event_count
		FROM read_parquet('%s')
		WHERE created_at >= current_date - interval '%d day'
		GROUP BY 1
		ORDER BY 1
	`, historyPath, days)

	rows, err := a.conn.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("토큰 추이 조회 실패: %w", err)
	}
	defer rows.Close()

	var results []TokenTrendPoint
	for rows.Next() {
		var point TokenTrendPoint
		if err := rows.Scan(&point.Day, &point.TotalTokens, &point.AvgAttention, &point.EventCount); err != nil {
			continue
		}
		results = append(results, point)
	}

	return results, nil
}

// AgentVersionStats returns performance statistics by agent version
func (a *AnalyticsDB) AgentVersionStats(ctx context.Context, historyPath, agentID string) ([]AgentVersionStat, error) {
	sqlQuery := fmt.Sprintf(`
		SELECT 
			agent_version,
			COUNT(*) as usage_count,
			AVG(attention_avg) as avg_attention,
			AVG(quality_score) as avg_quality,
			SUM(CASE WHEN outcome = 'success' THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as success_rate
		FROM read_parquet('%s')
		WHERE agent_id = '%s'
		GROUP BY agent_version
		ORDER BY agent_version DESC
	`, historyPath, agentID)

	rows, err := a.conn.Query(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("에이전트 버전 통계 조회 실패: %w", err)
	}
	defer rows.Close()

	var results []AgentVersionStat
	for rows.Next() {
		var stat AgentVersionStat
		if err := rows.Scan(&stat.Version, &stat.UsageCount, &stat.AvgAttention, &stat.AvgQuality, &stat.SuccessRate); err != nil {
			continue
		}
		results = append(results, stat)
	}

	return results, nil
}

// ExportToParquet exports query result to parquet format
func (a *AnalyticsDB) ExportToParquet(ctx context.Context, query, outputPath string) error {
	exportQuery := fmt.Sprintf("COPY (%s) TO '%s' (FORMAT PARQUET)", query, outputPath)
	_, err := a.conn.Exec(ctx, exportQuery)
	if err != nil {
		return fmt.Errorf("Parquet 내보내기 실패: %w", err)
	}
	return nil
}

// QueryJSON executes a query on a JSON file and returns results as maps
func (a *AnalyticsDB) QueryJSON(ctx context.Context, jsonPath, whereClause string) ([]map[string]interface{}, error) {
	fullQuery := fmt.Sprintf("SELECT * FROM read_json_auto('%s')", jsonPath)
	if whereClause != "" {
		fullQuery += " WHERE " + whereClause
	}

	rows, err := a.conn.Query(ctx, fullQuery)
	if err != nil {
		return nil, fmt.Errorf("JSON 쿼리 실패: %w", err)
	}
	defer rows.Close()

	// Get column names
	colTypes := rows.ColumnTypes()
	cols := make([]string, len(colTypes))
	for i, ct := range colTypes {
		cols[i] = ct.Name()
	}

	var results []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(cols))
		valuePtrs := make([]interface{}, len(cols))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		row := make(map[string]interface{})
		for i, col := range cols {
			row[col] = values[i]
		}
		results = append(results, row)
	}

	return results, nil
}

// Exec executes a SQL statement
func (a *AnalyticsDB) Exec(ctx context.Context, query string) error {
	_, err := a.conn.Exec(ctx, query)
	return err
}
