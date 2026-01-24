# Port: LM-gui-hierarchy

> GUI 세션 계층 트리뷰 - 실시간 세션 시각화

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | LM-gui-hierarchy |
| 타입 | atomic |
| 레이어 | LM (Service/UI) |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | LM-sse-stream |
| 예상 토큰 | 12,000 |

---

## 목표

세션 계층 구조(Build → Operator → Worker → Test)를 트리뷰로 시각화하고, 실시간 Attention 상태를 표시한다.

---

## 범위

### 포함

- 세션 계층 트리뷰 컴포넌트
- 실시간 Attention 표시
- Compact Alert 컴포넌트
- 체크포인트 복구 UI

### 제외

- 에이전트 관리 UI (별도 포트)
- 메시지 상세 뷰 (별도 포트)

---

## 작업 항목

### 1. API 엔드포인트 확장

- [x] `GET /api/v2/sessions/hierarchy` 응답 확장
  ```json
  {
    "root": {
      "id": "build-001",
      "type": "build",
      "title": "user-service 명세 설계",
      "status": "running",
      "attention": {
        "score": 0.72,
        "tokensUsed": 45000,
        "tokenBudget": 60000,
        "driftCount": 2,
        "lastCompact": "2026-01-23T10:30:00Z"
      },
      "compactCount": 3,
      "children": [
        {
          "id": "op-001",
          "type": "operator",
          "title": "user-entity-group",
          "status": "running",
          "attention": { "score": 0.85, ... },
          "children": [
            {
              "id": "worker-001",
              "type": "worker",
              "portId": "UserEntity",
              "status": "complete",
              "attention": { "score": 0.91, ... },
              "testSession": {
                "id": "test-001",
                "status": "passed",
                "attention": { "score": 0.88, ... }
              }
            }
          ]
        }
      ]
    }
  }
  ```

### 2. 세션 트리뷰 컴포넌트

- [x] `electron-gui/src/components/SessionHierarchyTree.tsx`
  ```typescript
  interface SessionNode {
      id: string
      type: 'build' | 'operator' | 'worker' | 'test'
      title: string
      portId?: string
      status: 'pending' | 'running' | 'complete' | 'failed' | 'blocked'
      attention: AttentionState
      compactCount?: number
      children?: SessionNode[]
      testSession?: SessionNode
  }
  
  export function SessionHierarchyTree({ root }: { root: SessionNode }) {
      // 트리 렌더링
  }
  ```

### 3. Attention 게이지 컴포넌트

- [x] `electron-gui/src/components/AttentionGauge.tsx` (기존 구현 활용)
  ```typescript
  interface AttentionGaugeProps {
      score: number        // 0.0 ~ 1.0
      tokensUsed: number
      tokenBudget: number
      status: 'focused' | 'drifting' | 'warning' | 'critical'
  }
  
  export function AttentionGauge({ score, tokensUsed, tokenBudget, status }: AttentionGaugeProps)
  ```

### 4. Compact Alert 컴포넌트

- [x] `electron-gui/src/components/CompactAlertBanner.tsx`
  ```typescript
  interface CompactAlertProps {
      compact: {
          id: string
          reason: string
          beforeTokens: number
          afterTokens: number
          preserved: string[]
          discarded: string[]
          timestamp: Date
          recoveryHint: string
      }
      onRecover: () => void
      onSplit: () => void
      onContinue: () => void
  }
  
  export function CompactAlert({ compact, onRecover, onSplit, onContinue }: CompactAlertProps)
  ```

### 5. 실시간 업데이트 연동

- [x] `electron-gui/src/components/SessionHierarchyTree.tsx`에 SSE 통합
  ```typescript
  export function SessionsPage() {
      const { data: hierarchy, refetch } = useQuery('hierarchy', fetchHierarchy)
      const { events } = useSSE(['session:update', 'attention:warning', 'compact:triggered'])
      
      useEffect(() => {
          // SSE 이벤트 수신 시 refetch
          if (events.some(e => e.type.startsWith('session:') || e.type.startsWith('attention:'))) {
              refetch()
          }
      }, [events])
      
      // Compact Alert 표시
      const compactEvent = events.find(e => e.type === 'compact:triggered')
      
      return (
          <div>
              {compactEvent && <CompactAlert compact={compactEvent.data} />}
              <SessionHierarchyTree root={hierarchy.root} />
          </div>
      )
  }
  ```

### 6. 체크포인트 복구 UI

- [x] `electron-gui/src/components/CompactAlertBanner.tsx`에 복구 버튼 포함
  ```typescript
  interface CheckpointDialogProps {
      sessionId: string
      checkpoints: Checkpoint[]
      onRestore: (checkpointId: string) => void
  }
  
  export function CheckpointDialog({ sessionId, checkpoints, onRestore }: CheckpointDialogProps)
  ```

### 7. 스타일링

- [x] Tailwind 클래스 적용
  - 세션 타입별 색상 (build=blue, operator=green, worker=yellow, test=purple)
  - Attention 상태별 색상 (focused=green, warning=yellow, critical=red)
  - 트리 구조 시각화 (indent, 연결선)

---

## UI 디자인

```
┌─────────────────────────────────────────────────────────────┐
│  Sessions                                      [Refresh] 🔄  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  ⚠️ COMPACT ALERT                                 [×]       │
│  ┌─────────────────────────────────────────────────────┐   │
│  │ Compact #3 발생 • 토큰: 58K → 12K                   │   │
│  │ [체크포인트 복구] [세션 분리] [계속 진행]            │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  📊 Build: user-service 명세 설계                           │
│  ├─ Status: running                                         │
│  ├─ Attention: ████████░░ 72%                              │
│  ├─ Tokens: 45,000 / 60,000                                │
│  └─ Compact: 3회                                           │
│                                                             │
│  └─ 📁 Operator: user-entity-group [running]               │
│     ├─ Attention: █████████░ 85%                           │
│     │                                                       │
│     ├─ 🔧 Worker: UserEntity [✓ complete]                  │
│     │  ├─ Attention: █████████▓ 91%                        │
│     │  └─ 🧪 Test: UserEntityTest [✓ passed]               │
│     │                                                       │
│     ├─ 🔧 Worker: UserRepository [● running]               │
│     │  ├─ Attention: ████████░░ 78%                        │
│     │  └─ 🧪 Test: UserRepositoryTest [○ waiting]          │
│     │                                                       │
│     └─ 🔧 Worker: UserService [○ pending]                  │
│        └─ 🧪 Test: UserServiceTest [○ pending]             │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## 완료 기준

- [x] 세션 계층 트리 정상 렌더링
- [x] 실시간 Attention 상태 업데이트
- [x] Compact Alert 표시 및 복구 버튼 동작
- [x] 세션 타입별 시각적 구분
- [x] 반응형 디자인 (Tailwind)
- [ ] 단위 테스트 (컴포넌트 테스트)

---

## 참조

- `specs/SESSION-AGENT-DESIGN.md` - 세션 계층 설계
- `specs/ELECTRON-DESIGN.md` - GUI 컴포넌트 설계
- `electron-gui/src/components/` - 기존 컴포넌트

---

<!-- pal:port:LM-gui-hierarchy -->
