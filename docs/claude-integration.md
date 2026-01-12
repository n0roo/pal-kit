# Claude Code 연계 가이드

> PAL Kit과 Claude Code를 연동하는 방법

---

## 1. 개요

PAL Kit은 Claude Code와 연동하여 포트 기반 개발을 지원합니다.
포트 명세에 따라 적절한 워커가 자동 선택되고, 컨텍스트가 구성됩니다.

### 핵심 기능

- **워커 자동 매핑**: 포트 타입 → 워커 자동 결정
- **컨텍스트 로딩**: 계층적 컨벤션 및 프롬프트 로딩
- **CLAUDE.md 업데이트**: 활성 워커/체크리스트 자동 표시
- **Hook 연동**: port-start/port-end 이벤트 처리

---

## 2. 컨텍스트 로딩 순서

포트 작업 시작 시 다음 순서로 컨텍스트가 로딩됩니다:

```
┌─────────────────────────────────────────────────────────────────┐
│                    Claude Context Loading                       │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  1. CLAUDE.md (프로젝트 기본 정보)                               │
│       ↓                                                         │
│  2. 패키지 컨벤션 (conventions/architecture.md)                  │
│       ↓                                                         │
│  3. 워커 공통 컨벤션 (conventions/workers/{category}/_common.md) │
│       ↓                                                         │
│  4. 워커 개별 컨벤션 ({worker.conventions_ref})                  │
│       ↓                                                         │
│  5. 포트 명세 (ports/{port-id}.md)                               │
│       ↓                                                         │
│  6. 워커 프롬프트 (agents/workers/{...}/{worker}.yaml → prompt)  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

---

## 3. 워커 매핑 규칙

### 3.1 Backend 워커

| 포트 타입 | 워커 | 선택 기준 |
|----------|------|-----------|
| tpl-server-l1-port | entity-worker | tech: jpa, orm, hibernate |
| tpl-server-l1-port | cache-worker | tech: redis, valkey, cache |
| tpl-server-l1-port | document-worker | tech: mongodb, document |
| tpl-server-lm-port | service-worker | - |
| tpl-server-l2-port | service-worker | 비즈니스 로직 |
| tpl-server-l2-port | router-worker | API Endpoint |
| tpl-server-l3-port | router-worker | - |
| tpl-server-test | test-worker | - |

### 3.2 Frontend 워커

| 포트 타입 | 워커 | 선택 기준 |
|----------|------|-----------|
| tpl-client-feature | frontend-engineer-worker | 오케스트레이션 |
| tpl-client-api-port | component-model-worker | API 연동 |
| tpl-client-query | component-model-worker | Data Fetching |
| tpl-client-component-port | component-ui-worker | UI 구현 |
| tpl-client-e2e | e2e-worker | - |
| tpl-unit-test | unit-tc-worker | - |

---

## 4. Hook 연동

### 4.1 port-start Hook

```bash
pal hook port-start <port-id>
```

수행 작업:
1. 포트 명세 분석
2. 워커 자동 결정
3. 컨텍스트 빌드
4. CLAUDE.md 업데이트
5. Rules 파일 생성

출력 예시:
```
▶️  포트 시작: my-feature-port
   Rules: .claude/rules/my-feature-port.md
   워커: Entity Worker (entity-worker)
   토큰: ~5000
   체크리스트: 8 항목
```

### 4.2 port-end Hook

```bash
pal hook port-end <port-id>
```

수행 작업:
1. 활성 워커 섹션 정리
2. Rules 파일 삭제
3. 포트 상태 complete로 변경

---

## 5. CLI 명령어

### 5.1 컨텍스트 관리

```bash
# 현재 컨텍스트 표시
pal context show

# Claude 통합 컨텍스트 (워커 프롬프트 포함)
pal context claude --port <port-id>

# 컨텍스트 새로고침
pal context reload

# CLAUDE.md에 주입
pal context inject
```

### 5.2 워커 관리

```bash
# 워커 목록
pal worker list
pal worker list --filter backend
pal worker list --filter frontend

# 워커 상세
pal worker show <worker-id>

# 워커 전환
pal worker switch <worker-id>
pal worker switch <worker-id> --port <port-id>

# 포트에 적합한 워커 찾기
pal worker map <port-id>
```

---

## 6. CLAUDE.md 구조

### 6.1 활성 워커 섹션

port-start 시 자동 생성:

```markdown
<!-- pal:active-worker:start -->
> 업데이트: 2024-01-15 14:30:00

### 현재 활성 워커

- **워커**: Entity Worker (`entity-worker`)
- **레이어**: L1
- **기술**: kotlin (jpa, hibernate, jooq)
- **포트**: `my-feature-port`

### 체크리스트

- [ ] Entity 클래스 구현 (Private Constructor, Factory Method)
- [ ] Repository 인터페이스 정의
- [ ] CommandService 구현 (@Transactional)
- [ ] QueryService 구현 (@Transactional(readOnly=true))
- [ ] 단위 테스트 작성

*컨텍스트 토큰: ~5000*
<!-- pal:active-worker:end -->
```

### 6.2 컨텍스트 섹션

```markdown
<!-- pal:context:start -->
> 마지막 업데이트: 2024-01-15 14:30:00

### 활성 세션
- **abc12345**: 기능 개발 (포트: my-feature-port)

### 포트 현황
- 🔄 running: 1
- ✅ complete: 5

### 진행 중인 작업
- **my-feature-port**: 사용자 인증 기능

### 에스컬레이션
- 없음
<!-- pal:context:end -->
```

---

## 7. 워커 YAML 구조

```yaml
agent:
  id: entity-worker
  name: Entity Worker
  type: worker
  layer: L1
  tech:
    language: kotlin
    frameworks: [jpa, hibernate, jooq]

  description: |
    L1 Domain 레이어에서 JPA 엔티티와 레포지토리를 담당하는 Worker.

  responsibilities:
    - Entity 클래스 구현
    - Repository 인터페이스 정의
    - CommandService 구현
    - QueryService 구현

  conventions_ref: conventions/agents/workers/backend/entity.md

  port_types:
    - tpl-server-l1-port

  checklist:
    - Entity 클래스 구현 (Private Constructor, Factory Method)
    - Repository 인터페이스 정의
    - CommandService 구현 (@Transactional)
    - QueryService 구현 (@Transactional(readOnly=true))

  prompt: |
    # Entity Worker

    당신은 L1 Domain 레이어 전문 Entity Worker입니다.
    JPA Entity와 Repository를 담당합니다.

    ## 핵심 규칙
    ...
```

---

## 8. 포트 명세에서 워커 힌트

포트 명세에 다음 필드를 포함하면 워커가 자동 결정됩니다:

```markdown
## 메타데이터

| 항목 | 값 |
|------|-----|
| 포트 타입 | tpl-server-l1-port |
| 레이어 | L1 |

## 기술

- jpa
- hibernate
```

또는:

```markdown
template: tpl-server-l1-port
tech: jpa, hibernate
```

---

## 9. 디렉토리 구조

```
project/
├── CLAUDE.md                    # 프로젝트 컨텍스트
├── .claude/
│   ├── settings.json            # Claude Code 설정
│   └── rules/                   # 활성 포트 Rules
│       └── my-feature-port.md
├── agents/
│   └── workers/
│       ├── backend/
│       │   ├── entity.yaml
│       │   ├── cache.yaml
│       │   ├── document.yaml
│       │   ├── service.yaml
│       │   ├── router.yaml
│       │   └── test.yaml
│       └── frontend/
│           ├── engineer.yaml
│           ├── model.yaml
│           ├── ui.yaml
│           ├── e2e.yaml
│           └── unit-tc.yaml
├── conventions/
│   ├── architecture.md
│   ├── agents/
│   │   └── workers/
│   │       ├── backend/
│   │       │   ├── _common.md
│   │       │   ├── entity.md
│   │       │   └── ...
│   │       └── frontend/
│   │           ├── _common.md
│   │           ├── engineer.md
│   │           └── ...
│   └── ui/
│       ├── mui.md
│       └── tailwind.md
└── ports/
    └── my-feature-port.md
```

---

## 10. 워크플로우 예시

### 새 기능 개발 시작

```bash
# 1. 포트 생성
pal port create my-feature --title "사용자 인증 기능"

# 2. 포트 시작 (자동 워커 할당)
pal hook port-start my-feature

# 3. 작업 수행 (Claude Code에서)
# - 워커의 체크리스트 따라 구현
# - conventions 참고

# 4. 포트 완료
pal hook port-end my-feature
```

### 수동 워커 전환

```bash
# 다른 워커로 전환 필요 시
pal worker switch router-worker --port my-feature
```

---

## 11. 문제 해결

### 워커가 자동 결정되지 않을 때

1. 포트 명세에 `port_type` 또는 `template` 필드 확인
2. `pal worker map <port-id>` 로 분석 확인
3. 수동 전환: `pal worker switch <worker-id>`

### 컨텍스트가 업데이트되지 않을 때

```bash
pal context reload
```

### CLAUDE.md가 없을 때

```bash
pal context inject --file CLAUDE.md
```

---

<!-- pal:doc:claude-integration -->
