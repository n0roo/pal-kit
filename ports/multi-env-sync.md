# Port: multi-env-sync

> PAL Kit 다중 접속환경 완전 동기화 시스템

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | multi-env-sync |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | session-tracking |
| 예상 복잡도 | high |

---

## 배경

여러 근무지(집, 회사 등)에서 PAL Kit을 사용할 때, 작업 내역(세션, 포트, 에스컬레이션 등)이 각 환경에 고립되어 연속성 있는 작업이 어렵다.

### 현재 문제점

1. **경로 종속성**: DB에 절대 경로(`/Users/n0roo/...`)가 저장되어 다른 환경에서 의미 없음
2. **수동 동기화 필요**: DB 파일 복사 시 경로 불일치로 데이터 손상
3. **Claude 데이터 분리**: `jsonl_path`, `transcript_path`는 Claude가 관리하여 환경 간 공유 불가

---

## 목표

환경 프로파일 기반 경로 추상화와 Git 기반 동기화로, 변환 과정 없이 여러 환경에서 작업 내역을 완전 공유한다.

---

## 핵심 설계

### 1. 환경 프로파일 시스템

```yaml
# ~/.pal/environments.yaml (로컬 전용, Git 제외)
version: 1
current: home-mac

environments:
  home-mac:
    id: "env-home-001"
    match:
      hostname: "n0roo-macbook"
      # 또는 path_exists로 감지
    paths:
      workspace: "/Users/n0roo/playground"
      claude_data: "/Users/n0roo/.claude"
      home: "/Users/n0roo"

  office-mac:
    id: "env-office-001"
    match:
      hostname: "company-macbook"
    paths:
      workspace: "/Users/work/projects"
      claude_data: "/Users/work/.claude"
      home: "/Users/work"
```

### 2. 경로 추상화

**저장 시 (절대 → 논리):**
```
/Users/n0roo/playground/CodeSpace/pal-kit
       ↓ 환경 감지 + 변환
$workspace/CodeSpace/pal-kit
```

**읽기 시 (논리 → 절대):**
```
$workspace/CodeSpace/pal-kit
       ↓ 현재 환경 적용
/Users/work/projects/CodeSpace/pal-kit
```

### 3. 데이터 분류

| 분류 | 데이터 | 동기화 | 비고 |
|------|--------|--------|------|
| **Core** | ports, pipelines, port_dependencies | O | 완전 동기화 |
| **Session Meta** | id, title, status, tokens, cost, port_id | O | 메타만 |
| **Session Local** | jsonl_path, transcript_path | X | 환경별 독립 |
| **Escalation** | escalations | O | 완전 동기화 |
| **Project** | projects (논리 경로) | O | 경로 추상화 |
| **Environment** | environments.yaml | X | 로컬 전용 |

### 4. 동기화 아키텍처

```
┌─────────────────────────────────────────────────────────────────┐
│  ~/.pal/                                                         │
│  ├── pal.db                 ← 로컬 캐시 (SQLite)                 │
│  ├── environments.yaml      ← 로컬 전용                          │
│  └── sync/                  ← Git 저장소 (동기화 대상)            │
│      ├── .git/                                                   │
│      ├── manifest.yaml      ← 동기화 메타정보                     │
│      ├── ports.yaml         ← 포트 상태                          │
│      ├── sessions.yaml      ← 세션 메타 (로컬 경로 제외)          │
│      ├── escalations.yaml                                        │
│      ├── pipelines.yaml                                          │
│      └── projects.yaml      ← 프로젝트 (논리 경로)                │
└─────────────────────────────────────────────────────────────────┘
```

---

## 범위

### 포함

- 환경 프로파일 관리 (`environments` 테이블, YAML)
- 경로 추상화 유틸리티 (논리 ↔ 절대 변환)
- 데이터 Export/Import (YAML 형식)
- Git 기반 동기화 (`pal sync push/pull`)
- 환경 자동 감지 (hostname, path_exists)
- 충돌 감지 및 알림

### 제외

- 원격 DB 직접 연결 (PostgreSQL, Turso 등)
- 실시간 동기화
- Claude 세션 파일(jsonl) 자체 동기화
- 자동 충돌 해결 (수동 선택)

---

## 작업 항목

### Phase 1: 환경 프로파일 시스템 (완료)

- [x] `environments` 테이블 스키마 추가 (DB v7)
- [x] `~/.pal/environments.yaml` 관리
- [x] 환경 자동 감지 로직 (hostname, path_exists)
- [x] `pal env` 명령어
  - [x] `pal env setup` - 현재 환경 등록
  - [x] `pal env list` - 환경 목록
  - [x] `pal env switch <name>` - 환경 전환
  - [x] `pal env current` - 현재 환경 표시
  - [x] `pal env detect` - 환경 자동 감지
  - [x] `pal env delete` - 환경 삭제

### Phase 2: 경로 추상화 (완료)

- [x] `internal/path/resolver.go` 구현
  - [x] `ToLogical(absPath) string` - 절대 → 논리
  - [x] `ToAbsolute(logicalPath) string` - 논리 → 절대
  - [x] `IsResolvable(logicalPath) bool` - 해석 가능 여부
  - [x] `Analyze(path)` - 경로 분석
  - [x] `BatchToLogical/BatchToAbsolute` - 배치 변환
- [x] `pal path` CLI 명령어
  - [x] `pal path to-logical` - 절대 → 논리
  - [x] `pal path to-absolute` - 논리 → 절대
  - [x] `pal path analyze` - 경로 분석
  - [x] `pal path migrate` - 기존 데이터 마이그레이션
- [x] 기존 데이터 마이그레이션 완료 (28개 경로)

### Phase 3: Export/Import (완료)

- [x] `internal/sync/types.go` - 동기화 데이터 타입 정의
- [x] `internal/sync/export.go` 구현
  - [x] `ExportPorts()` - 포트 및 의존성
  - [x] `ExportSessions()` - 세션 (로컬 경로 필드 제외)
  - [x] `ExportEscalations()`
  - [x] `ExportPipelines()` - 파이프라인 및 포트
  - [x] `ExportProjects()` - 논리 경로 사용
  - [x] `ExportToYAML()`, `ExportToFile()`
- [x] `internal/sync/import.go` 구현
  - [x] `ImportFromFile()`, `ImportFromYAML()`
  - [x] 중복/충돌 감지
  - [x] Merge 전략 (last_write_wins, keep_local, keep_remote, manual)
  - [x] Dry-run 모드
- [x] `pal sync` CLI 명령어
  - [x] `pal sync export` - YAML 내보내기
  - [x] `pal sync import` - YAML 가져오기
  - [x] `pal sync status` - 동기화 상태

### Phase 4: Git 동기화 (완료)

- [x] `~/.pal/sync/` Git 저장소 초기화
- [x] `pal sync init <remote>` - 원격 저장소 연결
- [x] `pal sync push` - 변경사항 푸시
  - [x] 로컬 DB → YAML export
  - [x] Git add, commit, push
- [x] `pal sync pull` - 변경사항 풀
  - [x] Git pull
  - [x] YAML → 로컬 DB import
  - [x] 충돌 감지 및 알림
- [x] `pal sync status` - 동기화 상태 (Git 정보 포함)

### Phase 5: 충돌 처리 (완료)

- [x] 충돌 감지 로직
  - [x] 같은 ID, 다른 데이터 비교
  - [x] 환경별 수정 시간 추적
  - [x] 필드별 차이점 분석
- [x] 충돌 해결 UI
  - [x] `pal sync conflicts` - 충돌 목록 확인
  - [x] `pal sync resolve` - 수동 해결
  - [x] `pal sync diff` - 환경별 버전 비교
- [x] Merge 전략
  - [x] `keep_local` - 로컬 데이터 유지
  - [x] `keep_remote` - 원격 데이터 사용
  - [x] `skip` - 충돌 건너뛰기

---

## DB 스키마 변경

```sql
-- v7: 환경 프로파일 지원

-- 환경 테이블
CREATE TABLE environments (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    hostname TEXT,
    config JSON,  -- paths 등
    is_current INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_active DATETIME
);

-- 동기화 메타
CREATE TABLE sync_meta (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- sessions에 환경 추적 추가
ALTER TABLE sessions ADD COLUMN created_env TEXT;
ALTER TABLE sessions ADD COLUMN last_env TEXT;

-- projects에 논리 경로 추가
ALTER TABLE projects ADD COLUMN logical_root TEXT;

-- 동기화 히스토리
CREATE TABLE sync_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    direction TEXT NOT NULL,  -- push, pull
    env_id TEXT,
    items_count INTEGER,
    conflicts_count INTEGER DEFAULT 0,
    synced_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| 환경 관리 | `internal/env/env.go` | 환경 프로파일 CRUD |
| 경로 해석기 | `internal/path/resolver.go` | 논리 ↔ 절대 경로 변환 |
| Export 서비스 | `internal/sync/export.go` | 데이터 내보내기 |
| Import 서비스 | `internal/sync/import.go` | 데이터 가져오기 |
| Git 동기화 | `internal/sync/git.go` | Git 기반 Push/Pull |
| 충돌 처리 | `internal/sync/conflict.go` | 충돌 감지 및 해결 |
| 동기화 타입 | `internal/sync/types.go` | 동기화 데이터 타입 정의 |
| CLI 명령어 | `internal/cli/env.go` | pal env 명령어 |
| CLI 명령어 | `internal/cli/path.go` | pal path 명령어 |
| CLI 명령어 | `internal/cli/sync.go` | pal sync 명령어 |
| 환경 설정 | `~/.pal/environments.yaml` | 로컬 환경 설정 |
| 동기화 저장소 | `~/.pal/sync/` | Git 저장소 |

---

## 사용 시나리오

### 최초 설정 (집 PC)

```bash
# 1. 환경 등록
pal env setup --name home-mac

# 2. 동기화 저장소 초기화
pal sync init git@github.com:user/pal-sync.git

# 3. 현재 데이터 푸시
pal sync push
```

### 다른 환경 설정 (회사 PC)

```bash
# 1. PAL Kit 설치
pal install

# 2. 환경 등록
pal env setup --name office-mac

# 3. 동기화 저장소 연결 및 풀
pal sync init git@github.com:user/pal-sync.git
pal sync pull

# 4. 자동으로 논리 경로 → 현재 환경 경로로 해석
```

### 일상 작업

```bash
# 작업 시작 전
pal sync pull

# 작업 종료 후
pal sync push

# 또는 자동 동기화 (hook 설정 시)
```

---

## 완료 기준

- [x] `pal env setup`으로 환경 등록 가능
- [x] 경로가 논리 경로로 저장되어 환경 간 이동 가능
- [x] `pal sync push/pull`로 Git 기반 동기화 동작
- [x] 다른 환경에서 만든 세션/포트 조회 가능
- [x] 충돌 발생 시 알림 및 수동 해결 가능
- [x] 기존 데이터 마이그레이션 완료

---

## 리스크 및 대응

| 리스크 | 영향 | 대응 |
|--------|------|------|
| Git 충돌 빈번 | 동기화 실패 | Last-write-wins 기본, 수동 해결 옵션 |
| 경로 해석 실패 | 데이터 접근 불가 | Fallback: 원본 경로 유지, 경고 표시 |
| 대용량 데이터 | Git 성능 저하 | 오래된 세션 아카이브, 증분 동기화 |
| 오프라인 작업 | 동기화 지연 | 로컬 우선, 온라인 시 병합 |

---

## 향후 확장

1. **자동 동기화**: session-end hook에서 자동 push
2. **선택적 동기화**: 프로젝트별 동기화 on/off
3. **원격 DB 지원**: Turso (SQLite 클라우드) 연동
4. **팀 협업**: 멀티 유저 동기화 (충돌 해결 고도화)

---

<!-- pal:port:status=complete -->
