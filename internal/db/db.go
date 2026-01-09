package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

const schemaVersion = 1

const schema = `
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
    CHECK (status IN ('pending', 'running', 'complete', 'failed', 'blocked'))
);

CREATE INDEX IF NOT EXISTS idx_ports_status ON ports(status);

-- 세션 관리
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
    last_compact_at DATETIME,
    CHECK (status IN ('running', 'complete', 'failed', 'cancelled')),
    FOREIGN KEY (port_id) REFERENCES ports(id)
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
    tokens_before INTEGER,
    CHECK (trigger_type IN ('auto', 'manual')),
    FOREIGN KEY (session_id) REFERENCES sessions(id)
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
    resolved_at DATETIME,
    CHECK (status IN ('open', 'resolved', 'dismissed'))
);

CREATE INDEX IF NOT EXISTS idx_escalations_status ON escalations(status);

-- 메타데이터
CREATE TABLE IF NOT EXISTS metadata (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
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
	// 스키마 적용
	if _, err := d.Exec(schema); err != nil {
		return fmt.Errorf("스키마 적용 실패: %w", err)
	}

	// 버전 저장
	_, err := d.Exec(`INSERT OR REPLACE INTO metadata (key, value, updated_at) VALUES ('schema_version', ?, CURRENT_TIMESTAMP)`, schemaVersion)
	if err != nil {
		return fmt.Errorf("버전 저장 실패: %w", err)
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
