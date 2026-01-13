# Entity Worker 컨벤션

> L1 Domain 레이어 - Spring Data JPA + Jooq 기반 엔티티 전문 Worker

---

## 1. 역할 정의

Entity Worker는 **L1 Domain 레이어**에서 JPA/ORM 기반 엔티티와 리포지토리를 구현하는 전문 Worker입니다.

### 1.1 담당 영역

- Entity 클래스 구현
- Repository 인터페이스 정의
- Jooq DSL Template 구현
- DTO, VO 구현
- CommandService / QueryService 구현

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| ORM | Spring Data JPA |
| Query DSL | Jooq |
| DB | PostgreSQL, MySQL, H2 |
| 언어 | Kotlin |

### 1.3 핵심 원칙

1. **단일 책임**: 하나의 서비스 = 하나의 엔티티 도메인
2. **CQS**: Query와 Command 서비스 분리
3. **Soft Delete**: 물리 삭제 금지
4. **Null Safety**: 명시적 null 처리

---

## 2. 디렉토리 구조

```
domain/{module}/
├── entities/           # JPA Entity
├── repository/         # Spring Data Repository
├── templates/          # Jooq DSL Template
├── models/             # DTO, VO
└── services/           # QueryService, CommandService
```

---

## 3. L1 레이어 규칙

### 3.1 의존성 규칙

```
L1 Domain은:
✅ Infrastructure(DB, 외부 시스템) 참조 가능
❌ 다른 L1 도메인 직접 참조 불가
❌ LM, L2 참조 불가
```

### 3.2 다른 도메인 참조 방법

```kotlin
// ❌ 잘못된 방법: 다른 도메인 직접 참조
class Order(
    val user: User  // User 도메인 직접 참조
)

// ✅ 올바른 방법: ID만 참조
class Order(
    val userId: Long  // ID만 저장
)
```

---

## 4. Entity 규칙

### 4.1 클래스 선언

JPA Entity는 반드시 **class**로 선언합니다 (data class 아님).

**이유:**
- JPA는 프록시 기반으로 동작하며, data class의 `copy()`, `equals()`, `hashCode()`가 예상치 못한 문제를 일으킬 수 있음
- 지연 로딩(Lazy Loading) 시 프록시 객체 생성이 필요

```kotlin
@Entity
@Table(name = "users")
class User(
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    @get:Column(name = "id")
    val id: Long = 0,

    @get:Column(name = "email", nullable = false, unique = true)
    var email: String,

    @get:Column(name = "deleted", nullable = false)
    var deleted: Boolean = false
)
```

### 4.2 어노테이션 위치

Kotlin에서 JPA 어노테이션은 **@get:** 으로 지정합니다.

**이유:**
- Kotlin의 프로퍼티는 field, getter, setter가 모두 생성됨
- JPA는 getter 기반으로 동작하므로 @get: 지정 필요

```kotlin
// ✅ 올바른 사용
@get:Column(name = "name", nullable = false)
var name: String

// ❌ 잘못된 사용 - JPA가 인식 못할 수 있음
@Column(name = "name", nullable = false)
var name: String
```

### 4.3 nullable 규칙

DB 컬럼의 nullable과 Kotlin 타입을 일치시킵니다.

```kotlin
// NOT NULL 컬럼
@get:Column(name = "required_field", nullable = false)
var requiredField: String  // non-null

// NULLABLE 컬럼
@get:Column(name = "optional_field", nullable = true)
var optionalField: String? = null  // nullable
```

### 4.4 Enum 처리

```kotlin
@get:Enumerated(EnumType.STRING)
@get:Column(name = "status", nullable = false)
var status: Status
```

**주의:** `EnumType.ORDINAL`은 사용하지 않습니다. 순서 변경 시 데이터 불일치 발생.

### 4.5 JSON 타입

```kotlin
@get:JdbcTypeCode(SqlTypes.JSON)
@get:Column(name = "metadata", columnDefinition = "jsonb")
var metadata: Map<String, Any>? = null
```

### 4.6 관계 매핑

**허용:**
- `@ManyToOne` - N:1 관계
- `@OneToOne` - 1:1 관계

**금지:**
- `@OneToMany` - 1:N 관계
- `@ManyToMany` - N:M 관계

**이유:**
- OneToMany, ManyToMany는 N+1 문제, 성능 이슈 발생 가능성 높음
- 필요 시 별도 쿼리(Jooq)로 처리

```kotlin
// ✅ 허용
@get:ManyToOne(fetch = FetchType.LAZY)
@get:JoinColumn(name = "parent_id")
var parent: ParentEntity? = null

// ❌ 금지
@get:OneToMany(mappedBy = "parent")
val children: List<ChildEntity> = emptyList()
```

---

## 5. Repository 규칙

### 5.1 기본 구조

```kotlin
interface UserRepository : JpaRepository<User, Long> {

    // Optional 반환 (단일 조회)
    fun findByIdAndDeleted(id: Long, deleted: Boolean): Optional<User>

    // List 반환 (복수 조회)
    fun findByStatusAndDeleted(status: Status, deleted: Boolean): List<User>

    // exists 체크
    fun existsByEmailAndDeleted(email: String, deleted: Boolean): Boolean
}
```

### 5.2 반환 타입 규칙

| 조회 유형 | 반환 타입 | 예시 |
|----------|----------|------|
| ID로 단일 조회 | `Optional<Entity>` | `findByIdAndDeleted()` |
| 조건으로 단일 조회 | `Optional<Entity>` | `findByEmailAndDeleted()` |
| 복수 조회 | `List<Entity>` | `findByStatusAndDeleted()` |
| 존재 여부 | `Boolean` | `existsByEmailAndDeleted()` |
| 개수 | `Long` | `countByStatusAndDeleted()` |

### 5.3 금지 패턴

```kotlin
// ❌ deleteBy 메서드 금지 - Soft Delete 정책 위반
fun deleteByParentId(parentId: Long)
fun deleteAllByStatus(status: Status)
```

---

## 6. Service 규칙

### 6.1 Query/Command 분리

```kotlin
// 조회 전용
class UserQueryService { }

// CUD 전용
class UserCommandService { }
```

### 6.2 조회 메서드 네이밍

| 상황 | 메서드명 | 반환 | null 시 |
|------|---------|------|---------|
| ID로 조회 (필수) | `getById(id)` | `Entity` | throw |
| ID로 조회 (선택) | `findById(id)` | `Entity?` | null |
| 조건으로 조회 | `findBy*(...)` | `Entity?` | null |
| 목록 조회 | `findAll*()` | `List<Entity>` | emptyList |

```kotlin
@Service
class UserQueryService(
    private val repository: UserRepository
) {
    // ID 조회 - 없으면 예외
    fun getById(id: Long): User {
        return repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException("User not found: $id") }
    }

    // ID 조회 - nullable
    fun findById(id: Long): User? {
        return repository.findByIdAndDeleted(id, false).orElse(null)
    }

    // 조건 조회 - nullable
    fun findByEmail(email: String): User? {
        return repository.findByEmailAndDeleted(email, false).orElse(null)
    }

    // 목록 조회 - empty list
    fun findByStatus(status: Status): List<User> {
        return repository.findByStatusAndDeleted(status, false)
    }
}
```

### 6.3 Command 메서드

```kotlin
@Service
class UserCommandService(
    private val repository: UserRepository
) {
    fun create(request: CreateUserRequest): User {
        val user = User(
            email = request.email,
            name = request.name
        )
        return repository.save(user)
    }

    fun update(id: Long, request: UpdateUserRequest): User {
        val user = repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException("User not found: $id") }

        user.name = request.name
        // 더티체킹으로 자동 저장 (또는 명시적 save)
        return repository.save(user)
    }

    // ✅ Soft Delete
    fun delete(id: Long): User {
        val user = repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException("User not found: $id") }

        user.deleted = true
        return repository.save(user)
    }
}
```

### 6.4 단일 엔티티 원칙

L1 Service는 **하나의 메서드에서 하나의 엔티티만** 조작합니다.

```kotlin
// ✅ 올바름 - 단일 엔티티
fun updateUser(id: Long, name: String): User

// ❌ 잘못됨 - 복수 엔티티 조작은 L2로
fun updateUserWithPosts(id: Long, ...): User
```

### 6.5 의존성 규칙

L1 Service는 다음만 참조 가능:
- 같은 도메인의 Repository
- 같은 도메인의 DSL Template

```kotlin
@Service
class UserQueryService(
    private val repository: UserRepository,        // ✅
    private val dslTemplate: UserQueryDSLTemplate, // ✅
    // private val postService: PostQueryService,  // ❌ 다른 L1 참조 금지
)
```

---

## 7. Jooq DSL Template 규칙

### 7.1 사용 시점

| 상황 | 도구 |
|------|------|
| 단순 조회, 고정 조건 | JPA Repository |
| 동적 필터, Optional 조건 | Jooq DSL |
| 복수 Entity Join | Jooq DSL + DTO |
| Bulk Insert/Update | Jooq DSL |

### 7.2 네이밍

```kotlin
// Query용
class UserQueryDSLTemplate

// Command용 (Bulk)
class UserCommandDSLTemplate
```

### 7.3 기본 구조

```kotlin
@Repository
class UserQueryDSLTemplate(
    private val dslContext: DSLContext
) {
    companion object {
        private val USER = Tables.USER
    }

    // 동적 필터 조회
    fun findByFilters(
        status: Status? = null,
        role: Role? = null
    ): List<Long> {
        var condition = USER.DELETED.eq(false)

        status?.let { condition = condition.and(USER.STATUS.eq(it.name)) }
        role?.let { condition = condition.and(USER.ROLE.eq(it.name)) }

        return dslContext.select(USER.ID)
            .from(USER)
            .where(condition)
            .fetchInto(Long::class.java)
    }
}
```

### 7.4 Null Safety 패턴

```kotlin
// count - elvis :0
fun countByStatus(status: Status): Long {
    return dslContext.selectCount()
        .from(USER)
        .where(USER.STATUS.eq(status.name))
        .and(USER.DELETED.eq(false))
        .fetchOne(0, Long::class.java) ?: 0
}

// list - elvis :emptyList()
fun findIdsByStatus(status: Status): List<Long> {
    return dslContext.select(USER.ID)
        .from(USER)
        .where(USER.STATUS.eq(status.name))
        .and(USER.DELETED.eq(false))
        .fetchInto(Long::class.java) ?: emptyList()
}
```

### 7.5 재사용 쿼리 분리

반복 사용되는 DTO 쿼리는 from절과 where절을 분리합니다.

```kotlin
@Repository
class UserQueryDSLTemplate(
    private val dslContext: DSLContext
) {
    companion object {
        private val USER = Tables.USER
        private val PROFILE = Tables.USER_PROFILE
    }

    // 공통 select + from
    private fun baseUserDtoQuery() = dslContext
        .select(
            USER.ID,
            USER.EMAIL,
            USER.NAME,
            PROFILE.AVATAR_URL
        )
        .from(USER)
        .innerJoin(PROFILE).on(USER.ID.eq(PROFILE.USER_ID))

    // 공통 기본 조건
    private fun baseCondition() = USER.DELETED.eq(false)

    // 활용 - ID로 조회
    fun findDtoById(id: Long): UserDto? {
        return baseUserDtoQuery()
            .where(baseCondition())
            .and(USER.ID.eq(id))
            .fetchOneInto(UserDto::class.java)
    }

    // 활용 - 상태로 조회
    fun findDtosByStatus(status: Status): List<UserDto> {
        return baseUserDtoQuery()
            .where(baseCondition())
            .and(USER.STATUS.eq(status.name))
            .fetchInto(UserDto::class.java) ?: emptyList()
    }

    // 활용 - 동적 필터
    fun findDtosByFilters(
        status: Status? = null,
        role: Role? = null
    ): List<UserDto> {
        var condition = baseCondition()

        status?.let { condition = condition.and(USER.STATUS.eq(it.name)) }
        role?.let { condition = condition.and(USER.ROLE.eq(it.name)) }

        return baseUserDtoQuery()
            .where(condition)
            .fetchInto(UserDto::class.java) ?: emptyList()
    }
}
```

### 7.6 Join 규칙

```kotlin
// ✅ innerJoin 기본 사용
dslContext.select(...)
    .from(USER)
    .innerJoin(ORDER).on(USER.ID.eq(ORDER.USER_ID))
    .where(...)

// ✅ leftJoin 필요 시 별도 메서드
fun findWithOptionalProfile(id: Long): UserWithProfileDto? {
    return dslContext.select(...)
        .from(USER)
        .leftJoin(PROFILE).on(USER.ID.eq(PROFILE.USER_ID))
        .where(USER.ID.eq(id))
        .fetchOneInto(UserWithProfileDto::class.java)
}
```

---

## 8. Soft Delete 정책

### 8.1 기본 패턴

```kotlin
// Entity에 deleted 필드
@get:Column(name = "deleted", nullable = false)
var deleted: Boolean = false

// 삭제 시
fun softDelete(id: Long): User {
    val user = repository.findByIdAndDeleted(id, false)
        .orElseThrow { NotFoundException() }
    user.deleted = true
    return repository.save(user)
}
```

### 8.2 일괄 삭제

```kotlin
// Repository에서 조회 후 개별 처리
fun softDeleteByParentId(parentId: Long): List<User> {
    val users = repository.findByParentIdAndDeleted(parentId, false)
    users.forEach { it.deleted = true }
    return repository.saveAll(users)
}

// 또는 Jooq로 Bulk Update
fun bulkSoftDelete(ids: List<Long>): Int {
    return dslContext.update(USER)
        .set(USER.DELETED, true)
        .where(USER.ID.`in`(ids))
        .execute()
}
```

### 8.3 금지 패턴

```kotlin
// ❌ 모두 금지
repository.delete(entity)
repository.deleteById(id)
repository.deleteAll()
repository.deleteByParentId(parentId)
```

---

## 9. 완료 체크리스트

### Entity
- [ ] class로 선언 (data class 아님)
- [ ] 어노테이션에 @get: 사용
- [ ] nullable과 Kotlin 타입 일치
- [ ] ManyToOne, OneToOne만 사용
- [ ] deleted 필드 포함

### Repository
- [ ] 단일 조회는 Optional 반환
- [ ] deleteBy* 메서드 없음
- [ ] deleted 조건 포함

### Service
- [ ] Query/Command 분리
- [ ] ID 조회: getById (throw), findById (nullable)
- [ ] 조건 조회: findBy* (nullable)
- [ ] 단일 엔티티만 조작
- [ ] Soft Delete만 사용

### Jooq
- [ ] count는 elvis :0
- [ ] list는 elvis :emptyList()
- [ ] 재사용 쿼리 from/where 분리
- [ ] innerJoin 기본, leftJoin 분리

---

## 10. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 다른 도메인 참조 필요 | Architect | LM 레이어로 올릴지 결정 |
| 복잡한 쿼리 최적화 | Architect | 쿼리 전략 결정 |
| OneToMany/ManyToMany 필요 | Architect | Jooq 쿼리로 대체 방안 |
| DB 스키마 변경 | User | 마이그레이션 승인 |

---

<!-- pal:convention:workers:backend:entity -->
