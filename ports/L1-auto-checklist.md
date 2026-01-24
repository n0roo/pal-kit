# Port: L1-auto-checklist

> 자동 체크리스트 검증 - pal_port_end 호출 시 자동 실행

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | L1-auto-checklist |
| 타입 | atomic |
| 레이어 | L1 (Core) |
| 상태 | pending |
| 우선순위 | high |
| 의존성 | LM-mcp-tools |
| 예상 토큰 | 6,000 |

---

## 목표

Claude가 `pal_port_end`를 호출하면 **자동으로** 빌드/테스트/린트 검증을 실행하고, 결과를 Claude에 피드백한다.

---

## 동작 흐름

```
Claude: "pal_port_end {id: 'user-entity'}"
    ↓
MCP Server: handlePortEnd()
    ↓
[자동 검증]
├── 빌드 체크 (go build / npm build)
├── 테스트 체크 (go test / npm test)  
└── 린트 체크 (golangci-lint / eslint)
    ↓
[결과에 따라]
├── 모두 통과 → 포트 complete + 성공 응답
└── 실패 있음 → 포트 blocked + 실패 상세 응답
    ↓
Claude: 결과 수신, 필요시 수정
```

---

## 작업 항목

### 1. 체크리스트 검증기

- [ ] `internal/checklist/verifier.go`
  ```go
  type Verifier struct {
      projectRoot string
      projectType ProjectType
  }
  
  type VerifyResult struct {
      Passed  bool         `json:"passed"`
      Items   []ItemResult `json:"items"`
      Summary string       `json:"summary"`
  }
  
  type ItemResult struct {
      ID       string `json:"id"`
      Name     string `json:"name"`
      Passed   bool   `json:"passed"`
      Output   string `json:"output,omitempty"`   // 성공 시 출력
      Error    string `json:"error,omitempty"`    // 실패 시 에러
      Duration int    `json:"duration_ms"`
      Required bool   `json:"required"`           // 필수 여부
  }
  
  func (v *Verifier) Verify() (*VerifyResult, error) {
      result := &VerifyResult{Passed: true}
      
      // 1. 빌드 체크 (필수)
      buildResult := v.checkBuild()
      result.Items = append(result.Items, buildResult)
      if !buildResult.Passed && buildResult.Required {
          result.Passed = false
      }
      
      // 2. 테스트 체크 (필수)
      testResult := v.checkTest()
      result.Items = append(result.Items, testResult)
      if !testResult.Passed && testResult.Required {
          result.Passed = false
      }
      
      // 3. 린트 체크 (권장, 실패해도 블록 안함)
      lintResult := v.checkLint()
      lintResult.Required = false
      result.Items = append(result.Items, lintResult)
      
      // 요약 생성
      result.Summary = v.generateSummary(result)
      
      return result, nil
  }
  ```

### 2. 프로젝트 타입별 명령어

- [ ] `internal/checklist/commands.go`
  ```go
  type ProjectType string
  
  const (
      ProjectGo     ProjectType = "go"
      ProjectNode   ProjectType = "node"
      ProjectPython ProjectType = "python"
  )
  
  func DetectProjectType(root string) ProjectType {
      if fileExists(filepath.Join(root, "go.mod")) {
          return ProjectGo
      }
      if fileExists(filepath.Join(root, "package.json")) {
          return ProjectNode
      }
      if fileExists(filepath.Join(root, "pyproject.toml")) {
          return ProjectPython
      }
      return ""
  }
  
  func (v *Verifier) checkBuild() ItemResult {
      result := ItemResult{ID: "build", Name: "빌드", Required: true}
      
      var cmd string
      switch v.projectType {
      case ProjectGo:
          cmd = "go build ./..."
      case ProjectNode:
          cmd = "npm run build --if-present"
      case ProjectPython:
          cmd = "python -m py_compile *.py"
      }
      
      output, err := runCommand(cmd)
      if err != nil {
          result.Passed = false
          result.Error = truncate(err.Error(), 500)
      } else {
          result.Passed = true
          result.Output = "성공"
      }
      
      return result
  }
  
  func (v *Verifier) checkTest() ItemResult {
      result := ItemResult{ID: "test", Name: "테스트", Required: true}
      
      var cmd string
      switch v.projectType {
      case ProjectGo:
          cmd = "go test ./... -v"
      case ProjectNode:
          cmd = "npm test --if-present"
      case ProjectPython:
          cmd = "pytest -v"
      }
      
      output, err := runCommand(cmd)
      if err != nil {
          result.Passed = false
          result.Error = truncate(output+err.Error(), 500)
      } else {
          result.Passed = true
          // 테스트 결과 파싱
          result.Output = parseTestSummary(output)
      }
      
      return result
  }
  ```

### 3. MCP 도구 연동

- [ ] `internal/mcp/server.go` - `handlePortEnd` 수정
  ```go
  func (s *MCPServer) handlePortEnd(params map[string]interface{}) (interface{}, error) {
      portID := params["id"].(string)
      
      // 체크리스트 검증
      verifier := checklist.NewVerifier(s.projectRoot)
      result, err := verifier.Verify()
      if err != nil {
          return nil, err
      }
      
      response := &PortEndResponse{
          PortID:    portID,
          Checklist: result,
      }
      
      if result.Passed {
          // 성공: 포트 완료
          s.portSvc.UpdateStatus(portID, "complete")
          response.Status = "complete"
          response.Message = "✅ 포트 완료"
      } else {
          // 실패: 포트 블록
          s.portSvc.UpdateStatus(portID, "blocked")
          response.Status = "blocked"
          response.Message = "❌ 체크리스트 검증 실패"
          response.NextAction = generateFixSuggestion(result)
      }
      
      return response, nil
  }
  
  func generateFixSuggestion(result *VerifyResult) string {
      var suggestions []string
      for _, item := range result.Items {
          if !item.Passed && item.Required {
              suggestions = append(suggestions, 
                  fmt.Sprintf("%s 실패: %s", item.Name, item.Error))
          }
      }
      return strings.Join(suggestions, "\n")
  }
  ```

---

## Claude가 수신하는 응답

**성공 시:**
```json
{
  "port_id": "user-entity",
  "status": "complete",
  "message": "✅ 포트 완료",
  "checklist": {
    "passed": true,
    "items": [
      {"id": "build", "name": "빌드", "passed": true, "output": "성공"},
      {"id": "test", "name": "테스트", "passed": true, "output": "15 passed"},
      {"id": "lint", "name": "린트", "passed": true, "output": "경고 없음"}
    ],
    "summary": "모든 검증 통과"
  }
}
```

**실패 시:**
```json
{
  "port_id": "user-entity",
  "status": "blocked",
  "message": "❌ 체크리스트 검증 실패",
  "next_action": "테스트 실패: TestUserCreate - expected 200, got 400",
  "checklist": {
    "passed": false,
    "items": [
      {"id": "build", "name": "빌드", "passed": true},
      {"id": "test", "name": "테스트", "passed": false, 
       "error": "--- FAIL: TestUserCreate (0.00s)\n    expected 200, got 400"},
      {"id": "lint", "name": "린트", "passed": true}
    ],
    "summary": "테스트 실패 1건"
  }
}
```

---

## Claude 동작 예시

```
Claude: [pal_port_end 호출]
    ↓
응답: {status: "blocked", next_action: "테스트 실패..."}
    ↓
Claude: "테스트가 실패했습니다. 수정하겠습니다."
Claude: [코드 수정]
Claude: [pal_port_end 재호출]
    ↓
응답: {status: "complete"}
    ↓
Claude: "포트가 완료되었습니다."
```

---

## 완료 기준

- [ ] pal_port_end 호출 시 자동 검증 실행
- [ ] 빌드/테스트/린트 체크 동작
- [ ] 실패 시 상세 에러 포함
- [ ] Claude가 결과 수신 후 자동 수정 가능

---

<!-- pal:port:L1-auto-checklist -->
