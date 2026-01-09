# Order Service 구현

> Port ID: port-002
> Status: pending
> Agent: service
> Depends: port-001
> Created: 2025-01-09

---

## 컨텍스트

### 상위 요구사항
주문 생성, 조회, 상태 변경 비즈니스 로직 구현

### 작업 목적
OrderService를 통해 주문 관련 비즈니스 로직을 캡슐화

### 선행 작업
- **port-001**: Order Entity, OrderStatus Enum (완료 필요)

---

## 입력

### 선행 작업 산출물
```
src/main/kotlin/domain/
├─ entity/Order.kt
├─ entity/OrderItem.kt
└─ enum/OrderStatus.kt
```

### 참조할 기존 코드
- `domain/entity/Order.kt` - Entity 구조
- `domain/enum/OrderStatus.kt` - 상태 전이 규칙

---

## 작업 범위 (배타적 소유권)

### 생성할 파일
```
src/main/kotlin/application/
├─ service/
│  └─ OrderService.kt
└─ repository/
   └─ OrderRepository.kt  (Interface)
```

### 구현할 기능
- [ ] OrderRepository Interface
- [ ] OrderService 클래스
- [ ] 주문 생성 로직
- [ ] 주문 조회 로직
- [ ] 주문 상태 변경 로직

---

## 컨벤션

### Service 규칙
```kotlin
@Service
@Transactional(readOnly = true)
class OrderService(
    private val orderRepository: OrderRepository
) {
    // 조회는 readOnly
    fun getOrder(id: Long): Order { ... }
    
    // 변경은 @Transactional
    @Transactional
    fun createOrder(request: CreateOrderRequest): Order { ... }
}
```

### Repository Interface 규칙
```kotlin
interface OrderRepository {
    fun findById(id: Long): Order?
    fun save(order: Order): Order
    // 구현체는 infrastructure 레이어에서
}
```

### 레이어 규칙
- ✅ domain.* 참조 가능
- ❌ infrastructure.* 직접 참조 금지
- ❌ Repository 구현체 참조 금지

---

## 검증

### 컴파일 명령
```bash
./gradlew compileKotlin
```

### 테스트 명령
```bash
./gradlew test --tests "*OrderService*"
```

### 완료 체크리스트
- [ ] 컴파일 성공
- [ ] Repository Interface 정의
- [ ] Service 메서드 구현
- [ ] 트랜잭션 어노테이션 정확

---

## 출력

### 완료 조건
- 모든 파일 생성 완료
- 컴파일 성공
- Service 단위 테스트 통과

### 후속 작업에 전달할 정보
- OrderService API
- OrderRepository Interface

---

## 실행 명령

```bash
# 선행 작업 확인
pal port show port-001  # status: complete 확인

# 작업 시작
pal lock acquire service
pal port status port-002 running

# 작업 완료 후
pal lock release service
pal port status port-002 complete
```
