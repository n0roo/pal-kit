# Router Worker 컨벤션

> L3 레이어 - API 라우팅 전문 Worker

---

## 1. 역할 정의

Router Worker는 **L3 Router 레이어**에서 HTTP API 엔드포인트를 구현하는 전문 Worker입니다.

### 1.1 담당 영역

- Controller 구현
- API 엔드포인트 정의
- Request/Response 변환
- 인증/인가 처리
- 예외 핸들링
- API 문서화 (OpenAPI)

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| 프레임워크 | Spring WebFlux, Spring MVC |
| 문서화 | SpringDoc OpenAPI |
| 검증 | Jakarta Validation |

---

## 2. L3 레이어 규칙

### 2.1 의존성 규칙

```
L3 Controller는:
✅ L2 CompositeService 참조 가능
✅ L1 QueryService 참조 가능 (단순 조회)
✅ LM Coordinator 참조 가능
❌ L1 CommandService 직접 참조 지양 (L2 통해서)
```

### 2.2 Controller 역할

```
HTTP Request → Controller → L2/L1 호출 → HTTP Response
        │           │              │            │
    검증/변환   라우팅/조합      비즈니스    결과 변환
```

---

## 3. Controller 구현 규칙

### 3.1 기본 Controller 구조

```kotlin
@RestController
@RequestMapping("/api/v1/orders")
@Tag(name = "Orders", description = "주문 관리 API")
class OrderController(
    private val checkoutCompositeService: CheckoutCompositeService,
    private val orderQueryService: OrderQueryService
) {
    @PostMapping
    @Operation(summary = "주문 생성", description = "새로운 주문을 생성합니다")
    @ApiResponses(
        ApiResponse(responseCode = "201", description = "주문 생성 성공"),
        ApiResponse(responseCode = "400", description = "잘못된 요청")
    )
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
    @Operation(summary = "주문 조회", description = "주문 상세 정보를 조회합니다")
    fun getOrder(
        @PathVariable id: Long,
        @AuthenticationPrincipal user: UserPrincipal
    ): ResponseEntity<OrderApiResponse> {
        val order = orderQueryService.findById(id)
            ?: throw OrderNotFoundException(id)

        // 권한 체크
        if (order.userId != user.id) {
            throw AccessDeniedException("Not authorized")
        }

        return ResponseEntity.ok(OrderApiResponse.from(order))
    }

    @GetMapping
    @Operation(summary = "주문 목록 조회")
    fun getOrders(
        @AuthenticationPrincipal user: UserPrincipal,
        @RequestParam(defaultValue = "0") page: Int,
        @RequestParam(defaultValue = "20") size: Int
    ): ResponseEntity<PagedResponse<OrderApiResponse>> {
        val orders = orderQueryService.findByUserId(user.id, PageRequest.of(page, size))
        return ResponseEntity.ok(PagedResponse.from(orders, OrderApiResponse::from))
    }
}
```

### 3.2 네이밍 컨벤션

| 구분 | 패턴 | 예시 |
|------|------|------|
| Controller | `{Feature}Controller` | `OrderController` |
| Request DTO | `{Action}{Feature}ApiRequest` | `CreateOrderApiRequest` |
| Response DTO | `{Feature}ApiResponse` | `OrderApiResponse` |
| 에러 Response | `ErrorResponse` | 공통 |

---

## 4. API DTO 규칙

### 4.1 Request DTO

```kotlin
data class CreateOrderApiRequest(
    @field:NotNull
    @field:Positive
    val productId: Long,

    @field:NotNull
    @field:Min(1)
    @field:Max(100)
    val quantity: Int,

    @field:NotNull
    val paymentMethod: PaymentMethod
) {
    fun toCheckoutRequest(userId: Long): CheckoutRequest {
        return CheckoutRequest(
            userId = userId,
            productId = productId,
            quantity = quantity,
            paymentMethod = paymentMethod
        )
    }
}
```

### 4.2 Response DTO

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

        fun from(result: CheckoutResult): OrderApiResponse {
            return OrderApiResponse(
                id = result.orderId,
                status = if (result.isSuccess) "COMPLETED" else "FAILED",
                totalPrice = MoneyResponse.from(result.priceResult.totalPrice),
                createdAt = Instant.now().toString()
            )
        }
    }
}

data class MoneyResponse(
    val amount: String,
    val currency: String
) {
    companion object {
        fun from(money: Money): MoneyResponse {
            return MoneyResponse(
                amount = money.amount.toPlainString(),
                currency = money.currency
            )
        }
    }
}
```

### 4.3 페이징 Response

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

## 5. 예외 핸들링

### 5.1 Global Exception Handler

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
            .body(ErrorResponse(
                code = "ACCESS_DENIED",
                message = e.message ?: "Access denied"
            ))
    }

    @ExceptionHandler(Exception::class)
    fun handleUnknown(e: Exception): ResponseEntity<ErrorResponse> {
        logger.error("Unexpected error", e)
        return ResponseEntity
            .status(HttpStatus.INTERNAL_SERVER_ERROR)
            .body(ErrorResponse(
                code = "INTERNAL_ERROR",
                message = "An unexpected error occurred"
            ))
    }
}
```

### 5.2 Error Response DTO

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

## 6. 인증/인가

### 6.1 인증 처리

```kotlin
@RestController
@RequestMapping("/api/v1/orders")
class OrderController(
    private val orderQueryService: OrderQueryService
) {
    @GetMapping("/{id}")
    @PreAuthorize("hasRole('USER')")
    fun getOrder(
        @PathVariable id: Long,
        @AuthenticationPrincipal user: UserPrincipal
    ): ResponseEntity<OrderApiResponse> {
        val order = orderQueryService.findById(id)
            ?: throw OrderNotFoundException(id)

        // 리소스 소유자 확인
        if (order.userId != user.id && !user.isAdmin()) {
            throw AccessDeniedException("Not authorized")
        }

        return ResponseEntity.ok(OrderApiResponse.from(order))
    }
}
```

### 6.2 UserPrincipal

```kotlin
data class UserPrincipal(
    val id: Long,
    val email: String,
    val roles: Set<String>
) {
    fun isAdmin(): Boolean = roles.contains("ADMIN")
}
```

---

## 7. API 버저닝

### 7.1 URL 기반 버저닝

```kotlin
// v1 API
@RestController
@RequestMapping("/api/v1/orders")
class OrderV1Controller { }

// v2 API (새 버전)
@RestController
@RequestMapping("/api/v2/orders")
class OrderV2Controller { }
```

### 7.2 하위 호환성

```kotlin
// v1 Response (유지)
data class OrderV1Response(
    val id: Long,
    val status: String,
    val price: String  // 단순 문자열
)

// v2 Response (개선)
data class OrderV2Response(
    val id: Long,
    val status: String,
    val price: MoneyResponse  // 구조화
)
```

---

## 8. 단위 테스트

### 8.1 Controller 테스트

```kotlin
@WebMvcTest(OrderController::class)
class OrderControllerTest {

    @Autowired
    lateinit var mockMvc: MockMvc

    @MockBean
    lateinit var checkoutCompositeService: CheckoutCompositeService

    @MockBean
    lateinit var orderQueryService: OrderQueryService

    @Test
    @WithMockUser(roles = ["USER"])
    fun `should create order successfully`() {
        // Given
        val request = CreateOrderApiRequest(
            productId = 1L,
            quantity = 2,
            paymentMethod = PaymentMethod.CARD
        )

        whenever(checkoutCompositeService.checkout(any()))
            .thenReturn(CheckoutResult(orderId = 100L, isSuccess = true))

        // When & Then
        mockMvc.perform(
            post("/api/v1/orders")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request))
        )
            .andExpect(status().isCreated)
            .andExpect(jsonPath("$.id").value(100))
    }

    @Test
    @WithMockUser(roles = ["USER"])
    fun `should return 400 for invalid request`() {
        val request = CreateOrderApiRequest(
            productId = -1L,  // Invalid
            quantity = 0,      // Invalid
            paymentMethod = PaymentMethod.CARD
        )

        mockMvc.perform(
            post("/api/v1/orders")
                .contentType(MediaType.APPLICATION_JSON)
                .content(objectMapper.writeValueAsString(request))
        )
            .andExpect(status().isBadRequest)
            .andExpect(jsonPath("$.code").value("VALIDATION_ERROR"))
    }
}
```

---

## 9. 파일 구조

```
api/
└── v1/
    ├── controller/
    │   └── OrderController.kt
    ├── request/
    │   └── CreateOrderApiRequest.kt
    ├── response/
    │   ├── OrderApiResponse.kt
    │   ├── MoneyResponse.kt
    │   └── PagedResponse.kt
    └── exception/
        ├── GlobalExceptionHandler.kt
        └── ErrorResponse.kt
```

---

## 10. 완료 체크리스트

- [ ] Controller 구현
- [ ] Request DTO 정의 (Validation 포함)
- [ ] Response DTO 정의
- [ ] 예외 핸들링 구현
- [ ] 인증/인가 적용
- [ ] OpenAPI 문서화
- [ ] Controller 테스트 작성

---

## 11. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| API 설계 변경 | Architect | API 버저닝 전략 |
| 인증 방식 변경 | Architect | 보안 구조 검토 |
| 복잡한 권한 체크 | Architect | 권한 모델 설계 |
| API 성능 이슈 | Architect | 캐싱/페이징 전략 |

---

<!-- pal:convention:workers:backend:router -->
