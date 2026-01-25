# Port: hook-sse-emit

> Hook SSE 이벤트 발행 - 모든 Hook에서 실시간 이벤트 전송

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | hook-sse-emit |
| 타입 | atomic |
| 레이어 | LM (Service) |
| 상태 | pending |
| 우선순위 | high |
| 의존성 | hook-auto-checkpoint, hook-checklist-gate |
| 예상 토큰 | 8,000 |

---

## 설계 원칙

**Hook이 실행될 때마다 SSE로 GUI에 실시간 전송**

```
[Claude Code Hook 실행]
    ↓
[PAL Kit Hook 처리]
    ↓
[SSE 이벤트 발행] ──────▶ [GUI: 실시간 업데이트]
    ↓
[Hook 완료]
```

**사용자는 GUI에서 실시간으로 상태 확인 (optional)**

---

## 범위

### 포함

- SSE 서버 구현 (HTTP API 확장)
- 각 Hook에서 이벤트 발행 코드 추가
- 이벤트 타입 정의
- 클라이언트 연결 관리

### 제외

- GUI 컴포넌트 (gui-realtime-view에서 처리)

---

## 작업 항목

### 1. SSE 서버

- [ ] `internal/server/sse.go` 생성
  ```go
  type SSEHub struct {
      clients    map[string]chan Event
      register   chan *Client
      unregister chan *Client
      broadcast  chan Event
      mu         sync.RWMutex
  }
  
  type Client struct {
      ID      string
      Channel chan Event
      Filters []string // 구독할 이벤트 타입
  }
  
  type Event struct {
      Type      string      `json:"type"`
      Timestamp time.Time   `json:"timestamp"`
      SessionID string      `json:"session_id,omitempty"`
      PortID    string      `json:"port_id,omitempty"`
      Data      interface{} `json:"data"`
  }
  
  var hub *SSEHub // 싱글톤
  
  func GetHub() *SSEHub {
      // 싱글톤 초기화
  }
  
  func (h *SSEHub) Broadcast(event Event) {
      h.broadcast <- event
  }
  
  func (h *SSEHub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
      // SSE 스트림 핸들러
      w.Header().Set("Content-Type", "text/event-stream")
      w.Header().Set("Cache-Control", "no-cache")
      w.Header().Set("Connection", "keep-alive")
      
      client := &Client{
          ID:      uuid.New().String(),
          Channel: make(chan Event, 100),
      }
      
      h.register <- client
      defer func() { h.unregister <- client }()
      
      for {
          select {
          case event := <-client.Channel:
              data, _ := json.Marshal(event)
              fmt.Fprintf(w, "data: %s\n\n", data)
              w.(http.Flusher).Flush()
          case <-r.Context().Done():
              return
          }
      }
  }
  ```

### 2. 이벤트 타입

- [ ] `internal/server/events.go` 생성
  ```go
  const (
      // 세션
      EventSessionStart = "session:start"
      EventSessionEnd   = "session:end"
      
      // Attention
      EventAttentionWarning  = "attention:warning"   // 80%
      EventAttentionCritical = "attention:critical"  // 90%
      
      // 체크포인트
      EventCheckpointCreated = "checkpoint:created"
      
      // Compact
      EventCompactTriggered = "compact:triggered"
      
      // 포트
      EventPortStart   = "port:start"
      EventPortEnd     = "port:end"
      EventPortBlocked = "port:blocked"  // 체크리스트 실패
      
      // 체크리스트
      EventChecklistPassed = "checklist:passed"
      EventChecklistFailed = "checklist:failed"
      
      // 에스컬레이션
      EventEscalation = "escalation:created"
  )
  
  // 이벤트 생성 헬퍼
  func NewSessionStartEvent(sessionID, title string) Event
  func NewCheckpointEvent(sessionID, checkpointID string, usage float64) Event
  func NewChecklistEvent(portID string, result *VerifyResult) Event
  ```

### 3. Hook에 이벤트 발행 추가

- [ ] `internal/cli/hook.go` 각 Hook 함수 수정

  **session-start:**
  ```go
  func runHookSessionStart(...) {
      // ... 기존 로직 ...
      
      // ★ SSE 이벤트 발행
      sse.GetHub().Broadcast(sse.Event{
          Type:      sse.EventSessionStart,
          Timestamp: time.Now(),
          SessionID: palSessionID,
          Data: map[string]interface{}{
              "title":       title,
              "project":     projectName,
              "sessionType": sessionType,
          },
      })
  }
  ```

  **pre-tool-use (체크포인트 생성 시):**
  ```go
  if usage >= 0.8 {
      cp, _ := checkpointSvc.CreateAuto(palSessionID, "auto_80")
      
      // ★ SSE 이벤트 발행
      sse.GetHub().Broadcast(sse.Event{
          Type:      sse.EventCheckpointCreated,
          Timestamp: time.Now(),
          SessionID: palSessionID,
          Data: map[string]interface{}{
              "checkpoint_id": cp.ID,
              "usage":         usage,
              "trigger":       "auto_80",
          },
      })
  }
  ```

  **port-end (체크리스트 결과):**
  ```go
  if !result.Passed {
      // ★ SSE 이벤트 발행
      sse.GetHub().Broadcast(sse.Event{
          Type:      sse.EventChecklistFailed,
          Timestamp: time.Now(),
          PortID:    portID,
          Data:      result,
      })
  } else {
      sse.GetHub().Broadcast(sse.Event{
          Type:      sse.EventChecklistPassed,
          Timestamp: time.Now(),
          PortID:    portID,
      })
  }
  ```

  **pre-compact:**
  ```go
  // ★ SSE 이벤트 발행
  sse.GetHub().Broadcast(sse.Event{
      Type:      sse.EventCompactTriggered,
      Timestamp: time.Now(),
      SessionID: palSessionID,
      Data: map[string]interface{}{
          "trigger":       trigger,
          "compactCount":  newCount,
      },
  })
  ```

### 4. API 엔드포인트

- [ ] `GET /api/v2/events/stream` - SSE 스트림
  ```
  Query params:
    - filter: 이벤트 타입 (쉼표 구분)
    - session: 특정 세션만
  
  Response:
    Content-Type: text/event-stream
    
    data: {"type":"checkpoint:created","timestamp":"...","data":{...}}
    data: {"type":"checklist:failed","timestamp":"...","data":{...}}
  ```

### 5. 서버에 SSE 등록

- [ ] `internal/server/server.go` 수정
  ```go
  func (s *Server) setupRoutes() {
      // ... 기존 라우트 ...
      
      // SSE 스트림
      s.router.GET("/api/v2/events/stream", s.handleSSE)
  }
  
  func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
      sse.GetHub().ServeHTTP(w, r)
  }
  ```

---

## 테스트

```bash
# 터미널 1: SSE 스트림 연결
curl -N http://localhost:8080/api/v2/events/stream

# 터미널 2: Claude Code에서 작업
# → 터미널 1에서 실시간 이벤트 수신 확인
```

---

## 참조

- `internal/server/server.go` - 기존 HTTP 서버
- `internal/server/websocket.go` - WebSocket 참고

---

<!-- pal:port:hook-sse-emit -->
