# Entity Worker Rules

> Claude 참조용 L1 JPA + Jooq 규칙

---

## Quick Reference

```
Layer: L1 Domain
Tech: Spring Data JPA + Jooq (Kotlin)
Pattern: CQS (Query/Command 분리)
Delete: Soft Delete Only
```

---

## Directory Structure

```
domain/{module}/
├── entities/      # JPA Entity (class, not data class)
├── repository/    # Spring Data Repository
├── templates/     # Jooq DSL Template
├── models/        # DTO, VO
└── services/      # QueryService, CommandService
```

---

## Entity Rules

### Declaration
- **MUST** use `class` (NOT `data class`)
- **MUST** use `@get:` prefix for JPA annotations
- **MUST** include `deleted: Boolean = false` field
- **MUST** match Kotlin nullability with DB nullable

### Allowed Relations
- `@ManyToOne` - OK
- `@OneToOne` - OK

### Forbidden Relations
- `@OneToMany` - NO (use Jooq query instead)
- `@ManyToMany` - NO (use Jooq query instead)

### Example
```kotlin
@Entity
@Table(name = "users")
class User(
    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    @get:Column(name = "id")
    val id: Long = 0,

    @get:Column(name = "email", nullable = false)
    var email: String,

    @get:Column(name = "name", nullable = true)
    var name: String? = null,

    @get:Enumerated(EnumType.STRING)
    @get:Column(name = "status", nullable = false)
    var status: Status,

    @get:ManyToOne(fetch = FetchType.LAZY)
    @get:JoinColumn(name = "parent_id")
    var parent: ParentEntity? = null,

    @get:Column(name = "deleted", nullable = false)
    var deleted: Boolean = false
)
```

---

## Repository Rules

### Return Types
| Query Type | Return | Example |
|------------|--------|---------|
| Single by ID | `Optional<E>` | `findByIdAndDeleted(id, false)` |
| Single by condition | `Optional<E>` | `findByEmailAndDeleted(email, false)` |
| Multiple | `List<E>` | `findByStatusAndDeleted(status, false)` |
| Exists | `Boolean` | `existsByEmailAndDeleted(email, false)` |
| Count | `Long` | `countByStatusAndDeleted(status, false)` |

### Forbidden
```kotlin
// NEVER use deleteBy* methods
fun deleteByParentId(parentId: Long)  // FORBIDDEN
fun deleteAllByStatus(status: Status) // FORBIDDEN
```

---

## Service Rules

### Naming Convention
| Situation | Method | Return | On null |
|-----------|--------|--------|---------|
| ID lookup (required) | `getById(id)` | `Entity` | throw |
| ID lookup (optional) | `findById(id)` | `Entity?` | null |
| Condition lookup | `findBy*(...)` | `Entity?` | null |
| List | `findAll*()` | `List<E>` | emptyList |

### QueryService Pattern
```kotlin
@Service
class UserQueryService(
    private val repository: UserRepository,
    private val dslTemplate: UserQueryDSLTemplate  // OK
    // private val otherService: OtherQueryService // FORBIDDEN
) {
    fun getById(id: Long): User =
        repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException("User not found: $id") }

    fun findById(id: Long): User? =
        repository.findByIdAndDeleted(id, false).orElse(null)

    fun findByStatus(status: Status): List<User> =
        repository.findByStatusAndDeleted(status, false)
}
```

### CommandService Pattern
```kotlin
@Service
class UserCommandService(
    private val repository: UserRepository
) {
    fun create(request: CreateUserRequest): User {
        val user = User(email = request.email, ...)
        return repository.save(user)
    }

    fun update(id: Long, request: UpdateUserRequest): User {
        val user = repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException("User not found: $id") }
        user.name = request.name
        return repository.save(user)
    }

    // Soft Delete ONLY
    fun delete(id: Long): User {
        val user = repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException("User not found: $id") }
        user.deleted = true
        return repository.save(user)
    }
}
```

### Dependency Rules
```
L1 Service can reference:
✅ Same domain Repository
✅ Same domain DSL Template
❌ Other L1 Services
❌ LM Services
❌ L2 Services
```

---

## Jooq DSL Template Rules

### When to Use
| Situation | Tool |
|-----------|------|
| Simple query, fixed condition | JPA Repository |
| Dynamic filter, optional condition | Jooq DSL |
| Multi-entity join | Jooq DSL + DTO |
| Bulk insert/update | Jooq DSL |

### Naming
- Query: `{Entity}QueryDSLTemplate`
- Command (Bulk): `{Entity}CommandDSLTemplate`

### Null Safety Pattern
```kotlin
// count - use elvis ?: 0
fun countByStatus(status: Status): Long =
    dslContext.selectCount()
        .from(USER)
        .where(USER.STATUS.eq(status.name))
        .and(USER.DELETED.eq(false))
        .fetchOne(0, Long::class.java) ?: 0

// list - use elvis ?: emptyList()
fun findIdsByStatus(status: Status): List<Long> =
    dslContext.select(USER.ID)
        .from(USER)
        .where(USER.STATUS.eq(status.name))
        .and(USER.DELETED.eq(false))
        .fetchInto(Long::class.java) ?: emptyList()
```

### Reusable Query Pattern
```kotlin
@Repository
class UserQueryDSLTemplate(private val dslContext: DSLContext) {

    // Base query (reusable)
    private fun baseUserDtoQuery() = dslContext
        .select(USER.ID, USER.EMAIL, USER.NAME, PROFILE.AVATAR_URL)
        .from(USER)
        .innerJoin(PROFILE).on(USER.ID.eq(PROFILE.USER_ID))

    // Base condition (reusable)
    private fun baseCondition() = USER.DELETED.eq(false)

    // Usage
    fun findDtoById(id: Long): UserDto? =
        baseUserDtoQuery()
            .where(baseCondition())
            .and(USER.ID.eq(id))
            .fetchOneInto(UserDto::class.java)

    fun findDtosByFilters(status: Status?, role: Role?): List<UserDto> {
        var condition = baseCondition()
        status?.let { condition = condition.and(USER.STATUS.eq(it.name)) }
        role?.let { condition = condition.and(USER.ROLE.eq(it.name)) }

        return baseUserDtoQuery()
            .where(condition)
            .fetchInto(UserDto::class.java) ?: emptyList()
    }
}
```

### Join Rules
- Default: `innerJoin`
- Optional relation: Create separate method with `leftJoin`

---

## Soft Delete Policy

### DO
```kotlin
user.deleted = true
repository.save(user)
```

### DO NOT
```kotlin
repository.delete(entity)      // FORBIDDEN
repository.deleteById(id)      // FORBIDDEN
repository.deleteAll()         // FORBIDDEN
repository.deleteByXxx(...)    // FORBIDDEN
```

### Bulk Soft Delete
```kotlin
// Option 1: Repository
fun softDeleteByParentId(parentId: Long): List<User> {
    val users = repository.findByParentIdAndDeleted(parentId, false)
    users.forEach { it.deleted = true }
    return repository.saveAll(users)
}

// Option 2: Jooq Bulk Update
fun bulkSoftDelete(ids: List<Long>): Int =
    dslContext.update(USER)
        .set(USER.DELETED, true)
        .where(USER.ID.`in`(ids))
        .execute()
```

---

## Checklist

### Entity
- [ ] `class` declaration (not `data class`)
- [ ] `@get:` prefix on annotations
- [ ] Kotlin nullability matches DB nullable
- [ ] Only `@ManyToOne`, `@OneToOne` relations
- [ ] `deleted: Boolean = false` field

### Repository
- [ ] Single query returns `Optional<E>`
- [ ] No `deleteBy*` methods
- [ ] All queries include `deleted` condition

### Service
- [ ] Query/Command separation
- [ ] `getById` throws, `findById` returns nullable
- [ ] Single entity per method
- [ ] Soft delete only

### Jooq
- [ ] `count` uses `?: 0`
- [ ] `list` uses `?: emptyList()`
- [ ] Reusable queries split base/condition
- [ ] `innerJoin` default, `leftJoin` separate method

---

## Escalation

| Situation | Target | Action |
|-----------|--------|--------|
| Need other domain reference | Architect | Decide LM layer promotion |
| Complex query optimization | Architect | Query strategy decision |
| Need OneToMany/ManyToMany | Architect | Jooq query alternative |
| DB schema change | User | Migration approval |

---

<!-- pal:rules:workers:backend:entity -->
