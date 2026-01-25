# Orchestration Port: v1.0-claude-integration

> PAL Kit v1.0 Claude 통합 고도화 - 완전 자동화 + MCP 도구

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | OP-v1.0-claude-integration |
| 타입 | orchestration |
| 상태 | complete |
| 우선순위 | high |
| 예상 기간 | 2주 |

---

## 설계 원칙

### 사용자는 PAL을 의식하지 않는다

```
사용자 ←→ Claude ←→ PAL Kit (투명)
```

- **Hook 자동 트리거**: Claude Code 훅에서 PAL 기능 자동 실행
- **MCP 도구**: Claude가 필요시 명시적 호출 (`pal_*` 도구)
- **사용자 투명성**: 사용자는 Claude와 대화만, PAL은 백그라운드

### 트리거 방식

| 기능 | 트리거 | 설명 |
|------|--------|------|
| 체크포인트 생성 | Hook (자동) | `pre-tool-use`에서 80% 도달 시 |
| 체크포인트 복구 | MCP (Claude) | Claude가 판단하여 호출 |
| 체크리스트 검증 | Hook (자동) | `port-end`에서 자동 실행 |
| SSE 이벤트 발행 | Hook (자동) | 각 훅에서 이벤트 발행 |
| 세션 계층 조회 | MCP (Claude) | Claude가 필요시 조회 |

---

## 포함 포트 (의존성 순서)

```
[Phase 1: Hook 자동화]
├── hook-auto-checkpoint    # pre-tool-use에서 80% 시 자동 체크포인트
├── hook-checklist-gate     # port-end에서 자동 체크리스트 검증
│
[Phase 2: 실시간 연동]  
├── hook-sse-emit           # 모든 Hook에서 SSE 이벤트 발행
├── gui-realtime-view       # GUI 실시간 업데이트
│
[Phase 3: MCP 도구 확장]
├── mcp-checkpoint-tools    # 체크포인트 MCP 도구
├── mcp-session-tools       # 세션/Attention MCP 도구
│
[Phase 4: 에이전트 확장]
├── agent-reviewer          # Reviewer 에이전트
└── agent-docs              # Docs 에이전트
```

---

## 의존성 그래프

```
[Hook 자동화 - 사용자 개입 없음]
hook-auto-checkpoint ─────┬──▶ hook-sse-emit ──▶ gui-realtime-view
hook-checklist-gate ──────┘

[MCP 도구 - Claude가 호출]
mcp-checkpoint-tools (독립)
mcp-session-tools (독립)

[에이전트 - Claude가 역할 수행]
agent-reviewer (독립)
agent-docs (독립)
```

---

## 완료 기준

**자동화 (사용자 개입 없음)**
- [x] 80% 토큰 시 Hook에서 자동 체크포인트 + Claude에 알림
- [x] port-end 시 자동 체크리스트 검증 + 실패 시 Claude에 피드백
- [x] 모든 Hook 이벤트가 SSE로 GUI에 실시간 전송

**MCP 도구 (Claude가 호출)**
- [x] Claude가 `pal_checkpoint restore`로 체크포인트 복구 가능
- [x] Claude가 `pal_session`으로 Attention 상태 확인 가능
- [x] Claude가 `pal_hierarchy`로 세션 계층 조회 가능

---

<!-- pal:port:OP-v1.0-claude-integration -->
