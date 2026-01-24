# Port: hook-checklist-gate

> Hook 체크리스트 게이트 - port-end에서 자동 검증 + Claude 피드백

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | hook-checklist-gate |
| 타입 | atomic |
| 레이어 | L1 (Hook) |
| 상태 | pending |
| 우선순위 | high |
| 의존성 | - |
| 예상 토큰 | 6,000 |

---

## 설계 원칙

**Claude가 체크리스트를 의식하고 작업한다**

```
[Claude: "pal hook port-end <id>" 호출]
    ↓
[PAL Kit: 자동 검증 실행]
    ├─ 빌드 체크
    ├─ 테스트 체크  
    └─ 린트 체크
    ↓
[결과에 따라]
    ├─ 성공 → 포트 완료, Claude에 성공 메시지
    └─ 실패 → 포트 블록, Claude에 실패 상세 + 수정 가이드
```

**핵심: 실패해도 사용자에게 묻지 않음 → Claude가 알아서 수정**

---

## 범위

### 포함

- `port-end` Hook에서 자동 체크리스트 검증
- 빌드/테스트/린트 자동 실행
- 실패 시 Claude에 상세 피드백 (stderr)
- 실패 시 포트 상태를 `blocked`로 유지

### 제외

- 수동 체크리스트 (Claude가 판단)
- 사용자 확인 요청

---

## 작업 항목

### 1. 체크리스트 검증기

- [ ] `internal/checklist/verifier.go` 생성
  ```go
  type VerifyResult struct {
      Passed   bool     `json:"passed"`
      Items    []ItemResult `json:"items"`
      Summary  string   `json:"summary"`
  }
  
  type ItemResult struct {
      ID          string `json:"id"`
      Description string `json:"description"`
      Passed      bool   `json:"passed"`
      Output      string `json:"output"`  // 실행 결과
      ErrorMsg    string `json:"error,omitempty"`
      Duration    time.Duration `json:"duration"`
  }
  
  type Verifier struct {
      projectRoot string
  }
  
  // 자동 검증 실행
  func (v *Verifier) Verify() (*VerifyResult, error) {
      result := &VerifyResult{Passed: true}
      
      // 1. 빌드 체크
      buildResult := v.checkBuild()
      result.Items = append(result.Items, buildResult)
      if !buildResult.Passed {
          result.Passed = false
      }
      
      // 2. 테스트 체크
      testResult := v.checkTest()
      result.Items = append(result.Items, testResult)
      if !testResult.Passed {
          result.Passed = false
      }
      
      // 3. 린트 체크 (경고만, 블록 안함)
      lintResult := v.checkLint()
      result.Items = append(result.Items, lintResult)
      
      return result, nil
  }
  
  func (v *Verifier) checkBuild() ItemResult {
      // Go: go build ./...
      // Node: npm run build
      // 프로젝트 타입 자동 감지
  }
  
  func (v *Verifier) checkTest() ItemResult {
      // Go: go test ./...
      // Node: npm test
  }
  
  func (v *Verifier) checkLint() ItemResult {
      // Go: golangci-lint run (있으면)
      // Node: npm run lint (있으면)
  }
  ```

### 2. port-end Hook 수정

- [ ] `internal/cli/hook.go` - `runHookPortEnd` 수정
  ```go
  func runHookPortEnd(cmd *cobra.Command, args []string) error {
      portID := args[0]
      // ...
      
      // ★ 자동 체크리스트 검증 (새로운 로직)
      if projectRoot != "" {
          verifier := checklist.NewVerifier(projectRoot)
          result, err := verifier.Verify()
          
          if err != nil {
              fmt.Fprintf(os.Stderr, "⚠️  [PAL Kit] 체크리스트 검증 실패: %v\n", err)
          } else if !result.Passed {
              // ★ 실패: 포트를 blocked로 유지, Claude에 피드백
              portSvc.UpdateStatus(portID, "blocked")
              
              fmt.Fprintf(os.Stderr, "\n")
              fmt.Fprintf(os.Stderr, "❌ [PAL Kit] 체크리스트 검증 실패 - 포트 완료 불가\n")
              fmt.Fprintf(os.Stderr, "\n")
              
              for _, item := range result.Items {
                  if !item.Passed {
                      fmt.Fprintf(os.Stderr, "   ❌ %s\n", item.Description)
                      if item.ErrorMsg != "" {
                          // 에러 메시지 첫 5줄만
                          lines := strings.Split(item.ErrorMsg, "\n")
                          for i, line := range lines {
                              if i >= 5 {
                                  fmt.Fprintf(os.Stderr, "      ... (생략)\n")
                                  break
                              }
                              fmt.Fprintf(os.Stderr, "      %s\n", line)
                          }
                      }
                  } else {
                      fmt.Fprintf(os.Stderr, "   ✅ %s\n", item.Description)
                  }
              }
              
              fmt.Fprintf(os.Stderr, "\n")
              fmt.Fprintf(os.Stderr, "💡 위 문제를 수정한 후 다시 port-end를 호출하세요.\n")
              fmt.Fprintf(os.Stderr, "\n")
              
              // JSON 출력 (Claude가 파싱 가능)
              if jsonOut {
                  json.NewEncoder(os.Stdout).Encode(map[string]interface{}{
                      "status":    "blocked",
                      "port":      portID,
                      "checklist": result,
                  })
              }
              
              return fmt.Errorf("체크리스트 검증 실패")
          } else {
              // ★ 성공
              fmt.Fprintf(os.Stderr, "✅ [PAL Kit] 체크리스트 검증 통과\n")
              for _, item := range result.Items {
                  fmt.Fprintf(os.Stderr, "   ✅ %s\n", item.Description)
              }
          }
      }
      
      // 기존 로직 (포트 완료 처리)
      // ...
  }
  ```

### 3. 프로젝트 타입 감지

- [ ] `internal/checklist/detect.go` 생성
  ```go
  type ProjectType string
  
  const (
      ProjectGo     ProjectType = "go"
      ProjectNode   ProjectType = "node"
      ProjectPython ProjectType = "python"
      ProjectUnknown ProjectType = "unknown"
  )
  
  func DetectProjectType(root string) ProjectType {
      if fileExists(filepath.Join(root, "go.mod")) {
          return ProjectGo
      }
      if fileExists(filepath.Join(root, "package.json")) {
          return ProjectNode
      }
      if fileExists(filepath.Join(root, "pyproject.toml")) || 
         fileExists(filepath.Join(root, "requirements.txt")) {
          return ProjectPython
      }
      return ProjectUnknown
  }
  
  func (p ProjectType) BuildCommand() string {
      switch p {
      case ProjectGo:
          return "go build ./..."
      case ProjectNode:
          return "npm run build"
      case ProjectPython:
          return "python -m py_compile"
      default:
          return ""
      }
  }
  
  func (p ProjectType) TestCommand() string {
      switch p {
      case ProjectGo:
          return "go test ./..."
      case ProjectNode:
          return "npm test"
      case ProjectPython:
          return "pytest"
      default:
          return ""
      }
  }
  ```

### 4. Claude 피드백 형식

**성공 시:**
```
✅ [PAL Kit] 체크리스트 검증 통과
   ✅ 빌드 성공
   ✅ 테스트 통과 (15 passed)
   ✅ 린트 경고 없음
```

**실패 시:**
```
❌ [PAL Kit] 체크리스트 검증 실패 - 포트 완료 불가

   ✅ 빌드 성공
   ❌ 테스트 실패
      --- FAIL: TestUserCreate (0.00s)
          user_test.go:42: expected 200, got 400
      FAIL
      ... (생략)
   ✅ 린트 경고 없음

💡 위 문제를 수정한 후 다시 port-end를 호출하세요.
```

---

## 동작 흐름

```
1. Claude가 코드 작성 완료
2. Claude: "pal hook port-end user-entity"
3. PAL Kit: 자동 검증 실행
4a. 성공 → 포트 complete, Claude에 성공 메시지
4b. 실패 → 포트 blocked, Claude에 실패 상세
5b. Claude가 에러 읽고 코드 수정
6b. Claude: "pal hook port-end user-entity" 재시도
7. 반복 → 성공
```

---

## 참조

- `internal/cli/hook.go` - 현재 port-end Hook
- `conventions/agents/workers/_common.md` - Worker 체크리스트 정의

---

<!-- pal:port:hook-checklist-gate -->
