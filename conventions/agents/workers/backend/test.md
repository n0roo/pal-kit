# Test Worker 컨벤션

> 테스트 전문 Worker - Worker 테스트 보완 담당

---

## 1. 역할 정의

Test Worker는 **다른 Worker들이 작성한 단위 테스트를 보완**하고, **테스트 인프라를 관리**하는 전문 Worker입니다.

### 1.1 담당 영역

- Worker 단위 테스트 보완
- 테스트 픽스처 관리
- 테스트 유틸리티 제공
- Mock/Stub 구현
- 테스트 커버리지 관리

### 1.2 역할 분담

| 테스트 유형 | 담당자 | 설명 |
|------------|--------|------|
| 단위 테스트 (기본) | 각 Worker | 구현과 함께 작성 |
| 단위 테스트 (보완) | **Test Worker** | 커버리지 보완 |
| 통합 테스트 | Tester Agent | 컴포넌트 연동 |
| E2E 테스트 | Tester Agent | 전체 흐름 |

---

## 2. 테스트 보완 규칙

### 2.1 보완 대상 식별

```markdown
## 테스트 보완 분석

### 현재 커버리지
- 라인: 75%
- 브랜치: 62%

### 미커버 영역
| 클래스 | 메서드 | 미커버 사유 |
|--------|--------|------------|
| OrderCommandService | cancel() | 에러 케이스 누락 |
| PricingCoordinator | calculatePrice() | 할인 없는 경우 |

### 보완 계획
1. OrderCommandService.cancel() 에러 케이스 추가
2. PricingCoordinator 할인 없는 케이스 추가
```

### 2.2 보완 테스트 작성

```kotlin
// 기존 Worker가 작성한 테스트
class OrderCommandServiceTest {
    @Test
    fun `should cancel pending order`() { /* ... */ }
}

// Test Worker가 보완한 테스트
class OrderCommandServiceTest {
    // ... 기존 테스트 ...

    // 보완: 에러 케이스
    @Test
    fun `should throw when cancelling non-existent order`() {
        // Given
        val nonExistentId = 999L

        // When & Then
        assertThrows<OrderException.NotFound> {
            orderCommandService.cancel(nonExistentId)
        }
    }

    @Test
    fun `should throw when cancelling completed order`() {
        // Given
        val order = createCompletedOrder()

        // When & Then
        assertThrows<OrderException.InvalidStatus> {
            orderCommandService.cancel(order.id)
        }
    }
}
```

---

## 3. 테스트 픽스처 관리

### 3.1 Fixture 클래스

```kotlin
object OrderFixture {

    fun createOrder(
        id: Long = 1L,
        userId: Long = 1L,
        status: OrderStatus = OrderStatus.PENDING,
        totalPrice: Money = Money(BigDecimal("10000"))
    ): Order {
        return Order(
            id = id,
            userId = userId,
            status = status,
            totalPrice = totalPrice,
            createdAt = LocalDateTime.now()
        )
    }

    fun createPendingOrder(userId: Long = 1L) = createOrder(status = OrderStatus.PENDING, userId = userId)
    fun createCompletedOrder(userId: Long = 1L) = createOrder(status = OrderStatus.COMPLETED, userId = userId)
    fun createCancelledOrder(userId: Long = 1L) = createOrder(status = OrderStatus.CANCELLED, userId = userId)
}
```

### 3.2 Request Fixture

```kotlin
object OrderRequestFixture {

    fun createOrderRequest(
        userId: Long = 1L,
        productId: Long = 1L,
        quantity: Int = 1
    ): CreateOrderRequest {
        return CreateOrderRequest(
            userId = userId,
            productId = productId,
            quantity = quantity
        )
    }

    fun createInvalidRequest() = createOrderRequest(quantity = -1)
}
```

---

## 4. Mock/Stub 관리

### 4.1 공통 Mock 설정

```kotlin
abstract class MockServiceTestBase {

    @Mock
    protected lateinit var orderRepository: OrderRepository

    @Mock
    protected lateinit var productQueryService: ProductQueryService

    @Mock
    protected lateinit var eventPublisher: ApplicationEventPublisher

    @BeforeEach
    fun setupMocks() {
        MockitoAnnotations.openMocks(this)
    }

    protected fun mockOrderExists(order: Order) {
        whenever(orderRepository.findByIdOrNull(order.id)).thenReturn(order)
    }

    protected fun mockOrderNotExists(orderId: Long) {
        whenever(orderRepository.findByIdOrNull(orderId)).thenReturn(null)
    }

    protected fun mockProductExists(product: Product) {
        whenever(productQueryService.findById(product.id)).thenReturn(product)
    }
}
```

### 4.2 Stub 구현

```kotlin
class StubPaymentService : PaymentService {

    private val responses = mutableMapOf<Long, PaymentResult>()

    fun stubSuccess(orderId: Long) {
        responses[orderId] = PaymentResult.success(transactionId = "TXN-$orderId")
    }

    fun stubFailure(orderId: Long, reason: String) {
        responses[orderId] = PaymentResult.failure(reason)
    }

    override fun process(orderId: Long, amount: Money): PaymentResult {
        return responses[orderId] ?: PaymentResult.success()
    }
}
```

---

## 5. 테스트 유틸리티

### 5.1 Assertion 유틸리티

```kotlin
object OrderAssertions {

    fun assertOrderCreated(order: Order) {
        assertThat(order.id).isNotNull()
        assertThat(order.status).isEqualTo(OrderStatus.PENDING)
        assertThat(order.createdAt).isNotNull()
    }

    fun assertOrderCompleted(order: Order) {
        assertThat(order.status).isEqualTo(OrderStatus.COMPLETED)
        assertThat(order.completedAt).isNotNull()
    }

    fun assertPriceCalculation(result: PriceResult, expectedTotal: BigDecimal) {
        assertThat(result.totalPrice.amount).isEqualByComparingTo(expectedTotal)
        assertThat(result.basePrice.amount)
            .isGreaterThanOrEqualTo(result.totalPrice.amount.subtract(result.tax.amount))
    }
}
```

### 5.2 테스트 데이터 빌더

```kotlin
class OrderBuilder {
    private var id: Long = 1L
    private var userId: Long = 1L
    private var status: OrderStatus = OrderStatus.PENDING
    private var items: MutableList<OrderItem> = mutableListOf()

    fun withId(id: Long) = apply { this.id = id }
    fun withUserId(userId: Long) = apply { this.userId = userId }
    fun withStatus(status: OrderStatus) = apply { this.status = status }
    fun withItem(productId: Long, quantity: Int, price: Money) = apply {
        items.add(OrderItem(productId, quantity, price))
    }

    fun build(): Order {
        return Order(
            id = id,
            userId = userId,
            status = status,
            items = items,
            createdAt = LocalDateTime.now()
        )
    }
}

// 사용 예
val order = OrderBuilder()
    .withUserId(123L)
    .withStatus(OrderStatus.PENDING)
    .withItem(1L, 2, Money(BigDecimal("5000")))
    .build()
```

---

## 6. 테스트 카테고리 관리

### 6.1 테스트 태그

```kotlin
@Target(AnnotationTarget.CLASS, AnnotationTarget.FUNCTION)
@Retention(AnnotationRetention.RUNTIME)
@Tag("slow")
annotation class SlowTest

@Target(AnnotationTarget.CLASS, AnnotationTarget.FUNCTION)
@Retention(AnnotationRetention.RUNTIME)
@Tag("integration")
annotation class IntegrationTest

@Target(AnnotationTarget.CLASS, AnnotationTarget.FUNCTION)
@Retention(AnnotationRetention.RUNTIME)
@Tag("database")
annotation class DatabaseTest
```

### 6.2 테스트 프로파일

```kotlin
// 빠른 테스트만 실행
@ActiveProfiles("test-fast")
class FastTestConfig

// 전체 테스트 실행
@ActiveProfiles("test-full")
class FullTestConfig
```

---

## 7. 테스트 커버리지 관리

### 7.1 커버리지 목표 검증

```kotlin
// build.gradle.kts
tasks.jacocoTestCoverageVerification {
    violationRules {
        rule {
            element = "CLASS"
            includes = listOf("com.example.domain.*")

            limit {
                counter = "LINE"
                minimum = "0.80".toBigDecimal()
            }

            limit {
                counter = "BRANCH"
                minimum = "0.70".toBigDecimal()
            }
        }
    }
}
```

### 7.2 커버리지 리포트

```markdown
## 커버리지 리포트

### 레이어별 커버리지
| 레이어 | 라인 | 브랜치 | 목표 달성 |
|--------|------|--------|----------|
| L1 Domain | 92% | 85% | ✅ |
| LM Shared | 88% | 78% | ✅ |
| L2 Feature | 82% | 74% | ✅ |
| L3 Router | 75% | 68% | ✅ |

### 보완 필요 영역
- CheckoutCompositeService.handlePaymentFailure(): 브랜치 50%
- OrderController.handleError(): 라인 60%
```

---

## 8. 파일 구조

```
test/
├── fixtures/
│   ├── OrderFixture.kt
│   ├── ProductFixture.kt
│   └── UserFixture.kt
├── builders/
│   ├── OrderBuilder.kt
│   └── ProductBuilder.kt
├── mocks/
│   ├── MockServiceTestBase.kt
│   └── StubPaymentService.kt
├── utils/
│   ├── OrderAssertions.kt
│   └── TestDataGenerator.kt
└── config/
    ├── TestConfig.kt
    └── TestProfiles.kt
```

---

## 9. 완료 체크리스트

- [ ] 커버리지 분석 완료
- [ ] 미커버 영역 식별
- [ ] 보완 테스트 작성
- [ ] 테스트 픽스처 정리
- [ ] Mock/Stub 구현
- [ ] 테스트 유틸리티 제공
- [ ] 커버리지 목표 달성 확인

---

## 10. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 커버리지 목표 미달 | Worker | 추가 테스트 요청 |
| 테스트 불가 코드 | Architect | 코드 리팩토링 검토 |
| 테스트 환경 이슈 | Manager | 환경 설정 요청 |
| Flaky 테스트 | Worker | 테스트 안정화 |

---

<!-- pal:convention:workers:backend:test -->
