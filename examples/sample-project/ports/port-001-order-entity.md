# Order Entity 구현

> Port ID: port-001
> Status: pending
> Agent: domain
> Created: 2025-01-09

---

## 컨텍스트

### 상위 요구사항
주문 관리 시스템의 핵심 도메인 모델 구현

### 작업 목적
Order Entity와 관련 Enum을 정의하여 주문 데이터의 기본 구조를 확립

### 선행 작업
- 없음 (최초 작업)

---

## 입력

### 참조할 기존 코드
- 없음 (신규 생성)

### 요구사항
- 주문 ID (자동 생성)
- 주문 상태 (PENDING → CONFIRMED → SHIPPED → DELIVERED / CANCELLED)
- 생성 일시
- 주문 항목 목록
- 총 금액

---

## 작업 범위 (배타적 소유권)

### 생성할 파일
```
src/main/kotlin/domain/
├─ entity/
│  ├─ Order.kt
│  └─ OrderItem.kt
└─ enum/
   └─ OrderStatus.kt
```

### 구현할 기능
- [ ] Order Entity 클래스
- [ ] OrderItem Entity 클래스  
- [ ] OrderStatus Enum
- [ ] 주문 상태 변경 도메인 로직

---

## 컨벤션

### Entity 규칙
```kotlin
@Entity
@Table(name = "테이블명")
class EntityName(
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    val id: Long = 0,
    
    // 불변 필드는 val
    // 가변 필드는 var
)
```

### Enum 규칙
```kotlin
enum class StatusName {
    VALUE1,
    VALUE2;
    
    // 상태 전이 로직은 Enum 내부에
    fun canTransitionTo(next: StatusName): Boolean { ... }
}
```

### 네이밍
- Entity: PascalCase, 단수형
- 테이블: snake_case, 복수형

---

## 검증

### 컴파일 명령
```bash
./gradlew compileKotlin
```

### 테스트 명령
```bash
./gradlew test --tests "*Order*"
```

### 완료 체크리스트
- [ ] 컴파일 성공
- [ ] Entity 어노테이션 정확
- [ ] Enum 상태 전이 로직 구현
- [ ] 도메인 로직 테스트 작성

---

## 출력

### 완료 조건
- 모든 파일 생성 완료
- 컴파일 성공
- 기본 테스트 통과

### 후속 작업에 전달할 정보
- Order Entity 구조
- OrderStatus Enum 정의
- 상태 전이 규칙

---

## 실행 명령

```bash
# 작업 시작
pal lock acquire domain
pal port status port-001 running

# 작업 완료 후
pal lock release domain
pal port status port-001 complete
```
