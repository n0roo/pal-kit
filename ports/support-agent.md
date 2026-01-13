# Port: support-agent

> 문서 관리 특화 에이전트 (Obsidian Template + docs-management 연동)

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | support-agent |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | docs-management |
| 예상 복잡도 | high |

---

## 목표

Obsidian 템플릿 기반으로 문서를 관리하고, docs-management 기능을 통해
코어 에이전트들에게 필요한 문서를 검색/제공하는 **문서 관리 특화 에이전트**.

**핵심 역할**:
- 포트 명세에 직접 관여하지 않음
- 도메인/비즈니스 특화 문서 제공
- 구조화된 문서 관리 체계 지원

---

## 핵심 기능

### 1. 문서 검색 및 제공 (서브세션)

빌더/플래너/아키텍트가 요청 시 관련 문서 검색:

```
Builder: "인증 도메인의 기존 설계 문서가 필요해"
         ↓
Support Agent (서브세션):
  1. docs-management로 검색
     - pal docs search "domain:auth type:l1"
  2. 관련 문서 수집
  3. 토큰 예산 내 요약/전달
         ↓
Builder: 문서 컨텍스트 확보
```

### 2. Obsidian 템플릿 제공

도메인/비즈니스 문서 작성을 위한 템플릿:

```markdown
# {{domain}} 도메인 명세

## 개요
{{overview}}

## 핵심 개념
- {{concept_1}}
- {{concept_2}}

## 비즈니스 규칙
- {{rule_1}}
- {{rule_2}}

## 관련 문서
- [[{{related_1}}]]
- [[{{related_2}}]]

---
tags: #domain/{{domain}} #type/business-spec
```

### 3. 문서 구조화 지원

사용자가 구조화된 문서 관리 체계를 유지하도록 지원:

```
프로젝트 문서 구조:
├── domains/               # 도메인별 비즈니스 명세
│   ├── auth/
│   │   ├── overview.md
│   │   └── rules.md
│   └── inventory/
├── architecture/          # 아키텍처 결정
│   └── adrs/
├── conventions/           # 코딩 컨벤션
└── references/            # 외부 참조 문서
```

---

## 에이전트 정의

### YAML 정의

```yaml
agent:
  id: support
  name: Support
  type: core

  description: |
    문서 관리 특화 에이전트.
    Obsidian 템플릿 기반 문서 관리와 docs-management 연동으로
    코어 에이전트들에게 필요한 문서를 검색/제공합니다.

    포트 명세에 직접 관여하지 않고,
    도메인/비즈니스 문서를 제공하는 지원 역할입니다.

  responsibilities:
    - docs-management 연동 문서 검색
    - 코어 에이전트에게 문서 제공
    - Obsidian 템플릿 기반 문서 생성 지원
    - 문서 구조화 가이드

  subsession_role: support
  parent_agents:
    - builder
    - planner
    - architect

  tools:
    - pal docs search
    - pal docs port
    - pal template create
```

### Rules 파일 (`agents/core/support.rules.md`)

```markdown
# Support Agent Rules

당신은 Support 에이전트입니다. 문서 관리 특화 역할을 수행합니다.

## 핵심 원칙

**당신은 문서 제공자입니다. 포트 결정자가 아닙니다.**

- 포트 명세를 직접 작성하거나 결정하지 않습니다
- 요청된 도메인/비즈니스 문서를 검색하여 제공합니다
- 문서 구조화를 지원합니다

## 문서 검색 가이드

### 도메인 문서 요청 시

1. docs-management로 검색
   ```bash
   pal docs search "domain:{domain} type:l1"
   pal docs search "domain:{domain} type:convention"
   ```
2. 관련 문서 목록 정리
3. 토큰 예산 고려하여 요약 제공

### 비즈니스 규칙 요청 시

1. 도메인별 비즈니스 명세 검색
2. 관련 규칙 추출
3. 컨텍스트와 함께 제공

## 문서 생성 지원

### 템플릿 사용

```bash
pal template create domain --name {domain}
pal template create business-rule --domain {domain}
```

## 제공 형식

문서 제공 시 다음 형식 사용:

```markdown
## 검색 결과

### 관련 문서 (N건)
- {path}: {summary}

### 핵심 내용
{extracted_content}

### 참고
- 토큰 사용: ~{tokens}
- 추가 문서가 필요하면 요청해주세요
```

## 절대 금지

- ❌ 포트 명세 직접 작성
- ❌ 기술적 결정 제안
- ❌ 코드 작성
- ❌ 요청 범위 외 문서 제공
```

---

## docs-management 연동

### 검색 쿼리 예시

```bash
# 도메인별 L1 문서
pal docs search "type:l1 domain:auth"

# 컨벤션 문서
pal docs search "type:convention"

# 포트 명세
pal docs port auth-service --deps

# 특정 태그
pal docs search "tag:jwt tag:security"
```

### 토큰 예산 관리

```yaml
token_budget:
  default: 5000       # 기본 제공 토큰
  max: 15000          # 최대 토큰
  summary_threshold: 2000  # 요약 필요 기준
```

---

## 구현 가이드

### P1: 에이전트 정의

- [x] `agents/core/support.yaml` 작성
- [x] `agents/core/support.rules.md` 작성
- [x] `conventions/agents/core/support.md` 작성

### P2: docs-management 연동

- [x] 검색 쿼리 래퍼 함수 (`pal docs search`, `pal docs context`)
- [x] 토큰 예산 기반 문서 선택 (`--budget` 플래그)
- [x] 요약 생성 로직 (2000 토큰 초과 시 자동 요약)
- [x] `pal docs get` 명령어 추가

### P3: 템플릿 시스템

- [x] 도메인 명세 템플릿 (`domain-spec`)
- [x] 비즈니스 규칙 템플릿 (`business-rule`)
- [x] ADR 템플릿 (`adr`)
- [x] `pal docs template` CLI 연동

### P4: 문서 구조화 가이드

- [x] 권장 디렉토리 구조 문서화 (conventions/agents/core/support.md)
- [x] 문서 초기화 명령어 (`pal docs init`)
- [x] 구조 검증 (`pal docs lint`)

---

## 완료 기준

- [x] Support 에이전트 정의 완료
- [x] 빌더/플래너/아키텍트 요청 시 문서 검색/제공 가능
- [x] Obsidian 템플릿 기반 문서 생성 지원
- [x] 문서 구조화 가이드 제공

---

<!-- pal:port:status=complete -->
