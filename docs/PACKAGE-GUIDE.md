# 패키지 시스템 가이드

> 에이전트 패키지 생성 및 관리 가이드

---

## 1. 패키지란?

**패키지(Package)**는 에이전트를 묶는 상위 구조입니다.

```
Package
├── 기술 스택 (Tech)
├── 아키텍처 (Architecture)
├── 방법론 (Methodology)
├── 워커 목록 (Workers)
└── Core 오버라이드 (CoreOverrides)
```

### 1.1 패키지가 필요한 이유

- **일관성**: 프로젝트 전체에 동일한 컨벤션 적용
- **재사용**: 여러 프로젝트에서 동일한 설정 재사용
- **상속**: 기본 패키지를 확장하여 커스터마이징
- **자동화**: 워커 할당 및 컨벤션 로딩 자동화

---

## 2. 기본 패키지

PAL Kit은 PA-Layered 아키텍처 기반 기본 패키지를 제공합니다.

### 2.1 PA-Layered Backend

```yaml
# packages/pa-layered-backend.yaml
package:
  id: pa-layered-backend
  name: PA-Layered Backend
  tech:
    language: kotlin
    frameworks: [spring-boot, spring-data-jpa, jooq]
  architecture:
    name: PA-Layered
    layers: [L1, LM, L2, L3]
  workers:
    - entity-worker
    - cache-worker
    - document-worker
    - service-worker
    - router-worker
    - test-worker
```

### 2.2 PA-Layered Frontend

```yaml
# packages/pa-layered-frontend.yaml
package:
  id: pa-layered-frontend
  name: PA-Layered Frontend
  tech:
    language: typescript
    frameworks: [react, next.js, tanstack-query]
  architecture:
    name: PA-Layered-Frontend
    layers: [Orchestration, Logic, View, Test]
  workers:
    - frontend-engineer-worker
    - component-model-worker
    - component-ui-worker
    - e2e-worker
    - unit-tc-worker
```

---

## 3. 패키지 스키마

### 3.1 전체 스키마

```yaml
package:
  # 필수 필드
  id: string              # 고유 ID (예: my-backend)
  name: string            # 표시 이름 (예: My Backend Package)
  version: string         # 시맨틱 버전 (예: 1.0.0)

  # 선택 필드
  description: string     # 패키지 설명
  extends: string         # 상속할 부모 패키지 ID

  # 기술 스택
  tech:
    language: string      # 주 언어 (kotlin, typescript, go 등)
    frameworks: string[]  # 프레임워크 목록
    build_tool: string    # 빌드 도구 (gradle, npm, go 등)
    runtime: string       # 런타임 (jvm, node 등)

  # 아키텍처
  architecture:
    name: string          # 아키텍처 이름
    layers: string[]      # 레이어 목록
    conventions_ref: string  # 컨벤션 경로
    dependency_rule: string  # 의존성 규칙 설명

  # 방법론
  methodology:
    port_driven: boolean  # 포트 명세 기반 개발
    cqs: boolean          # Command/Query 분리
    event_driven: boolean # 이벤트 기반 통신

  # 워커 목록
  workers: string[]       # 워커 ID 목록

  # Core 에이전트 오버라이드
  core_overrides:
    builder:
      conventions_ref: string
      port_templates: string[]
    architect:
      conventions_ref: string
      validation_rules: string[]
    # ... 다른 Core 에이전트
```

### 3.2 필수 vs 선택

| 필드 | 필수 | 설명 |
|------|------|------|
| id | ✅ | 고유 식별자 |
| name | ✅ | 표시 이름 |
| version | ❌ | 기본값: "1.0.0" |
| tech.language | ✅ | 주 언어 |
| architecture.name | ✅ | 아키텍처 이름 |
| architecture.layers | ✅ | 레이어 목록 (최소 1개) |
| methodology | ❌ | 방법론 (상속 가능) |
| workers | ❌ | 워커 목록 (상속 가능) |

---

## 4. 사용자 정의 패키지 만들기

### 4.1 CLI로 생성

```bash
# 새 패키지 생성
pal package create my-backend --extends pa-layered-backend --lang kotlin

# 결과
# packages/my-backend.yaml 생성됨
```

### 4.2 직접 작성

```yaml
# packages/my-backend.yaml
package:
  id: my-backend
  name: My Custom Backend
  version: "1.0.0"
  extends: pa-layered-backend  # 기본 패키지 상속

  # 기술 스택 오버라이드
  tech:
    frameworks:
      - spring-boot
      - spring-data-jpa
      - jooq
      - spring-security  # 추가

  # 추가 워커
  workers:
    - auth-worker  # 사용자 정의 워커 추가

  # Core 오버라이드
  core_overrides:
    builder:
      port_templates:
        - templates/my-templates/auth-port.md
```

### 4.3 프로젝트에 적용

```bash
# 패키지 적용
pal package use my-backend

# 확인
pal package show my-backend
```

---

## 5. 상속 (Extends)

### 5.1 상속 동작

```yaml
# 부모 패키지
package:
  id: parent
  tech:
    language: kotlin
  workers:
    - worker-a
    - worker-b

# 자식 패키지
package:
  id: child
  extends: parent
  tech:
    frameworks: [spring]  # 추가
  workers:
    - worker-c  # 추가

# 결과 (child)
# tech.language: kotlin (상속)
# tech.frameworks: [spring] (오버라이드)
# workers: [worker-a, worker-b, worker-c] (병합)
```

### 5.2 상속 규칙

| 필드 | 상속 방식 |
|------|----------|
| tech.language | 자식이 비어있으면 부모 사용 |
| tech.frameworks | 자식이 있으면 오버라이드 |
| architecture | 자식이 있으면 오버라이드 |
| methodology | 부울 OR (하나라도 true면 true) |
| workers | 병합 (부모 + 자식) |
| core_overrides | 병합 (자식이 우선) |

### 5.3 다단계 상속

```yaml
# base → backend → my-backend
pa-layered-base
    └── pa-layered-backend
            └── my-backend
```

---

## 6. 패키지 관리 명령어

### 6.1 목록 조회

```bash
pal package list

# 출력
📦 패키지 목록

🏛️  Base 패키지:
   pa-layered-base         PA-Layered Base

⚙️  Backend 패키지:
   pa-layered-backend      PA-Layered Backend (extends: pa-layered-base)

🎨 Frontend 패키지:
   pa-layered-frontend     PA-Layered Frontend (extends: pa-layered-base)
```

### 6.2 상세 조회

```bash
pal package show pa-layered-backend

# 출력
📦 패키지: PA-Layered Backend
──────────────────────────────
ID:      pa-layered-backend
버전:    1.0.0

🔧 기술 스택
   언어:       kotlin
   프레임워크: spring-boot, spring-data-jpa, jooq

🏗️  아키텍처
   이름:       PA-Layered
   레이어:     L1 → LM → L2 → L3

👷 워커
   - entity-worker
   - cache-worker
   - service-worker
   - router-worker
   - test-worker
```

### 6.3 검증

```bash
pal package validate

# 또는 특정 패키지만
pal package validate my-backend
```

### 6.4 워커 목록

```bash
pal package workers pa-layered-backend

# 출력
👷 pa-layered-backend 패키지 워커

   - entity-worker
   - cache-worker
   - document-worker
   - service-worker
   - router-worker
   - test-worker
```

---

## 7. 패키지 디렉토리 구조

### 7.1 프로젝트 구조

```
my-project/
├── .pal/
│   └── config.yaml     # package: my-backend 설정
├── packages/           # 프로젝트 패키지
│   └── my-backend.yaml
├── agents/             # 에이전트 정의
├── conventions/        # 컨벤션 문서
└── ports/              # 포트 명세
```

### 7.2 전역 패키지

```
~/.pal/
├── packages/           # 전역 패키지 (모든 프로젝트에서 사용)
│   ├── pa-layered-base.yaml
│   ├── pa-layered-backend.yaml
│   └── pa-layered-frontend.yaml
└── agents/             # 전역 에이전트 템플릿
```

### 7.3 우선순위

```
프로젝트 packages/ > .pal/packages/ > ~/.pal/packages/
```

동일 ID의 패키지가 있으면 프로젝트 패키지가 우선합니다.

---

## 8. 활용 예시

### 8.1 팀 공통 패키지

```yaml
# packages/team-standard.yaml
package:
  id: team-standard
  name: 팀 표준 패키지
  extends: pa-layered-backend

  tech:
    frameworks:
      - spring-boot
      - spring-data-jpa
      - spring-security
      - spring-cloud-openfeign

  methodology:
    port_driven: true
    cqs: true
    event_driven: true

  core_overrides:
    builder:
      port_templates:
        - templates/team/api-port.md
        - templates/team/event-port.md

    architect:
      validation_rules:
        - team-naming-convention
        - team-layer-dependency
```

### 8.2 마이크로서비스 패키지

```yaml
# packages/microservice.yaml
package:
  id: microservice
  name: Microservice Package
  extends: pa-layered-backend

  tech:
    frameworks:
      - spring-boot
      - spring-cloud
      - kafka
      - redis

  workers:
    - entity-worker
    - cache-worker
    - event-worker      # 이벤트 처리 워커
    - integration-worker # 외부 연동 워커
    - router-worker
```

---

## 9. 트러블슈팅

### 9.1 패키지를 찾을 수 없음

```bash
Error: 패키지 'my-pkg'을(를) 찾을 수 없습니다
```

**확인 사항:**
1. 파일이 `packages/` 디렉토리에 있는지
2. 파일 확장자가 `.yaml` 또는 `.yml`인지
3. `package.id` 값이 올바른지

### 9.2 순환 상속 오류

```bash
Error: 순환 상속 감지: pkg-a
```

상속 관계에서 순환이 발생했습니다. `extends` 관계를 확인하세요.

### 9.3 검증 오류

```bash
pal package validate my-pkg

❌ my-pkg: 오류
   - Tech.Language가 필요합니다
   - Architecture.Layers가 필요합니다
```

필수 필드를 추가하거나 `extends`로 상속받으세요.

---

## 10. 베스트 프랙티스

1. **기본 패키지 상속**: 처음부터 만들지 말고 기본 패키지를 상속
2. **버전 관리**: 패키지 변경 시 version 업데이트
3. **문서화**: description에 패키지 목적 명시
4. **팀 패키지**: 팀 표준을 패키지로 정의하여 공유
5. **검증 실행**: 변경 후 `pal package validate` 실행

---

<!-- pal:docs:package-guide -->
