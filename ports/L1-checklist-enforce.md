# Port: L1-checklist-enforce

> 완료 체크리스트 강제 검증 - 포트 완료 시 품질 게이트

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | L1-checklist-enforce |
| 타입 | atomic |
| 레이어 | L1 (Core) |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | - |
| 예상 토큰 | 6,000 |

---

## 목표

`pal hook port-end` 시 에이전트별 완료 체크리스트를 검증하고, 미충족 시 포트 완료를 블록한다.

---

## 범위

### 포함

- 에이전트별 체크리스트 정의 로딩
- 자동 검증 로직 (빌드, 테스트, 린트)
- Hook에서 강제 검증
- 검증 실패 시 에스컬레이션

### 제외

- 체크리스트 UI (GUI 별도 포트)
- 수동 체크리스트 확인 (기존 유지)

---

## 작업 항목

### 1. 체크리스트 스키마 정의

- [ ] `internal/checklist/types.go` 생성
  ```go
  type ChecklistItem struct {
      ID          string
      Description string
      Type        string  // auto, manual
      Command     string  // 자동 검증 시 실행할 명령
      Required    bool
  }
  
  type Checklist struct {
      AgentType string
      PortType  string
      Items     []ChecklistItem
  }
  
  type CheckResult struct {
      ItemID   string
      Passed   bool
      Message  string
      Duration time.Duration
  }
  ```

### 2. 자동 검증 항목

- [ ] `internal/checklist/verifier.go` 생성
  ```go
  // 자동 검증 가능 항목
  type AutoVerifier struct {
      BuildCheck   func(projectRoot string) (bool, string)
      TestCheck    func(projectRoot string) (bool, string)
      LintCheck    func(projectRoot string) (bool, string)
      ConventionCheck func(files []string) (bool, string)
  }
  
  func (v *AutoVerifier) Verify(items []ChecklistItem) []CheckResult
  ```

### 3. 에이전트별 기본 체크리스트

- [ ] Worker 공통:
  ```yaml
  checklist:
    - id: build
      description: "빌드 성공"
      type: auto
      command: "go build ./... || npm run build"
      required: true
    - id: test
      description: "테스트 통과"
      type: auto
      command: "go test ./... || npm test"
      required: true
    - id: lint
      description: "린트 경고 없음"
      type: auto
      command: "golangci-lint run || npm run lint"
      required: false
    - id: convention
      description: "컨벤션 준수"
      type: manual
      required: true
  ```

### 4. Hook 연동

- [ ] `internal/cli/hook.go` 수정
  ```go
  func handlePortEnd(portID string) error {
      // 1. 포트에 할당된 에이전트 확인
      agent := getPortAgent(portID)
      
      // 2. 체크리스트 로드
      checklist := loadChecklist(agent.Type)
      
      // 3. 자동 검증 실행
      results := verifier.Verify(checklist.Items)
      
      // 4. 필수 항목 실패 시 블록
      for _, r := range results {
          if !r.Passed && r.Required {
              // 에스컬레이션 생성
              createEscalation(portID, r)
              return fmt.Errorf("체크리스트 미충족: %s", r.Message)
          }
      }
      
      // 5. 포트 완료 처리
      return completePort(portID)
  }
  ```

### 5. 에스컬레이션 연동

- [ ] 검증 실패 시 자동 에스컬레이션 생성
  ```go
  type ChecklistEscalation struct {
      PortID      string
      FailedItems []CheckResult
      Suggestion  string
  }
  ```

### 6. CLI 명령어

- [ ] `pal checklist show <agent-type>` - 체크리스트 조회
- [ ] `pal checklist verify <port-id>` - 수동 검증
- [ ] `pal checklist skip <port-id> <item-id>` - 항목 스킵 (사유 필수)

---

## 완료 기준

- [ ] `pal hook port-end` 시 자동 검증 실행
- [ ] 빌드/테스트 실패 시 포트 완료 블록
- [ ] 실패 시 에스컬레이션 자동 생성
- [ ] `pal checklist verify`로 수동 검증 가능
- [ ] 단위 테스트 작성 및 통과

---

## 테스트 시나리오

```bash
# 1. 빌드 실패 상태에서 포트 종료 시도
pal hook port-end test-port
# → "체크리스트 미충족: 빌드 실패" 에러

# 2. 빌드 성공, 테스트 실패
pal hook port-end test-port
# → "체크리스트 미충족: 테스트 실패" 에러

# 3. 모든 항목 통과
pal hook port-end test-port
# → 포트 완료
```

---

## 참조

- `specs/agent-improvement-proposal.md` - 완료 체크리스트 표준화 제안
- `conventions/agents/workers/_common.md` - Worker 공통 체크리스트
- `internal/escalation/escalation.go` - 에스컬레이션 시스템

---

<!-- pal:port:L1-checklist-enforce -->
