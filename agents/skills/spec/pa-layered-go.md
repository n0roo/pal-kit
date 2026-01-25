# PA-Layered Go Spec Skill

> Go 백엔드 PA-Layered 아키텍처 명세 스킬

---

## 도메인 특성

### 핵심 개념

PA-Layered는 Port/Adapter 기반 CQS Layered Architecture입니다.

```
L2: Feature Composition (~CompositeService)
    ↓
LM: Shared Composition (~Coordinator)
    ↓
L1: Domain (~CommandService, ~QueryService)
```

### 레이어 역할

| 레이어 | 역할 | 네이밍 |
|--------|------|--------|
| L1 | 도메인 로직, CRUD | {Entity}CommandService, {Entity}QueryService |
| LM | 공유 조율 로직 | {Feature}Coordinator |
| L2 | 기능/유스케이스 | {Feature}CompositeService |

### 의존성 규칙

```
┌────────────────┬─────────┬─────────┬─────────┐
│ From → To      │ L1      │ LM      │ L2      │
├────────────────┼─────────┼─────────┼─────────┤
│ L1 Domain      │ ❌      │ ❌      │ ❌      │
│ LM Shared      │ ✅      │ ⚠️ 단방향│ ❌      │
│ L2 Feature     │ ✅      │ ✅      │ ❌ Event│
└────────────────┴─────────┴─────────┴─────────┘
```

---

## 템플릿

### L1 Domain Port

```yaml
---
type: port
layer: L1
domain: {domain}
title: "{Entity} Domain Service"
priority: {priority}
dependencies: []
---

# {Entity} Domain Port

## 목표
{Entity} 엔티티의 도메인 로직 구현

## 범위
- Command: 생성, 수정, 삭제
- Query: 조회, 검색, 집계

## 인터페이스

### Command Service
- Create{Entity}(ctx, input) -> (entity, error)
- Update{Entity}(ctx, id, input) -> (entity, error)
- Delete{Entity}(ctx, id) -> error

### Query Service
- Get{Entity}(ctx, id) -> (entity, error)
- List{Entities}(ctx, filter) -> ([]entity, error)
- Search{Entities}(ctx, query) -> ([]entity, error)

## 엔티티

| 필드 | 타입 | 설명 |
|------|------|------|
| ID | string | 고유 식별자 |
| ... | ... | ... |

## 검증 규칙
- [ ] Create 시 필수 필드 검증
- [ ] Update 시 존재 여부 확인
- [ ] Delete 시 참조 무결성 확인
```

### LM Coordinator Port

```yaml
---
type: port
layer: LM
domain: {domain}
title: "{Feature} Coordinator"
priority: {priority}
dependencies: [L1-xxx, L1-yyy]
---

# {Feature} Coordinator Port

## 목표
여러 L1 도메인을 조율하는 공유 로직

## 범위
- 교차 도메인 조율
- 공유 비즈니스 규칙
- 이벤트 발행

## 의존성
- L1-xxx: {역할}
- L1-yyy: {역할}

## 인터페이스

### Coordinator
- Coordinate{Action}(ctx, input) -> (result, error)

## 조율 흐름
1. L1-xxx 호출
2. 결과 기반 L1-yyy 호출
3. 이벤트 발행 (필요시)

## 검증 규칙
- [ ] L1만 의존함 (L2 의존 금지)
- [ ] 단방향 LM 의존만 허용
```

### L2 Feature Port

```yaml
---
type: port
layer: L2
domain: {domain}
title: "{Feature} Feature"
priority: {priority}
dependencies: [L1-xxx, LM-yyy]
---

# {Feature} Feature Port

## 목표
{Feature} 유스케이스 구현

## 범위
- 유스케이스 시나리오
- API 엔드포인트
- 외부 어댑터 연동

## 의존성
- L1-xxx: {역할}
- LM-yyy: {역할}

## 인터페이스

### Composite Service
- Execute{UseCase}(ctx, input) -> (result, error)

### API Endpoints
| Method | Path | 설명 |
|--------|------|------|
| POST | /api/v1/{feature} | {description} |
| GET | /api/v1/{feature}/{id} | {description} |

## 검증 규칙
- [ ] L2-to-L2 직접 의존 없음 (Event만)
- [ ] API 계약 정의됨
```

---

## 컨벤션

### 패키지 구조

```
internal/
├── {domain}/           # L1 도메인
│   ├── command.go      # CommandService
│   ├── query.go        # QueryService
│   ├── entity.go       # 엔티티 정의
│   └── repository.go   # 리포지토리 인터페이스
│
├── {feature}/          # LM 또는 L2
│   ├── coordinator.go  # LM: Coordinator
│   └── composite.go    # L2: CompositeService
│
└── adapter/            # 외부 어댑터
    ├── http/           # HTTP 어댑터
    ├── grpc/           # gRPC 어댑터
    └── repository/     # DB 어댑터
```

### 네이밍 규칙

| 구성요소 | 패턴 | 예시 |
|----------|------|------|
| L1 Command | {Entity}CommandService | UserCommandService |
| L1 Query | {Entity}QueryService | UserQueryService |
| LM | {Feature}Coordinator | PricingCoordinator |
| L2 | {Feature}CompositeService | CheckoutCompositeService |
| Repository | {Entity}Repository | UserRepository |
| Entity | {Entity} | User, Order |

### 에러 처리

```go
// 도메인 에러 정의
var (
    ErrNotFound     = errors.New("not found")
    ErrInvalidInput = errors.New("invalid input")
    ErrConflict     = errors.New("conflict")
)

// 래핑
return fmt.Errorf("user not found: %w", ErrNotFound)
```

---

## 검증 기준

### L1 체크리스트
- [ ] 다른 도메인 직접 참조 없음
- [ ] Repository 인터페이스 정의됨
- [ ] 도메인 에러 정의됨
- [ ] Command/Query 분리됨

### LM 체크리스트
- [ ] L1만 의존함
- [ ] L2 의존 없음
- [ ] 단방향 LM 의존
- [ ] 이벤트 발행 정의됨 (필요시)

### L2 체크리스트
- [ ] L2-to-L2 직접 의존 없음
- [ ] API 계약 정의됨
- [ ] 인증/인가 고려됨

---

## 워커 매핑

| 포트 유형 | 워커 | 파일 |
|-----------|------|------|
| L1 Entity | worker-go:entity | entity.yaml |
| L1 Service | worker-go:service | service.yaml |
| API Router | worker-go:router | router.yaml |
| Test | worker-go:test | test.yaml |

---

## 참조 문서

- mcp-docs/00-System/pa-layered/ARCHITECTURE.md
- mcp-docs/00-System/pa-layered/DEPENDENCY-RULES.md
- conventions/go/pa-layered.md
