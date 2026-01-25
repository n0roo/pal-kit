# Phase 5: 데이터 관리 명세

> Port ID: data-management
> 상태: draft
> 우선순위: high
> 의존성: -

---

## 개요

PAL Kit의 핵심 데이터(SQLite, DuckDB, Vault)에 대한 백업, 복원, 임포트/엑스포트 기능을 제공합니다.

---

## 현재 상태 분석

### 데이터 저장소 구조

```
.pal/
├── pal.db              # SQLite (OLTP) - 세션, 포트, 에이전트 등
├── pal.db-shm          # SQLite shared memory
├── pal.db-wal          # SQLite write-ahead log
├── analytics/
│   ├── docs-index.json     # DuckDB 문서 색인
│   ├── conventions.json    # DuckDB 컨벤션
│   └── token-history.parquet  # DuckDB 토큰 히스토리
├── sessions/           # 세션 로그 (JSON)
├── decisions/          # 결정 기록
└── context/            # 컨텍스트 캐시

vault/                  # Knowledge Base (Obsidian Vault)
├── .pal-kb/
│   └── index.db        # KB 색인 (SQLite)
├── 00-System/
├── 10-Domains/
├── 20-Projects/
├── 30-References/
└── 40-Archive/
```

### 현재 문제점

1. **백업 기능 부재**: 데이터 손실 위험
2. **이관 어려움**: 프로젝트/머신 간 데이터 이동 불편
3. **버전 관리 불가**: 스키마 변경 시 호환성 문제
4. **선택적 내보내기 없음**: 전체 또는 없음

---

## 개선 사항

### 5.1 백업 시스템

**백업 형식**:
```
backup-2026-01-25T10-30-00.pal.tar.gz
├── manifest.json           # 백업 메타데이터
├── sqlite/
│   └── pal.db              # SQLite 스냅샷
├── duckdb/
│   ├── docs-index.json
│   ├── conventions.json
│   └── token-history.parquet
├── vault/                  # Vault 전체 또는 선택
│   └── ...
└── sessions/               # 세션 로그
    └── ...
```

**manifest.json 스키마**:
```json
{
  "version": "1.0",
  "created_at": "2026-01-25T10:30:00Z",
  "pal_version": "1.0.0",
  "schema_version": 10,
  "project": {
    "name": "my-project",
    "root": "/path/to/project"
  },
  "contents": {
    "sqlite": true,
    "duckdb": true,
    "vault": true,
    "sessions": true
  },
  "stats": {
    "sessions": 150,
    "ports": 45,
    "agents": 12,
    "documents": 230
  },
  "checksum": {
    "sqlite": "sha256:...",
    "vault": "sha256:..."
  }
}
```

**CLI 명령어**:
```bash
# 전체 백업
pal backup create
# → backup-2026-01-25T10-30-00.pal.tar.gz

# 이름 지정
pal backup create --name "before-migration"
# → before-migration.pal.tar.gz

# 선택적 백업
pal backup create --include sqlite,vault --exclude sessions

# 특정 경로로 백업
pal backup create --output /backups/

# 백업 목록
pal backup list
# NAME                              SIZE      DATE
# backup-2026-01-25T10-30-00        45MB      2026-01-25 10:30
# before-migration                   38MB      2026-01-24 15:00

# 백업 정보
pal backup info backup-2026-01-25T10-30-00.pal.tar.gz
```

**구현**:
```go
// internal/backup/backup.go (신규)

type BackupService struct {
    projectRoot string
    palDir      string
    vaultPath   string
}

type BackupOptions struct {
    Name       string
    OutputDir  string
    Include    []string  // sqlite, duckdb, vault, sessions
    Exclude    []string
    Compress   bool
}

type BackupManifest struct {
    Version       string            `json:"version"`
    CreatedAt     time.Time         `json:"created_at"`
    PalVersion    string            `json:"pal_version"`
    SchemaVersion int               `json:"schema_version"`
    Project       ProjectInfo       `json:"project"`
    Contents      map[string]bool   `json:"contents"`
    Stats         BackupStats       `json:"stats"`
    Checksum      map[string]string `json:"checksum"`
}

func (s *BackupService) Create(opts BackupOptions) (*BackupResult, error) {
    // 1. 임시 디렉토리 생성
    tmpDir, err := os.MkdirTemp("", "pal-backup-")
    if err != nil {
        return nil, err
    }
    defer os.RemoveAll(tmpDir)

    manifest := &BackupManifest{
        Version:       "1.0",
        CreatedAt:     time.Now(),
        PalVersion:    version.Version,
        SchemaVersion: db.SchemaVersion,
        Contents:      make(map[string]bool),
    }

    // 2. SQLite 백업 (WAL 체크포인트 후 복사)
    if s.shouldInclude("sqlite", opts) {
        if err := s.backupSQLite(tmpDir); err != nil {
            return nil, err
        }
        manifest.Contents["sqlite"] = true
    }

    // 3. DuckDB 백업
    if s.shouldInclude("duckdb", opts) {
        if err := s.backupDuckDB(tmpDir); err != nil {
            return nil, err
        }
        manifest.Contents["duckdb"] = true
    }

    // 4. Vault 백업
    if s.shouldInclude("vault", opts) {
        if err := s.backupVault(tmpDir); err != nil {
            return nil, err
        }
        manifest.Contents["vault"] = true
    }

    // 5. Sessions 백업
    if s.shouldInclude("sessions", opts) {
        if err := s.backupSessions(tmpDir); err != nil {
            return nil, err
        }
        manifest.Contents["sessions"] = true
    }

    // 6. Manifest 작성
    s.writeManifest(tmpDir, manifest)

    // 7. 압축
    outputPath := s.getOutputPath(opts)
    if err := s.compress(tmpDir, outputPath); err != nil {
        return nil, err
    }

    return &BackupResult{
        Path:     outputPath,
        Manifest: manifest,
    }, nil
}

// SQLite 백업 (WAL 안전 처리)
func (s *BackupService) backupSQLite(tmpDir string) error {
    srcDB := filepath.Join(s.palDir, "pal.db")
    dstDB := filepath.Join(tmpDir, "sqlite", "pal.db")

    os.MkdirAll(filepath.Dir(dstDB), 0755)

    // WAL 체크포인트 실행
    db, err := sql.Open("sqlite3", srcDB)
    if err != nil {
        return err
    }
    defer db.Close()

    _, err = db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
    if err != nil {
        return err
    }

    // 파일 복사
    return copyFile(srcDB, dstDB)
}
```

**변경 파일**:
- `internal/backup/backup.go`: 백업 서비스
- `internal/backup/manifest.go`: 매니페스트 처리
- `internal/cli/backup.go`: CLI 명령어

---

### 5.2 복원 시스템

**CLI 명령어**:
```bash
# 전체 복원
pal backup restore backup-2026-01-25T10-30-00.pal.tar.gz

# 선택적 복원
pal backup restore backup.tar.gz --only sqlite

# 미리보기 (dry-run)
pal backup restore backup.tar.gz --dry-run

# 기존 데이터 유지하면서 병합
pal backup restore backup.tar.gz --merge

# 강제 복원 (확인 없이)
pal backup restore backup.tar.gz --force
```

**구현**:
```go
// internal/backup/restore.go (신규)

type RestoreOptions struct {
    Only    []string  // 특정 항목만 복원
    DryRun  bool      // 미리보기
    Merge   bool      // 병합 모드
    Force   bool      // 강제 실행
}

type RestoreResult struct {
    Restored []string
    Skipped  []string
    Errors   []error
}

func (s *BackupService) Restore(archivePath string, opts RestoreOptions) (*RestoreResult, error) {
    // 1. 아카이브 검증
    manifest, err := s.readManifest(archivePath)
    if err != nil {
        return nil, fmt.Errorf("invalid backup: %w", err)
    }

    // 2. 스키마 버전 확인
    if manifest.SchemaVersion > db.SchemaVersion {
        return nil, fmt.Errorf("backup schema version %d is newer than current %d",
            manifest.SchemaVersion, db.SchemaVersion)
    }

    // 3. Dry-run 모드
    if opts.DryRun {
        return s.previewRestore(manifest, opts), nil
    }

    // 4. 확인 (--force가 없으면)
    if !opts.Force {
        if !s.confirmRestore(manifest) {
            return nil, fmt.Errorf("restore cancelled")
        }
    }

    result := &RestoreResult{}

    // 5. 압축 해제
    tmpDir, err := s.extract(archivePath)
    if err != nil {
        return nil, err
    }
    defer os.RemoveAll(tmpDir)

    // 6. 각 항목 복원
    if s.shouldRestore("sqlite", manifest, opts) {
        if err := s.restoreSQLite(tmpDir, opts.Merge); err != nil {
            result.Errors = append(result.Errors, err)
        } else {
            result.Restored = append(result.Restored, "sqlite")
        }
    }

    if s.shouldRestore("duckdb", manifest, opts) {
        if err := s.restoreDuckDB(tmpDir, opts.Merge); err != nil {
            result.Errors = append(result.Errors, err)
        } else {
            result.Restored = append(result.Restored, "duckdb")
        }
    }

    if s.shouldRestore("vault", manifest, opts) {
        if err := s.restoreVault(tmpDir, opts.Merge); err != nil {
            result.Errors = append(result.Errors, err)
        } else {
            result.Restored = append(result.Restored, "vault")
        }
    }

    return result, nil
}

// 스키마 마이그레이션 (구버전 백업 복원 시)
func (s *BackupService) migrateSchema(fromVersion, toVersion int) error {
    // 버전별 마이그레이션 스크립트 실행
    for v := fromVersion; v < toVersion; v++ {
        migration := migrations[v]
        if err := migration.Up(s.db); err != nil {
            return err
        }
    }
    return nil
}
```

---

### 5.3 임포트/엑스포트

**엑스포트 형식**:
```bash
# 세션 내보내기 (JSON)
pal export sessions --format json --output sessions.json
pal export sessions --format csv --output sessions.csv

# 포트 내보내기
pal export ports --status complete --format json

# 에이전트 내보내기
pal export agents --format yaml

# KB 문서 내보내기
pal export kb --section 10-Domains --format markdown
pal export kb --query "auth" --format json

# 통계 내보내기
pal export stats --format csv
```

**임포트**:
```bash
# 세션 임포트
pal import sessions sessions.json --merge

# 포트 임포트
pal import ports ports.yaml --skip-existing

# 에이전트 임포트 (다른 프로젝트에서)
pal import agents /other/project/.pal/agents.yaml

# KB 문서 임포트
pal import kb documents/ --section 20-Projects
```

**구현**:
```go
// internal/export/export.go (신규)

type ExportService struct {
    sessionSvc *session.Service
    portSvc    *port.Service
    agentSvc   *agent.Service
    kbSvc      *kb.Service
}

type ExportFormat string

const (
    FormatJSON     ExportFormat = "json"
    FormatYAML     ExportFormat = "yaml"
    FormatCSV      ExportFormat = "csv"
    FormatMarkdown ExportFormat = "markdown"
)

type ExportOptions struct {
    Format   ExportFormat
    Output   string
    Filter   map[string]string
    Fields   []string  // 특정 필드만
    Pretty   bool
}

func (s *ExportService) ExportSessions(opts ExportOptions) error {
    sessions, err := s.sessionSvc.List(opts.Filter)
    if err != nil {
        return err
    }

    return s.writeOutput(sessions, opts)
}

func (s *ExportService) ExportPorts(opts ExportOptions) error {
    ports, err := s.portSvc.List(opts.Filter["status"], 0)
    if err != nil {
        return err
    }

    return s.writeOutput(ports, opts)
}

// 범용 출력 작성기
func (s *ExportService) writeOutput(data any, opts ExportOptions) error {
    var content []byte
    var err error

    switch opts.Format {
    case FormatJSON:
        if opts.Pretty {
            content, err = json.MarshalIndent(data, "", "  ")
        } else {
            content, err = json.Marshal(data)
        }
    case FormatYAML:
        content, err = yaml.Marshal(data)
    case FormatCSV:
        content, err = s.toCSV(data)
    }

    if err != nil {
        return err
    }

    if opts.Output == "" || opts.Output == "-" {
        fmt.Println(string(content))
        return nil
    }

    return os.WriteFile(opts.Output, content, 0644)
}
```

```go
// internal/export/import.go (신규)

type ImportOptions struct {
    Merge       bool  // 기존 데이터와 병합
    SkipExisting bool  // 중복 건너뛰기
    DryRun      bool  // 미리보기
}

type ImportResult struct {
    Imported int
    Skipped  int
    Errors   []ImportError
}

func (s *ExportService) ImportSessions(path string, opts ImportOptions) (*ImportResult, error) {
    // 파일 읽기
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    // 형식 자동 감지
    var sessions []session.Session
    if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
        err = yaml.Unmarshal(data, &sessions)
    } else {
        err = json.Unmarshal(data, &sessions)
    }
    if err != nil {
        return nil, err
    }

    result := &ImportResult{}

    for _, s := range sessions {
        // 중복 확인
        existing, _ := s.sessionSvc.Get(s.ID)
        if existing != nil {
            if opts.SkipExisting {
                result.Skipped++
                continue
            }
            if !opts.Merge {
                result.Errors = append(result.Errors, ImportError{
                    ID:     s.ID,
                    Reason: "already exists",
                })
                continue
            }
        }

        if opts.DryRun {
            result.Imported++
            continue
        }

        if err := s.sessionSvc.Create(s); err != nil {
            result.Errors = append(result.Errors, ImportError{
                ID:     s.ID,
                Reason: err.Error(),
            })
        } else {
            result.Imported++
        }
    }

    return result, nil
}
```

---

### 5.4 자동 백업

**설정**:
```yaml
# .pal/config.yaml

backup:
  auto_enabled: true
  schedule: "daily"       # daily, weekly, on_session_end
  retention:
    count: 7              # 최근 7개 유지
    days: 30              # 또는 30일 이내
  location: ".pal/backups"
  include:
    - sqlite
    - sessions
  exclude:
    - vault              # Vault는 별도 관리
```

**구현**:
```go
// internal/backup/scheduler.go (신규)

type BackupScheduler struct {
    service *BackupService
    config  BackupConfig
}

func (s *BackupScheduler) Start(ctx context.Context) error {
    if !s.config.AutoEnabled {
        return nil
    }

    switch s.config.Schedule {
    case "daily":
        return s.runDaily(ctx)
    case "weekly":
        return s.runWeekly(ctx)
    case "on_session_end":
        // Hook에서 트리거
        return nil
    }
    return nil
}

func (s *BackupScheduler) runDaily(ctx context.Context) error {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()

    for {
        select {
        case <-ticker.C:
            s.createAndCleanup()
        case <-ctx.Done():
            return nil
        }
    }
}

func (s *BackupScheduler) createAndCleanup() {
    // 백업 생성
    result, err := s.service.Create(BackupOptions{
        Include: s.config.Include,
        Exclude: s.config.Exclude,
    })
    if err != nil {
        log.Printf("auto backup failed: %v", err)
        return
    }

    log.Printf("auto backup created: %s", result.Path)

    // 오래된 백업 정리
    s.cleanup()
}

func (s *BackupScheduler) cleanup() {
    backups, _ := s.service.List()

    // retention.count 초과분 삭제
    if len(backups) > s.config.Retention.Count {
        for _, b := range backups[s.config.Retention.Count:] {
            os.Remove(b.Path)
        }
    }

    // retention.days 초과분 삭제
    cutoff := time.Now().AddDate(0, 0, -s.config.Retention.Days)
    for _, b := range backups {
        if b.CreatedAt.Before(cutoff) {
            os.Remove(b.Path)
        }
    }
}
```

**Hook 연동** (session-end 시 백업):
```go
// hook.go

func runHookSessionEnd(input HookInput) HookOutput {
    // ... 기존 로직 ...

    // 자동 백업 트리거
    if config.Backup.Schedule == "on_session_end" {
        go backupScheduler.createAndCleanup()
    }

    return output
}
```

---

### 5.5 데이터 무결성 검사

**CLI 명령어**:
```bash
# 전체 검사
pal data check

# 개별 검사
pal data check --sqlite
pal data check --vault
pal data check --duckdb

# 복구 시도
pal data repair --sqlite
```

**구현**:
```go
// internal/backup/integrity.go (신규)

type IntegrityChecker struct {
    palDir    string
    vaultPath string
}

type CheckResult struct {
    Component string
    Status    string   // ok, warning, error
    Issues    []Issue
}

type Issue struct {
    Severity string
    Message  string
    Fix      string   // 수정 방법
}

func (c *IntegrityChecker) CheckSQLite() (*CheckResult, error) {
    result := &CheckResult{Component: "sqlite"}

    db, err := sql.Open("sqlite3", filepath.Join(c.palDir, "pal.db"))
    if err != nil {
        result.Status = "error"
        result.Issues = append(result.Issues, Issue{
            Severity: "critical",
            Message:  "cannot open database",
            Fix:      "pal backup restore <backup>",
        })
        return result, nil
    }
    defer db.Close()

    // PRAGMA integrity_check
    var integrityResult string
    db.QueryRow("PRAGMA integrity_check").Scan(&integrityResult)
    if integrityResult != "ok" {
        result.Status = "error"
        result.Issues = append(result.Issues, Issue{
            Severity: "critical",
            Message:  integrityResult,
            Fix:      "pal data repair --sqlite",
        })
    }

    // 외래키 검사
    rows, _ := db.Query("PRAGMA foreign_key_check")
    for rows.Next() {
        var table, rowid, parent, fkid string
        rows.Scan(&table, &rowid, &parent, &fkid)
        result.Issues = append(result.Issues, Issue{
            Severity: "warning",
            Message:  fmt.Sprintf("foreign key violation: %s.%s -> %s", table, fkid, parent),
        })
    }

    if len(result.Issues) == 0 {
        result.Status = "ok"
    } else if result.Status != "error" {
        result.Status = "warning"
    }

    return result, nil
}

func (c *IntegrityChecker) CheckVault() (*CheckResult, error) {
    result := &CheckResult{Component: "vault"}

    // 1. 디렉토리 구조 확인
    requiredDirs := []string{
        "00-System", "10-Domains", "20-Projects", "30-References", "40-Archive",
    }
    for _, dir := range requiredDirs {
        path := filepath.Join(c.vaultPath, dir)
        if _, err := os.Stat(path); os.IsNotExist(err) {
            result.Issues = append(result.Issues, Issue{
                Severity: "warning",
                Message:  fmt.Sprintf("missing directory: %s", dir),
                Fix:      "pal kb init",
            })
        }
    }

    // 2. 깨진 링크 검사
    brokenLinks := c.findBrokenLinks()
    for _, link := range brokenLinks {
        result.Issues = append(result.Issues, Issue{
            Severity: "warning",
            Message:  fmt.Sprintf("broken link: %s -> %s", link.From, link.To),
            Fix:      fmt.Sprintf("update link in %s", link.From),
        })
    }

    // 3. 색인 동기화 확인
    if !c.isIndexSynced() {
        result.Issues = append(result.Issues, Issue{
            Severity: "info",
            Message:  "index out of sync",
            Fix:      "pal kb index",
        })
    }

    if len(result.Issues) == 0 {
        result.Status = "ok"
    } else {
        result.Status = "warning"
    }

    return result, nil
}

// SQLite 복구
func (c *IntegrityChecker) RepairSQLite() error {
    dbPath := filepath.Join(c.palDir, "pal.db")
    backupPath := dbPath + ".backup"

    // 1. 현재 DB 백업
    copyFile(dbPath, backupPath)

    // 2. 새 DB로 덤프 & 복원
    db, _ := sql.Open("sqlite3", dbPath)
    defer db.Close()

    // .dump 실행
    rows, _ := db.Query("SELECT sql FROM sqlite_master WHERE type='table'")
    // ... 복원 로직

    return nil
}
```

---

### 5.6 GUI 통합

**백업 관리 UI**:
```
┌───────────────────────────────────────────────────────────┐
│ 데이터 관리                                    [검사] [백업] │
├───────────────────────────────────────────────────────────┤
│                                                           │
│  상태                                                     │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ SQLite: ✅ OK (15.2 MB)                             │  │
│  │ DuckDB: ✅ OK (3.4 MB)                              │  │
│  │ Vault:  ⚠️ 2 broken links                           │  │
│  │ Sessions: ✅ OK (156 files)                         │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                           │
│  최근 백업                                                 │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ NAME                    SIZE    DATE        ACTION  │  │
│  │ auto-2026-01-25         45MB    Today 10:30 [복원]  │  │
│  │ before-migration        38MB    Yesterday   [복원]  │  │
│  │ auto-2026-01-23         42MB    2 days ago  [삭제]  │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                           │
│  자동 백업 설정                                            │
│  ┌─────────────────────────────────────────────────────┐  │
│  │ [x] 자동 백업 활성화                                 │  │
│  │ 주기: [매일 ▼]  보관: [7개 ▼] 또는 [30일 ▼]         │  │
│  │ 포함: [x] SQLite [x] Sessions [ ] Vault             │  │
│  └─────────────────────────────────────────────────────┘  │
│                                                           │
│  임포트/엑스포트                                           │
│  [세션 내보내기] [포트 내보내기] [에이전트 내보내기]         │
│  [데이터 임포트...]                                        │
│                                                           │
└───────────────────────────────────────────────────────────┘
```

**컴포넌트**:
```typescript
// electron-gui/src/pages/DataManagement.tsx (신규)

export default function DataManagement() {
  const { status, backups, checkIntegrity, createBackup, restore } = useDataManagement()
  const [autoBackupConfig, setAutoBackupConfig] = useState<AutoBackupConfig>()

  return (
    <div className="h-full flex flex-col p-4">
      <h1 className="text-xl font-semibold mb-4">데이터 관리</h1>

      {/* Status cards */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <StatusCard
          name="SQLite"
          status={status.sqlite}
          size={status.sqliteSize}
        />
        {/* ... */}
      </div>

      {/* Backup list */}
      <BackupList
        backups={backups}
        onRestore={restore}
        onDelete={deleteBackup}
      />

      {/* Auto backup config */}
      <AutoBackupConfig
        config={autoBackupConfig}
        onChange={setAutoBackupConfig}
      />

      {/* Export/Import buttons */}
      <div className="flex gap-2">
        <ExportButton type="sessions" />
        <ExportButton type="ports" />
        <ImportButton />
      </div>
    </div>
  )
}
```

---

## CLI 명령어 요약

```bash
# 백업
pal backup create [--name NAME] [--include X,Y] [--output DIR]
pal backup list
pal backup info <file>
pal backup restore <file> [--only X] [--merge] [--dry-run]

# 엑스포트
pal export sessions [--format json|csv|yaml] [--output FILE]
pal export ports [--status STATUS] [--format FORMAT]
pal export agents [--format yaml]
pal export kb [--section SECTION] [--query QUERY]
pal export stats [--format csv]

# 임포트
pal import sessions <file> [--merge] [--skip-existing]
pal import ports <file>
pal import agents <file>
pal import kb <dir> [--section SECTION]

# 무결성
pal data check [--sqlite] [--vault] [--duckdb]
pal data repair --sqlite
pal data stats
```

---

## 구현 순서

```
5.1 백업 시스템        (기반)
  ↓
5.2 복원 시스템
  ↓
5.3 임포트/엑스포트
  ↓
5.4 자동 백업
  ↓
5.5 무결성 검사
  ↓
5.6 GUI 통합
```

---

## 테스트 계획

### 단위 테스트

```go
// internal/backup/backup_test.go

func TestBackupCreate(t *testing.T) {
    // 전체 백업 생성 확인
}

func TestBackupRestore(t *testing.T) {
    // 복원 후 데이터 일치 확인
}

func TestBackupMerge(t *testing.T) {
    // 병합 모드 동작 확인
}

func TestIntegrityCheck(t *testing.T) {
    // 무결성 검사 정확성
}
```

### 통합 테스트

```bash
# 백업 → 복원 사이클
./test_backup_restore_cycle.sh

# 스키마 마이그레이션
./test_schema_migration.sh

# 자동 백업
./test_auto_backup.sh
```

---

## 완료 기준

- [ ] `pal backup create/restore` 동작
- [ ] 선택적 백업/복원 (sqlite, duckdb, vault, sessions)
- [ ] 스키마 버전 검증 및 마이그레이션
- [ ] `pal export/import` 5가지 타입 지원
- [ ] 자동 백업 스케줄링 동작
- [ ] `pal data check/repair` 동작
- [ ] GUI에서 백업 관리 가능
- [ ] 모든 테스트 통과

---

## 관련 문서

- [ROADMAP-CLAUDE-INTEGRATION.md](../ROADMAP-CLAUDE-INTEGRATION.md)
- [internal/db/schema.go](../../internal/db/schema.go)
- [internal/kb/](../../internal/kb/)
