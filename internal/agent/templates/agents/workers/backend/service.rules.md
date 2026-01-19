# Service Worker Rules

> Claude 참조용 LM/L2 서비스 조합 규칙

---

## Quick Reference

```
Layer: LM (Shared) / L2 (Feature)
Tech: Spring Boot + Spring TX (Kotlin)
Pattern: Coordinator (LM) / CompositeService (L2)
TX: Method-level transaction boundary
```

---

## Layer Definition

```
LM (Shared Composition)
├── Multiple Features share
├── Generic composition logic
├── Event listeners
└── Examples: PricingCoordinator, InventoryCoordinator

L2 (Feature Composition)
├── Feature-specific
├── Business flow orchestration
└── Examples: CheckoutCompositeService, OrderCompositeService
```

---

## Directory Structure

```
# LM Layer
shared/
└── pricing/
    ├── PricingCoordinator.kt
    ├── PriceResult.kt
    └── TaxCalculator.kt

# L2 Layer
feature/
└── checkout/
    ├── CheckoutCompositeService.kt
    ├── CheckoutRequest.kt
    ├── CheckoutResult.kt
    └── CheckoutCompletedEvent.kt
```

---

## Dependency Rules

### LM Coordinator
```
LM Coordinator can reference:
✅ L1 Services (multiple domains)
✅ Infrastructure
⚠️ Other LM (one-way only, no circular)
❌ L2 Services
```

### L2 CompositeService
```
L2 CompositeService can reference:
✅ L1 Services
✅ LM Coordinators
✅ Infrastructure
❌ Other L2 Services (event only)
```

---

## LM Coordinator Pattern

```kotlin
@Service
@Transactional(readOnly = true)
class PricingCoordinator(
    private val productQueryService: ProductQueryService,    // L1
    private val discountQueryService: DiscountQueryService,  // L1
    private val taxCalculator: TaxCalculator
) {
    fun calculatePrice(productId: Long, quantity: Int): PriceResult {
        // L1 calls
        val product = productQueryService.findById(productId)
            ?: throw ProductException.NotFound(productId)

        val discount = discountQueryService.findActiveDiscount(productId)

        // Composition logic
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

---

## L2 CompositeService Pattern

```kotlin
@Service
class CheckoutCompositeService(
    private val pricingCoordinator: PricingCoordinator,       // LM
    private val orderCommandService: OrderCommandService,     // L1
    private val paymentService: PaymentService,               // L1
    private val eventPublisher: ApplicationEventPublisher
) {
    @Transactional
    fun checkout(request: CheckoutRequest): CheckoutResult {
        // 1. Price calculation (LM)
        val priceResult = pricingCoordinator.calculatePrice(
            request.productId,
            request.quantity
        )

        // 2. Create order (L1)
        val orderId = orderCommandService.create(
            CreateOrderRequest(
                userId = request.userId,
                productId = request.productId,
                quantity = request.quantity,
                totalPrice = priceResult.totalPrice
            )
        )

        // 3. Payment (L1)
        val paymentResult = paymentService.process(
            orderId = orderId,
            amount = priceResult.totalPrice
        )

        // 4. Complete or cancel
        if (paymentResult.isSuccess) {
            orderCommandService.complete(orderId)
            eventPublisher.publishEvent(CheckoutCompletedEvent(orderId))
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

---

## Transaction Management

### Basic Rules
- **LM Coordinator**: `@Transactional(readOnly = true)` for queries
- **L2 CompositeService**: `@Transactional` at method level
- **External API**: Outside transaction boundary

### Split Transaction (External API)
```kotlin
@Service
class CheckoutCompositeService(
    private val orderCommandService: OrderCommandService,
    private val paymentService: PaymentService,
    private val transactionTemplate: TransactionTemplate
) {
    fun checkout(request: CheckoutRequest): CheckoutResult {
        // TX 1: Create order
        val orderId = transactionTemplate.execute {
            orderCommandService.create(request.toOrderRequest())
        }!!

        // External API (no TX)
        val paymentResult = paymentService.process(orderId, request.amount)

        // TX 2: Update status
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

---

## Saga Pattern (Compensation)

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
            // Step 1: Create order
            orderId = orderCommandService.create(request.toOrderRequest())

            // Step 2: Reserve inventory
            inventoryService.reserve(request.productId, request.quantity)
            inventoryReserved = true

            // Step 3: Payment
            val paymentResult = paymentService.process(orderId, request.amount)
            if (!paymentResult.isSuccess) {
                throw PaymentFailedException(paymentResult.errorMessage)
            }

            // Step 4: Complete
            orderCommandService.complete(orderId)
            return OrderResult.success(orderId)

        } catch (e: Exception) {
            // Compensation
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

## Event-Based L2 Communication

### Publisher (L2-Checkout)
```kotlin
@Service
class CheckoutCompositeService(
    private val eventPublisher: ApplicationEventPublisher
) {
    fun checkout(request: CheckoutRequest): CheckoutResult {
        // ... checkout logic ...

        // Publish event to other L2
        eventPublisher.publishEvent(
            CheckoutCompletedEvent(orderId = orderId, userId = request.userId)
        )

        return result
    }
}
```

### Listener (LM or L2-Notification)
```kotlin
@Service
class NotificationEventListener {

    @EventListener
    fun onCheckoutCompleted(event: CheckoutCompletedEvent) {
        sendOrderConfirmation(event.orderId, event.userId)
    }
}
```

---

## DTO Rules

### Request DTO
```kotlin
data class CheckoutRequest(
    val userId: Long,
    val productId: Long,
    val quantity: Int,
    val paymentMethod: PaymentMethod
)
```

### Result DTO
```kotlin
data class CheckoutResult(
    val orderId: Long,
    val paymentResult: PaymentResult,
    val priceResult: PriceResult
) {
    val isSuccess: Boolean
        get() = paymentResult.isSuccess
}
```

---

## Checklist

### LM Coordinator
- [ ] `@Transactional(readOnly = true)` for queries
- [ ] Only L1 service references
- [ ] No L2 references
- [ ] Composition logic implemented

### L2 CompositeService
- [ ] `@Transactional` at method level
- [ ] L1 and LM references only
- [ ] No direct L2 references (use events)
- [ ] Proper transaction boundary
- [ ] Event publishing if needed

### Common
- [ ] Request/Result DTO defined
- [ ] Unit tests with mocks
- [ ] Error handling strategy

---

## Escalation

| Situation | Target | Action |
|-----------|--------|--------|
| L2 → L2 direct reference needed | Architect | Event or LM refactoring |
| Complex transaction boundary | Architect | Saga pattern review |
| External API failure handling | Architect | Circuit breaker review |
| Performance issue | Architect | Async processing review |

---

<!-- pal:rules:workers:backend:service -->
