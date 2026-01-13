# Router Worker Rules

> Claude 참조용 L3 HTTP API 규칙

---

## Quick Reference

```
Layer: L3 Router
Tech: Spring MVC / WebFlux + Spring Security (Kotlin)
Pattern: Controller + DTO + ExceptionHandler
Auth: @PreAuthorize, @AuthenticationPrincipal
```

---

## Directory Structure

```
api/
└── v1/
    ├── controller/    # @RestController
    ├── request/       # Request DTO with validation
    ├── response/      # Response DTO
    └── exception/     # @ControllerAdvice, ErrorResponse
```

---

## Dependency Rules

```
L3 Controller can reference:
✅ L2 CompositeService
✅ L1 QueryService (simple read-only)
✅ LM Coordinator
❌ L1 CommandService directly (use L2)
```

---

## Controller Rules

### Basic Structure
```kotlin
@RestController
@RequestMapping("/api/v1/orders")
@Tag(name = "Orders", description = "Order API")
class OrderController(
    private val checkoutCompositeService: CheckoutCompositeService,
    private val orderQueryService: OrderQueryService
) {
    @PostMapping
    @Operation(summary = "Create order")
    @PreAuthorize("isAuthenticated()")
    fun createOrder(
        @Valid @RequestBody request: CreateOrderApiRequest,
        @AuthenticationPrincipal user: UserPrincipal
    ): ResponseEntity<OrderApiResponse> {
        val result = checkoutCompositeService.checkout(
            request.toCheckoutRequest(user.id)
        )
        return ResponseEntity
            .status(HttpStatus.CREATED)
            .body(OrderApiResponse.from(result))
    }

    @GetMapping("/{id}")
    @Operation(summary = "Get order")
    fun getOrder(
        @PathVariable id: Long,
        @AuthenticationPrincipal user: UserPrincipal
    ): ResponseEntity<OrderApiResponse> {
        val order = orderQueryService.findById(id)
            ?: throw OrderNotFoundException(id)
        return ResponseEntity.ok(OrderApiResponse.from(order))
    }
}
```

### Naming Convention
| Component | Pattern | Example |
|-----------|---------|---------|
| Controller | `{Feature}Controller` | `OrderController` |
| Request DTO | `{Action}{Feature}ApiRequest` | `CreateOrderApiRequest` |
| Response DTO | `{Feature}ApiResponse` | `OrderApiResponse` |
| Error | `ErrorResponse` | Common |

---

## REST API Rules

| Action | Method | URL Pattern | Response Code |
|--------|--------|-------------|---------------|
| List | GET | /resources | 200, 204 |
| Get | GET | /resources/{id} | 200, 404 |
| Create | POST | /resources | 201 |
| Update | PUT | /resources/{id} | 200, 202 |
| Patch | PATCH | /resources/{id} | 200, 202 |
| Delete | DELETE | /resources/{id} | 200, 202 |

---

## Request DTO Rules

### With Validation
```kotlin
data class CreateOrderApiRequest(
    @field:NotNull
    @field:Positive
    val productId: Long,

    @field:NotNull
    @field:Min(1)
    @field:Max(100)
    val quantity: Int,

    @field:NotBlank
    val shippingAddress: String,

    val couponCode: String? = null
) {
    fun toCheckoutRequest(userId: Long): CheckoutRequest {
        return CheckoutRequest(
            userId = userId,
            productId = productId,
            quantity = quantity
        )
    }
}
```

---

## Response DTO Rules

### Basic Response
```kotlin
data class OrderApiResponse(
    val id: Long,
    val status: String,
    val totalPrice: MoneyResponse,
    val createdAt: String
) {
    companion object {
        fun from(order: Order): OrderApiResponse {
            return OrderApiResponse(
                id = order.id,
                status = order.status.name,
                totalPrice = MoneyResponse.from(order.totalPrice),
                createdAt = order.createdAt.format(DateTimeFormatter.ISO_DATE_TIME)
            )
        }
    }
}
```

### Paged Response
```kotlin
data class PagedResponse<T>(
    val content: List<T>,
    val page: Int,
    val size: Int,
    val totalElements: Long,
    val totalPages: Int,
    val hasNext: Boolean
) {
    companion object {
        fun <T, R> from(page: Page<T>, mapper: (T) -> R): PagedResponse<R> {
            return PagedResponse(
                content = page.content.map(mapper),
                page = page.number,
                size = page.size,
                totalElements = page.totalElements,
                totalPages = page.totalPages,
                hasNext = page.hasNext()
            )
        }
    }
}
```

---

## Exception Handler Rules

### Global Handler
```kotlin
@RestControllerAdvice
class GlobalExceptionHandler {

    @ExceptionHandler(OrderNotFoundException::class)
    fun handleNotFound(e: OrderNotFoundException): ResponseEntity<ErrorResponse> {
        return ResponseEntity
            .status(HttpStatus.NOT_FOUND)
            .body(ErrorResponse(
                code = "ORDER_NOT_FOUND",
                message = e.message ?: "Order not found"
            ))
    }

    @ExceptionHandler(MethodArgumentNotValidException::class)
    fun handleValidation(e: MethodArgumentNotValidException): ResponseEntity<ErrorResponse> {
        val errors = e.bindingResult.fieldErrors.map {
            FieldError(it.field, it.defaultMessage ?: "Invalid value")
        }
        return ResponseEntity
            .status(HttpStatus.BAD_REQUEST)
            .body(ErrorResponse(
                code = "VALIDATION_ERROR",
                message = "Validation failed",
                errors = errors
            ))
    }

    @ExceptionHandler(AccessDeniedException::class)
    fun handleAccessDenied(e: AccessDeniedException): ResponseEntity<ErrorResponse> {
        return ResponseEntity
            .status(HttpStatus.FORBIDDEN)
            .body(ErrorResponse(code = "ACCESS_DENIED", message = e.message ?: "Access denied"))
    }
}
```

### Error Response DTO
```kotlin
data class ErrorResponse(
    val code: String,
    val message: String,
    val errors: List<FieldError>? = null,
    val timestamp: String = Instant.now().toString()
)

data class FieldError(
    val field: String,
    val message: String
)
```

---

## Auth Rules

### @PreAuthorize
```kotlin
@PreAuthorize("isAuthenticated()")       // Login required
@PreAuthorize("hasRole('USER')")         // Role check
@PreAuthorize("hasRole('ADMIN')")        // Admin only
```

### @AuthenticationPrincipal
```kotlin
@GetMapping("/{id}")
fun getOrder(
    @PathVariable id: Long,
    @AuthenticationPrincipal user: UserPrincipal
): ResponseEntity<OrderApiResponse> {
    val order = orderQueryService.findById(id)
        ?: throw OrderNotFoundException(id)

    // Resource ownership check
    if (order.userId != user.id && !user.isAdmin()) {
        throw AccessDeniedException("Not authorized")
    }

    return ResponseEntity.ok(OrderApiResponse.from(order))
}
```

---

## API Versioning

### URL-based (Preferred)
```kotlin
@RestController
@RequestMapping("/api/v1/orders")
class OrderV1Controller { }

@RestController
@RequestMapping("/api/v2/orders")
class OrderV2Controller { }
```

---

## Checklist

### Controller
- [ ] `@RestController` annotation
- [ ] Proper `@RequestMapping` with version
- [ ] OpenAPI annotations (`@Tag`, `@Operation`)
- [ ] Auth annotations (`@PreAuthorize`)

### Request DTO
- [ ] Jakarta Validation annotations
- [ ] `toXxxRequest()` conversion method
- [ ] Proper nullability

### Response DTO
- [ ] `from()` companion function
- [ ] ISO format for dates

### Exception
- [ ] Global `@ControllerAdvice`
- [ ] Proper HTTP status codes
- [ ] Consistent error response format

---

## Escalation

| Situation | Target | Action |
|-----------|--------|--------|
| API design change | Architect | Versioning strategy |
| Auth method change | Architect | Security architecture |
| Complex permission | Architect | Permission model design |
| API performance | Architect | Caching/pagination strategy |

---

<!-- pal:rules:workers:backend:router -->
