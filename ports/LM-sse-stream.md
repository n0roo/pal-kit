# Port: LM-sse-stream

> SSE 실시간 스트림 - GUI 실시간 이벤트 연동

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | LM-sse-stream |
| 타입 | atomic |
| 레이어 | LM (Service) |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | L1-auto-checkpoint, L1-checklist-enforce |
| 예상 토큰 | 10,000 |

---

## 목표

Server-Sent Events (SSE)를 통해 PAL Kit 이벤트를 실시간으로 GUI에 전송한다.

---

## 범위

### 포함

- SSE 엔드포인트 구현
- 이벤트 타입 정의
- 이벤트 발행 시스템
- 클라이언트 연결 관리

### 제외

- WebSocket (SSE로 충분)
- GUI 컴포넌트 (LM-gui-hierarchy에서 처리)

---

## 작업 항목

### 1. 이벤트 타입 정의

- [x] `internal/server/events/types.go` 생성
  ```go
  type EventType string
  
  const (
      // 세션 이벤트
      EventSessionStart    EventType = "session:start"
      EventSessionEnd      EventType = "session:end"
      EventSessionUpdate   EventType = "session:update"
      
      // Attention 이벤트
      EventAttentionWarning  EventType = "attention:warning"   // 80%
      EventAttentionCritical EventType = "attention:critical"  // 90%
      EventCompactTriggered  EventType = "compact:triggered"
      EventCheckpointCreated EventType = "checkpoint:created"
      
      // 포트 이벤트
      EventPortStart    EventType = "port:start"
      EventPortEnd      EventType = "port:end"
      EventPortBlocked  EventType = "port:blocked"
      
      // 체크리스트 이벤트
      EventChecklistFailed EventType = "checklist:failed"
      EventChecklistPassed EventType = "checklist:passed"
      
      // 에스컬레이션 이벤트
      EventEscalationCreated EventType = "escalation:created"
      EventEscalationResolved EventType = "escalation:resolved"
      
      // 메시지 이벤트
      EventMessageReceived EventType = "message:received"
  )
  
  type Event struct {
      Type      EventType   `json:"type"`
      Timestamp time.Time   `json:"timestamp"`
      SessionID string      `json:"session_id,omitempty"`
      PortID    string      `json:"port_id,omitempty"`
      Data      interface{} `json:"data"`
  }
  ```

### 2. SSE 서버 구현

- [x] `internal/server/sse.go` 생성
  ```go
  type SSEServer struct {
      clients    map[string]chan Event
      register   chan *SSEClient
      unregister chan *SSEClient
      broadcast  chan Event
      mu         sync.RWMutex
  }
  
  type SSEClient struct {
      ID       string
      Events   chan Event
      Filters  []EventType  // 구독할 이벤트 타입
  }
  
  func (s *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request)
  func (s *SSEServer) Broadcast(event Event)
  func (s *SSEServer) SendTo(clientID string, event Event)
  ```

### 3. API 엔드포인트

- [x] `GET /api/v2/events/stream` - SSE 스트림 연결
  ```
  Query params:
    - filter: 이벤트 타입 필터 (쉼표 구분)
    - session_id: 특정 세션만 구독
  
  Response:
    Content-Type: text/event-stream
    
    data: {"type":"session:update","timestamp":"...","data":{...}}
    
    data: {"type":"attention:warning","timestamp":"...","data":{...}}
  ```

### 4. 이벤트 발행자 연동

- [x] `internal/server/events/publisher.go` 생성
  ```go
  type Publisher struct {
      sse *SSEServer
  }
  
  func (p *Publisher) PublishSessionStart(session *Session)
  func (p *Publisher) PublishAttentionWarning(sessionID string, usage float64)
  func (p *Publisher) PublishCompactTriggered(event *CompactEvent)
  func (p *Publisher) PublishChecklistFailed(portID string, results []CheckResult)
  ```

### 5. 기존 코드 연동

- [x] `internal/attention/attention.go` 수정
  - 80% 도달 시 `EventAttentionWarning` 발행
  - 90% 도달 시 `EventAttentionCritical` 발행
  - Compact 발생 시 `EventCompactTriggered` 발행

- [x] `internal/cli/hook.go` 수정
  - `port-start` 시 `EventPortStart` 발행
  - `port-end` 시 `EventPortEnd` 발행
  - 체크리스트 실패 시 `EventChecklistFailed` 발행

- [x] `internal/session/session.go` 수정
  - 세션 시작/종료 시 이벤트 발행

### 6. 클라이언트 예제 (React)

- [x] `electron-gui/src/hooks/useSSE.ts`
  ```typescript
  export function useSSE(filters?: EventType[]) {
      const [events, setEvents] = useState<Event[]>([])
      const [connected, setConnected] = useState(false)
      
      useEffect(() => {
          const url = `http://localhost:8080/api/v2/events/stream?filter=${filters?.join(',')}`
          const eventSource = new EventSource(url)
          
          eventSource.onmessage = (e) => {
              const event = JSON.parse(e.data)
              setEvents(prev => [...prev.slice(-99), event])
          }
          
          eventSource.onopen = () => setConnected(true)
          eventSource.onerror = () => setConnected(false)
          
          return () => eventSource.close()
      }, [filters])
      
      return { events, connected }
  }
  ```

---

## 완료 기준

- [x] `/api/v2/events/stream` 엔드포인트 동작
- [x] Attention 80%/90% 이벤트 실시간 수신
- [x] Compact 이벤트 실시간 수신
- [x] 체크리스트 실패 이벤트 수신
- [x] 클라이언트 연결/해제 정상 처리
- [ ] 단위 테스트 작성 및 통과

---

## 테스트 시나리오

```bash
# 1. SSE 연결 테스트
curl -N http://localhost:8080/api/v2/events/stream

# 2. 이벤트 발생 시뮬레이션
pal hook session-start --session test-session
# → SSE에서 session:start 이벤트 수신 확인

# 3. Attention 경고 시뮬레이션
# 80% 토큰 사용 시 attention:warning 이벤트 수신 확인
```

---

## 참조

- `internal/server/server.go` - 기존 HTTP 서버
- `internal/server/websocket.go` - 기존 WebSocket (참고)
- `specs/DETAILED-DESIGN-v1.0.md` - 실시간 모니터링 설계

---

<!-- pal:port:LM-sse-stream -->
