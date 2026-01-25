package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const schemaVersion = 11

// 기본 테이블 (v1 호환)
const schemaBase = `
-- Lock 관리
CREATE TABLE IF NOT EXISTS locks (
    resource TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    acquired_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 포트 관리
CREATE TABLE IF NOT EXISTS ports (
    id TEXT PRIMARY KEY,
    title TEXT,
    status TEXT DEFAULT 'pending',
    session_id TEXT,
    file_path TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    cost_usd REAL DEFAULT 0,
    duration_secs INTEGER DEFAULT 0,
    agent_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_ports_status ON ports(status);

-- 세션 관리 (기본)
CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    port_id TEXT,
    title TEXT,
    status TEXT DEFAULT 'running',
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    ended_at DATETIME,
    jsonl_path TEXT,
    input_tokens INTEGER DEFAULT 0,
    output_tokens INTEGER DEFAULT 0,
    cache_read_tokens INTEGER DEFAULT 0,
    cache_create_tokens INTEGER DEFAULT 0,
    cost_usd REAL DEFAULT 0,
    compact_count INTEGER DEFAULT 0,
    last_compact_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_port ON sessions(port_id);
CREATE INDEX IF NOT EXISTS idx_sessions_started ON sessions(started_at);

-- 컴팩션 히스토리
CREATE TABLE IF NOT EXISTS compactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    triggered_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    trigger_type TEXT DEFAULT 'auto',
    context_summary TEXT,
    tokens_before INTEGER
);

CREATE INDEX IF NOT EXISTS idx_compactions_session ON compactions(session_id);

-- 에스컬레이션
CREATE TABLE IF NOT EXISTS escalations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    from_session TEXT,
    from_port TEXT,
    issue TEXT NOT NULL,
    status TEXT DEFAULT 'open',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolved_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_escalations_status ON escalations(status);

-- 메타데이터
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
`

// v2 추가 테이블
const schemaV2 = `
-- 파이프라인
CREATE TABLE IF NOT EXISTS pipelines (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    session_id TEXT,
    status TEXT DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_pipelines_status ON pipelines(status);

-- 파이프라인 포트
CREATE TABLE IF NOT EXISTS pipeline_ports (
    pipeline_id TEXT NOT NULL,
    port_id TEXT NOT NULL,
    group_order INTEGER DEFAULT 0,
    status TEXT DEFAULT 'pending',
    PRIMARY KEY (pipeline_id, port_id)
);

CREATE INDEX IF NOT EXISTS idx_pipeline_ports_group ON pipeline_ports(pipeline_id, group_order);

-- 포트 의존성
CREATE TABLE IF NOT EXISTS port_dependencies (
    port_id TEXT NOT NULL,
    depends_on TEXT NOT NULL,
    PRIMARY KEY (port_id, depends_on)
);
`

// v3 추가 테이블
const schemaV3 = `
-- 세션 이벤트 (히스토리)
CREATE TABLE IF NOT EXISTS session_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    session_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    event_data TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_session_events_session ON session_events(session_id);
CREATE INDEX IF NOT EXISTS idx_session_events_type ON session_events(event_type);
CREATE INDEX IF NOT EXISTS idx_session_events_created ON session_events(created_at);
`

// v4 추가 테이블 (전역 구조)
const schemaV4 = `
-- 등록된 프로젝트
CREATE TABLE IF NOT EXISTS projects (
    root TEXT PRIMARY KEY,
    name TEXT,
    description TEXT,
    last_active DATETIME,
    session_count INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    total_cost REAL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_projects_active ON projects(last_active);
CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);

-- sessions에 프로젝트 인덱스
CREATE INDEX IF NOT EXISTS idx_sessions_project ON sessions(project_root);
`

// v5 추가 테이블 (Manifest 시스템)
const schemaV5 = `
-- 파일 Manifest (프로젝트별 파일 추적)
CREATE TABLE IF NOT EXISTS file_manifests (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_root TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_type TEXT NOT NULL,
    hash TEXT NOT NULL,
    size INTEGER DEFAULT 0,
    mtime DATETIME,
    managed_by TEXT DEFAULT 'pal',
    status TEXT DEFAULT 'synced',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_root, file_path)
);

CREATE INDEX IF NOT EXISTS idx_manifests_project ON file_manifests(project_root);
CREATE INDEX IF NOT EXISTS idx_manifests_status ON file_manifests(project_root, status);
CREATE INDEX IF NOT EXISTS idx_manifests_type ON file_manifests(project_root, file_type);

-- 파일 변경 히스토리 (대시보드용)
CREATE TABLE IF NOT EXISTS file_changes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project_root TEXT NOT NULL,
    file_path TEXT NOT NULL,
    change_type TEXT NOT NULL,
    old_hash TEXT,
    new_hash TEXT,
    changed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    session_id TEXT
);

CREATE INDEX IF NOT EXISTS idx_changes_project ON file_changes(project_root, changed_at);
CREATE INDEX IF NOT EXISTS idx_changes_session ON file_changes(session_id);
`

// v6: 포트 스키마 확장 (세션 추적용)
const schemaV6 = `
-- ports 테이블에 추적 컬럼 추가는 migrate()에서 처리

-- 포트 인덱스 추가
CREATE INDEX IF NOT EXISTS idx_ports_session ON ports(session_id);
`

// v7: 다중 환경 동기화 지원
const schemaV7 = `
-- 환경 프로파일
CREATE TABLE IF NOT EXISTS environments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    hostname TEXT,
    paths JSON,
    is_current INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_active DATETIME
);

CREATE INDEX IF NOT EXISTS idx_environments_current ON environments(is_current);
CREATE INDEX IF NOT EXISTS idx_environments_name ON environments(name);

-- 동기화 메타정보
CREATE TABLE IF NOT EXISTS sync_meta (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 동기화 히스토리
CREATE TABLE IF NOT EXISTS sync_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    direction TEXT NOT NULL,
    env_id TEXT,
    items_count INTEGER DEFAULT 0,
    conflicts_count INTEGER DEFAULT 0,
    synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sync_history_env ON sync_history(env_id);
CREATE INDEX IF NOT EXISTS idx_sync_history_time ON sync_history(synced_at);
`

// v8: 문서 관리 (docs-management)
const schemaV8 = `
-- 문서 인덱스
CREATE TABLE IF NOT EXISTS documents (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    type TEXT,
    domain TEXT,
    status TEXT DEFAULT 'active',
    priority TEXT,
    tokens INTEGER DEFAULT 0,
    summary TEXT,
    content_hash TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_documents_type ON documents(type);
CREATE INDEX IF NOT EXISTS idx_documents_domain ON documents(domain);
CREATE INDEX IF NOT EXISTS idx_documents_status ON documents(status);
CREATE INDEX IF NOT EXISTS idx_documents_path ON documents(path);

-- 문서 태그
CREATE TABLE IF NOT EXISTS document_tags (
    document_id TEXT NOT NULL,
    tag TEXT NOT NULL,
    PRIMARY KEY (document_id, tag),
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_document_tags_tag ON document_tags(tag);

-- 문서 링크 (의존성)
CREATE TABLE IF NOT EXISTS document_links (
    from_id TEXT NOT NULL,
    to_id TEXT NOT NULL,
    link_type TEXT NOT NULL,
    PRIMARY KEY (from_id, to_id),
    FOREIGN KEY (from_id) REFERENCES documents(id) ON DELETE CASCADE,
    FOREIGN KEY (to_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_document_links_from ON document_links(from_id);
CREATE INDEX IF NOT EXISTS idx_document_links_to ON document_links(to_id);
CREATE INDEX IF NOT EXISTS idx_document_links_type ON document_links(link_type);
`

// v9: 코드 마커 인덱싱
const schemaV9 = `
-- 코드 마커
CREATE TABLE IF NOT EXISTS code_markers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    port TEXT NOT NULL,
    layer TEXT,
    domain TEXT,
    adapter TEXT,
    generated INTEGER DEFAULT 0,
    file_path TEXT NOT NULL,
    line INTEGER NOT NULL,
    project_root TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(file_path, line)
);

CREATE INDEX IF NOT EXISTS idx_markers_port ON code_markers(port);
CREATE INDEX IF NOT EXISTS idx_markers_layer ON code_markers(layer);
CREATE INDEX IF NOT EXISTS idx_markers_domain ON code_markers(domain);
CREATE INDEX IF NOT EXISTS idx_markers_generated ON code_markers(generated);
CREATE INDEX IF NOT EXISTS idx_markers_file ON code_markers(file_path);

-- 마커 의존성 (포트 간 의존 관계)
CREATE TABLE IF NOT EXISTS code_marker_deps (
    from_port TEXT NOT NULL,
    to_port TEXT NOT NULL,
    PRIMARY KEY (from_port, to_port)
);

CREATE INDEX IF NOT EXISTS idx_marker_deps_from ON code_marker_deps(from_port);
CREATE INDEX IF NOT EXISTS idx_marker_deps_to ON code_marker_deps(to_port);

-- 마커와 포트 명세 연결
CREATE TABLE IF NOT EXISTS marker_port_links (
    marker_port TEXT NOT NULL,
    document_id TEXT NOT NULL,
    PRIMARY KEY (marker_port, document_id),
    FOREIGN KEY (document_id) REFERENCES documents(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_marker_port_links_port ON marker_port_links(marker_port);
CREATE INDEX IF NOT EXISTS idx_marker_port_links_doc ON marker_port_links(document_id);
`

// v10: PAL Kit v1.0 재설계 - 세션 계층, 에이전트 버전, 메시지 패싱
const schemaV10 = `
-- ============================================================
-- 에이전트 관리 (버전 관리 지원)
-- ============================================================

-- 에이전트 정의
CREATE TABLE IF NOT EXISTS agents (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    type TEXT NOT NULL,                        -- spec, operator, worker, test
    description TEXT,
    capabilities TEXT,                         -- JSON: 에이전트 능력
    current_version INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agents_type ON agents(type);
CREATE INDEX IF NOT EXISTS idx_agents_name ON agents(name);

-- 에이전트 버전 히스토리
CREATE TABLE IF NOT EXISTS agent_versions (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    version INTEGER NOT NULL,
    spec_content TEXT NOT NULL,                -- 전체 에이전트 명세
    spec_hash TEXT,                            -- 내용 해시 (변경 감지)
    change_summary TEXT,                       -- 변경 사항 요약
    change_reason TEXT,                        -- 변경 이유
    avg_attention_score REAL,                  -- 이 버전의 평균 Attention
    avg_completion_rate REAL,
    avg_token_efficiency REAL,
    usage_count INTEGER DEFAULT 0,
    status TEXT DEFAULT 'active',              -- active, deprecated, experimental
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, version)
);

CREATE INDEX IF NOT EXISTS idx_agent_versions_agent ON agent_versions(agent_id, version DESC);
CREATE INDEX IF NOT EXISTS idx_agent_versions_status ON agent_versions(status);

-- 에이전트 성능 기록
CREATE TABLE IF NOT EXISTS agent_performance (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    agent_version INTEGER NOT NULL,
    session_id TEXT NOT NULL,
    attention_avg REAL,
    attention_min REAL,
    token_used INTEGER,
    compact_count INTEGER,
    completion_time_seconds INTEGER,
    outcome TEXT,                              -- success, partial, failed
    quality_score REAL,                        -- 0.0~1.0
    feedback TEXT,
    improvement_suggestions TEXT,              -- JSON
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_agent_performance_agent ON agent_performance(agent_id, agent_version);
CREATE INDEX IF NOT EXISTS idx_agent_performance_session ON agent_performance(session_id);

-- ============================================================
-- 세션 Attention 추적
-- ============================================================

CREATE TABLE IF NOT EXISTS session_attention (
    session_id TEXT PRIMARY KEY,
    port_id TEXT,
    current_context_hash TEXT,
    loaded_tokens INTEGER DEFAULT 0,
    available_tokens INTEGER,
    focus_score REAL DEFAULT 1.0,
    drift_count INTEGER DEFAULT 0,
    last_compaction_at DATETIME,
    loaded_files TEXT,                         -- JSON: 로드된 파일 목록
    loaded_conventions TEXT,                   -- JSON: 주입된 컨벤션
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_session_attention_focus ON session_attention(focus_score);

-- ============================================================
-- Compact 이벤트 상세 추적
-- ============================================================

CREATE TABLE IF NOT EXISTS compact_events (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    trigger_reason TEXT,                       -- token_limit, user_request, auto
    before_tokens INTEGER,
    after_tokens INTEGER,
    preserved_context TEXT,                    -- JSON: 보존된 컨텍스트
    discarded_context TEXT,                    -- JSON: 버려진 컨텍스트
    checkpoint_before TEXT,
    recovery_hint TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_compact_events_session ON compact_events(session_id);
CREATE INDEX IF NOT EXISTS idx_compact_events_time ON compact_events(created_at);

-- ============================================================
-- 메시지 패싱
-- ============================================================

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    conversation_id TEXT NOT NULL,             -- 대화 그룹 (포트 단위)
    from_session TEXT NOT NULL,
    to_session TEXT,                           -- NULL = broadcast
    type TEXT NOT NULL,                        -- request, response, report, escalation
    subtype TEXT,                              -- impl_complete, test_pass, test_fail, blocked
    payload TEXT NOT NULL,                     -- JSON 구조화된 내용
    attention_score REAL,                      -- 0.0~1.0 중요도
    context_snapshot TEXT,                     -- 메시지 시점의 컨텍스트 요약
    token_count INTEGER,                       -- 이 메시지의 토큰 수
    cumulative_tokens INTEGER,                 -- 대화 누적 토큰
    status TEXT DEFAULT 'pending',             -- pending, delivered, processed, expired
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    processed_at DATETIME,
    port_id TEXT,
    priority INTEGER DEFAULT 5                 -- 1(highest)~10(lowest)
);

CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id, created_at);
CREATE INDEX IF NOT EXISTS idx_messages_to_session ON messages(to_session, status);
CREATE INDEX IF NOT EXISTS idx_messages_port ON messages(port_id);
CREATE INDEX IF NOT EXISTS idx_messages_status ON messages(status);

-- ============================================================
-- Worker 세션 관리
-- ============================================================

CREATE TABLE IF NOT EXISTS worker_sessions (
    id TEXT PRIMARY KEY,
    orchestration_id TEXT,
    port_id TEXT NOT NULL,
    worker_type TEXT NOT NULL,                 -- impl, test, impl_test_pair, single
    impl_session_id TEXT,
    test_session_id TEXT,
    status TEXT DEFAULT 'pending',
    substatus TEXT,                            -- coding, building, testing, reviewing
    result TEXT,                               -- JSON: {output, metrics, errors}
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_worker_sessions_port ON worker_sessions(port_id);
CREATE INDEX IF NOT EXISTS idx_worker_sessions_orch ON worker_sessions(orchestration_id);
CREATE INDEX IF NOT EXISTS idx_worker_sessions_status ON worker_sessions(status);

-- ============================================================
-- 포트 Handoff (컨텍스트 전달)
-- ============================================================

CREATE TABLE IF NOT EXISTS port_handoffs (
    id TEXT PRIMARY KEY,
    from_port_id TEXT NOT NULL,
    to_port_id TEXT NOT NULL,
    handoff_type TEXT,                         -- api_contract, file_list, type_def
    content TEXT NOT NULL,                     -- JSON: 구조화된 전달 정보
    token_count INTEGER,
    max_token_budget INTEGER DEFAULT 2000,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_port_handoffs_from ON port_handoffs(from_port_id);
CREATE INDEX IF NOT EXISTS idx_port_handoffs_to ON port_handoffs(to_port_id);

-- ============================================================
-- Build 세션 산출물
-- ============================================================

CREATE TABLE IF NOT EXISTS build_outputs (
    id TEXT PRIMARY KEY,
    build_session_id TEXT NOT NULL,
    operator_count INTEGER DEFAULT 0,
    worker_count INTEGER DEFAULT 0,
    test_count INTEGER DEFAULT 0,
    total_ports INTEGER DEFAULT 0,
    port_hierarchy TEXT,                       -- JSON: 전체 포트 트리
    dependency_graph TEXT,                     -- JSON: 포트 간 의존성
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_build_outputs_session ON build_outputs(build_session_id);

-- ============================================================
-- Orchestration 포트 (관리 명세)
-- ============================================================

CREATE TABLE IF NOT EXISTS orchestration_ports (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    atomic_ports TEXT NOT NULL,                -- JSON: [{port_id, order, depends_on}]
    status TEXT DEFAULT 'pending',             -- pending, running, complete, failed
    current_port_id TEXT,
    progress_percent INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME,
    completed_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_orchestration_ports_status ON orchestration_ports(status);
`

// v11 추가 테이블 (Worker 간 직접 통신)
const schemaV11 = `
-- ============================================================
-- Worker 간 직접 통신 채널
-- ============================================================

CREATE TABLE IF NOT EXISTS direct_channels (
    id TEXT PRIMARY KEY,
    session_a TEXT NOT NULL,                   -- Worker A (보통 Impl Worker)
    session_b TEXT NOT NULL,                   -- Worker B (보통 Test Worker)
    port_id TEXT,                              -- 연관된 포트
    orchestration_id TEXT,                     -- 연관된 Orchestration
    status TEXT DEFAULT 'active',              -- active, closed
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    closed_at DATETIME,
    FOREIGN KEY (session_a) REFERENCES sessions(id),
    FOREIGN KEY (session_b) REFERENCES sessions(id)
);

CREATE INDEX IF NOT EXISTS idx_direct_channels_sessions ON direct_channels(session_a, session_b);
CREATE INDEX IF NOT EXISTS idx_direct_channels_port ON direct_channels(port_id);
CREATE INDEX IF NOT EXISTS idx_direct_channels_status ON direct_channels(status);

-- ============================================================
-- 직접 통신 메시지
-- ============================================================

CREATE TABLE IF NOT EXISTS direct_messages (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    from_session TEXT NOT NULL,
    to_session TEXT NOT NULL,
    message_type TEXT NOT NULL,                -- result, feedback, query, ack
    payload TEXT,                              -- JSON: 메시지 내용
    delivered_at DATETIME,
    processed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (channel_id) REFERENCES direct_channels(id)
);

CREATE INDEX IF NOT EXISTS idx_direct_messages_channel ON direct_messages(channel_id, created_at);
CREATE INDEX IF NOT EXISTS idx_direct_messages_to ON direct_messages(to_session, delivered_at);
CREATE INDEX IF NOT EXISTS idx_direct_messages_type ON direct_messages(message_type);

-- ============================================================
-- 피드백 루프 추적
-- ============================================================

CREATE TABLE IF NOT EXISTS feedback_loops (
    id TEXT PRIMARY KEY,
    channel_id TEXT NOT NULL,
    impl_session TEXT NOT NULL,
    test_session TEXT NOT NULL,
    port_id TEXT,
    max_retries INTEGER DEFAULT 3,
    current_retry INTEGER DEFAULT 0,
    status TEXT DEFAULT 'running',             -- running, success, failed, escalated
    last_feedback_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    FOREIGN KEY (channel_id) REFERENCES direct_channels(id)
);

CREATE INDEX IF NOT EXISTS idx_feedback_loops_status ON feedback_loops(status);
CREATE INDEX IF NOT EXISTS idx_feedback_loops_port ON feedback_loops(port_id);
`

// DB wraps sql.DB with helper methods
type DB struct {
	*sql.DB
	path string
}

// Open opens or creates the database
func Open(path string) (*DB, error) {
	// 디렉토리 생성
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("디렉토리 생성 실패: %w", err)
	}

	db, err := sql.Open("sqlite3", path+"?_foreign_keys=on&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("DB 열기 실패: %w", err)
	}

	// 연결 테스트
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("DB 연결 실패: %w", err)
	}

	d := &DB{DB: db, path: path}

	// 스키마 자동 초기화
	if err := d.Init(); err != nil {
		db.Close()
		return nil, fmt.Errorf("스키마 초기화 실패: %w", err)
	}

	return d, nil
}

// Init initializes the database schema
func (d *DB) Init() error {
	// 1. 기본 스키마 적용 (v1 호환)
	if _, err := d.Exec(schemaBase); err != nil {
		return fmt.Errorf("기본 스키마 적용 실패: %w", err)
	}

	// 2. 마이그레이션 실행
	if err := d.migrate(); err != nil {
		return fmt.Errorf("마이그레이션 실패: %w", err)
	}

	// 3. v2 테이블 적용
	if _, err := d.Exec(schemaV2); err != nil {
		return fmt.Errorf("v2 스키마 적용 실패: %w", err)
	}

	// 4. v3 테이블 적용
	if _, err := d.Exec(schemaV3); err != nil {
		return fmt.Errorf("v3 스키마 적용 실패: %w", err)
	}

	// 5. v4 테이블 적용
	if _, err := d.Exec(schemaV4); err != nil {
		return fmt.Errorf("v4 스키마 적용 실패: %w", err)
	}

	// 6. v5 테이블 적용 (Manifest)
	if _, err := d.Exec(schemaV5); err != nil {
		return fmt.Errorf("v5 스키마 적용 실패: %w", err)
	}

	// 7. v6 적용 (포트 추적)
	if _, err := d.Exec(schemaV6); err != nil {
		return fmt.Errorf("v6 스키마 적용 실패: %w", err)
	}

	// 8. v7 적용 (다중 환경 동기화)
	if _, err := d.Exec(schemaV7); err != nil {
		return fmt.Errorf("v7 스키마 적용 실패: %w", err)
	}

	// 9. v8 적용 (문서 관리)
	if _, err := d.Exec(schemaV8); err != nil {
		return fmt.Errorf("v8 스키마 적용 실패: %w", err)
	}

	// 10. v9 적용 (코드 마커)
	if _, err := d.Exec(schemaV9); err != nil {
		return fmt.Errorf("v9 스키마 적용 실패: %w", err)
	}

	// 11. v10 적용 (PAL Kit v1.0 재설계)
	if _, err := d.Exec(schemaV10); err != nil {
		return fmt.Errorf("v10 스키마 적용 실패: %w", err)
	}

	// 12. v11 적용 (Worker 간 직접 통신)
	if _, err := d.Exec(schemaV11); err != nil {
		return fmt.Errorf("v11 스키마 적용 실패: %w", err)
	}

	// 13. 버전 저장
	_, err := d.Exec(`INSERT OR REPLACE INTO metadata (key, value, updated_at) VALUES ('schema_version', ?, CURRENT_TIMESTAMP)`, schemaVersion)
	if err != nil {
		return fmt.Errorf("버전 저장 실패: %w", err)
	}

	return nil
}

// migrate runs database migrations
func (d *DB) migrate() error {
	currentVersion, _ := d.GetVersion()

	// v1 -> v2: sessions 테이블에 새 컬럼 추가
	if currentVersion < 2 {
		// 컬럼 존재 여부 확인 후 추가
		d.Exec(`ALTER TABLE sessions ADD COLUMN session_type TEXT DEFAULT 'single'`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN parent_session TEXT`)
	}

	// v2 -> v3: 프로젝트 정보 및 Claude 세션 ID 추가
	if currentVersion < 3 {
		d.Exec(`ALTER TABLE sessions ADD COLUMN claude_session_id TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN project_root TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN project_name TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN transcript_path TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN cwd TEXT`)
	}

	// v3 -> v4: projects 테이블은 schemaV4에서 생성
	// 추가 마이그레이션 필요 시 여기에 추가

	// v5 -> v6: ports 테이블에 추적 컬럼 추가
	if currentVersion < 6 {
		d.Exec(`ALTER TABLE ports ADD COLUMN input_tokens INTEGER DEFAULT 0`)
		d.Exec(`ALTER TABLE ports ADD COLUMN output_tokens INTEGER DEFAULT 0`)
		d.Exec(`ALTER TABLE ports ADD COLUMN cost_usd REAL DEFAULT 0`)
		d.Exec(`ALTER TABLE ports ADD COLUMN duration_secs INTEGER DEFAULT 0`)
		d.Exec(`ALTER TABLE ports ADD COLUMN agent_id TEXT`)
	}

	// v6 -> v7: 다중 환경 동기화 지원
	if currentVersion < 7 {
		// sessions에 환경 추적 컬럼 추가
		d.Exec(`ALTER TABLE sessions ADD COLUMN created_env TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN last_env TEXT`)
		// projects에 논리 경로 컬럼 추가
		d.Exec(`ALTER TABLE projects ADD COLUMN logical_root TEXT`)
	}

	// v9 -> v10: 세션 계층 구조 확장
	if currentVersion < 10 {
		// sessions 테이블에 계층 관련 컬럼 추가
		d.Exec(`ALTER TABLE sessions ADD COLUMN parent_id TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN root_id TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN depth INTEGER DEFAULT 0`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN path TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN type TEXT DEFAULT 'single'`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN agent_id TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN agent_version INTEGER`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN substatus TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN attention_score REAL`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN token_budget INTEGER`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN context_snapshot TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN checkpoint_id TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN output_summary TEXT`)
		
		// 인덱스 추가
		d.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_parent ON sessions(parent_id)`)
		d.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_root ON sessions(root_id)`)
		d.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_type ON sessions(type)`)
		d.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_path ON sessions(path)`)

		// ports 테이블에 타입 컬럼 추가
		d.Exec(`ALTER TABLE ports ADD COLUMN port_type TEXT DEFAULT 'atomic'`)
		d.Exec(`CREATE INDEX IF NOT EXISTS idx_ports_type ON ports(port_type)`)

		// port_dependencies 확장
		d.Exec(`ALTER TABLE port_dependencies ADD COLUMN dependency_type TEXT`)
		d.Exec(`ALTER TABLE port_dependencies ADD COLUMN required_outputs TEXT`)
		d.Exec(`ALTER TABLE port_dependencies ADD COLUMN satisfied INTEGER DEFAULT 0`)
		d.Exec(`ALTER TABLE port_dependencies ADD COLUMN satisfied_at DATETIME`)

		// escalations 확장
		d.Exec(`ALTER TABLE escalations ADD COLUMN to_session TEXT`)
		d.Exec(`ALTER TABLE escalations ADD COLUMN type TEXT`)
		d.Exec(`ALTER TABLE escalations ADD COLUMN severity TEXT DEFAULT 'medium'`)
		d.Exec(`ALTER TABLE escalations ADD COLUMN context TEXT`)
		d.Exec(`ALTER TABLE escalations ADD COLUMN suggestion TEXT`)
		d.Exec(`ALTER TABLE escalations ADD COLUMN resolution TEXT`)
	}

	// v10 -> v11: 세션 식별 강화 (Phase 1.2)
	if currentVersion < 11 {
		// sessions 테이블에 세션 식별 컬럼 추가
		d.Exec(`ALTER TABLE sessions ADD COLUMN tty TEXT`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN parent_pid INTEGER`)
		d.Exec(`ALTER TABLE sessions ADD COLUMN fingerprint TEXT`)

		// 인덱스 추가
		d.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_fingerprint ON sessions(fingerprint)`)
		d.Exec(`CREATE INDEX IF NOT EXISTS idx_sessions_claude_id ON sessions(claude_session_id)`)
	}

	return nil
}

// GetVersion returns current schema version
func (d *DB) GetVersion() (int, error) {
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

// Path returns the database file path
func (d *DB) Path() string {
	return d.path
}
