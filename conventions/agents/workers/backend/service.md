# Service Worker 컨벤션

> LM/L2 레이어 - 서비스 조합 전문 Worker

---

## 1. 역할 정의

Service Worker는 **LM(Shared Composition) 및 L2(Feature Composition) 레이어**에서 여러 도메인 서비스를 조합하는 전문 Worker입니다.

### 1.1 담당 영역

| 레이어 | 역할                                         | 산출물                   |
|--------|--------------------------------------------|-----------------------|
| **LM** | 공유 가능한 도메인 조합, L2 내부에서 선언되는 Event의 명시적 수신처 | Coordinator, Listener |
| **L2** | Feature 단위 비즈니스 조합                         | CompositeService      |

### 1.2 LM vs L2 구분

```
LM (Shared Composition)
├── 여러 Feature에서 공유
├── 범용적 조합 로직
└── 예: PricingCoordinator, InventoryCoordinator
└──    CustomEventListener, CustomEventListnerService

L2 (Feature Composition)
├── 특정 Feature 전용
├── 비즈니스 플로우 조합
└── 예: CheckoutCompositeService, OrderCompositeService
```

---

## 2. LM 레이어 규칙

### 2.1 Coordinator 패턴

```kotlin
@Service
@Transactional
class PricingCoordinator(
    private val productQueryService: ProductQueryService,
    private val discountQueryService: DiscountQueryService,
    private val taxCalculator: TaxCalculator
) {
    /**
     * 여러 도메인의 정보를 조합하여 최종 가격 계산
     */
    fun calculatePrice(productId: Long, quantity: Int): PriceResult {
        // L1 서비스들 호출
        val product = productQueryService.findById(productId)
            ?: throw ProductException.NotFound(productId)

        val discount = discountQueryService.findActiveDiscount(productId)

        // 조합 로직
        val basePrice = product.price * quantity
        val discountedPrice = discount?.apply(basePrice) ?: basePrice
        val tax = taxCalculator.calculate(discountedPrice)

        return PriceResult(
            basePrice = basePrice,
            discount = basePrice - discountedPrice,
            tax = tax,
            totalPrice = discountedPrice + tax
        )
    }
}
```

### 2.2 LM 의존성 규칙

```
LM Coordinator는:
✅ L1 서비스 참조 가능 (여러 도메인)
✅ Infrastructure 참조 가능
⚠️ 다른 LM은 단방향만 가능 (순환 금지)
❌ L2 참조 불가
```

---

## 3. L2 레이어 규칙

### 3.1 CompositeService 패턴

```kotlin
@Service
@Transactional
class CheckoutCompositeService(
    private val orderCommandService: OrderCommandService,
    private val orderQueryService: OrderQueryService,
    private val pricingCoordinator: PricingCoordinator,
    private val paymentService: PaymentService,
    private val eventPublisher: ApplicationEventPublisher
) {
    /**
     * 체크아웃 전체 플로우 조합
     */
    fun checkout(request: CheckoutRequest): CheckoutResult {
        // 1. 가격 계산 (LM 호출)
        val priceResult = pricingCoordinator.calculatePrice(
            request.productId,
            request.quantity
        )

        // 2. 주문 생성 (L1 호출)
        val orderId = orderCommandService.create(
            CreateOrderRequest(
                userId = request.userId,
                productId = request.productId,
                quantity = request.quantity,
                totalPrice = priceResult.totalPrice
            )
        )

        // 3. 결제 처리
        val paymentResult = paymentService.process(
            orderId = orderId,
            amount = priceResult.totalPrice
        )

        // 4. 주문 완료 처리
        if (paymentResult.isSuccess) {
            orderCommandService.complete(orderId)
            eventPublisher.publishEvent(OrderCompletedEvent(orderId))
        } else {
            orderCommandService.cancel(orderId)
        }

        return CheckoutResult(
            orderId = orderId,
            paymentResult = paymentResult,
            priceResult = priceResult
        )
    }
}
```

### 3.2 L2 의존성 규칙

```
L2 CompositeService는:
✅ L1 서비스 참조 가능
✅ LM Coordinator 참조 가능
✅ Infrastructure 참조 가능
❌ 다른 L2 직접 참조 불가 (Event로만)
```

---

## 4. 트랜잭션 관리

### 4.1 트랜잭션 경계

```kotlin
@Service
class CheckoutCompositeService(
    private val orderCommandService: OrderCommandService,
    private val paymentService: PaymentService,
    private val transactionTemplate: TransactionTemplate
) {
    fun checkout(request: CheckoutRequest): CheckoutResult {
        // 주문 생성은 트랜잭션 내
        val orderId = transactionTemplate.execute {
            orderCommandService.create(request.toOrderRequest())
        }!!

        // 결제는 트랜잭션 외부 (외부 API)
        val paymentResult = paymentService.process(orderId, request.amount)

        // 결과에 따라 별도 트랜잭션
        transactionTemplate.execute {
            if (paymentResult.isSuccess) {
                orderCommandService.complete(orderId)
            } else {
                orderCommandService.cancel(orderId)
            }
        }

        return CheckoutResult(orderId, paymentResult)
    }
}
```

### 4.2 보상 트랜잭션 (Saga)

```kotlin
@Service
class OrderSagaCompositeService(
    private val orderCommandService: OrderCommandService,
    private val inventoryService: InventoryService,
    private val paymentService: PaymentService
) {
    fun processOrder(request: OrderRequest): OrderResult {
        var orderId: Long? = null
        var inventoryReserved = false

        try {
            // Step 1: 주문 생성
            orderId = orderCommandService.create(request.toOrderRequest())

            // Step 2: 재고 예약
            inventoryService.reserve(request.productId, request.quantity)
            inventoryReserved = true

            // Step 3: 결제 처리
            val paymentResult = paymentService.process(orderId, request.amount)

            if (!paymentResult.isSuccess) {
                throw PaymentFailedException(paymentResult.errorMessage)
            }

            // Step 4: 완료
            orderCommandService.complete(orderId)

            return OrderResult.success(orderId)

        } catch (e: Exception) {
            // 보상 트랜잭션
            if (inventoryReserved) {
                inventoryService.release(request.productId, request.quantity)
            }
            orderId?.let { orderCommandService.cancel(it) }

            return OrderResult.failure(e.message)
        }
    }
}
```

---

## 5. 이벤트 기반 연동

### 5.1 다른 L2와의 통신

```kotlin
// L2-Checkout에서 이벤트 발행
@Service
class CheckoutCompositeService(
    private val eventPublisher: ApplicationEventPublisher
) {
    fun checkout(request: CheckoutRequest): CheckoutResult {
        // ... 체크아웃 로직 ...

        // 다른 L2에게 이벤트로 알림
        eventPublisher.publishEvent(
            CheckoutCompletedEvent(
                orderId = orderId,
                userId = request.userId
            )
        )

        return result
    }
}

// L2-Notification에서 이벤트 수신
@Service
class NotificationCompositeService {

    @EventListener
    fun onCheckoutCompleted(event: CheckoutCompletedEvent) {
        // 알림 처리
        sendOrderConfirmation(event.orderId, event.userId)
    }
}
```

---

## 6. DTO 규칙

### 6.1 Request/Response DTO

```kotlin
// Request DTO
data class CheckoutRequest(
    val userId: Long,
    val productId: Long,
    val quantity: Int,
    val paymentMethod: PaymentMethod
)

// Response DTO
data class CheckoutResult(
    val orderId: Long,
    val paymentResult: PaymentResult,
    val priceResult: PriceResult
) {
    val isSuccess: Boolean
        get() = paymentResult.isSuccess
}

// 내부 결과 DTO
data class PriceResult(
    val basePrice: Money,
    val discount: Money,
    val tax: Money,
    val totalPrice: Money
)
```

---

## 7. 단위 테스트

### 7.1 Coordinator 테스트

```kotlin
class PricingCoordinatorTest {

    @Mock lateinit var productQueryService: ProductQueryService
    @Mock lateinit var discountQueryService: DiscountQueryService
    @Mock lateinit var taxCalculator: TaxCalculator

    @InjectMocks lateinit var pricingCoordinator: PricingCoordinator

    @Test
    fun `should calculate price with discount`() {
        // Given
        val product = Product(price = Money(BigDecimal("10000")))
        val discount = Discount(rate = 0.1) // 10% 할인

        whenever(productQueryService.findById(1L)).thenReturn(product)
        whenever(discountQueryService.findActiveDiscount(1L)).thenReturn(discount)
        whenever(taxCalculator.calculate(any())).thenReturn(Money(BigDecimal("900")))

        // When
        val result = pricingCoordinator.calculatePrice(1L, 1)

        // Then
        assertThat(result.basePrice.amount).isEqualByComparingTo("10000")
        assertThat(result.discount.amount).isEqualByComparingTo("1000")
        assertThat(result.totalPrice.amount).isEqualByComparingTo("9900")
    }
}
```

### 7.2 CompositeService 테스트

```kotlin
class CheckoutCompositeServiceTest {

    @Mock lateinit var orderCommandService: OrderCommandService
    @Mock lateinit var pricingCoordinator: PricingCoordinator
    @Mock lateinit var paymentService: PaymentService

    @InjectMocks lateinit var checkoutService: CheckoutCompositeService

    @Test
    fun `should complete checkout when payment succeeds`() {
        // Given
        val request = CheckoutRequest(userId = 1L, productId = 1L, quantity = 1)
        val priceResult = PriceResult(totalPrice = Money(BigDecimal("10000")))

        whenever(pricingCoordinator.calculatePrice(1L, 1)).thenReturn(priceResult)
        whenever(orderCommandService.create(any())).thenReturn(100L)
        whenever(paymentService.process(100L, priceResult.totalPrice))
            .thenReturn(PaymentResult.success())

        // When
        val result = checkoutService.checkout(request)

        // Then
        assertThat(result.isSuccess).isTrue()
        verify(orderCommandService).complete(100L)
    }

    @Test
    fun `should cancel order when payment fails`() {
        // Given
        val request = CheckoutRequest(userId = 1L, productId = 1L, quantity = 1)

        whenever(orderCommandService.create(any())).thenReturn(100L)
        whenever(paymentService.process(any(), any()))
            .thenReturn(PaymentResult.failure("Insufficient balance"))

        // When
        val result = checkoutService.checkout(request)

        // Then
        assertThat(result.isSuccess).isFalse()
        verify(orderCommandService).cancel(100L)
    }
}
```

---

## 8. 파일 구조

```
# LM 레이어
shared/
└── pricing/
    ├── PricingCoordinator.kt
    ├── PriceResult.kt
    └── TaxCalculator.kt

# L2 레이어
feature/
└── checkout/
    ├── CheckoutCompositeService.kt
    ├── CheckoutRequest.kt
    ├── CheckoutResult.kt
    └── CheckoutCompletedEvent.kt
```

---

## 9. 완료 체크리스트

### LM 작업 시
- [ ] Coordinator 구현
- [ ] 도메인 서비스 조합 로직
- [ ] DTO 정의
- [ ] 단위 테스트 작성

### L2 작업 시
- [ ] CompositeService 구현
- [ ] 비즈니스 플로우 조합
- [ ] 트랜잭션 경계 정의
- [ ] 이벤트 발행 (필요시)
- [ ] Request/Response DTO 정의
- [ ] 단위 테스트 작성

---

## 10. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| L2→L2 직접 참조 필요 | Architect | Event 또는 LM으로 리팩토링 |
| 트랜잭션 경계 복잡 | Architect | Saga 패턴 검토 |
| 외부 API 장애 처리 | Architect | Circuit Breaker 검토 |
| 성능 이슈 | Architect | 비동기 처리 검토 |

---

<!-- pal:convention:workers:backend:service -->
