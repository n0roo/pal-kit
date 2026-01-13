# Port: docs-management

> pa-retriever 문서 관리 기능 → PAL Kit 이식

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | docs-management |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | session-tracking |
| 예상 복잡도 | high |

---

## 목표

pa-retriever의 문서 관리 기능을 PAL Kit에 통합하여, 포트 명세 작성 및 아키텍처 문서 참조를 원활하게 한다.

---

## pa-retriever 기능 분석

### 현재 pa-retriever 제공 기능

| 기능 | MCP Tool | 용도 |
|------|----------|------|
| 문서 검색 | `search_docs` | type, domain, status, tag 필터링 |
| 포트 조회 | `find_port` | Port 이름/별칭으로 검색 |
| 문서 조회 | `get_doc` | 특정 문서 전체 내용 |
| 아키텍처 | `get_architecture` | PA-Layered 핵심 규칙 |
| 타입별 목록 | `list_by_type` | l1, l2, lm, template 등 |
| 스키마 정보 | `get_index_schema` | 검색 필드/연산자 |
| 통계 | `get_stats` | 문서 통계 |

### 인덱스 스키마 (pa-retriever)

```yaml
builtin_fields:
  - type      # l1, l2, lm, template, guide, spec, ...
  - domain    # 도메인명
  - status    # draft, active, stable, ...
  - priority  # critical, high, medium
  - author
  - tag
  - tokens    # 토큰 수
  - created
  - updated
  - path

operators:
  - AND, OR, NOT
  - ":", "<", ">", "<=", ">="

special:
  - "has:<field>"
  - "no:<field>"
  - "links_to:<id>"
  - "linked_by:<id>"
```

---

## PAL Kit 이식 설계

### 1. 문서 인덱싱 (내부 통합)

pa-retriever는 외부 MCP였지만, PAL Kit은 내부 기능으로 통합.

```
┌─────────────────────────────────────────────┐
│              PAL Kit 문서 관리               │
├─────────────────────────────────────────────┤
│                                              │
│  프로젝트 문서       │  아키텍처 문서          │
│  ├── ports/*.md     │  ├── conventions/     │
│  ├── agents/*.yaml  │  ├── packages/*.yaml  │
│  └── .pal/          │  └── templates/       │
│      ├── sessions/  │                       │
│      └── decisions/ │                       │
│                                              │
├─────────────────────────────────────────────┤
│              내부 인덱스 (SQLite)            │
│  documents 테이블                           │
│  - id, path, type, domain, status          │
│  - tokens, summary, content_hash           │
└─────────────────────────────────────────────┘
```

### 2. CLI 명령어 추가

```bash
# 문서 검색
pal docs search "type:l1 AND domain:order"
pal docs search "tag:pa-layered"

# 포트 조회
pal docs port entity        # Port 명세 찾기
pal docs port --deps entity # 의존성 포함

# 아키텍처 참조
pal docs arch               # PA-Layered 요약
pal docs arch --full        # 전체 문서

# 타입별 목록
pal docs list --type l1
pal docs list --type template

# 통계
pal docs stats
```

### 3. 훅 연동 (컨텍스트 자동 로드)

```
pal hook port-start <id>
  ↓
1. Port 명세 파싱 → type, domain 추출
2. 관련 문서 자동 검색
   - type:l1 AND domain:{domain}
   - conventions/agents/workers/backend/{layer}.md
3. 토큰 예산 내에서 컨텍스트 주입
```

### 4. DB 스키마 확장

```sql
-- documents 테이블 추가
CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    type TEXT,                    -- l1, l2, lm, template, convention, port
    domain TEXT,
    status TEXT DEFAULT 'active',
    priority TEXT,
    tokens INTEGER DEFAULT 0,
    summary TEXT,
    content_hash TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 문서 태그
CREATE TABLE document_tags (
    document_id TEXT,
    tag TEXT,
    PRIMARY KEY (document_id, tag),
    FOREIGN KEY (document_id) REFERENCES documents(id)
);

-- 문서 링크 (의존성)
CREATE TABLE document_links (
    from_id TEXT,
    to_id TEXT,
    link_type TEXT,  -- depends_on, references, implements
    PRIMARY KEY (from_id, to_id),
    FOREIGN KEY (from_id) REFERENCES documents(id),
    FOREIGN KEY (to_id) REFERENCES documents(id)
);
```

---

## 구현 우선순위

### P1: 기본 인덱싱

- [x] documents 테이블 생성 (DB 마이그레이션 v8)
- [x] 문서 스캔 및 인덱싱 (`pal docs index`)
- [x] frontmatter 파싱 (type, domain, status 추출)

### P2: 검색 기능

- [x] `pal docs search` 명령어
- [x] 기본 필터링 (type, domain, status)
- [x] 토큰 기반 결과 제한

### P3: 훅 연동

- [x] port-start 시 관련 문서 자동 검색
- [x] Context Budget 관리 (50K 토큰 예산)
- [x] .claude/rules/ 에 문서 참조 주입

### P4: 고급 기능

- [x] 문서 링크 추적 (links_to, linked_by)
- [x] 태그 기반 검색
- [x] 문서 통계 (`pal docs stats`)

---

## 이식 범위 정의

### 포함

- 로컬 프로젝트 문서 인덱싱
- 검색 및 필터링
- 포트 명세 자동 검색
- 컨벤션 자동 로드

### 제외 (pa-retriever 유지)

- Obsidian Vault 전역 검색
- 외부 프로젝트 문서 참조
- 복잡한 시맨틱 검색

---

## 완료 기준

- [x] `pal docs search` 로 프로젝트 문서 검색 가능
- [x] port-start 시 관련 컨벤션 자동 로드
- [x] 토큰 예산 내 컨텍스트 관리

---

## 파일 변경 목록

| 파일 | 변경 내용 | 상태 |
|------|----------|------|
| `internal/db/db.go` | 스키마 v8 (documents, document_tags, document_links) | ✅ 완료 |
| `internal/document/document.go` | 문서 인덱싱 서비스 | ✅ 완료 |
| `internal/cli/docs.go` | CLI 명령어 (index, search, port, stats) | ✅ 완료 |
| `internal/cli/hook.go` | port-start 문서 컨텍스트 로딩 | ✅ 완료 |
| `internal/rules/rules.go` | AppendToRule 메서드 | ✅ 완료 |

---

<!-- pal:port:status=complete -->
