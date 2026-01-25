# Phase 2: Context 관리 명세

> Port ID: context-management
> 상태: draft
> 우선순위: high
> 의존성: hook-enhancement

---

## 개요

Claude가 항상 최적의 컨텍스트를 유지할 수 있도록:
- 토큰 예산 내 최대 정보 제공
- 중복 제거 및 우선순위 기반 로딩
- Compact 복구 강화

---

## 현재 상태 분석

### 컨텍스트 흐름

```
Claude Code 시작
    │
    ▼
CLAUDE.md 로드 (자동)
    │
    ├─ <!-- pal:context:start --> ... <!-- pal:context:end -->
    │   └─ 활성 세션, 포트 현황, 진행 중 작업
    │
    ▼
.claude/rules/*.md 로드 (자동)
    │
    ├─ {port-id}.md (활성 포트)
    ├─ workflow.md (워크플로우)
    └─ pal-usage.md (사용 규칙)
    │
    ▼
Session Briefing (.pal/context/session-briefing.md)
    │
    ▼
작업 진행
```

### 현재 구현 파일

| 파일 | 역할 | 라인 수 |
|------|------|---------|
| `internal/context/context.go` | 기본 컨텍스트 서비스 | 367 |
| `internal/context/claude.go` | Claude 통합 | 14,145 |
| `internal/context/injection.go` | CLAUDE.md 주입 | 10,572 |

### 식별된 문제점

1. **토큰 예산 관리 없음**: 컨텍스트 크기 제한 없이 로드
2. **중복 정보**: 같은 내용이 여러 곳에 반복
3. **우선순위 없음**: 모든 문서를 동등하게 로드
4. **Compact 복구 불완전**: 손실된 컨텍스트 복구 어려움

---

## 개선 사항

### 2.1 컨텍스트 예산 관리

**설정 구조**:
```yaml
# .pal/config.yaml
context:
  # 총 토큰 예산
  token_budget: 15000

  # 우선순위별 할당 (합계 100%)
  allocation:
    port_spec: 40      # 현재 작업 포트 명세
    conventions: 25    # 컨벤션/가이드
    recent_changes: 15 # 최근 변경 사항
    related_docs: 15   # 관련 문서
    session_info: 5    # 세션 정보

  # 로딩 전략
  strategy: priority   # priority | fifo | recent

  # 최소 보장
  minimum:
    port_spec: 2000    # 최소 토큰 보장
    conventions: 1000
```

**토큰 카운팅**:
```go
// internal/context/tokens.go

// 토큰 카운터 인터페이스
type TokenCounter interface {
    Count(text string) int
    CountFile(path string) (int, error)
}

// 근사 카운터 (빠름, 영문 기준 4자=1토큰)
type ApproximateCounter struct{}

func (c *ApproximateCounter) Count(text string) int {
    // 영문: ~4 chars/token
    // 한글: ~2 chars/token
    // 코드: ~3 chars/token
    return estimateTokens(text)
}

// tiktoken 기반 정확한 카운터 (느림)
type TiktokenCounter struct {
    encoding *tiktoken.Encoding
}
```

**예산 관리자**:
```go
// internal/context/budget.go

type BudgetManager struct {
    Config     ContextConfig
    Counter    TokenCounter
    Allocation map[string]int // 카테고리별 할당된 토큰
    Used       map[string]int // 카테고리별 사용된 토큰
}

// 컨텍스트 항목 추가
func (bm *BudgetManager) AddItem(category string, content string, priority int) error {
    tokens := bm.Counter.Count(content)
    remaining := bm.Allocation[category] - bm.Used[category]

    if tokens > remaining {
        if bm.Config.Strategy == "priority" {
            // 우선순위 낮은 항목 제거
            bm.trimLowPriority(category, tokens-remaining)
        } else {
            return ErrBudgetExceeded
        }
    }

    bm.Used[category] += tokens
    return nil
}

// 예산 리포트
func (bm *BudgetManager) Report() BudgetReport {
    return BudgetReport{
        Total:      bm.Config.TokenBudget,
        Used:       bm.totalUsed(),
        Remaining:  bm.Config.TokenBudget - bm.totalUsed(),
        ByCategory: bm.Used,
    }
}
```

**변경 파일**:
- `internal/context/tokens.go`: 토큰 카운팅 (신규)
- `internal/context/budget.go`: 예산 관리 (신규)
- `internal/config/config.go`: 설정 추가

---

### 2.2 CLAUDE.md 자동 업데이트 개선

**현재 구조**:
```markdown
<!-- pal:context:start -->
> 마지막 업데이트: 2026-01-25 13:04:19

### 활성 세션
- **2219551e**: -

### 포트 현황
- ⏳ pending: 1
- 🔄 running: 2
- ✅ complete: 12

### 진행 중인 작업
- **knowledge-base**: Knowledge Base 구조 관리

<!-- pal:context:end -->
```

**개선된 구조**:
```markdown
<!-- pal:context:start -->
## PAL Kit 컨텍스트
> 자동 생성됨 | 마지막 업데이트: 2026-01-25 14:30:00

### 현재 작업
| 항목 | 상태 |
|------|------|
| **세션** | main (#abc123) - 활성 30분 |
| **포트** | `user-auth` - 사용자 인증 구현 |
| **진행률** | ████░░░░░░ 40% (2/5 작업) |

### 로드된 컨텍스트
```
📄 Port 명세    user-auth.md          2,100 tokens ✓
📘 Convention   go-backend.md         1,500 tokens ✓
📘 Convention   api-design.md           800 tokens ✓
📚 관련 문서    jwt-guide.md            600 tokens (대기)
───────────────────────────────────────────────────
                                Total: 5,000 / 15,000 tokens
```

### 최근 변경 (10분 이내)
- `internal/auth/handler.go` - 5분 전 (Login 함수 추가)
- `internal/auth/service.go` - 8분 전 (ValidateToken 수정)

### 빠른 명령어
```bash
pal port show user-auth    # 포트 상세
pal context status         # 컨텍스트 상태
pal context reload         # 컨텍스트 리로드
pal kb search "jwt"        # KB 검색
```

### 작업 가이드
> 현재 포트: **user-auth**
>
> 다음 작업: JWT 토큰 검증 로직 구현
> 참고: `30-References/jwt-guide.md` 활용 가능

<!-- pal:context:end -->
```

**구현 구조**:
```go
// internal/context/injection.go

type ContextSection struct {
    CurrentWork    CurrentWorkInfo
    LoadedContext  []LoadedDocument
    RecentChanges  []FileChange
    QuickCommands  []string
    WorkGuide      string
}

type CurrentWorkInfo struct {
    SessionID    string
    SessionAge   time.Duration
    PortID       string
    PortTitle    string
    Progress     int // 0-100
    TasksDone    int
    TasksTotal   int
}

type LoadedDocument struct {
    Type     string // port_spec, convention, related
    Name     string
    Path     string
    Tokens   int
    Loaded   bool
    Priority int
}

func (s *Service) GenerateContextSection() string {
    // 1. 현재 작업 정보 수집
    // 2. 로드된 문서 목록 생성
    // 3. 최근 변경 파일 추적
    // 4. 빠른 명령어 생성
    // 5. 작업 가이드 생성
    // 6. 마크다운 렌더링
}
```

**변경 파일**:
- `internal/context/injection.go`: 섹션 구조 개선
- `internal/context/templates/`: 템플릿 파일들

---

### 2.3 Rules 파일 동적 생성

**현재 문제**:
- Rules 파일이 수동 관리
- 포트 상태 변화 반영 안 됨

**개선된 Lifecycle**:
```
Port 활성화 (pal hook port-start)
    │
    ▼
┌─────────────────────────────────────────┐
│ Rules 파일 생성                          │
│                                          │
│ 1. Port 명세 로드                        │
│    → .claude/rules/{port-id}.md         │
│                                          │
│ 2. 관련 Convention 로드                  │
│    → .claude/rules/conv-{name}.md       │
│                                          │
│ 3. 의존 Port 요약 생성                   │
│    → .claude/rules/dependencies.md      │
│                                          │
│ 4. 토큰 예산 내 관련 문서                │
│    → .claude/rules/related-{n}.md       │
└─────────────────────────────────────────┘
    │
    ▼
작업 진행
    │
    ▼
Port 비활성화 (pal hook port-end)
    │
    ▼
┌─────────────────────────────────────────┐
│ Rules 파일 정리                          │
│                                          │
│ - .claude/rules/{port-id}.md 삭제       │
│ - .claude/rules/conv-*.md 삭제          │
│ - .claude/rules/dependencies.md 갱신    │
└─────────────────────────────────────────┘
```

**Rules 생성 로직**:
```go
// internal/context/rules.go

type RulesManager struct {
    RulesDir      string // .claude/rules/
    BudgetManager *BudgetManager
    KBService     *kb.Service
}

// 포트 활성화 시 Rules 생성
func (rm *RulesManager) ActivatePort(port *Port) error {
    // 1. Port 명세 → rules/{port-id}.md
    portRules := rm.generatePortRules(port)
    rm.writeRules(port.ID+".md", portRules)

    // 2. Convention 로드
    for _, conv := range port.Conventions {
        content, _ := rm.loadConvention(conv)
        rm.writeRules("conv-"+conv+".md", content)
    }

    // 3. 의존성 요약
    if len(port.Dependencies) > 0 {
        depSummary := rm.generateDependencySummary(port.Dependencies)
        rm.writeRules("dependencies.md", depSummary)
    }

    // 4. KB에서 관련 문서 검색
    if rm.KBService != nil {
        related := rm.KBService.SearchByTags(port.Tags, 3)
        for i, doc := range related {
            rm.writeRules(fmt.Sprintf("related-%d.md", i), doc.Summary)
        }
    }

    return nil
}

// 포트 비활성화 시 Rules 정리
func (rm *RulesManager) DeactivatePort(portID string) error {
    // 해당 포트 관련 파일 삭제
    rm.removeRules(portID + ".md")
    rm.removeRulesPattern("conv-*.md")
    // dependencies.md는 다른 활성 포트가 있으면 유지

    return nil
}
```

**Rules 파일 템플릿**:
```markdown
<!-- .claude/rules/{port-id}.md -->
# Port: {port-id}

> 이 파일은 자동 생성됩니다. 직접 수정하지 마세요.
> 생성: {timestamp}

## 작업 목표
{port.title}

## 상세 설명
{port.description}

## 작업 항목
{port.tasks as checklist}

## 기술 결정
{port.decisions}

## 참고 사항
{port.notes}

---
**PAL 명령어**:
- `pal port show {port-id}` - 상태 확인
- `pal hook port-end {port-id}` - 작업 완료
```

**변경 파일**:
- `internal/context/rules.go`: Rules 관리 (신규)
- `internal/cli/hook.go`: port-start/end에서 호출

---

### 2.4 Compact 복구 강화

**현재 문제**:
- Compact 후 컨텍스트 손실
- 복구 정보 불충분
- 체크포인트 없음

**체크포인트 시스템**:
```go
// internal/context/checkpoint.go

type Checkpoint struct {
    ID            string            `json:"id"`
    SessionID     string            `json:"session_id"`
    PortID        string            `json:"port_id,omitempty"`
    CreatedAt     time.Time         `json:"created_at"`
    TokensUsed    int               `json:"tokens_used"`

    // 상태 스냅샷
    ActivePort    *PortSnapshot     `json:"active_port,omitempty"`
    LoadedDocs    []DocSnapshot     `json:"loaded_docs"`
    RecentChanges []FileChange      `json:"recent_changes"`
    PendingTasks  []string          `json:"pending_tasks"`

    // 복구 정보
    RecoveryPrompt string           `json:"recovery_prompt"`
    RecoveryContext string          `json:"recovery_context"`
}

type PortSnapshot struct {
    ID          string   `json:"id"`
    Title       string   `json:"title"`
    Progress    int      `json:"progress"`
    CurrentTask string   `json:"current_task"`
    CompletedTasks []string `json:"completed_tasks"`
}
```

**Compact 흐름**:
```
pre-compact Hook
    │
    ▼
┌─────────────────────────────────────────┐
│ 체크포인트 생성                          │
│                                          │
│ 1. 현재 상태 수집                        │
│    - 활성 포트                           │
│    - 로드된 문서                         │
│    - 최근 변경 파일                      │
│    - 진행 중인 작업                      │
│                                          │
│ 2. 복구 프롬프트 생성                    │
│    "이전 작업 요약 + 다음 단계"          │
│                                          │
│ 3. 체크포인트 저장                       │
│    .pal/checkpoints/{id}.json           │
└─────────────────────────────────────────┘
    │
    ▼
Compact 발생
    │
    ▼
notification Hook (Compact 복구)
    │
    ▼
┌─────────────────────────────────────────┐
│ 컨텍스트 복구                            │
│                                          │
│ 1. 최신 체크포인트 로드                  │
│                                          │
│ 2. 복구 프롬프트 출력                    │
│    "이전에 {작업}을 진행 중이었습니다.  │
│     다음 단계: {task}"                  │
│                                          │
│ 3. Rules 파일 재생성                     │
│                                          │
│ 4. CLAUDE.md 업데이트                   │
└─────────────────────────────────────────┘
```

**복구 프롬프트 템플릿**:
```go
func (c *Checkpoint) GenerateRecoveryPrompt() string {
    return fmt.Sprintf(`## 컨텍스트 복구

### 이전 작업 상태
- **포트**: %s - %s
- **진행률**: %d%%
- **현재 작업**: %s

### 최근 변경 파일
%s

### 다음 단계
%s

### 복구된 컨텍스트
포트 명세와 관련 문서가 다시 로드되었습니다.
\`pal context status\`로 확인할 수 있습니다.

---
작업을 계속하시겠습니까?`,
        c.ActivePort.ID, c.ActivePort.Title,
        c.ActivePort.Progress,
        c.ActivePort.CurrentTask,
        formatRecentChanges(c.RecentChanges),
        formatPendingTasks(c.PendingTasks),
    )
}
```

**변경 파일**:
- `internal/context/checkpoint.go`: 체크포인트 시스템 (신규)
- `internal/cli/hook.go`: pre-compact, notification 개선

---

## CLI 명령어 추가

```bash
# 컨텍스트 상태 확인
pal context status
# 출력:
# Context Budget: 5,000 / 15,000 tokens (33%)
#
# Loaded Documents:
#   📄 user-auth.md (port)        2,100 tokens
#   📘 go-backend.md (conv)       1,500 tokens
#   📘 api-design.md (conv)         800 tokens
#   📚 jwt-guide.md (related)       600 tokens (pending)
#
# Active Port: user-auth (40% complete)
# Last Checkpoint: 10 minutes ago

# 컨텍스트 리로드
pal context reload [--force]

# 체크포인트 목록
pal context checkpoints

# 체크포인트 복구
pal context restore <checkpoint-id>
```

---

## 구현 순서

```
2.1 컨텍스트 예산 관리 (기반)
  ↓
2.2 CLAUDE.md 자동 업데이트 개선
  ↓
2.3 Rules 파일 동적 생성
  ↓
2.4 Compact 복구 강화
```

---

## 테스트 계획

### 단위 테스트

```go
func TestTokenCounting(t *testing.T) {
    // 영문, 한글, 코드 각각 테스트
}

func TestBudgetManager(t *testing.T) {
    // 예산 초과 시 trim 동작 확인
}

func TestCheckpointRestore(t *testing.T) {
    // 체크포인트 저장/복구 확인
}
```

### 통합 테스트

```bash
# 컨텍스트 예산 테스트
./test_context_budget.sh

# Compact 복구 테스트
./test_compact_recovery.sh
```

---

## 완료 기준

- [ ] 토큰 예산 15K 이내 유지
- [ ] CLAUDE.md에 로드된 문서 목록 표시
- [ ] 포트 활성화 시 Rules 자동 생성
- [ ] Compact 후 95% 이상 컨텍스트 복구
- [ ] `pal context status` 명령어 동작
- [ ] 모든 테스트 통과

---

## 관련 문서

- [ROADMAP-CLAUDE-INTEGRATION.md](../ROADMAP-CLAUDE-INTEGRATION.md)
- [phase1-hook-enhancement.md](./phase1-hook-enhancement.md)
- [internal/context/](../../internal/context/)
