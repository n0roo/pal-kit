# Phase 4: Knowledge Base 통합 명세

> Port ID: kb-integration
> 상태: draft
> 우선순위: medium
> 의존성: hook-enhancement, context-management

---

## 개요

Knowledge Base(KB)를 Claude 컨텍스트 시스템과 긴밀하게 통합하여 지식 기반 작업 지원을 강화합니다.

---

## 현재 상태 분석

### KB 서비스 구조

```go
// internal/kb/kb.go

type Service struct {
    vaultPath   string
    indexPath   string
    db          *sql.DB
}

// 주요 기능
- Init(): KB 초기화, 디렉토리 구조 생성
- Index(): 문서 색인
- Search(): 검색
- GetTOC(): 목차 조회
```

### 디렉토리 구조

```
vault/
├── 00-System/           # 시스템 문서
├── 10-Domains/          # 도메인 지식
├── 20-Projects/         # 프로젝트 문서
├── 30-References/       # 참조 문서
├── 40-Archive/          # 아카이브
└── .pal-kb/
    ├── index.db         # SQLite 색인
    └── toc-cache.json   # 목차 캐시
```

### 현재 검색 구조

```sql
-- .pal-kb/index.db

CREATE TABLE documents (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    title TEXT,
    section TEXT,
    doc_type TEXT,
    summary TEXT,
    tags TEXT,           -- JSON array
    aliases TEXT,        -- JSON array
    token_count INTEGER,
    created_at DATETIME,
    updated_at DATETIME
);

CREATE VIRTUAL TABLE documents_fts USING fts5(
    title, summary, content, tags
);
```

### 식별된 문제점

1. **컨텍스트 연동 부재**: KB 검색 결과가 Claude 컨텍스트에 자동 주입되지 않음
2. **분류체계 미활용**: 태그/도메인 기반 필터링이 제한적
3. **토큰 예산 미지원**: 검색 결과가 토큰 예산을 고려하지 않음
4. **실시간 업데이트 없음**: 문서 변경 시 색인이 자동 갱신되지 않음
5. **GUI 통합 부족**: KB 브라우징/편집 UI 미흡

---

## 개선 사항

### 4.1 컨텍스트 자동 로딩

**현재**:
```
사용자 요청 → Claude 처리 (KB 무관)
```

**개선**:
```
사용자 요청 → KB 검색 → 관련 문서 로드 → Claude 처리
                │
                └─ 토큰 예산 내에서 최적 문서 선택
```

**구현**:
```go
// internal/kb/context_loader.go (신규)

type ContextLoader struct {
    kb         *Service
    tokenBudget int
}

type LoadRequest struct {
    Query       string            // 검색 쿼리
    Domain      string            // 도메인 필터
    DocTypes    []string          // 문서 타입 필터
    TokenBudget int               // 토큰 예산
    Priority    []string          // 우선 로드할 문서 ID
}

type LoadResult struct {
    Documents   []LoadedDoc       // 로드된 문서
    TotalTokens int               // 사용된 토큰
    Truncated   []string          // 예산 초과로 제외된 문서
}

type LoadedDoc struct {
    ID          string
    Title       string
    Path        string
    Content     string            // 요약 또는 전체
    TokenCount  int
    Relevance   float64           // 관련도 점수
}

func (cl *ContextLoader) Load(req LoadRequest) (*LoadResult, error) {
    result := &LoadResult{}
    remainingBudget := req.TokenBudget

    // 1. 우선 문서 로드
    for _, id := range req.Priority {
        doc, err := cl.kb.GetDocument(id)
        if err != nil {
            continue
        }
        if doc.TokenCount <= remainingBudget {
            result.Documents = append(result.Documents, cl.toLoadedDoc(doc, 1.0))
            remainingBudget -= doc.TokenCount
        } else {
            result.Truncated = append(result.Truncated, id)
        }
    }

    // 2. 검색으로 관련 문서 찾기
    searchResults := cl.kb.Search(SearchRequest{
        Query:    req.Query,
        Domain:   req.Domain,
        DocTypes: req.DocTypes,
        Limit:    20,
    })

    // 3. 관련도 순으로 예산 내 로드
    for _, sr := range searchResults {
        if sr.TokenCount <= remainingBudget {
            result.Documents = append(result.Documents,
                cl.toLoadedDoc(sr, sr.Relevance))
            remainingBudget -= sr.TokenCount
        } else {
            // 요약만 로드
            summary := cl.kb.GetSummary(sr.ID)
            if summary.TokenCount <= remainingBudget {
                result.Documents = append(result.Documents,
                    cl.toSummaryDoc(sr, summary))
                remainingBudget -= summary.TokenCount
            } else {
                result.Truncated = append(result.Truncated, sr.ID)
            }
        }
    }

    result.TotalTokens = req.TokenBudget - remainingBudget
    return result, nil
}
```

**Hook 연동**:
```go
// hook.go session-start에서 호출

func injectKBContext(sessionID string, userRequest string) error {
    loader := kb.NewContextLoader(kbSvc, config.Context.TokenBudget.RelatedDocs)

    // 현재 포트의 도메인 확인
    port := portSvc.GetActive(sessionID)
    domain := ""
    if port != nil {
        domain = port.Domain
    }

    result, err := loader.Load(kb.LoadRequest{
        Query:       userRequest,
        Domain:      domain,
        TokenBudget: config.Context.TokenBudget.RelatedDocs,
    })
    if err != nil {
        return err
    }

    // stderr로 Claude에 전달
    if len(result.Documents) > 0 {
        fmt.Fprintln(os.Stderr, "\n📚 관련 문서:")
        for _, doc := range result.Documents {
            fmt.Fprintf(os.Stderr, "- %s (%d tokens)\n", doc.Title, doc.TokenCount)
        }
        fmt.Fprintln(os.Stderr, "")
    }

    return nil
}
```

**변경 파일**:
- `internal/kb/context_loader.go`: 컨텍스트 로더
- `internal/cli/hook.go`: session-start에서 KB 컨텍스트 로드

---

### 4.2 분류체계 기반 검색

**Taxonomy 스키마**:
```yaml
# vault/_taxonomy/domains.yaml

domains:
  - id: auth
    name: 인증/인가
    description: 인증, 권한, 보안 관련
    aliases: [authentication, authorization, security]
    parent: null

  - id: data
    name: 데이터
    description: 데이터 모델, 저장, 처리
    aliases: [database, storage, model]
    parent: null

  - id: ui
    name: UI/UX
    description: 사용자 인터페이스
    aliases: [frontend, component, design]
    parent: null

# vault/_taxonomy/doc-types.yaml

doc_types:
  - id: port
    name: Port 명세
    template: templates/port.md
    required_fields: [domain, priority]

  - id: adr
    name: Architecture Decision
    template: templates/adr.md
    required_fields: [decision_date, decision_makers]

  - id: concept
    name: 개념 문서
    template: templates/concept.md
    required_fields: [domain]

  - id: guide
    name: 가이드
    template: templates/guide.md
    required_fields: []
```

**분류체계 파서**:
```go
// internal/kb/taxonomy.go (신규)

type Taxonomy struct {
    Domains  []Domain  `yaml:"domains"`
    DocTypes []DocType `yaml:"doc_types"`
    Tags     []Tag     `yaml:"tags"`
}

type Domain struct {
    ID          string   `yaml:"id"`
    Name        string   `yaml:"name"`
    Description string   `yaml:"description"`
    Aliases     []string `yaml:"aliases"`
    Parent      string   `yaml:"parent"`
}

func (s *Service) LoadTaxonomy() (*Taxonomy, error) {
    taxonomy := &Taxonomy{}

    // domains.yaml 로드
    domainsPath := filepath.Join(s.vaultPath, "_taxonomy", "domains.yaml")
    if data, err := os.ReadFile(domainsPath); err == nil {
        yaml.Unmarshal(data, taxonomy)
    }

    // doc-types.yaml 로드
    // tags.yaml 로드

    return taxonomy, nil
}

// 별칭 포함 검색
func (s *Service) SearchWithTaxonomy(query string, domain string) []SearchResult {
    taxonomy := s.LoadTaxonomy()

    // 도메인 별칭 확장
    expandedDomains := []string{domain}
    for _, d := range taxonomy.Domains {
        if d.ID == domain {
            expandedDomains = append(expandedDomains, d.Aliases...)
        }
    }

    // 확장된 도메인으로 검색
    return s.searchWithDomains(query, expandedDomains)
}
```

**변경 파일**:
- `internal/kb/taxonomy.go`: 분류체계 파서
- `internal/kb/service.go`: 분류체계 기반 검색
- `internal/kb/init.go`: 초기화 시 _taxonomy/ 생성

---

### 4.3 실시간 색인 업데이트

**파일 감시**:
```go
// internal/kb/watcher.go (신규)

type Watcher struct {
    kb      *Service
    watcher *fsnotify.Watcher
    debounce time.Duration
}

func (w *Watcher) Start(ctx context.Context) error {
    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        return err
    }
    w.watcher = watcher

    // vault 디렉토리 감시
    err = filepath.Walk(w.kb.vaultPath, func(path string, info os.FileInfo, err error) error {
        if info.IsDir() && !strings.HasPrefix(info.Name(), ".") {
            return watcher.Add(path)
        }
        return nil
    })

    go w.processEvents(ctx)
    return nil
}

func (w *Watcher) processEvents(ctx context.Context) {
    pendingUpdates := make(map[string]time.Time)
    ticker := time.NewTicker(w.debounce)

    for {
        select {
        case event := <-w.watcher.Events:
            if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
                if strings.HasSuffix(event.Name, ".md") {
                    pendingUpdates[event.Name] = time.Now()
                }
            }
            if event.Op&fsnotify.Remove != 0 {
                w.kb.RemoveFromIndex(event.Name)
            }

        case <-ticker.C:
            // Debounce된 업데이트 처리
            now := time.Now()
            for path, t := range pendingUpdates {
                if now.Sub(t) > w.debounce {
                    w.kb.IndexFile(path)
                    delete(pendingUpdates, path)
                }
            }

        case <-ctx.Done():
            return
        }
    }
}
```

**증분 색인**:
```go
// internal/kb/index.go (확장)

func (s *Service) IndexFile(path string) error {
    // 파일 파싱
    doc, err := s.parseDocument(path)
    if err != nil {
        return err
    }

    // 기존 문서 확인
    existing, _ := s.GetDocumentByPath(path)

    if existing != nil {
        // 업데이트
        return s.updateDocument(doc)
    } else {
        // 새로 추가
        return s.insertDocument(doc)
    }
}

func (s *Service) RemoveFromIndex(path string) error {
    return s.db.Exec("DELETE FROM documents WHERE path = ?", path)
}
```

**변경 파일**:
- `internal/kb/watcher.go`: 파일 감시
- `internal/kb/index.go`: 증분 색인
- `internal/server/server.go`: 서버 시작 시 Watcher 실행

---

### 4.4 API 확장

**새로운 엔드포인트**:
```go
// internal/server/api_kb.go (신규)

// KB 상태
GET  /api/v2/kb/status
Response: {
    "initialized": true,
    "vault_path": "/path/to/vault",
    "document_count": 150,
    "last_indexed": "2026-01-25T10:00:00Z",
    "sections": {
        "00-System": 20,
        "10-Domains": 45,
        ...
    }
}

// 컨텍스트 로드 (Claude 연동용)
POST /api/v2/kb/context/load
Request: {
    "query": "인증 시스템",
    "domain": "auth",
    "token_budget": 3000,
    "priority": ["port-auth-001"]
}
Response: {
    "documents": [...],
    "total_tokens": 2800,
    "truncated": []
}

// 분류체계 조회
GET  /api/v2/kb/taxonomy
Response: {
    "domains": [...],
    "doc_types": [...],
    "tags": [...]
}

// 문서 추천
POST /api/v2/kb/recommend
Request: {
    "port_id": "user-service",
    "context": "사용자 인증 구현"
}
Response: {
    "recommendations": [
        {
            "id": "concept-auth",
            "title": "인증 개념",
            "reason": "포트 도메인(auth)과 일치"
        },
        ...
    ]
}

// TOC 갱신
POST /api/v2/kb/toc/:section/generate
Request: {
    "depth": 3,
    "include_summary": true
}
Response: {
    "path": "10-Domains/_toc.md",
    "entries": 25
}
```

**변경 파일**:
- `internal/server/api_kb.go`: KB API 핸들러
- `internal/server/api_v2.go`: 라우트 등록

---

### 4.5 GUI 통합

**KB 페이지 개선**:
```
┌───────────────────────────────────────────────────────────────┐
│ Knowledge Base                              [인덱스] [새로고침] │
├──────────┬────────────────────────────────────────────────────┤
│ Sections │  TOC / Search Results          │  Document View    │
│          │                                │                   │
│ 00-System│  ▼ 개요                        │  # 인증 가이드    │
│ 10-Domains│   └ 시작하기                   │                   │
│ 20-Projects│ ▼ 도메인                      │  ## 개요          │
│ 30-Refs  │   ├ 인증                       │  인증 시스템의... │
│ 40-Archive│   └ 데이터                     │                   │
│          │                                │  ## 구현          │
│ ───────  │ [검색: auth    🔍]             │  1. JWT 설정      │
│ Taxonomy │                                │  2. 미들웨어      │
│ ───────  │ 검색 결과:                     │                   │
│ 도메인    │ - 인증 가이드 (90%)           │  [편집] [삭제]    │
│ 문서타입  │ - JWT 설정 (85%)              │                   │
│ 태그     │                                │                   │
└──────────┴────────────────────────────────┴───────────────────┘
```

**컴포넌트**:
```typescript
// electron-gui/src/pages/KnowledgeBase.tsx (확장)

// 추가할 기능:
// 1. Taxonomy 필터 사이드바
// 2. 검색 결과 관련도 표시
// 3. 문서 추천 패널
// 4. 인라인 편집

// electron-gui/src/components/kb/KBTaxonomyFilter.tsx (신규)
export function KBTaxonomyFilter({
  domains,
  docTypes,
  selectedDomain,
  selectedDocType,
  onFilterChange,
}: KBTaxonomyFilterProps) {
  return (
    <div className="space-y-4">
      <div>
        <h4 className="text-sm font-medium mb-2">도메인</h4>
        {domains.map(d => (
          <button
            key={d.id}
            onClick={() => onFilterChange({ domain: d.id })}
            className={clsx(
              'block w-full text-left px-2 py-1 rounded text-sm',
              selectedDomain === d.id ? 'bg-primary-600/20' : 'hover:bg-dark-700'
            )}
          >
            {d.name}
          </button>
        ))}
      </div>
      {/* 문서 타입, 태그 유사하게 */}
    </div>
  )
}

// electron-gui/src/components/kb/KBRecommendations.tsx (신규)
export function KBRecommendations({ portId }: { portId: string }) {
  const { recommendations, loading } = useKBRecommendations(portId)

  if (loading) return <Spinner />

  return (
    <div className="p-4 bg-dark-800 rounded-lg">
      <h4 className="text-sm font-medium mb-2">📚 추천 문서</h4>
      {recommendations.map(r => (
        <div key={r.id} className="py-2 border-b border-dark-700">
          <a href={`#/kb/${r.id}`} className="text-primary-400 hover:underline">
            {r.title}
          </a>
          <p className="text-xs text-dark-400">{r.reason}</p>
        </div>
      ))}
    </div>
  )
}
```

**변경 파일**:
- `electron-gui/src/pages/KnowledgeBase.tsx`: 페이지 확장
- `electron-gui/src/components/kb/KBTaxonomyFilter.tsx`: 분류 필터
- `electron-gui/src/components/kb/KBRecommendations.tsx`: 추천 패널
- `electron-gui/src/hooks/useKB.ts`: 추천 훅 추가

---

### 4.6 Claude 통합 심화

**MCP Tool 추가**:
```go
// internal/mcp/kb_tools.go (신규)

var kbTools = []mcp.Tool{
    {
        Name:        "kb_search",
        Description: "Knowledge Base에서 문서 검색",
        InputSchema: kbSearchSchema,
        Handler:     handleKBSearch,
    },
    {
        Name:        "kb_load_context",
        Description: "관련 문서를 컨텍스트로 로드",
        InputSchema: kbLoadContextSchema,
        Handler:     handleKBLoadContext,
    },
    {
        Name:        "kb_recommend",
        Description: "현재 작업에 관련된 문서 추천",
        InputSchema: kbRecommendSchema,
        Handler:     handleKBRecommend,
    },
    {
        Name:        "kb_create_doc",
        Description: "새 문서 생성",
        InputSchema: kbCreateDocSchema,
        Handler:     handleKBCreateDoc,
    },
}

func handleKBSearch(input KBSearchInput) (*KBSearchOutput, error) {
    results := kbSvc.Search(SearchRequest{
        Query:    input.Query,
        Domain:   input.Domain,
        DocTypes: input.DocTypes,
        Limit:    input.Limit,
    })

    return &KBSearchOutput{
        Results: results,
        Total:   len(results),
    }, nil
}

func handleKBLoadContext(input KBLoadContextInput) (*KBLoadContextOutput, error) {
    loader := kb.NewContextLoader(kbSvc, input.TokenBudget)
    result, err := loader.Load(kb.LoadRequest{
        Query:       input.Query,
        Domain:      input.Domain,
        TokenBudget: input.TokenBudget,
        Priority:    input.Priority,
    })
    if err != nil {
        return nil, err
    }

    // 로드된 문서 내용 반환
    return &KBLoadContextOutput{
        Documents: result.Documents,
        TotalTokens: result.TotalTokens,
    }, nil
}
```

**Prompt 추가**:
```go
// internal/mcp/kb_prompts.go (신규)

var kbPrompts = []mcp.Prompt{
    {
        Name:        "kb_context",
        Description: "현재 작업에 관련된 KB 문서를 로드합니다",
        Arguments: []mcp.PromptArgument{
            {Name: "query", Description: "검색 쿼리", Required: true},
            {Name: "domain", Description: "도메인 필터", Required: false},
        },
        Handler: func(args map[string]string) string {
            loader := kb.NewContextLoader(kbSvc, 3000)
            result, _ := loader.Load(kb.LoadRequest{
                Query:  args["query"],
                Domain: args["domain"],
            })

            var sb strings.Builder
            sb.WriteString("# 관련 문서\n\n")
            for _, doc := range result.Documents {
                sb.WriteString(fmt.Sprintf("## %s\n\n", doc.Title))
                sb.WriteString(doc.Content)
                sb.WriteString("\n\n---\n\n")
            }
            return sb.String()
        },
    },
}
```

**변경 파일**:
- `internal/mcp/kb_tools.go`: KB MCP 도구
- `internal/mcp/kb_prompts.go`: KB MCP 프롬프트
- `internal/mcp/server.go`: 도구/프롬프트 등록

---

## 구현 순서

```
4.1 컨텍스트 자동 로딩  (핵심)
  ↓
4.2 분류체계 기반 검색
  ↓
4.3 실시간 색인 업데이트
  ↓
4.4 API 확장
  ↓
4.5 GUI 통합
  ↓
4.6 Claude 통합 심화
```

---

## 테스트 계획

### 단위 테스트

```go
// internal/kb/context_loader_test.go

func TestContextLoaderTokenBudget(t *testing.T) {
    // 토큰 예산 내에서 문서 로드 확인
}

func TestContextLoaderPriority(t *testing.T) {
    // 우선순위 문서가 먼저 로드되는지 확인
}

func TestTaxonomySearch(t *testing.T) {
    // 별칭으로 검색 시 정확한 결과 반환
}
```

### 통합 테스트

```bash
# test/integration/kb_test.sh

# 1. 컨텍스트 로드 테스트
./test_context_load.sh

# 2. 실시간 색인 테스트
./test_realtime_index.sh

# 3. MCP 도구 테스트
./test_mcp_kb_tools.sh
```

---

## 완료 기준

- [ ] session-start에서 관련 KB 문서 자동 로드
- [ ] 분류체계(도메인, 문서타입, 태그) 기반 검색
- [ ] 파일 변경 시 자동 색인 갱신
- [ ] API 엔드포인트 6개 추가
- [ ] GUI에서 Taxonomy 필터, 추천 문서 표시
- [ ] MCP 도구 4개, 프롬프트 1개 추가
- [ ] 모든 테스트 통과

---

## 관련 문서

- [ROADMAP-CLAUDE-INTEGRATION.md](../ROADMAP-CLAUDE-INTEGRATION.md)
- [knowledge-base port](../../.claude/rules/knowledge-base.md)
- [internal/kb/](../../internal/kb/)
