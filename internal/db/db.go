package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const schemaVersion = 2

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
    completed_at DATETIME
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

	return &DB{DB: db, path: path}, nil
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

	// 4. 버전 저장
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
