# Worker 에이전트 공통 컨벤션

> 모든 Worker 에이전트가 공유하는 규칙

---

## 1. Worker의 역할

Worker는 **포트(Port) 명세를 기반으로 실제 코드를 구현**하는 에이전트입니다.

### 1.1 핵심 원칙

- **포트 주도 개발**: 포트 명세가 작업의 범위와 산출물을 정의
- **레이어 전문화**: 각 Worker는 특정 레이어(L1/LM/L2/L3)에 특화
- **단위 테스트 책임**: Worker는 자신이 구현한 코드의 단위 테스트 작성

### 1.2 Worker 유형

| 레이어 | Backend Worker | Frontend Worker |
|--------|----------------|-----------------|
| L1 (Domain) | entity-worker, cache-worker, document-worker | component-model-worker |
| LM (Shared) | service-worker | - |
| L2 (Feature) | service-worker | component-ui-worker |
| L3 (Router) | router-worker | frontend-engineer-worker |
| Test | test-worker | e2e-worker, unit-tc-worker |

---

## 2. 포트 기반 작업 프로세스

### 2.1 작업 시작

```markdown
## 포트 수신: {포트 ID}

**레이어**: {L1/LM/L2/L3}
**타입**: {포트 타입}
**의존성**: {선행 포트 목록}
**완료조건**: {TC 체크리스트}
```

### 2.2 포트 명세 확인 항목

1. **레이어 확인**: 자신의 전문 레이어와 일치하는지
2. **의존성 확인**: 선행 포트 산출물 존재 여부
3. **완료조건(TC) 확인**: 구체적인 완료 기준 파악
4. **산출물 확인**: 생성해야 할 파일 목록

### 2.3 작업 흐름

```
포트 수신 → 명세 분석 → 구현 → 단위 테스트 → 완료 보고
     │           │          │          │            │
   확인        계획       코드작성   테스트작성    게이트 제출
```

---

## 3. CQS(Command Query Separation) 규칙

### 3.1 Command 서비스 규칙

```kotlin
// Command는 상태를 변경하고, void 또는 ID만 반환
interface OrderCommandService {
    fun create(request: CreateOrderRequest): OrderId
    fun update(id: OrderId, request: UpdateOrderRequest)
    fun delete(id: OrderId)
}
```

**금지 사항:**
- Command에서 복잡한 조회 결과 반환
- Command에서 다른 도메인 직접 조회

**허용 사항:**
- Command 내부에서 Private Query (Master DB 직접 조회)
- 생성된 ID 반환

### 3.2 Query 서비스 규칙

```kotlin
// Query는 조회만, 상태 변경 없음
interface OrderQueryService {
    fun findById(id: OrderId): Order?
    fun findAll(criteria: OrderCriteria): List<Order>
    fun count(criteria: OrderCriteria): Long
}
```

**금지 사항:**
- Query에서 상태 변경
- Query에서 Command 호출

---

## 4. 단위 테스트 규칙

### 4.1 테스트 책임

| 테스트 유형 | 담당 | 범위 |
|------------|------|------|
| 단위 테스트 | **Worker** | 자신이 구현한 코드 |
| 통합 테스트 | Tester | 여러 컴포넌트 연동 |
| E2E 테스트 | Tester | 전체 흐름 |

### 4.2 단위 테스트 패턴

```kotlin
class OrderCommandServiceTest {

    @Test
    fun `should create order with valid request`() {
        // Given
        val request = CreateOrderRequest(...)

        // When
        val orderId = orderCommandService.create(request)

        // Then
        assertThat(orderId).isNotNull()
    }

    @Test
    fun `should throw exception when product not found`() {
        // Given
        val request = CreateOrderRequest(productId = "invalid")

        // When & Then
        assertThrows<ProductNotFoundException> {
            orderCommandService.create(request)
        }
    }
}
```

### 4.3 커버리지 목표

| 레이어 | 라인 커버리지 | 브랜치 커버리지 |
|--------|--------------|----------------|
| L1 Domain | 90%+ | 80%+ |
| LM Shared | 85%+ | 75%+ |
| L2 Feature | 80%+ | 70%+ |
| L3 Router | 70%+ | 60%+ |

---

## 5. 에러 처리 규칙

### 5.1 도메인 예외 정의

```kotlin
// L1 레이어에서 도메인 예외 정의
sealed class OrderException(message: String) : RuntimeException(message) {
    class NotFound(id: OrderId) : OrderException("Order not found: $id")
    class InvalidStatus(status: OrderStatus) : OrderException("Invalid status: $status")
    class InsufficientStock(productId: ProductId) : OrderException("Insufficient stock")
}
```

### 5.2 예외 전파 규칙

```
L1 (도메인 예외) → LM (변환/래핑) → L2 (비즈니스 예외) → L3 (HTTP 응답)
```

- L1: 도메인 규칙 위반 예외
- LM: 조합 과정 예외 (트랜잭션 실패 등)
- L2: 비즈니스 규칙 위반
- L3: HTTP 상태 코드로 변환

---

## 6. 코드 스타일 규칙

### 6.1 네이밍 컨벤션

| 레이어 | 패턴 | 예시 |
|--------|------|------|
| L1 Entity | `{Entity}` | `Order`, `Product`, `User` |
| L1 Command | `{Entity}CommandService` | `OrderCommandService` |
| L1 Query | `{Entity}QueryService` | `OrderQueryService` |
| L1 Repository | `{Entity}Repository` | `OrderRepository` |
| LM Coordinator | `{Feature}Coordinator` | `PricingCoordinator` |
| L2 Composite | `{Feature}CompositeService` | `CheckoutCompositeService` |
| L3 Controller | `{Feature}Controller` | `OrderController` |

### 6.2 파일 구조

```
domain/
└── {domain}/
    ├── model/
    │   ├── {Entity}.kt
    │   ├── {Entity}Repository.kt
    │   └── {ValueObject}.kt
    ├── command/
    │   └── {Entity}CommandService.kt
    └── query/
        └── {Entity}QueryService.kt
```

---

## 7. 에스컬레이션 규칙

### 7.1 Worker가 에스컬레이션해야 하는 상황

| 상황 | 대상 | 에스컬레이션 내용 |
|------|------|-----------------|
| 포트 범위 초과 | Builder | 새 포트 필요 |
| 아키텍처 결정 필요 | Architect | 기술 선택, 패턴 결정 |
| 의존성 문제 | Manager | 선행 포트 미완료 |
| 테스트 불가 | Tester | 테스트 환경 문제 |
| 요구사항 불명확 | User | 기능 명확화 |

### 7.2 에스컬레이션 형식

```markdown
## 에스컬레이션 요청

**출처**: {worker-type}
**포트**: {port-id}
**유형**: {범위초과/아키텍처/의존성/테스트/요구사항}
**내용**: {구체적인 상황 설명}
**제안**: {가능하다면 해결 방안 제안}
```

---

## 8. 완료 보고 규칙

### 8.1 완료 체크리스트

- [ ] 포트 명세의 모든 완료조건(TC) 충족
- [ ] 단위 테스트 작성 완료
- [ ] 테스트 전체 통과
- [ ] 빌드 성공
- [ ] 린터 경고 없음

### 8.2 완료 보고 형식

```markdown
## 포트 완료 보고: {포트 ID}

### 산출물
| 파일 | 설명 |
|------|------|
| domain/orders/model/Order.kt | 주문 엔티티 |
| domain/orders/command/OrderCommandService.kt | 주문 Command 서비스 |

### 테스트
- 작성: 15개
- 통과: 15개
- 커버리지: 92%

### 완료조건 체크
- [x] TC-001: Order 엔티티 구현
- [x] TC-002: CRUD 동작 확인
- [x] TC-003: 유효성 검증 동작

### 비고
{특이사항 또는 다음 작업자 참고사항}
```

---

## 9. Worker 간 협업 규칙

### 9.1 산출물 참조

- 다른 Worker의 산출물은 **인터페이스**로만 참조
- 구현체 직접 참조 금지
- ID 참조 선호 (객체 참조 최소화)

### 9.2 의존성 규칙

```
entity-worker 산출물 → service-worker 사용
         │                    │
    (Repository)         (Service)
```

---

## 10. PAL 명령어

```bash
# 포트 상세 보기
pal port show <id>

# 포트 의존성 확인
pal port deps <id>

# 포트 파일 목록
pal port files <id>

# 빌드 실행
pal build

# 테스트 실행
pal test

# 작업 시작/종료
pal hook port-start <id>
pal hook port-end <id>
```

---

<!-- pal:convention:workers:common -->
