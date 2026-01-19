# Test Worker Rules

> Claude 참조용 테스트 보완 및 인프라 규칙

---

## Quick Reference

```
Layer: Test
Tech: JUnit5 + MockK/Mockito + AssertJ + Spring Test (Kotlin)
Pattern: Given-When-Then
Role: Worker test supplement + Test infrastructure
```

---

## Role Definition

| Test Type | Owner | Description |
|-----------|-------|-------------|
| Unit Test (basic) | Each Worker | Written with implementation |
| Unit Test (supplement) | **Test Worker** | Coverage supplement |
| Integration Test | Tester Agent | Component integration |
| E2E Test | Tester Agent | Full flow |

---

## Directory Structure

```
test/
├── fixtures/        # Test fixtures
│   ├── OrderFixture.kt
│   └── ProductFixture.kt
├── builders/        # Test data builders
│   └── OrderBuilder.kt
├── mocks/           # Mock/Stub implementations
│   ├── MockServiceTestBase.kt
│   └── StubPaymentService.kt
├── utils/           # Assertion utilities
│   └── OrderAssertions.kt
└── config/          # Test configuration
    └── TestConfig.kt
```

---

## Given-When-Then Pattern

```kotlin
@Test
fun `should create order when valid request`() {
    // Given
    val userId = createTestUser()
    val product = createTestProduct(price = 10000)
    val request = CreateOrderRequest(productId = product.id, quantity = 2)

    // When
    val result = orderCommandService.create(userId, request)

    // Then
    assertThat(result.id).isNotNull()
    assertThat(result.status).isEqualTo(OrderStatus.PENDING)
    assertThat(result.totalPrice.amount).isEqualByComparingTo("20000")
}
```

---

## Test Fixture Rules

### Object Fixture
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

    fun createPendingOrder(userId: Long = 1L) =
        createOrder(status = OrderStatus.PENDING, userId = userId)

    fun createCompletedOrder(userId: Long = 1L) =
        createOrder(status = OrderStatus.COMPLETED, userId = userId)

    fun createCancelledOrder(userId: Long = 1L) =
        createOrder(status = OrderStatus.CANCELLED, userId = userId)
}
```

### Request Fixture
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

## Test Builder Pattern

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

// Usage
val order = OrderBuilder()
    .withUserId(123L)
    .withStatus(OrderStatus.PENDING)
    .withItem(1L, 2, Money(BigDecimal("5000")))
    .build()
```

---

## Mock/Stub Rules

### Mock Test Base
```kotlin
abstract class MockServiceTestBase {

    @Mock
    protected lateinit var orderRepository: OrderRepository

    @Mock
    protected lateinit var productQueryService: ProductQueryService

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
}
```

### Stub Implementation
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

## Assertion Utilities

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
    }
}
```

---

## Test Supplement Rules

### Identify Coverage Gaps
```markdown
## Coverage Analysis

### Current Coverage
- Line: 75%
- Branch: 62%

### Uncovered Areas
| Class | Method | Reason |
|-------|--------|--------|
| OrderCommandService | cancel() | Error case missing |
| PricingCoordinator | calculatePrice() | No-discount case |
```

### Supplement Tests
```kotlin
// Original test by Worker
@Test
fun `should cancel pending order`() { /* ... */ }

// Supplemented by Test Worker
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
```

---

## Coverage Goals

| Layer | Line | Branch |
|-------|------|--------|
| L1 Domain | 90%+ | 80%+ |
| LM Shared | 85%+ | 75%+ |
| L2 Feature | 80%+ | 70%+ |
| L3 Router | 70%+ | 60%+ |

---

## Test Tags

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

---

## Checklist

### Coverage Analysis
- [ ] Run coverage report
- [ ] Identify uncovered areas
- [ ] Prioritize by importance

### Test Supplement
- [ ] Error cases covered
- [ ] Edge cases covered
- [ ] Null handling tested

### Test Infrastructure
- [ ] Fixtures created
- [ ] Builders created (if complex objects)
- [ ] Mocks/Stubs provided
- [ ] Assertion utilities provided

### Quality
- [ ] Given-When-Then pattern
- [ ] Descriptive test names
- [ ] No test interdependency
- [ ] No flaky tests

---

## Escalation

| Situation | Target | Action |
|-----------|--------|--------|
| Coverage goal not met | Worker | Request additional tests |
| Untestable code | Architect | Code refactoring review |
| Test environment issue | Manager | Environment setup |
| Flaky test | Worker | Test stabilization |

---

<!-- pal:rules:workers:backend:test -->
