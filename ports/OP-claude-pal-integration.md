# Orchestration Port: claude-pal-integration

> Claude Code ↔ PAL Kit 완전 통합 - 자동화된 에이전트 오케스트레이션

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | OP-claude-pal-integration |
| 타입 | orchestration |
| 상태 | pending |
| 우선순위 | critical |
| 예상 기간 | 2주 |

---

## 목표

사용자가 `pal` 명령어를 직접 호출하지 않고, **Claude와의 모든 작업에서 PAL Kit이 자동으로 연동**되는 구조를 구현한다.

### 핵심 원칙

```
사용자 ←→ Claude ←→ PAL Kit (백그라운드)
         ↑
    사용자는 Claude와만 대화
    PAL Kit은 Claude를 통해 자동 동작
```

---

## 연동 방식

### 1. Hook 기반 자동 트리거 (Claude Code Hook System)

```yaml
# .claude/settings.json
hooks:
  PreToolUse:
    - command: "pal hook pre-tool --tool $TOOL_NAME"
      triggers: [Write, Edit, Bash]
      
  PostToolUse:
    - command: "pal hook post-tool --tool $TOOL_NAME --result $RESULT"
      triggers: [Write, Edit, Bash, Task]
      
  Notification:
    - command: "pal hook notification --type $TYPE"
      triggers: [compact, error]
      
  SubagentSpawn:
    - command: "pal hook subagent-spawn --task $TASK_ID"
```

### 2. MCP 도구 (Claude가 직접 호출)

```yaml
# Claude Desktop / Claude Code MCP 설정
mcpServers:
  pal-kit:
    command: "pal"
    args: ["mcp-server"]
    tools:
      - pal_status          # 현재 상태 조회
      - pal_checkpoint      # 체크포인트 생성/복구
      - pal_port_start      # 포트 시작
      - pal_port_end        # 포트 완료
      - pal_escalate        # 에스컬레이션
      - pal_handoff         # 핸드오프 생성
      - pal_context         # 컨텍스트 조회/주입
```

### 3. CLAUDE.md 자동 주입

```markdown
# CLAUDE.md (자동 생성/관리)

## PAL Kit 연동

이 프로젝트는 PAL Kit으로 관리됩니다. 다음 규칙을 따르세요:

### 작업 시작 시
- `pal_status` 도구로 현재 상태 확인
- 활성 포트가 있으면 해당 포트 컨텍스트 로드

### 작업 중
- 주요 결정 시 `pal_checkpoint` 호출
- 80% 토큰 경고 시 자동 체크포인트

### 작업 완료 시
- `pal_port_end` 호출로 체크리스트 검증
- 검증 실패 시 자동 피드백 수신

### 문제 발생 시
- `pal_escalate`로 에스컬레이션
```

---

## 포함 포트 (의존성 순서)

```
[Phase 1: Hook 시스템 구축]
├── L1-hook-system          # Claude Code Hook 완전 연동
│
[Phase 2: MCP 서버 고도화]  
├── LM-mcp-tools            # Claude가 호출하는 MCP 도구
│
[Phase 3: 자동화 로직]
├── L1-auto-checkpoint      # 토큰 기반 자동 체크포인트
├── L1-auto-checklist       # 포트 완료 시 자동 검증
│
[Phase 4: 컨텍스트 관리]
├── LM-context-injection    # 자동 컨텍스트 주입
└── LM-compact-recovery     # Compact 시 자동 복구
```

---

## 시나리오별 동작

### 시나리오 1: 새 작업 시작

```
사용자: "User 엔티티 구현해줘"
    ↓
Claude: [내부] pal_status 호출 → 현재 상태 확인
Claude: [내부] pal_port_start 호출 → user-entity 포트 시작
Claude: [내부] 컨벤션/컨텍스트 자동 로드
Claude: "User 엔티티를 구현하겠습니다. 현재 포트: user-entity"
Claude: [코드 작성 시작]
```

### 시나리오 2: 토큰 80% 도달

```
[Hook: PreToolUse]
    ↓
pal hook pre-tool: 토큰 80% 감지
    ↓
[자동] pal_checkpoint 생성
    ↓
Claude: "⚠️ 토큰 80% 도달. 체크포인트를 생성했습니다."
Claude: [계속 작업]
```

### 시나리오 3: 작업 완료

```
Claude: "User 엔티티 구현이 완료되었습니다."
Claude: [내부] pal_port_end 호출
    ↓
[자동 검증]
├── 빌드 확인 → ✓
├── 테스트 확인 → ✓
└── 컨벤션 확인 → ✓
    ↓
Claude: "✅ 체크리스트 통과. 포트 완료."
```

### 시나리오 4: 검증 실패

```
Claude: [내부] pal_port_end 호출
    ↓
[자동 검증]
├── 빌드 확인 → ✓
├── 테스트 확인 → ✗ (2개 실패)
└── 컨벤션 확인 → ✓
    ↓
Claude: "❌ 테스트 2개 실패. 수정하겠습니다."
Claude: [자동으로 테스트 실패 원인 분석 및 수정]
```

### 시나리오 5: Compact 발생

```
[Hook: Notification - compact]
    ↓
pal hook notification --type compact
    ↓
[자동] 마지막 체크포인트 정보 조회
[자동] 복구 힌트 생성
    ↓
Claude: "Compact가 발생했습니다. 마지막 체크포인트에서 복구합니다."
Claude: [컨텍스트 복구]
Claude: [작업 계속]
```

### 시나리오 6: 서브에이전트 스폰 (Task tool)

```
Claude: [Task tool 사용하여 Worker 스폰]
    ↓
[Hook: SubagentSpawn]
    ↓
pal hook subagent-spawn --task $TASK_ID
    ↓
[자동] 세션 계층 연결 (parent-child)
[자동] 포트 할당
[자동] Handoff 컨텍스트 생성
    ↓
Worker Claude: [자동으로 컨텍스트 수신]
Worker Claude: [작업 시작]
```

---

## 완료 기준

- [ ] 사용자가 `pal` 명령어를 직접 호출하지 않아도 됨
- [ ] Claude가 자연스럽게 PAL 도구 호출
- [ ] Hook으로 자동 체크포인트/검증
- [ ] Compact 발생 시 자동 복구
- [ ] 서브에이전트 자동 연결

---

## 예상 토큰 예산

| 포트 | 예상 토큰 | 비고 |
|------|----------|------|
| L1-hook-system | 8,000 | Hook 핸들러 구현 |
| LM-mcp-tools | 10,000 | MCP 도구 확장 |
| L1-auto-checkpoint | 6,000 | 자동 체크포인트 |
| L1-auto-checklist | 6,000 | 자동 체크리스트 |
| LM-context-injection | 8,000 | 컨텍스트 주입 |
| LM-compact-recovery | 6,000 | Compact 복구 |
| **총합** | **~44,000** | |

---

<!-- pal:port:OP-claude-pal-integration -->
