package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/marcboeker/go-duckdb/v2"
)

// DuckDB 스키마 버전
const duckDBSchemaVersion = 1

// DuckDB 스키마 (SQLite 호환 + DuckDB 최적화)
const duckDBSchema = `
-- ============================================================
-- 기본 테이블
-- ============================================================

-- 메타데이터
CREATE TABLE IF NOT EXISTS metadata (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT now()
);

-- Lock 관리
CREATE TABLE IF NOT EXISTS locks (
    resource VARCHAR PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    acquired_at TIMESTAMP DEFAULT now()
);

-- 프로젝트
CREATE TABLE IF NOT EXISTS projects (
    root VARCHAR PRIMARY KEY,
    name VARCHAR,
    description VARCHAR,
    logical_root VARCHAR,
    last_active TIMESTAMP,
    session_count INTEGER DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    total_cost DOUBLE DEFAULT 0,
    created_at TIMESTAMP DEFAULT now()
);

-- ============================================================
-- 세션 관리 (계층 구조 지원)
-- ============================================================

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    port_id VARCHAR,
    title VARCHAR,
    status VARCHAR DEFAULT 'running',
    
    -- 계층 구조
    parent_id VARCHAR,
    root_id VARCHAR,
    depth INTEGER DEFAULT 0,
    path VARCHAR,
    type VARCHAR DEFAULT 'single',
    
    -- 에이전트 연결
    agent_id VARCHAR,
    agent_version INTEGER,
    
    -- 상태 추적
    substatus VARCHAR,
    attention_score DOUBLE,
    token_budget INTEGER,
    context_snapshot VARCHAR,
    checkpoint_id VARCHAR,
    output_summary VARCHAR,
    
    -- 프로젝트 정보
    project_root VARCHAR,
    project_name VARCHAR,
    claude_session_id VARCHAR,
    transcript_path VARCHAR,
    cwd VARCHAR,
    session_type VARCHAR DEFAULT 'single',
    parent_session VARCHAR,
    
    -- 환경 정보
    created_env VARCHAR,
    last_env VARCHAR,
    
    -- 타임스탬프
    started_at TIMESTAMP DEFAULT now(),
    ended_at TIMESTAMP,
    
    -- 사용량
    input_tokens BIGINT DEFAULT 0,
    output_tokens BIGINT DEFAULT 0,
    cache_read_tokens BIGINT DEFAULT 0,
    cache_create_tokens BIGINT DEFAULT 0,
    cost_usd DOUBLE DEFAULT 0,
    compact_count INTEGER DEFAULT 0,
    last_compact_at TIMESTAMP,
    jsonl_path VARCHAR
);

CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_port ON sessions(port_id);
CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at);
CREATE INDEX IF NOT EXISTS idx_sessions_parent ON sessions(parent_id);
CREATE INDEX IF NOT EXISTS idx_sessions_root ON sessions(root_id);
CREATE INDEX IF NOT EXISTS idx_sessions_type ON sessions(type);
CREATE INDEX IF NOT EXISTS idx_sessions_path ON sessions(path);
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_root);

-- 세션 이벤트
CREATE TABLE IF NOT EXISTS session_events (
    id INTEGER PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    event_type VARCHAR NOT NULL,
    event_data VARCHAR,
    created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_session_events_session ON session_events(session_id);
CREATE INDEX IF NOT EXISTS idx_session_events_type ON session_events(event_type);
CREATE INDEX IF NOT EXISTS idx_session_events_created ON session_events(created_at);

-- 세션 Attention 추적
CREATE TABLE IF NOT EXISTS session_attention (
    session_id VARCHAR PRIMARY KEY,
    port_id VARCHAR,
    current_context_hash VARCHAR,
    loaded_tokens INTEGER DEFAULT 0,
    available_tokens INTEGER,
    token_budget INTEGER,
    focus_score DOUBLE DEFAULT 1.0,
    drift_score DOUBLE DEFAULT 0.0,
    drift_count INTEGER DEFAULT 0,
    last_compaction_at TIMESTAMP,
    loaded_files VARCHAR,
    loaded_conventions VARCHAR,
    updated_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_session_attention_focus ON session_attention(focus_score);

-- Compact 이벤트
CREATE TABLE IF NOT EXISTS compact_events (
    id VARCHAR PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    trigger_reason VARCHAR,
    before_tokens INTEGER,
    after_tokens INTEGER,
    preserved_context VARCHAR,
    discarded_context VARCHAR,
    checkpoint_before VARCHAR,
    recovery_hint VARCHAR,
    created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_compact_events_session ON compact_events(session_id);
CREATE INDEX IF NOT EXISTS idx_compact_events_time ON compact_events(created_at);

-- 컴팩션 히스토리 (레거시 호환)
CREATE TABLE IF NOT EXISTS compactions (
    id INTEGER PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    triggered_at TIMESTAMP DEFAULT now(),
    trigger_type VARCHAR DEFAULT 'auto',
    context_summary VARCHAR,
    tokens_before INTEGER
);

CREATE INDEX IF NOT EXISTS idx_compactions_session ON compactions(session_id);

-- ============================================================
-- 포트 관리
-- ============================================================

CREATE TABLE IF NOT EXISTS ports (
    id VARCHAR PRIMARY KEY,
    title VARCHAR,
    status VARCHAR DEFAULT 'pending',
    port_type VARCHAR DEFAULT 'atomic',
    session_id VARCHAR,
    file_path VARCHAR,
    agent_id VARCHAR,
    
    -- 타임스탬프
    created_at TIMESTAMP DEFAULT now(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- 사용량
    input_tokens BIGINT DEFAULT 0,
    output_tokens BIGINT DEFAULT 0,
    cost_usd DOUBLE DEFAULT 0,
    duration_secs INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_ports_status ON ports(status);
CREATE INDEX IF NOT EXISTS idx_ports_session ON ports(session_id);
CREATE INDEX IF NOT EXISTS idx_ports_type ON ports(port_type);

-- 포트 의존성
CREATE TABLE IF NOT EXISTS port_dependencies (
    port_id VARCHAR NOT NULL,
    depends_on VARCHAR NOT NULL,
    dependency_type VARCHAR,
    required_outputs VARCHAR,
    satisfied INTEGER DEFAULT 0,
    satisfied_at TIMESTAMP,
    PRIMARY KEY (port_id, depends_on)
);

-- 포트 Handoff
CREATE TABLE IF NOT EXISTS port_handoffs (
    id VARCHAR PRIMARY KEY,
    from_port_id VARCHAR NOT NULL,
    to_port_id VARCHAR NOT NULL,
    handoff_type VARCHAR,
    content VARCHAR NOT NULL,
    token_count INTEGER,
    max_token_budget INTEGER DEFAULT 2000,
    created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_port_handoffs_from ON port_handoffs(from_port_id);
CREATE INDEX IF NOT EXISTS idx_port_handoffs_to ON port_handoffs(to_port_id);

-- ============================================================
-- 에이전트 관리
-- ============================================================

CREATE TABLE IF NOT EXISTS agents (
    id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    description VARCHAR,
    capabilities VARCHAR,
    current_version INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agents_type ON agents(type);
CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);

-- 에이전트 버전
CREATE TABLE IF NOT EXISTS agent_versions (
    id VARCHAR PRIMARY KEY,
    agent_id VARCHAR NOT NULL,
    version INTEGER NOT NULL,
    spec_content VARCHAR NOT NULL,
    spec_hash VARCHAR,
    change_summary VARCHAR,
    change_reason VARCHAR,
    avg_attention_score DOUBLE,
    avg_completion_rate DOUBLE,
    avg_token_efficiency DOUBLE,
    usage_count INTEGER DEFAULT 0,
    status VARCHAR DEFAULT 'active',
    created_at TIMESTAMP DEFAULT now(),
    UNIQUE(agent_id, version)
);

CREATE INDEX IF NOT EXISTS idx_agent_versions_agent ON agent_versions(agent_id, version);
CREATE INDEX IF NOT EXISTS idx_agent_versions_status ON agent_versions(status);

-- 에이전트 성능
CREATE TABLE IF NOT EXISTS agent_performance (
    id VARCHAR PRIMARY KEY,
    agent_id VARCHAR NOT NULL,
    agent_version INTEGER NOT NULL,
    session_id VARCHAR NOT NULL,
    attention_avg DOUBLE,
    attention_min DOUBLE,
    token_used INTEGER,
    compact_count INTEGER,
    completion_time_seconds INTEGER,
    outcome VARCHAR,
    quality_score DOUBLE,
    feedback VARCHAR,
    improvement_suggestions VARCHAR,
    created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_performance_agent ON agent_performance(agent_id, agent_version);
CREATE INDEX IF NOT EXISTS idx_agent_performance_session ON agent_performance(session_id);

-- ============================================================
-- 메시지 패싱
-- ============================================================

CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR PRIMARY KEY,
    conversation_id VARCHAR NOT NULL,
    from_session VARCHAR NOT NULL,
    to_session VARCHAR,
    type VARCHAR NOT NULL,
    subtype VARCHAR,
    payload VARCHAR NOT NULL,
    attention_score DOUBLE,
    context_snapshot VARCHAR,
    token_count INTEGER,
    cumulative_tokens INTEGER,
    status VARCHAR DEFAULT 'pending',
    port_id VARCHAR,
    priority INTEGER DEFAULT 5,
    created_at TIMESTAMP DEFAULT now(),
    processed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_to_session ON messages(to_session, status);
CREATE INDEX IF NOT EXISTS idx_messages_port ON messages(port_id);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);

-- ============================================================
-- Orchestration
-- ============================================================

CREATE TABLE IF NOT EXISTS orchestration_ports (
    id VARCHAR PRIMARY KEY,
    title VARCHAR NOT NULL,
    description VARCHAR,
    atomic_ports VARCHAR NOT NULL,
    status VARCHAR DEFAULT 'pending',
    current_port_id VARCHAR,
    progress_percent INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_orchestration_ports_status ON orchestration_ports(status);

-- 파이프라인
CREATE TABLE IF NOT EXISTS pipelines (
    id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL,
    session_id VARCHAR,
    status VARCHAR DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT now(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_pipelines_status ON pipelines(status);

-- 파이프라인 포트
CREATE TABLE IF NOT EXISTS pipeline_ports (
    pipeline_id VARCHAR NOT NULL,
    port_id VARCHAR NOT NULL,
    group_order INTEGER DEFAULT 0,
    status VARCHAR DEFAULT 'pending',
    PRIMARY KEY (pipeline_id, port_id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_ports_group ON pipeline_ports(pipeline_id, group_order);

-- Worker 세션
CREATE TABLE IF NOT EXISTS worker_sessions (
    id VARCHAR PRIMARY KEY,
    orchestration_id VARCHAR,
    port_id VARCHAR NOT NULL,
    worker_type VARCHAR NOT NULL,
    impl_session_id VARCHAR,
    test_session_id VARCHAR,
    status VARCHAR DEFAULT 'pending',
    substatus VARCHAR,
    result VARCHAR,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_worker_sessions_port ON worker_sessions(port_id);
CREATE INDEX IF NOT EXISTS idx_worker_sessions_orch ON worker_sessions(orchestration_id);
CREATE INDEX IF NOT EXISTS idx_worker_sessions_status ON worker_sessions(status);

-- Build 세션 산출물
CREATE TABLE IF NOT EXISTS build_outputs (
    id VARCHAR PRIMARY KEY,
    build_session_id VARCHAR NOT NULL,
    operator_count INTEGER DEFAULT 0,
    worker_count INTEGER DEFAULT 0,
    test_count INTEGER DEFAULT 0,
    total_ports INTEGER DEFAULT 0,
    port_hierarchy VARCHAR,
    dependency_graph VARCHAR,
    created_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_build_outputs_session ON build_outputs(build_session_id);

-- ============================================================
-- 에스컬레이션
-- ============================================================

CREATE TABLE IF NOT EXISTS escalations (
    id INTEGER PRIMARY KEY,
    from_session VARCHAR,
    from_port VARCHAR,
    to_session VARCHAR,
    issue VARCHAR NOT NULL,
    type VARCHAR,
    severity VARCHAR DEFAULT 'medium',
    context VARCHAR,
    suggestion VARCHAR,
    resolution VARCHAR,
    status VARCHAR DEFAULT 'open',
    created_at TIMESTAMP DEFAULT now(),
    resolved_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_escalations_status ON escalations(status);

-- ============================================================
-- 문서 관리
-- ============================================================

CREATE TABLE IF NOT EXISTS documents (
    id VARCHAR PRIMARY KEY,
    path VARCHAR NOT NULL UNIQUE,
    type VARCHAR,
    domain VARCHAR,
    status VARCHAR DEFAULT 'active',
    priority VARCHAR,
    tokens INTEGER DEFAULT 0,
    summary VARCHAR,
    content_hash VARCHAR,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_documents_type ON documents(type);
CREATE INDEX IF NOT EXISTS idx_documents_domain ON documents(domain);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_path ON documents(path);

-- 문서 태그
CREATE TABLE IF NOT EXISTS document_tags (
    document_id VARCHAR NOT NULL,
    tag VARCHAR NOT NULL,
    PRIMARY KEY (document_id, tag)
);

CREATE INDEX IF NOT EXISTS idx_document_tags_tag ON document_tags(tag);

-- 문서 링크
CREATE TABLE IF NOT EXISTS document_links (
    from_id VARCHAR NOT NULL,
    to_id VARCHAR NOT NULL,
    link_type VARCHAR NOT NULL,
    PRIMARY KEY (from_id, to_id)
);

CREATE INDEX IF NOT EXISTS idx_document_links_from ON document_links(from_id);
CREATE INDEX IF NOT EXISTS idx_document_links_to ON document_links(to_id);
CREATE INDEX IF NOT EXISTS idx_document_links_type ON document_links(link_type);

-- ============================================================
-- 코드 마커
-- ============================================================

CREATE TABLE IF NOT EXISTS code_markers (
    id INTEGER PRIMARY KEY,
    port VARCHAR NOT NULL,
    layer VARCHAR,
    domain VARCHAR,
    adapter VARCHAR,
    generated INTEGER DEFAULT 0,
    file_path VARCHAR NOT NULL,
    line INTEGER NOT NULL,
    project_root VARCHAR,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    UNIQUE(file_path, line)
);

CREATE INDEX IF NOT EXISTS idx_markers_port ON code_markers(port);
CREATE INDEX IF NOT EXISTS idx_markers_layer ON code_markers(layer);
CREATE INDEX IF NOT EXISTS idx_markers_domain ON code_markers(domain);
CREATE INDEX IF NOT EXISTS idx_markers_generated ON code_markers(generated);
CREATE INDEX IF NOT EXISTS idx_markers_file ON code_markers(file_path);

-- 마커 의존성
CREATE TABLE IF NOT EXISTS code_marker_deps (
    from_port VARCHAR NOT NULL,
    to_port VARCHAR NOT NULL,
    PRIMARY KEY (from_port, to_port)
);

CREATE INDEX IF NOT EXISTS idx_marker_deps_from ON code_marker_deps(from_port);
CREATE INDEX IF NOT EXISTS idx_marker_deps_to ON code_marker_deps(to_port);

-- 마커-포트 링크
CREATE TABLE IF NOT EXISTS marker_port_links (
    marker_port VARCHAR NOT NULL,
    document_id VARCHAR NOT NULL,
    PRIMARY KEY (marker_port, document_id)
);

CREATE INDEX IF NOT EXISTS idx_marker_port_links_port ON marker_port_links(marker_port);
CREATE INDEX IF NOT EXISTS idx_marker_port_links_doc ON marker_port_links(document_id);

-- ============================================================
-- Manifest
-- ============================================================

CREATE TABLE IF NOT EXISTS file_manifests (
    id INTEGER PRIMARY KEY,
    project_root VARCHAR NOT NULL,
    file_path VARCHAR NOT NULL,
    file_type VARCHAR NOT NULL,
    hash VARCHAR NOT NULL,
    size INTEGER DEFAULT 0,
    mtime TIMESTAMP,
    managed_by VARCHAR DEFAULT 'pal',
    status VARCHAR DEFAULT 'synced',
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now(),
    UNIQUE(project_root, file_path)
);

CREATE INDEX IF NOT EXISTS idx_manifests_project ON file_manifests(project_root);
CREATE INDEX IF NOT EXISTS idx_manifests_status ON file_manifests(project_root, status);
CREATE INDEX IF NOT EXISTS idx_manifests_type ON file_manifests(project_root, file_type);

-- 파일 변경
CREATE TABLE IF NOT EXISTS file_changes (
    id INTEGER PRIMARY KEY,
    project_root VARCHAR NOT NULL,
    file_path VARCHAR NOT NULL,
    change_type VARCHAR NOT NULL,
    old_hash VARCHAR,
    new_hash VARCHAR,
    changed_at TIMESTAMP DEFAULT now(),
    session_id VARCHAR
);

CREATE INDEX IF NOT EXISTS idx_changes_project ON file_changes(project_root, changed_at);
CREATE INDEX IF NOT EXISTS idx_changes_session ON file_changes(session_id);

-- ============================================================
-- 환경
-- ============================================================

CREATE TABLE IF NOT EXISTS environments (
    id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL UNIQUE,
    hostname VARCHAR,
    paths VARCHAR,
    is_current INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT now(),
    last_active TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_environments_current ON environments(is_current);
CREATE INDEX IF NOT EXISTS idx_environments_name ON environments(name);

-- 동기화 메타
CREATE TABLE IF NOT EXISTS sync_meta (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT now()
);

-- 동기화 히스토리
CREATE TABLE IF NOT EXISTS sync_history (
    id INTEGER PRIMARY KEY,
    direction VARCHAR NOT NULL,
    env_id VARCHAR,
    items_count INTEGER DEFAULT 0,
    conflicts_count INTEGER DEFAULT 0,
    synced_at TIMESTAMP DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sync_history_env ON sync_history(env_id);
CREATE INDEX IF NOT EXISTS idx_sync_history_time ON sync_history(synced_at);

-- ============================================================
-- DuckDB 전용: 시퀀스 (AUTOINCREMENT 대체)
-- ============================================================

CREATE SEQUENCE IF NOT EXISTS seq_session_events START 1;
CREATE SEQUENCE IF NOT EXISTS seq_compactions START 1;
CREATE SEQUENCE IF NOT EXISTS seq_escalations START 1;
CREATE SEQUENCE IF NOT EXISTS seq_code_markers START 1;
CREATE SEQUENCE IF NOT EXISTS seq_file_manifests START 1;
CREATE SEQUENCE IF NOT EXISTS seq_file_changes START 1;
CREATE SEQUENCE IF NOT EXISTS seq_sync_history START 1;
`

// DuckDB wraps sql.DB for DuckDB
type DuckDB struct {
	*sql.DB
	path string
}

// OpenDuckDB opens or creates a DuckDB database
func OpenDuckDB(path string) (*DuckDB, error) {
	// 디렉토리 생성
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	// DuckDB 연결
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, fmt.Errorf("DuckDB 열기 실패: %w", err)
	}

	// 연결 테스트
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("DuckDB 연결 실패: %w", err)
	}

	d := &DuckDB{DB: db, path: path}

	// 스키마 초기화
	if err := d.Init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("스키마 초기화 실패: %w", err)
	}

	return d, nil
}

// Init initializes the DuckDB schema
func (d *DuckDB) Init() error {
	// 스키마 적용
	if _, err := d.Exec(duckDBSchema); err != nil {
		return fmt.Errorf("스키마 적용 실패: %w", err)
	}

	// 버전 저장 (DuckDB는 now() 사용)
	_, err := d.Exec(`
		INSERT INTO metadata (key, value, updated_at) 
		VALUES ('schema_version', ?, now())
		ON CONFLICT (key) DO UPDATE SET value = excluded.value, updated_at = now()
	`, duckDBSchemaVersion)
	if err != nil {
		return fmt.Errorf("버전 저장 실패: %w", err)
	}

	return nil
}

// Path returns the database file path
func (d *DuckDB) Path() string {
	return d.path
}

// GetVersion returns current schema version
func (d *DuckDB) GetVersion() (int, error) {
	var version int
	err := d.QueryRow(`SELECT CAST(value AS INTEGER) FROM metadata WHERE key = 'schema_version'`).Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return version, nil
}

// IsDuckDB checks if path is a DuckDB file
func IsDuckDB(path string) bool {
	// DuckDB 파일 시그니처 확인
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	// DuckDB 매직 바이트 확인 (첫 4바이트)
	magic := make([]byte, 4)
	if _, err := f.Read(magic); err != nil {
		return false
	}

	// DuckDB 시그니처: "DUCK" 또는 버전에 따라 다름
	return string(magic) == "DUCK"
}
