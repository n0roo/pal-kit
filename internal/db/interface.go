package db

import (
	"database/sql"
	"os"
)

// Database is the common interface for SQLite and DuckDB
type Database interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Prepare(query string) (*sql.Stmt, error)
	Close() error
	Path() string
	GetVersion() (int, error)
	GetDB() *sql.DB
}

// Ensure both types implement Database interface
var _ Database = (*DB)(nil)
var _ Database = (*DuckDB)(nil)

// GetDB returns the underlying sql.DB for DB (SQLite)
func (d *DB) GetDB() *sql.DB {
	return d.DB
}

// GetDB returns the underlying sql.DB for DuckDB
func (d *DuckDB) GetDB() *sql.DB {
	return d.DB
}

// DBType represents the database type
type DBType string

const (
	TypeSQLite DBType = "sqlite"
	TypeDuckDB DBType = "duckdb"
)

// OpenAuto opens the appropriate database based on environment or settings
func OpenAuto(basePath string) (Database, DBType, error) {
	dbType := os.Getenv("PAL_DB_TYPE")
	
	// 환경변수로 DuckDB 지정된 경우
	if dbType == "duckdb" {
		duckdbPath := GetDuckDBPath(basePath)
		db, err := OpenDuckDB(duckdbPath)
		if err != nil {
			// DuckDB 실패 시 SQLite 폴백
			sqliteDB, sqliteErr := Open(basePath)
			if sqliteErr != nil {
				return nil, "", err // 원래 DuckDB 에러 반환
			}
			return sqliteDB, TypeSQLite, nil
		}
		return db, TypeDuckDB, nil
	}

	// DuckDB 파일이 있고 SQLite가 없으면 DuckDB 사용
	duckdbPath := GetDuckDBPath(basePath)
	if _, err := os.Stat(duckdbPath); err == nil {
		if _, err := os.Stat(basePath); os.IsNotExist(err) {
			db, err := OpenDuckDB(duckdbPath)
			if err == nil {
				return db, TypeDuckDB, nil
			}
		}
	}

	// 기본: SQLite
	db, err := Open(basePath)
	if err != nil {
		return nil, "", err
	}
	return db, TypeSQLite, nil
}

// MustOpen opens database or panics
func MustOpen(basePath string) Database {
	db, _, err := OpenAuto(basePath)
	if err != nil {
		panic(err)
	}
	return db
}
