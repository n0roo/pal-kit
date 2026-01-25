package handoff

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/n0roo/pal-kit/internal/db"
)

// MaxTokenBudget is the default maximum token budget for handoffs
const MaxTokenBudget = 2000

// HandoffType defines the type of handoff
type HandoffType string

const (
	TypeAPIContract HandoffType = "api_contract"
	TypeFileList    HandoffType = "file_list"
	TypeTypeDef     HandoffType = "type_def"
	TypeSchema      HandoffType = "schema"
	TypeConfig      HandoffType = "config"
	TypeCustom      HandoffType = "custom"
)

// Handoff represents a handoff between ports
type Handoff struct {
	ID             string      `json:"id"`
	FromPortID     string      `json:"from_port_id"`
	ToPortID       string      `json:"to_port_id"`
	Type           HandoffType `json:"type"`
	Content        interface{} `json:"content"`
	TokenCount     int         `json:"token_count"`
	MaxTokenBudget int         `json:"max_token_budget"`
	CreatedAt      time.Time   `json:"created_at"`
}

// APIContractContent represents API contract handoff content
type APIContractContent struct {
	Entity      string           `json:"entity,omitempty"`
	Interface   string           `json:"interface,omitempty"`
	Methods     []MethodDef      `json:"methods,omitempty"`
	Fields      []FieldDef       `json:"fields,omitempty"`
	Constraints []string         `json:"constraints,omitempty"`
	Notes       string           `json:"notes,omitempty"`
}

// MethodDef represents a method definition
type MethodDef struct {
	Name       string   `json:"name"`
	Params     []string `json:"params,omitempty"`
	Returns    string   `json:"returns,omitempty"`
	Throws     []string `json:"throws,omitempty"`
	Desc       string   `json:"desc,omitempty"`
}

// FieldDef represents a field definition
type FieldDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Constraints string `json:"constraints,omitempty"`
	Desc        string `json:"desc,omitempty"`
}

// FileListContent represents file list handoff content
type FileListContent struct {
	Files []FileInfo `json:"files"`
}

// FileInfo represents file information
type FileInfo struct {
	Path     string `json:"path"`
	Purpose  string `json:"purpose,omitempty"`
	Modified bool   `json:"modified,omitempty"`
}

// TypeDefContent represents type definition handoff content
type TypeDefContent struct {
	TypeName   string   `json:"type_name"`
	Package    string   `json:"package,omitempty"`
	FilePath   string   `json:"file_path,omitempty"`
	Definition string   `json:"definition"`
	Imports    []string `json:"imports,omitempty"`
}

// SchemaContent represents schema handoff content
type SchemaContent struct {
	TableName  string      `json:"table_name,omitempty"`
	Columns    []ColumnDef `json:"columns,omitempty"`
	Indexes    []string    `json:"indexes,omitempty"`
	ForeignKeys []string   `json:"foreign_keys,omitempty"`
}

// ColumnDef represents a column definition
type ColumnDef struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Nullable    bool   `json:"nullable,omitempty"`
	PrimaryKey  bool   `json:"primary_key,omitempty"`
	Default     string `json:"default,omitempty"`
}

// Store handles handoff persistence
type Store struct {
	db *db.DB
}

// NewStore creates a new handoff store
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// Create creates a new handoff
func (s *Store) Create(fromPortID, toPortID string, handoffType HandoffType, content interface{}) (*Handoff, error) {
	// Serialize content
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("콘텐츠 직렬화 실패: %w", err)
	}

	// Estimate token count (rough: 4 chars = 1 token)
	tokenCount := len(string(contentJSON)) / 4

	// Check budget
	if tokenCount > MaxTokenBudget {
		return nil, fmt.Errorf("토큰 제한 초과: %d > %d (최대)", tokenCount, MaxTokenBudget)
	}

	id := uuid.New().String()
	now := time.Now()

	_, err = s.db.Exec(`
		INSERT INTO port_handoffs (id, from_port_id, to_port_id, handoff_type, content, token_count, max_token_budget, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, id, fromPortID, toPortID, handoffType, string(contentJSON), tokenCount, MaxTokenBudget, now)

	if err != nil {
		return nil, fmt.Errorf("Handoff 생성 실패: %w", err)
	}

	return &Handoff{
		ID:             id,
		FromPortID:     fromPortID,
		ToPortID:       toPortID,
		Type:           handoffType,
		Content:        content,
		TokenCount:     tokenCount,
		MaxTokenBudget: MaxTokenBudget,
		CreatedAt:      now,
	}, nil
}

// CreateAPIContract creates an API contract handoff
func (s *Store) CreateAPIContract(fromPortID, toPortID string, content APIContractContent) (*Handoff, error) {
	return s.Create(fromPortID, toPortID, TypeAPIContract, content)
}

// CreateFileList creates a file list handoff
func (s *Store) CreateFileList(fromPortID, toPortID string, files []FileInfo) (*Handoff, error) {
	return s.Create(fromPortID, toPortID, TypeFileList, FileListContent{Files: files})
}

// CreateTypeDef creates a type definition handoff
func (s *Store) CreateTypeDef(fromPortID, toPortID string, content TypeDefContent) (*Handoff, error) {
	return s.Create(fromPortID, toPortID, TypeTypeDef, content)
}

// CreateSchema creates a schema handoff
func (s *Store) CreateSchema(fromPortID, toPortID string, content SchemaContent) (*Handoff, error) {
	return s.Create(fromPortID, toPortID, TypeSchema, content)
}

// Get retrieves a handoff by ID
func (s *Store) Get(id string) (*Handoff, error) {
	var h Handoff
	var contentJSON string
	var handoffType sql.NullString

	err := s.db.QueryRow(`
		SELECT id, from_port_id, to_port_id, handoff_type, content, token_count, max_token_budget, created_at
		FROM port_handoffs WHERE id = ?
	`, id).Scan(&h.ID, &h.FromPortID, &h.ToPortID, &handoffType, &contentJSON, &h.TokenCount, &h.MaxTokenBudget, &h.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("Handoff '%s'을(를) 찾을 수 없습니다", id)
	}
	if err != nil {
		return nil, err
	}

	if handoffType.Valid {
		h.Type = HandoffType(handoffType.String)
	}

	// Deserialize content
	var content interface{}
	if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
		h.Content = contentJSON // fallback to raw string
	} else {
		h.Content = content
	}

	return &h, nil
}

// GetForPort retrieves all handoffs for a port (as receiver)
func (s *Store) GetForPort(toPortID string) ([]*Handoff, error) {
	rows, err := s.db.Query(`
		SELECT id, from_port_id, to_port_id, handoff_type, content, token_count, max_token_budget, created_at
		FROM port_handoffs
		WHERE to_port_id = ?
		ORDER BY created_at
	`, toPortID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanHandoffs(rows)
}

// GetFromPort retrieves all handoffs from a port (as sender)
func (s *Store) GetFromPort(fromPortID string) ([]*Handoff, error) {
	rows, err := s.db.Query(`
		SELECT id, from_port_id, to_port_id, handoff_type, content, token_count, max_token_budget, created_at
		FROM port_handoffs
		WHERE from_port_id = ?
		ORDER BY created_at
	`, fromPortID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanHandoffs(rows)
}

// GetBetweenPorts retrieves handoffs between two specific ports
func (s *Store) GetBetweenPorts(fromPortID, toPortID string) ([]*Handoff, error) {
	rows, err := s.db.Query(`
		SELECT id, from_port_id, to_port_id, handoff_type, content, token_count, max_token_budget, created_at
		FROM port_handoffs
		WHERE from_port_id = ? AND to_port_id = ?
		ORDER BY created_at
	`, fromPortID, toPortID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	return s.scanHandoffs(rows)
}

func (s *Store) scanHandoffs(rows *sql.Rows) ([]*Handoff, error) {
	var handoffs []*Handoff
	for rows.Next() {
		var h Handoff
		var contentJSON string
		var handoffType sql.NullString

		err := rows.Scan(&h.ID, &h.FromPortID, &h.ToPortID, &handoffType, &contentJSON,
			&h.TokenCount, &h.MaxTokenBudget, &h.CreatedAt)
		if err != nil {
			continue
		}

		if handoffType.Valid {
			h.Type = HandoffType(handoffType.String)
		}

		var content interface{}
		if err := json.Unmarshal([]byte(contentJSON), &content); err != nil {
			h.Content = contentJSON
		} else {
			h.Content = content
		}

		handoffs = append(handoffs, &h)
	}

	return handoffs, nil
}

// GetTotalTokens calculates total tokens of all handoffs for a port
func (s *Store) GetTotalTokens(toPortID string) (int, error) {
	var total int
	err := s.db.QueryRow(`
		SELECT COALESCE(SUM(token_count), 0) FROM port_handoffs WHERE to_port_id = ?
	`, toPortID).Scan(&total)
	return total, err
}

// Validate validates a handoff content against budget
func Validate(content interface{}, maxBudget int) error {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return fmt.Errorf("콘텐츠 검증 실패: %w", err)
	}

	tokenCount := len(string(contentJSON)) / 4
	if tokenCount > maxBudget {
		return fmt.Errorf("토큰 제한 초과: %d > %d", tokenCount, maxBudget)
	}

	return nil
}

// EstimateTokens estimates the token count for content
func EstimateTokens(content interface{}) (int, error) {
	contentJSON, err := json.Marshal(content)
	if err != nil {
		return 0, err
	}
	return len(string(contentJSON)) / 4, nil
}

// Builder helps build handoff content
type Builder struct {
	fromPortID string
	toPortID   string
	content    map[string]interface{}
}

// NewBuilder creates a new handoff builder
func NewBuilder(fromPortID, toPortID string) *Builder {
	return &Builder{
		fromPortID: fromPortID,
		toPortID:   toPortID,
		content:    make(map[string]interface{}),
	}
}

// AddEntity adds entity information
func (b *Builder) AddEntity(name string, fields []FieldDef) *Builder {
	b.content["entity"] = name
	b.content["fields"] = fields
	return b
}

// AddMethods adds method definitions
func (b *Builder) AddMethods(methods []MethodDef) *Builder {
	b.content["methods"] = methods
	return b
}

// AddFiles adds file information
func (b *Builder) AddFiles(files []FileInfo) *Builder {
	b.content["files"] = files
	return b
}

// AddConstraints adds constraints
func (b *Builder) AddConstraints(constraints []string) *Builder {
	b.content["constraints"] = constraints
	return b
}

// AddNotes adds notes
func (b *Builder) AddNotes(notes string) *Builder {
	b.content["notes"] = notes
	return b
}

// Add adds a custom key-value pair
func (b *Builder) Add(key string, value interface{}) *Builder {
	b.content[key] = value
	return b
}

// EstimateTokens estimates the current token count
func (b *Builder) EstimateTokens() (int, error) {
	return EstimateTokens(b.content)
}

// Build builds the handoff and validates the budget
func (b *Builder) Build() (*APIContractContent, error) {
	tokens, err := b.EstimateTokens()
	if err != nil {
		return nil, err
	}

	if tokens > MaxTokenBudget {
		return nil, fmt.Errorf("토큰 제한 초과: %d > %d. 콘텐츠를 줄이세요", tokens, MaxTokenBudget)
	}

	content := &APIContractContent{}
	if entity, ok := b.content["entity"].(string); ok {
		content.Entity = entity
	}
	if fields, ok := b.content["fields"].([]FieldDef); ok {
		content.Fields = fields
	}
	if methods, ok := b.content["methods"].([]MethodDef); ok {
		content.Methods = methods
	}
	if constraints, ok := b.content["constraints"].([]string); ok {
		content.Constraints = constraints
	}
	if notes, ok := b.content["notes"].(string); ok {
		content.Notes = notes
	}

	return content, nil
}

// Summarize creates a summary of handoff content for display
func Summarize(h *Handoff) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("Type: %s", h.Type))
	parts = append(parts, fmt.Sprintf("Tokens: %d/%d", h.TokenCount, h.MaxTokenBudget))
	parts = append(parts, fmt.Sprintf("From: %s → To: %s", h.FromPortID, h.ToPortID))

	contentJSON, _ := json.MarshalIndent(h.Content, "", "  ")
	if len(contentJSON) > 500 {
		parts = append(parts, fmt.Sprintf("Content: %s...", string(contentJSON[:500])))
	} else {
		parts = append(parts, fmt.Sprintf("Content: %s", string(contentJSON)))
	}

	return strings.Join(parts, "\n")
}

// MergeHandoffs merges multiple handoffs into a single context
func MergeHandoffs(handoffs []*Handoff) (map[string]interface{}, int, error) {
	merged := make(map[string]interface{})
	totalTokens := 0

	for _, h := range handoffs {
		key := fmt.Sprintf("%s_%s", h.FromPortID, h.Type)
		merged[key] = h.Content
		totalTokens += h.TokenCount
	}

	if totalTokens > MaxTokenBudget*2 { // Allow some overhead for merged context
		return nil, totalTokens, fmt.Errorf("병합된 컨텍스트가 너무 큽니다: %d 토큰", totalTokens)
	}

	return merged, totalTokens, nil
}
