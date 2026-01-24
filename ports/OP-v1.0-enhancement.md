# Orchestration Port: v1.0-enhancement

> PAL Kit v1.0 고도화 - 자동화 및 실시간 모니터링 강화

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | OP-v1.0-enhancement |
| 타입 | orchestration |
| 상태 | complete |
| 우선순위 | high |
| 예상 기간 | 2주 |

---

## 목표

PAL Kit v1.0의 핵심 누락 기능을 구현하여 실용성을 확보한다:
1. 자동 체크포인트로 Compact 대비
2. SSE 실시간 스트림으로 GUI 연동
3. 완료 체크리스트 강제로 품질 보장
4. 신규 에이전트(Reviewer, Docs)로 워크플로우 완성

---

## 포함 포트 (의존성 순서)

```
[Phase 1: 자동화 기반]
├── L1-auto-checkpoint      # 자동 체크포인트 시스템
├── L1-checklist-enforce    # 완료 체크리스트 강제
│
[Phase 2: 실시간 연동]
├── LM-sse-stream          # SSE 실시간 스트림
├── LM-gui-hierarchy       # GUI 세션 계층 트리뷰
│
[Phase 3: 에이전트 확장]
├── L2-agent-reviewer      # Reviewer 에이전트
└── L2-agent-docs          # Docs 에이전트
```

---

## 의존성 그래프

```
L1-auto-checkpoint ─────────────┐
                                ├──▶ LM-sse-stream ──▶ LM-gui-hierarchy
L1-checklist-enforce ───────────┘
                                     
L2-agent-reviewer ◀── (독립)
L2-agent-docs ◀── (독립)
```

---

## 완료 기준

- [ ] 80% 토큰 도달 시 자동 체크포인트 생성
- [ ] `pal hook port-end` 시 체크리스트 검증 실패하면 블록
- [ ] GUI에서 실시간 Compact Alert 수신
- [ ] 세션 계층 트리뷰에서 Attention 상태 표시
- [ ] Reviewer 에이전트로 코드 리뷰 가능
- [ ] Docs 에이전트로 문서 자동 생성 가능

---

## 예상 토큰 예산

| 포트 | 예상 토큰 | 비고 |
|------|----------|------|
| L1-auto-checkpoint | 8,000 | attention 패키지 확장 |
| L1-checklist-enforce | 6,000 | hook 패키지 수정 |
| LM-sse-stream | 10,000 | server 패키지 확장 |
| LM-gui-hierarchy | 12,000 | React 컴포넌트 |
| L2-agent-reviewer | 5,000 | 템플릿 + 컨벤션 |
| L2-agent-docs | 5,000 | 템플릿 + 컨벤션 |
| **총합** | **~46,000** | |

---

<!-- pal:port:OP-v1.0-enhancement -->
