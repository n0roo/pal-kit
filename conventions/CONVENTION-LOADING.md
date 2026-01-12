# 컨벤션 로딩 메커니즘

> PAL Kit 에이전트 컨벤션 로딩 및 적용 규칙

---

## 1. 개요

PAL Kit은 에이전트 유형에 따라 적절한 컨벤션을 자동으로 로드하고 적용합니다.

### 1.1 로딩 순서

```
Package 컨벤션 (선택)
       ↓
에이전트 공통 컨벤션
       ↓
에이전트 개별 컨벤션
       ↓
포트 명세 (작업 시)
```

### 1.2 컨벤션 우선순위

후순위 컨벤션이 선순위 컨벤션을 **오버라이드**합니다.

```
낮음 ← Package → Agent Common → Agent Specific → Port Spec → 높음
```

---

## 2. 컨벤션 디렉토리 구조

```
conventions/
├── CONVENTION-LOADING.md      # 이 문서
├── agents/
│   ├── core/
│   │   ├── _common.md         # Core 공통
│   │   ├── builder.md
│   │   ├── planner.md
│   │   ├── architect.md
│   │   ├── manager.md
│   │   ├── tester.md
│   │   └── logger.md
│   └── workers/
│       ├── _common.md         # Worker 공통
│       ├── backend/
│       │   ├── entity.md
│       │   ├── cache.md
│       │   ├── document.md
│       │   ├── service.md
│       │   ├── router.md
│       │   └── test.md
│       └── frontend/
│           ├── engineer.md
│           ├── model.md
│           ├── ui.md
│           ├── e2e.md
│           └── unit-tc.md
└── packages/                   # 패키지별 오버라이드 (선택)
    └── kotlin-spring/
        └── overrides.md
```

---

## 3. 로딩 규칙

### 3.1 Core 에이전트 로딩

| 에이전트 | 로드되는 컨벤션 |
|---------|---------------|
| Builder | `core/_common.md` + `core/builder.md` |
| Planner | `core/_common.md` + `core/planner.md` |
| Architect | `core/_common.md` + `core/architect.md` |
| Manager | `core/_common.md` + `core/manager.md` |
| Tester | `core/_common.md` + `core/tester.md` |
| Logger | `core/_common.md` + `core/logger.md` |

### 3.2 Worker 에이전트 로딩

| 워커 | 로드되는 컨벤션 |
|------|---------------|
| entity-worker | `workers/_common.md` + `workers/backend/entity.md` |
| cache-worker | `workers/_common.md` + `workers/backend/cache.md` |
| document-worker | `workers/_common.md` + `workers/backend/document.md` |
| service-worker | `workers/_common.md` + `workers/backend/service.md` |
| router-worker | `workers/_common.md` + `workers/backend/router.md` |
| test-worker | `workers/_common.md` + `workers/backend/test.md` |
| frontend-engineer-worker | `workers/_common.md` + `workers/frontend/engineer.md` |
| component-model-worker | `workers/_common.md` + `workers/frontend/model.md` |
| component-ui-worker | `workers/_common.md` + `workers/frontend/ui.md` |
| e2e-worker | `workers/_common.md` + `workers/frontend/e2e.md` |
| unit-tc-worker | `workers/_common.md` + `workers/frontend/unit-tc.md` |

---

## 4. 컨벤션 매칭 알고리즘

### 4.1 에이전트 ID → 컨벤션 파일 매핑

```typescript
function getConventionPaths(agentId: string): string[] {
  const paths: string[] = [];

  // 1. Core 에이전트인 경우
  if (CORE_AGENTS.includes(agentId)) {
    paths.push('agents/core/_common.md');
    paths.push(`agents/core/${agentId}.md`);
    return paths;
  }

  // 2. Worker 에이전트인 경우
  paths.push('agents/workers/_common.md');

  // Worker 타입에 따라 경로 결정
  const workerType = extractWorkerType(agentId);
  const category = getWorkerCategory(workerType); // backend | frontend

  paths.push(`agents/workers/${category}/${workerType}.md`);

  return paths;
}

// 워커 ID에서 타입 추출
function extractWorkerType(workerId: string): string {
  // entity-worker → entity
  // component-ui-worker → ui
  const mapping = {
    'entity-worker': 'entity',
    'cache-worker': 'cache',
    'document-worker': 'document',
    'service-worker': 'service',
    'router-worker': 'router',
    'test-worker': 'test',
    'frontend-engineer-worker': 'engineer',
    'component-model-worker': 'model',
    'component-ui-worker': 'ui',
    'e2e-worker': 'e2e',
    'unit-tc-worker': 'unit-tc',
  };
  return mapping[workerId] || workerId.replace('-worker', '');
}
```

### 4.2 카테고리 결정

```typescript
const BACKEND_WORKERS = ['entity', 'cache', 'document', 'service', 'router', 'test'];
const FRONTEND_WORKERS = ['engineer', 'model', 'ui', 'e2e', 'unit-tc'];

function getWorkerCategory(workerType: string): 'backend' | 'frontend' {
  if (BACKEND_WORKERS.includes(workerType)) return 'backend';
  if (FRONTEND_WORKERS.includes(workerType)) return 'frontend';
  throw new Error(`Unknown worker type: ${workerType}`);
}
```

---

## 5. 포트 시작 시 컨벤션 로딩

### 5.1 로딩 흐름

```
pal hook port-start <port-id>
        │
        ↓
   포트 명세 로드
        │
        ↓
   할당된 워커 확인
        │
        ↓
   컨벤션 파일 로드
        │
        ↓
   Claude 컨텍스트 구성
```

### 5.2 컨텍스트 구성 예시

```markdown
## 현재 세션 컨텍스트

### 활성 컨벤션
1. Worker 공통 규칙 (workers/_common.md)
2. Entity Worker 규칙 (workers/backend/entity.md)

### 현재 포트
- ID: L1-Order
- 레이어: L1
- 의존성: 없음

### 완료조건
- [ ] Order 엔티티 구현
- [ ] OrderRepository 구현
- [ ] OrderCommandService 구현
- [ ] OrderQueryService 구현
- [ ] 단위 테스트 작성
```

---

## 6. 패키지 오버라이드

### 6.1 패키지 컨벤션 적용

특정 기술 스택(패키지)에서 기본 컨벤션을 오버라이드할 수 있습니다.

```yaml
# .pal/config.yaml
package: kotlin-spring
conventions:
  overrides:
    - packages/kotlin-spring/overrides.md
```

### 6.2 오버라이드 예시

```markdown
# packages/kotlin-spring/overrides.md

## Kotlin Spring 패키지 오버라이드

### Entity 규칙 추가
- `@Entity` 어노테이션 필수
- Kotlin data class 대신 일반 class 사용
- equals/hashCode는 ID 기반으로 구현

### Repository 규칙 추가
- Spring Data JPA `JpaRepository` 상속
- 커스텀 쿼리는 `@Query` 어노테이션 사용
```

---

## 7. Claude 통합

### 7.1 시스템 프롬프트 구성

PAL Kit은 다음 순서로 Claude 시스템 프롬프트를 구성합니다:

```
1. PAL Kit 기본 지침
2. 프로젝트 컨텍스트 (CLAUDE.md)
3. 패키지 컨벤션 (선택)
4. 에이전트 공통 컨벤션
5. 에이전트 개별 컨벤션
6. 현재 포트 명세 (작업 시)
```

### 7.2 컨벤션 마커

각 컨벤션 파일은 마커로 식별됩니다:

```markdown
<!-- pal:convention:core:builder -->
<!-- pal:convention:workers:common -->
<!-- pal:convention:workers:backend:entity -->
```

### 7.3 로딩 확인

```bash
# 현재 로드된 컨벤션 확인
pal convention list

# 특정 에이전트의 컨벤션 미리보기
pal convention preview entity-worker
```

---

## 8. 컨벤션 관리 명령어

```bash
# 컨벤션 목록 조회
pal convention list

# 컨벤션 검증
pal convention validate

# 컨벤션 미리보기
pal convention preview <agent-id>

# 컨벤션 적용 테스트
pal convention test <port-id>

# 새 컨벤션 생성
pal convention create <type> <name>
```

---

## 9. 컨벤션 작성 가이드

### 9.1 필수 섹션

모든 컨벤션 파일은 다음 섹션을 포함해야 합니다:

1. **역할 정의** - 에이전트의 책임과 범위
2. **핵심 규칙** - 반드시 따라야 할 규칙
3. **완료 체크리스트** - 작업 완료 기준
4. **에스컬레이션 기준** - 언제, 누구에게 에스컬레이션

### 9.2 컨벤션 마커 규칙

```markdown
<!-- pal:convention:{category}:{subcategory}:{name} -->
```

예시:
- `<!-- pal:convention:core:builder -->`
- `<!-- pal:convention:workers:common -->`
- `<!-- pal:convention:workers:backend:entity -->`
- `<!-- pal:convention:workers:frontend:ui -->`

---

## 10. 트러블슈팅

### 10.1 컨벤션이 로드되지 않는 경우

1. 파일 경로 확인
2. 마커 태그 확인
3. `pal convention validate` 실행

### 10.2 충돌하는 규칙

후순위 컨벤션이 우선합니다. 충돌이 발생하면:
1. 포트 명세 > 에이전트 개별 > 에이전트 공통 > 패키지
2. 명시적 규칙 > 암묵적 규칙

### 10.3 커스텀 컨벤션 추가

```bash
# 커스텀 컨벤션 디렉토리 생성
mkdir -p conventions/custom

# 설정에 추가
pal config set conventions.custom ./conventions/custom
```

---

<!-- pal:convention:loading-mechanism -->
