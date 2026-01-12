# Entity Worker 컨벤션

> L1 Domain 레이어 - JPA/ORM 엔티티 전문 Worker

---

## 1. 역할 정의

Entity Worker는 **L1 Domain 레이어**에서 JPA/ORM 기반 엔티티와 리포지토리를 구현하는 전문 Worker입니다.

### 1.1 담당 영역

- Entity 클래스 구현
- Repository 인터페이스 정의
- Value Object 구현
- CommandService / QueryService 구현
- 도메인 이벤트 정의

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| ORM | JPA, Hibernate, Exposed |
| DB | PostgreSQL, MySQL, H2 |
| 언어 | Kotlin, Java |

---

## 2. L1 레이어 규칙

### 2.1 의존성 규칙

```
L1 Domain은:
✅ Infrastructure(DB, 외부 시스템) 참조 가능
❌ 다른 L1 도메인 직접 참조 불가
❌ LM, L2 참조 불가
```

### 2.2 다른 도메인 참조 방법

```kotlin
// ❌ 잘못된 방법: 다른 도메인 직접 참조
class Order(
    val user: User  // User 도메인 직접 참조
)

// ✅ 올바른 방법: ID만 참조
class Order(
    val userId: UserId  // ID만 저장
)
```

---

## 3. Entity 구현 규칙

### 3.1 Entity 클래스 구조

```kotlin
@Entity
@Table(name = "orders")
class Order private constructor(
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    val id: Long = 0,

    @Column(nullable = false)
    val userId: Long,

    @Enumerated(EnumType.STRING)
    var status: OrderStatus = OrderStatus.PENDING,

    @Column(nullable = false)
    val createdAt: LocalDateTime = LocalDateTime.now(),

    var updatedAt: LocalDateTime = LocalDateTime.now()
) {
    companion object {
        fun create(userId: Long): Order {
            return Order(userId = userId)
        }
    }

    fun complete() {
        require(status == OrderStatus.PENDING) { "Only pending orders can be completed" }
        status = OrderStatus.COMPLETED
        updatedAt = LocalDateTime.now()
    }

    fun cancel() {
        require(status != OrderStatus.COMPLETED) { "Completed orders cannot be cancelled" }
        status = OrderStatus.CANCELLED
        updatedAt = LocalDateTime.now()
    }
}
```

### 3.2 Entity 설계 원칙

| 원칙 | 설명 |
|------|------|
| Private Constructor | Factory Method로만 생성 |
| 불변 ID | val로 선언, 변경 불가 |
| 도메인 로직 포함 | 상태 변경은 메서드로 |
| 유효성 검증 | 생성/변경 시 require로 검증 |

### 3.3 Value Object

```kotlin
@Embeddable
data class Money(
    val amount: BigDecimal,
    val currency: String = "KRW"
) {
    init {
        require(amount >= BigDecimal.ZERO) { "Amount must be non-negative" }
    }

    operator fun plus(other: Money): Money {
        require(currency == other.currency) { "Currency mismatch" }
        return Money(amount + other.amount, currency)
    }
}
```

---

## 4. Repository 규칙

### 4.1 Repository 인터페이스

```kotlin
interface OrderRepository : JpaRepository<Order, Long> {
    fun findByUserId(userId: Long): List<Order>
    fun findByStatus(status: OrderStatus): List<Order>
    fun findByUserIdAndStatus(userId: Long, status: OrderStatus): List<Order>
}
```

### 4.2 Custom Repository (복잡한 쿼리)

```kotlin
interface OrderRepositoryCustom {
    fun findWithCriteria(criteria: OrderCriteria): List<Order>
}

class OrderRepositoryImpl(
    private val queryFactory: JPAQueryFactory
) : OrderRepositoryCustom {

    override fun findWithCriteria(criteria: OrderCriteria): List<Order> {
        return queryFactory
            .selectFrom(order)
            .where(
                userIdEq(criteria.userId),
                statusEq(criteria.status)
            )
            .fetch()
    }
}
```

---

## 5. Command/Query Service 규칙

### 5.1 CommandService

```kotlin
@Service
@Transactional
class OrderCommandService(
    private val orderRepository: OrderRepository
) {
    fun create(request: CreateOrderRequest): Long {
        val order = Order.create(
            userId = request.userId
        )
        return orderRepository.save(order).id
    }

    fun complete(orderId: Long) {
        val order = orderRepository.findByIdOrNull(orderId)
            ?: throw OrderException.NotFound(orderId)
        order.complete()
    }

    fun cancel(orderId: Long) {
        val order = orderRepository.findByIdOrNull(orderId)
            ?: throw OrderException.NotFound(orderId)
        order.cancel()
    }
}
```

### 5.2 QueryService

```kotlin
@Service
@Transactional(readOnly = true)
class OrderQueryService(
    private val orderRepository: OrderRepository
) {
    fun findById(id: Long): Order? {
        return orderRepository.findByIdOrNull(id)
    }

    fun findByUserId(userId: Long): List<Order> {
        return orderRepository.findByUserId(userId)
    }

    fun findAll(criteria: OrderCriteria): List<Order> {
        return orderRepository.findWithCriteria(criteria)
    }
}
```

---

## 6. 도메인 이벤트

### 6.1 이벤트 정의

```kotlin
sealed class OrderEvent {
    data class Created(val orderId: Long, val userId: Long) : OrderEvent()
    data class Completed(val orderId: Long) : OrderEvent()
    data class Cancelled(val orderId: Long, val reason: String) : OrderEvent()
}
```

### 6.2 이벤트 발행

```kotlin
@Service
class OrderCommandService(
    private val orderRepository: OrderRepository,
    private val eventPublisher: ApplicationEventPublisher
) {
    fun complete(orderId: Long) {
        val order = orderRepository.findByIdOrNull(orderId)
            ?: throw OrderException.NotFound(orderId)
        order.complete()
        eventPublisher.publishEvent(OrderEvent.Completed(orderId))
    }
}
```

---

## 7. 단위 테스트

### 7.1 Entity 테스트

```kotlin
class OrderTest {

    @Test
    fun `should create order with pending status`() {
        val order = Order.create(userId = 1L)

        assertThat(order.status).isEqualTo(OrderStatus.PENDING)
        assertThat(order.userId).isEqualTo(1L)
    }

    @Test
    fun `should complete pending order`() {
        val order = Order.create(userId = 1L)

        order.complete()

        assertThat(order.status).isEqualTo(OrderStatus.COMPLETED)
    }

    @Test
    fun `should throw when completing non-pending order`() {
        val order = Order.create(userId = 1L)
        order.complete()

        assertThrows<IllegalArgumentException> {
            order.complete()
        }
    }
}
```

### 7.2 Repository 테스트

```kotlin
@DataJpaTest
class OrderRepositoryTest {

    @Autowired
    lateinit var orderRepository: OrderRepository

    @Test
    fun `should save and find order`() {
        val order = Order.create(userId = 1L)
        val saved = orderRepository.save(order)

        val found = orderRepository.findByIdOrNull(saved.id)

        assertThat(found).isNotNull
        assertThat(found?.userId).isEqualTo(1L)
    }
}
```

---

## 8. 파일 구조

```
domain/
└── orders/
    ├── model/
    │   ├── Order.kt
    │   ├── OrderStatus.kt
    │   ├── OrderRepository.kt
    │   └── Money.kt (Value Object)
    ├── command/
    │   ├── OrderCommandService.kt
    │   └── CreateOrderRequest.kt
    ├── query/
    │   ├── OrderQueryService.kt
    │   └── OrderCriteria.kt
    └── event/
        └── OrderEvent.kt
```

---

## 9. 완료 체크리스트

- [ ] Entity 클래스 구현 (Factory Method, 도메인 로직)
- [ ] Value Object 구현 (필요시)
- [ ] Repository 인터페이스 정의
- [ ] CommandService 구현 (생성, 수정, 삭제)
- [ ] QueryService 구현 (조회)
- [ ] 도메인 이벤트 정의 (필요시)
- [ ] 단위 테스트 작성 (Entity, Service)
- [ ] Repository 테스트 작성

---

## 10. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 다른 도메인 참조 필요 | Architect | LM 레이어로 올릴지 결정 |
| 복잡한 쿼리 최적화 | Architect | 쿼리 전략 결정 |
| 이벤트 처리 방식 | Architect | 동기/비동기 결정 |
| DB 스키마 변경 | User | 마이그레이션 승인 |

---

<!-- pal:convention:workers:backend:entity -->
