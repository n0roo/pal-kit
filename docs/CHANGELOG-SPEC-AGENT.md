# Spec Agent System - 변경사항 문서

> 날짜: 2026-01-25
> 버전: PAL Kit v1.0

---

## 개요

복잡한 다중 도메인 명세를 체계적으로 관리하기 위한 **Spec Agent 시스템**을 구현했습니다.
검증된 LLM 워크플로우 접근법(Claude Skills, 에이전트 전문화)을 기반으로 설계되었습니다.

---

## 추가된 파일

### Core Agents (6개)

| 파일 | 설명 |
|------|------|
| `agents/core/spec.yaml` | Spec Agent 오케스트레이터 정의 |
| `agents/core/spec.rules.md` | Spec Agent 실행 규칙 |
| `agents/core/spec-writer.yaml` | 명세 작성 서브에이전트 |
| `agents/core/spec-writer.rules.md` | Spec Writer 실행 규칙 |
| `agents/core/spec-reviewer.yaml` | 명세 검토 서브에이전트 |
| `agents/core/spec-reviewer.rules.md` | Spec Reviewer 실행 규칙 |

### Domain Skills (6개)

| 파일 | 설명 |
|------|------|
| `agents/skills/spec/SKILL.md` | 스킬 시스템 개요 |
| `agents/skills/spec/pa-layered-go.md` | Go PA-Layered 아키텍처 명세 스킬 |
| `agents/skills/spec/spring-msa.md` | Spring Cloud MSA 명세 스킬 |
| `agents/skills/spec/react-client.md` | React 프론트엔드 명세 스킬 |
| `agents/skills/spec/electron.md` | Electron 데스크톱 앱 명세 스킬 |
| `agents/skills/spec/cloud-infra.md` | 클라우드 인프라 (IaC, K8s) 명세 스킬 |

### Conventions (3개)

| 파일 | 설명 |
|------|------|
| `conventions/agents/core/spec.md` | Spec Agent 컨벤션 |
| `conventions/agents/core/spec-writer.md` | Spec Writer 컨벤션 |
| `conventions/agents/core/spec-reviewer.md` | Spec Reviewer 컨벤션 |

### 템플릿 (embed)

위 모든 파일이 `internal/agent/templates/`에도 복사되어 `pal init` 시 프로젝트에 주입됩니다.

---

## 아키텍처

### 에이전트 계층

```
┌─────────────────────────────────────────────────────────────┐
│                     Spec Agent (Orchestrator)                │
│  - 워크플로우 조율                                            │
│  - 프로젝트 모드 판단 (strict/loose)                          │
│  - 도메인 스킬 로드                                           │
└─────────────────────────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
┌───────────────┐    ┌───────────────┐    ┌───────────────┐
│  Planner      │    │  Spec Writer  │    │ Spec Reviewer │
│  (분석)        │    │   (작성)       │    │   (검토)       │
└───────────────┘    └───────────────┘    └───────────────┘
                              │
                              ▼
                    ┌───────────────┐
                    │   Architect   │
                    │ (아키텍처 검토) │
                    └───────────────┘
```

### 워크플로우

```
분석 → 참조 → 초안 → 검토 → 수정 → 확정
       ↑__________________|
       (피드백 반복)
```

| 스테이지 | 서브에이전트 | 역할 |
|----------|-------------|------|
| 분석 | Planner | 요구사항 분해 |
| 참조 | Support | KB 검색, 문서 제공 |
| 초안/수정 | Spec Writer | 명세 작성 |
| 검토 | Spec Reviewer | 품질 검토 (완전성, 명확성) |
| 검토 | Architect | 아키텍처 검토 (레이어, 의존성) |

---

## 핵심 기능

### 1. 프로젝트 모드

| 모드 | 대상 | 명세 수준 |
|------|------|----------|
| **strict** | 업무 프로젝트 | 완전한 명세, PA-Layered 준수, 의존성 그래프 필수 |
| **loose** | 실험 프로젝트 | 최소 명세, 핵심 요구사항만 |

### 2. 도메인 스킬

| 도메인 | 스킬 파일 | 템플릿 |
|--------|----------|--------|
| Go 백엔드 | pa-layered-go.md | L1 Domain, LM Coordinator, L2 Feature |
| Spring MSA | spring-msa.md | Domain Service, API Gateway, Event Handler |
| React | react-client.md | Feature, Component, API Adapter |
| Electron | electron.md | Main Process, Preload, Renderer |
| Cloud | cloud-infra.md | Terraform Module, K8s Manifest, CI/CD |

### 3. 품질 검토 기준

| 기준 | 설명 |
|------|------|
| 완전성 (Completeness) | 필수 섹션, 요구사항 매핑 |
| 명확성 (Clarity) | 목표/범위 구체성, 모호함 없음 |
| 일관성 (Consistency) | 용어, 형식, 네이밍 통일 |
| 추적성 (Traceability) | 요구사항 연결, 의존성 명확 |

### 4. PAL Kit 연동

```bash
pal port create <id>      # 포트 생성
pal hook port-start <id>  # 작업 시작
pal kb search "query"     # KB 검색
pal hook port-end <id>    # 작업 완료
```

---

## 사용 방법

### 자동 스킬 로드

Spec Agent는 키워드를 감지하여 자동으로 도메인 스킬을 로드합니다:

| 키워드 | 로드되는 스킬 |
|--------|--------------|
| Go, L1, LM, L2, PA-Layered | pa-layered-go.md |
| Spring, MSA, Gateway, Kafka | spring-msa.md |
| React, Redux, Hook, Component | react-client.md |
| Electron, IPC, Main Process | electron.md |
| Terraform, AWS, K8s, Docker | cloud-infra.md |

### 프로젝트 모드 판단

| 신호 | 판단 |
|------|------|
| "프로덕션", "업무", "팀 프로젝트" | strict 모드 |
| "실험", "POC", "테스트", "개인" | loose 모드 |
| 불명확 | 사용자에게 확인 |

### Builder 연계

명세 확정 후 구현이 필요하면 Builder에게 Handoff:

```markdown
## Builder Handoff

### 포트 명세
{확정된 명세 경로}

### 작업지시서
{작업지시서 내용}

### 의존성 그래프
{포트 간 관계}

### 추가 정보
- 예상 토큰: ~{tokens}
- 워커 추천: {worker-type}
```

---

## 기술 결정

### Anthropic 연구 기반

- **에이전트 전문화**: 범용 에이전트보다 전문 에이전트가 성능 우수
- **단순하게 시작**: 필요에 따라 복잡성 추가
- **3-에이전트 패턴**: Research → Write → Review

### Claude Skills 프레임워크

- SKILL.md 파일로 특정 작업 방법 교육
- 도구 사용법이 아닌 "어떻게 사용하는지" 가르침
- 동적 지시사항 로딩

### Spec-Kit Autopilot 개념

- 의도(intent) 기반 개발
- 명세 명령 자동 오케스트레이션

---

## 버그 수정

### Electron GUI 중복 선언 오류

**문제**: `App.tsx`에서 `Settings` 이름 충돌
- lucide-react의 `Settings` 아이콘
- pages/Settings 컴포넌트

**해결**: 페이지 컴포넌트를 `SettingsPage`로 rename

```tsx
// Before
import Settings from './pages/Settings'
<Route path="/settings" element={<Settings />} />

// After
import SettingsPage from './pages/Settings'
<Route path="/settings" element={<SettingsPage />} />
```

---

## 검증

### 템플릿 주입 테스트

```bash
# 테스트 결과
$ pal init (in /tmp/pal-init-test)

✅ 총 78개 파일 설치
✅ Spec Agent 관련 15개 파일 포함
  - agents/core/spec*.yaml, spec*.rules.md (6개)
  - conventions/agents/core/spec*.md (3개)
  - agents/skills/spec/*.md (6개)
```

### 빌드 테스트

```bash
$ go build -o /dev/null ./...
# 성공 (오류 없음)
```

---

## 관련 문서

- [ONBOARDING.md](./ONBOARDING.md) - PAL Kit 온보딩 가이드
- [CLAUDE.md](../CLAUDE.md) - 프로젝트 컨텍스트
- [agents/core/spec.yaml](../agents/core/spec.yaml) - Spec Agent 정의
- [agents/skills/spec/SKILL.md](../agents/skills/spec/SKILL.md) - 스킬 개요
