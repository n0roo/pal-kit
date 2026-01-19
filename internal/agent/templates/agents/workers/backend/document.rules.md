# Document Worker Rules

> Claude 참조용 L1 MongoDB 규칙

---

## Quick Reference

```
Layer: L1 Domain
Tech: Spring Data MongoDB + MongoTemplate (Kotlin)
Pattern: CQS (Query/Command 분리)
Delete: Soft Delete Only
Model: data class (immutable, copy + save)
```

---

## Directory Structure

```
domain/{module}/
├── document/      # MongoDB Document (data class)
├── repository/    # MongoRepository
├── templates/     # MongoTemplate DSL (dynamic query)
├── services/      # QueryService, CommandService
└── models/        # DTO
```

---

## Document Rules

### Declaration
- **MUST** use `data class`
- **MUST** use `String` for ID (or ObjectId + idHex)
- **MUST** include `deleted: Boolean = false` field
- **SHOULD** define indexes with annotations

### Basic Structure
```kotlin
@Document(collection = "users")
data class UserDocument(
    @Id
    val id: String? = null,

    @Indexed(name = "user_filter_idhex")
    val idHex: String,

    val name: String,

    @Indexed(unique = true)
    val email: String,

    val deleted: Boolean = false,

    val createdAt: Instant = Instant.now()
)
```

### Nested vs DBRef

| Situation | Pattern |
|-----------|---------|
| 1:1, always loaded together | Nested |
| 1:N, small list (<100) | Nested |
| 1:N, large list (>100) | DBRef or ID reference |
| Referenced by multiple docs | DBRef or ID reference |
| Independent modification | DBRef or ID reference |

**Nested Example:**
```kotlin
@Document(collection = "orders")
data class OrderDocument(
    @Id val id: String? = null,
    val items: List<OrderItem> = emptyList(),  // Nested
    val shippingAddress: Address? = null       // Nested
)

// No @Document annotation
data class OrderItem(
    val productId: String,
    val quantity: Int,
    val price: BigDecimal
)
```

**DBRef Example:**
```kotlin
@Document(collection = "orders")
data class OrderDocument(
    @Id val id: String? = null,
    @DBRef val customer: CustomerDocument? = null  // Reference
)
```

---

## Index Rules

### Index Types
| Type | Annotation | Use Case |
|------|------------|----------|
| Single | `@Indexed` | Single field search |
| Unique | `@Indexed(unique = true)` | Prevent duplicates |
| Compound | `@CompoundIndex` | Multi-field search |
| TTL | `@Indexed(expireAfterSeconds = N)` | Auto-expire |

### Example
```kotlin
@Document(collection = "users")
@CompoundIndex(def = "{'status': 1, 'deleted': 1}")
data class UserDocument(
    @Indexed val status: String,
    @Indexed(unique = true) val email: String,
    @Indexed val createdAt: LocalDateTime
)
```

---

## Repository Rules

### Return Types
| Query Type | Return | Example |
|------------|--------|---------|
| Single by ID | `Optional<D>` | `findByIdAndDeleted(id, false)` |
| Single by condition | `Optional<D>` | `findByEmailAndDeleted(email, false)` |
| Multiple | `List<D>` | `findByStatusAndDeleted(status, false)` |
| Exists | `Boolean` | `existsByEmailAndDeleted(email, false)` |

### Forbidden
```kotlin
// NEVER use deleteBy* methods
fun deleteByUserId(userId: String)  // FORBIDDEN
fun deleteAll()                     // FORBIDDEN
```

---

## DSLTemplate (MongoTemplate)

### When to Use
| Situation | Tool |
|-----------|------|
| Simple query, fixed condition | MongoRepository |
| Dynamic filter, optional condition | MongoTemplate |
| Complex aggregation | MongoTemplate |
| Pagination with sorting | MongoTemplate |

### Use PojoFields for Field Names
```kotlin
@Repository
class UserDocumentQueryDSLTemplate(
    private val mongoTemplate: MongoTemplate
) {
    private fun generateConditions(
        userId: Long,
        userTypes: UserTypes,
        otherCondition: String? = null
    ): Query {
        var criteria = Criteria.where(PojoFields.Users.ID).`is`(userId)
            .and(PojoFields.Users.TYPES).`is`(userTypes)
            .and(PojoFields.Users.DELETED).`is`(false)

        otherCondition?.let {
            criteria = criteria.and(PojoFields.Users.OTHER_CONDITION).`is`(it)
        }

        return Query.query(criteria)
    }

    fun countUsers(userId: Long, userTypes: UserTypes, otherCondition: String? = null): Long =
        mongoTemplate.count(generateConditions(userId, userTypes, otherCondition), Users::class.java)

    fun findUsers(
        userId: Long,
        userTypes: UserTypes,
        otherCondition: String? = null,
        pageNo: Int,
        itemsPerPage: Int
    ): List<Users> {
        val query = generateConditions(userId, userTypes, otherCondition)
        query.with(Sort.by(Sort.Direction.DESC, PojoFields.Users.CREATED_AT_MILLIS))
        query.with(PageRequest.of(pageNo - 1, itemsPerPage))
        return mongoTemplate.find(query, Users::class.java)
    }
}
```

---

## Service Rules

### Naming Convention
| Situation | Method | Return | On null |
|-----------|--------|--------|---------|
| ID lookup (required) | `getById(id)` | `Document` | throw |
| ID lookup (optional) | `findById(id)` | `Document?` | null |
| Condition lookup | `findBy*(...)` | `Document?` | null |
| List | `findAll*()` | `List<D>` | emptyList |

### QueryService Pattern
```kotlin
@Service
class UserDocumentQueryService(
    private val repository: UserDocumentRepository
) {
    fun getById(id: String): UserDocument =
        repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException("Document not found: $id") }

    fun findById(id: String): UserDocument? =
        repository.findByIdAndDeleted(id, false).orElse(null)

    fun findByStatus(status: String): List<UserDocument> =
        repository.findByStatusAndDeleted(status, false)
}
```

### CommandService Pattern (copy + save)
```kotlin
@Service
class UserDocumentCommandService(
    private val repository: UserDocumentRepository
) {
    fun create(request: CreateRequest): UserDocument =
        repository.save(UserDocument(
            name = request.name,
            email = request.email
        ))

    // copy + save pattern (data class)
    fun updateName(id: String, name: String): UserDocument {
        val doc = repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException() }
        return repository.save(doc.copy(name = name))
    }

    // Soft Delete (copy + save)
    fun delete(id: String): UserDocument {
        val doc = repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException() }
        return repository.save(doc.copy(deleted = true))
    }
}
```

### Dependency Rules
```
L1 Service can reference:
✅ Same domain Repository
✅ Same domain DSLTemplate (MongoTemplate)
❌ Other L1 Services
❌ LM Services
❌ L2 Services
```

---

## Soft Delete Policy

### DO (copy + save)
```kotlin
val doc = repository.findByIdAndDeleted(id, false).orElseThrow { ... }
repository.save(doc.copy(deleted = true))
```

### DO NOT
```kotlin
repository.delete(doc)        // FORBIDDEN
repository.deleteById(id)     // FORBIDDEN
repository.deleteAll()        // FORBIDDEN
```

---

## Checklist

### Document
- [ ] `data class` declaration
- [ ] `String` ID (or ObjectId + idHex)
- [ ] Nested vs DBRef properly chosen
- [ ] `deleted: Boolean = false` field
- [ ] Required indexes defined

### Repository
- [ ] Single query returns `Optional<D>`
- [ ] No `deleteBy*` methods
- [ ] All queries include `deleted` condition

### Service
- [ ] Query/Command separation
- [ ] `getById` throws, `findById` returns nullable
- [ ] `copy + save` pattern for updates
- [ ] Soft delete only

### DSLTemplate
- [ ] Use `PojoFields` for field names
- [ ] Reusable condition methods
- [ ] Pagination with sort

---

## Escalation

| Situation | Target | Action |
|-----------|--------|--------|
| Complex aggregation query | Architect | Query optimization review |
| Schema change | User | Migration approval |
| Index performance issue | Architect | Index strategy review |
| Sharding needed | User | Infrastructure setup |

---

<!-- pal:rules:workers:backend:document -->
