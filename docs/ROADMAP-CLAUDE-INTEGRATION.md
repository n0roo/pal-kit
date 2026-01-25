# PAL Kit - Claude 통합 고도화 로드맵

> 작성일: 2026-01-25
> 목표: PAL Kit과 Claude Code/Desktop의 긴밀한 통합

---

## 개요

PAL Kit을 Claude와 더 밀접하게 통합하여, Claude가 프로젝트 컨텍스트를 정확히 이해하고 효율적으로 작업할 수 있도록 개선합니다.

### 개선 우선순위

1. **Hook 시스템 개선** - Claude Code와의 양방향 통신 강화
2. **Context 관리** - 컨텍스트 최적화 및 자동화
3. **에이전트 워크플로우** - 멀티 에이전트 패턴 고도화
4. **Knowledge Base 연동** - 지식 기반 컨텍스트 로딩
5. **데이터 관리** - 백업, 복원, 임포트/엑스포트

---

## 현재 상태 요약

### Hook 시스템 (12개 Hook 구현됨)

| Hook | 역할 | 상태 |
|------|------|------|
| session-start | 세션 시작, 컨텍스트 주입 | ✅ |
| session-end | 세션 종료, usage 수집 | ✅ |
| pre-tool-use | 도구 사용 전 검증 | ✅ |
| post-tool-use | 도구 사용 후 처리 | ✅ |
| port-start/end | 포트 작업 추적 | ✅ |
| subagent | 서브에이전트 연결 | ✅ |
| notification | Compact 복구 | ✅ |

### Context 관리

- CLAUDE.md 자동 주입 (마커 기반)
- Builder 에이전트 자동 활성화
- Session 브리핑 생성
- Workflow Rules 자동 작성

### 에이전트 워크플로우

- 계층적 세션: main → operator → worker
- Worker Pair: Impl + Test 동시 스폰
- Orchestration 기반 의존성 관리

### Knowledge Base

- 기본 구조 정의 (00-System ~ 40-Archive)
- `pal kb init/status` 구현
- 색인/검색 미구현

---

## Phase 1: Hook 시스템 개선

### 목표

Claude Code Hook과 PAL Kit의 통합을 강화하여:
- 더 정확한 컨텍스트 전달
- 실시간 상태 피드백
- 작업 추적 완전성

### 1.1 Hook 응답 최적화

**현재 문제**:
- Hook 출력이 Claude에 항상 전달되지 않음
- stderr 메시지가 무시되는 경우 있음

**개선 방안**:
```go
// hook.go - 응답 구조 개선
type EnhancedHookOutput struct {
    Decision      string            `json:"decision"`       // approve/block
    Reason        string            `json:"reason"`
    Context       map[string]any    `json:"context"`        // 추가 컨텍스트
    Notifications []Notification    `json:"notifications"`  // 알림 목록
    Suggestions   []string          `json:"suggestions"`    // Claude에 제안
}
```

**작업 항목**:
- [ ] Hook 응답에 structured context 추가
- [ ] Claude에 전달되는 suggestions 필드 활용
- [ ] notification hook에서 컨텍스트 복구 정보 개선

### 1.2 세션 식별 강화

**현재 문제**:
- 복수 터미널에서 세션 구분 어려움
- cwd만으로는 충분하지 않음

**개선 방안**:
```go
// session.go - 식별 강화
type SessionIdentifier struct {
    CWD           string    // 작업 디렉토리
    TTY           string    // 터미널 식별자
    ParentPID     int       // 부모 프로세스
    StartTime     time.Time // 시작 시각
    Fingerprint   string    // 복합 해시
}
```

**작업 항목**:
- [ ] TTY 정보 수집 (`tty` 명령 또는 환경변수)
- [ ] 세션 fingerprint 생성 로직
- [ ] FindActiveSession 개선

### 1.3 포트 추적 강제화

**현재 문제**:
- Edit/Write 시 포트 없으면 경고만 (강제 아님)
- 추적되지 않는 작업 발생

**개선 방안**:
```
PreToolUse Flow:
  Edit/Write 감지
    ↓
  활성 포트 확인
    ├─ 있음 → approve
    └─ 없음
        ├─ 설정: strict → block + 포트 생성 안내
        └─ 설정: warn → approve + stderr 경고
```

**작업 항목**:
- [ ] 프로젝트별 strict/warn 모드 설정
- [ ] 포트 없을 때 자동 생성 제안
- [ ] untracked_edit 이벤트 강화

### 1.4 Hook 이벤트 확장

**새로운 이벤트 타입**:
- `context_loaded`: 컨텍스트 로드 완료
- `agent_switched`: 에이전트 전환
- `dependency_resolved`: 의존성 해결
- `quality_check`: 코드 품질 검사 결과

**작업 항목**:
- [ ] 이벤트 타입 스키마 확장
- [ ] SSE 이벤트 발행 연동
- [ ] GUI에서 실시간 표시

---

## Phase 2: Context 관리

### 목표

Claude가 항상 최적의 컨텍스트를 갖도록:
- 토큰 예산 내 최대 정보
- 중복 제거
- 우선순위 기반 로딩

### 2.1 컨텍스트 예산 관리

**현재 문제**:
- 컨텍스트 크기 제한 없음
- 불필요한 정보 중복

**개선 방안**:
```yaml
# .pal/config.yaml
context:
  token_budget: 15000      # 최대 토큰
  priorities:
    - type: port_spec      # 현재 작업 포트
      weight: 1.0
    - type: convention     # 컨벤션
      weight: 0.8
    - type: related_docs   # 관련 문서
      weight: 0.5
```

**작업 항목**:
- [ ] 토큰 카운팅 로직 (tiktoken 또는 근사)
- [ ] 우선순위 기반 컨텍스트 선택
- [ ] 예산 초과 시 자동 trim

### 2.2 CLAUDE.md 자동 업데이트 개선

**현재 구조**:
```markdown
<!-- pal:context:start -->
> 마지막 업데이트: ...
### 활성 세션
### 포트 현황
### 진행 중인 작업
<!-- pal:context:end -->
```

**개선 방안**:
```markdown
<!-- pal:context:start -->
## PAL Kit 컨텍스트 (자동 생성)

### 현재 작업
- **활성 포트**: `user-auth` - 사용자 인증 구현
- **세션**: main (#abc123), 시작: 10분 전
- **진행률**: 3/5 작업 완료

### 컨텍스트 로드됨
- [x] Port 명세: user-auth.md (2.1K tokens)
- [x] Convention: go-backend.md (1.5K tokens)
- [ ] 관련: jwt-guide.md (사용 가능, 0.8K)

### 최근 변경
- `internal/auth/handler.go` - 5분 전
- `internal/auth/service.go` - 8분 전

### PAL 명령어
```bash
pal port show user-auth    # 포트 상태
pal context reload         # 컨텍스트 리로드
pal kb search "jwt"        # KB 검색
```
<!-- pal:context:end -->
```

**작업 항목**:
- [ ] 컨텍스트 섹션 구조 개선
- [ ] 로드된 문서 목록 표시
- [ ] 최근 변경 파일 추적
- [ ] PAL 명령어 힌트 추가

### 2.3 Rules 파일 동적 생성

**현재 문제**:
- `.claude/rules/` 파일이 정적
- 포트 상태 변화 반영 안 됨

**개선 방안**:
```
Port 활성화 시:
  1. Port 명세 로드 → rules/{port-id}.md 생성
  2. 관련 Convention 로드 → rules/convention-{name}.md
  3. 의존 Port 요약 → rules/dependencies.md

Port 비활성화 시:
  1. rules/{port-id}.md 삭제
  2. 의존성 정리
```

**작업 항목**:
- [ ] Rules 파일 lifecycle 관리
- [ ] Convention 자동 로딩
- [ ] 의존성 요약 생성

### 2.4 Compact 복구 강화

**현재 문제**:
- Compact 후 컨텍스트 손실
- 복구 정보 불충분

**개선 방안**:
```go
// CompactRecovery 구조
type CompactRecovery struct {
    CheckpointID    string          // 체크포인트 ID
    SessionSummary  string          // 세션 요약
    ActivePort      *PortSpec       // 현재 포트 명세
    RecentChanges   []FileChange    // 최근 변경
    PendingTasks    []string        // 남은 작업
    RecoveryPrompt  string          // 복구 프롬프트
}
```

**작업 항목**:
- [ ] 체크포인트 저장 (pre-compact)
- [ ] 복구 프롬프트 템플릿
- [ ] 자동 컨텍스트 리로드

---

## Phase 3: 에이전트 워크플로우

### 목표

멀티 에이전트 패턴을 고도화하여:
- 명확한 역할 분담
- 효율적인 위임
- 추적 가능한 워크플로우

### 3.1 에이전트 역할 재정의

**현재 에이전트**:
- Builder, Planner, Architect, Operator, Worker, Tester

**개선된 역할 구조**:
```
┌─────────────────────────────────────────────────────┐
│                    USER REQUEST                      │
└───────────────────────┬─────────────────────────────┘
                        ▼
┌─────────────────────────────────────────────────────┐
│  SPEC PHASE                                          │
│  ┌─────────┐  ┌──────────┐  ┌─────────────┐        │
│  │ Planner │→ │ Architect│→ │ Spec Writer │        │
│  └─────────┘  └──────────┘  └─────────────┘        │
│       ↓            ↓              ↓                 │
│   분석/분해    기술결정      명세 작성               │
└───────────────────────┬─────────────────────────────┘
                        ▼
┌─────────────────────────────────────────────────────┐
│  EXECUTION PHASE                                     │
│  ┌──────────┐  ┌────────────┐                       │
│  │ Operator │→ │ Workers    │                       │
│  └──────────┘  │ ├─ Impl    │                       │
│       ↓        │ └─ Test    │                       │
│   워커 관리    └────────────┘                       │
└───────────────────────┬─────────────────────────────┘
                        ▼
┌─────────────────────────────────────────────────────┐
│  VALIDATION PHASE                                    │
│  ┌──────────┐  ┌──────────┐  ┌────────┐            │
│  │ Reviewer │→ │ Tester   │→ │ Logger │            │
│  └──────────┘  └──────────┘  └────────┘            │
│       ↓            ↓            ↓                   │
│   코드 리뷰    통합 테스트    기록/문서화            │
└─────────────────────────────────────────────────────┘
```

**작업 항목**:
- [ ] 에이전트 역할 문서 정리
- [ ] Phase 전환 로직 구현
- [ ] 각 에이전트별 rules.md 개선

### 3.2 위임 프로토콜 개선

**현재 문제**:
- 위임 시 컨텍스트 전달 불완전
- 결과 수집 비효율

**개선 방안**:
```go
// DelegationContext
type DelegationContext struct {
    FromAgent     string          // 위임자
    ToAgent       string          // 수임자
    TaskType      string          // 작업 유형
    Objective     string          // 목표 (1줄)
    Context       string          // 필수 컨텍스트
    Constraints   []string        // 제약 조건
    ExpectedOutput string         // 기대 결과물
    TokenBudget   int             // 할당 토큰
}
```

**작업 항목**:
- [ ] 위임 컨텍스트 구조화
- [ ] 결과 수집 표준화
- [ ] 실패 시 에스컬레이션 경로

### 3.3 Worker Pair 개선

**현재 문제**:
- Impl과 Test가 독립적으로 실행
- 테스트 실패 시 피드백 루프 없음

**개선 방안**:
```
Worker Pair Flow:
  Impl Worker → 코드 작성
       ↓
  Test Worker → 테스트 작성/실행
       ↓
  결과 확인
    ├─ Pass → 완료
    └─ Fail → Impl Worker에 피드백 → 재작업
```

**작업 항목**:
- [ ] Worker 간 메시지 패싱 개선
- [ ] 피드백 루프 구현
- [ ] 최대 재시도 횟수 설정

### 3.4 세션 계층 시각화

**작업 항목**:
- [ ] CLI에서 세션 트리 표시 (`pal session tree`)
- [ ] GUI에서 실시간 워크플로우 뷰
- [ ] 각 단계별 진행 상태 추적

---

## Phase 4: Knowledge Base 연동

### 목표

KB를 활용하여 Claude에 더 나은 컨텍스트 제공:
- 관련 문서 자동 로딩
- 태그 기반 검색
- 이전 결정 참조

### 4.1 KB 색인 완성

**작업 항목**:
- [ ] `pal kb index` 구현
  - 제목, 요약, 태그, 별칭 색인
  - SQLite FTS5 활용
- [ ] `pal kb search` 구현
  - 키워드 검색
  - 태그 필터링
  - 토큰 예산 내 결과

### 4.2 컨텍스트 자동 로딩

**연동 포인트**:
```
Port 활성화 시:
  1. Port의 domain 태그 추출
  2. KB에서 관련 문서 검색
  3. 토큰 예산 내 상위 N개 로드
  4. Rules 파일에 참조 추가
```

**작업 항목**:
- [ ] Port → KB 태그 매핑
- [ ] 자동 문서 추천 로직
- [ ] 로드된 문서 캐싱

### 4.3 이전 결정(ADR) 참조

**작업 항목**:
- [ ] ADR 자동 감지 (세션 종료 시)
- [ ] 관련 ADR 검색 및 표시
- [ ] 충돌 검사 (기존 결정과 다른 경우)

### 4.4 프로젝트 동기화

**작업 항목**:
- [ ] `pal kb sync` 구현
  - 프로젝트 ports/ → KB 20-Projects/
  - 프로젝트 decisions/ → KB 20-Projects/decisions/
- [ ] 양방향 vs 단방향 정책 결정
- [ ] 변경 감지 및 증분 동기화

---

## Phase 5: 데이터 관리

### 목표

PAL Kit의 핵심 데이터에 대한 안정적인 관리:
- 백업/복원으로 데이터 손실 방지
- 임포트/엑스포트로 이관 용이성
- 무결성 검사로 데이터 신뢰성

### 5.1 백업 시스템

**대상 데이터**:
```
.pal/
├── pal.db              # SQLite (세션, 포트, 에이전트)
├── analytics/          # DuckDB (통계, 색인)
├── sessions/           # 세션 로그
└── decisions/          # 결정 기록

vault/                  # Knowledge Base
├── .pal-kb/index.db    # KB 색인
└── 문서들...
```

**작업 항목**:
- [ ] `pal backup create` - 전체/선택적 백업
- [ ] `pal backup restore` - 복원 (스키마 마이그레이션 포함)
- [ ] `pal backup list` - 백업 목록
- [ ] 자동 백업 스케줄링 (daily/weekly/on_session_end)

### 5.2 임포트/엑스포트

**엑스포트 형식**:
- JSON, YAML, CSV 지원
- 선택적 필드 내보내기

**작업 항목**:
- [ ] `pal export sessions/ports/agents/kb` - 데이터 내보내기
- [ ] `pal import` - 데이터 가져오기 (병합/덮어쓰기 옵션)

### 5.3 무결성 검사

**작업 항목**:
- [ ] `pal data check` - 전체 무결성 검사
  - SQLite: PRAGMA integrity_check, foreign key 검사
  - Vault: 깨진 링크, 누락 파일
  - DuckDB: 색인 동기화 상태
- [ ] `pal data repair` - 복구 시도

### 5.4 GUI 통합

**작업 항목**:
- [ ] 데이터 관리 페이지 추가
- [ ] 상태 모니터링 (크기, 건수, 마지막 백업)
- [ ] 백업/복원 UI
- [ ] 자동 백업 설정 UI

---

## 구현 일정 (예상)

| Phase | 작업 | 우선순위 | 복잡도 |
|-------|------|----------|--------|
| 1.1 | Hook 응답 최적화 | High | Medium |
| 1.2 | 세션 식별 강화 | High | Low |
| 1.3 | 포트 추적 강제화 | High | Low |
| 1.4 | Hook 이벤트 확장 | Medium | Medium |
| 2.1 | 컨텍스트 예산 관리 | High | High |
| 2.2 | CLAUDE.md 개선 | Medium | Medium |
| 2.3 | Rules 동적 생성 | High | Medium |
| 2.4 | Compact 복구 강화 | High | Medium |
| 3.1 | 에이전트 역할 재정의 | Medium | Low |
| 3.2 | 위임 프로토콜 개선 | Medium | High |
| 3.3 | Worker Pair 개선 | Medium | High |
| 3.4 | 세션 계층 시각화 | Low | Medium |
| 4.1 | KB 색인 완성 | High | High |
| 4.2 | 컨텍스트 자동 로딩 | High | Medium |
| 4.3 | ADR 참조 | Medium | Medium |
| 4.4 | 프로젝트 동기화 | Medium | Medium |
| 5.1 | 백업 시스템 | High | Medium |
| 5.2 | 임포트/엑스포트 | Medium | Medium |
| 5.3 | 무결성 검사 | Medium | Low |
| 5.4 | GUI 데이터 관리 | Low | Medium |

---

## 성공 지표

1. **Hook 시스템**
   - 모든 Edit/Write가 포트와 연결됨 (추적률 100%)
   - 세션 식별 정확도 99%+

2. **Context 관리**
   - 토큰 예산 준수 (15K 이내)
   - Compact 후 컨텍스트 복구 성공률 95%+

3. **에이전트 워크플로우**
   - 위임 성공률 90%+
   - Worker Pair 첫 시도 성공률 80%+

4. **Knowledge Base**
   - 검색 응답 시간 < 100ms
   - 관련 문서 적중률 85%+

5. **데이터 관리**
   - 백업/복원 성공률 100%
   - 스키마 마이그레이션 자동 처리
   - 무결성 검사 통과율 99%+

---

## 다음 단계

1. 이 문서를 기반으로 Port 명세 작성
2. Phase 1부터 순차 구현
3. 각 Phase 완료 시 통합 테스트
4. 사용자 피드백 수집 및 개선

---

## 관련 문서

- [ARCHITECTURE.md](./ARCHITECTURE.md) - 전체 아키텍처
- [claude-integration.md](./claude-integration.md) - 기존 통합 문서
- [PAL-KIT-MANUAL.md](./PAL-KIT-MANUAL.md) - 사용 매뉴얼
