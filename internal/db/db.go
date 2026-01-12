package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const schemaVersion = 7

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

	// 9. 버전 저장
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
