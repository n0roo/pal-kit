# Document Worker 컨벤션

> L1 Domain 레이어 - MongoDB/문서DB 전문 Worker

---

## 1. 역할 정의

Document Worker는 **L1 Domain 레이어**에서 MongoDB 등 문서형 데이터베이스를 다루는 전문 Worker입니다.

### 1.1 담당 영역

- Document 모델 구현
- MongoDB Repository 구현
- 집계(Aggregation) 파이프라인
- 인덱스 설계
- 스키마 마이그레이션

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| DB | MongoDB, Elasticsearch |
| ODM | Spring Data MongoDB |
| 언어 | Kotlin, Java |

---

## 2. 핵심 원칙

| 원칙 | 설명 |
|------|------|
| 단일 책임 | 하나의 서비스 = 하나의 도큐먼트 도메인 |
| CQS | Query와 Command 서비스 분리 |
| Soft Delete | 물리 삭제 금지, `deleted` 필드 사용 |
| Null Safety | 명시적 null 처리 (`getById` → throw, `findById` → nullable) |
| ID 관리 | ObjectId 사용 시 `idHex: String`으로 hexString 선언 |

---

## 3. 디렉토리 구조

```
domain/{module}/
├── document/           # MongoDB Document
├── repository/         # MongoRepository
├── services/           # QueryService, CommandService
├── templates/          # MongoTemplate 기반 (필요 시)
└── models/             # DTO
```

---

## 4. Document 모델

### 4.1 기본 Document 구조

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

### 4.2 Nested vs DBRef

**Nested (내장 문서)** - 함께 조회되는 데이터

```kotlin
@Document(collection = "orders")
data class OrderDocument(
    @Id
    val id: String? = null,

    // ✅ Nested - 항상 함께 로드
    val items: List<OrderItem> = emptyList(),
    val shippingAddress: Address? = null
)

// 별도 @Document 없음
data class OrderItem(
    val productId: String,
    val quantity: Int,
    val price: BigDecimal
)
```

**DBRef (참조)** - 독립적으로 관리되는 데이터

```kotlin
@Document(collection = "orders")
data class OrderDocument(
    @Id
    val id: String? = null,

    // ✅ DBRef - 별도 컬렉션 참조
    @DBRef
    val customer: CustomerDocument? = null
)
```

**선택 기준:**

| 상황 | 패턴 |
|------|------|
| 1:1, 항상 함께 조회 | Nested |
| 1:N, 작은 리스트 (<100) | Nested |
| 1:N, 큰 리스트 (>100) | DBRef 또는 ID 참조 |
| 여러 문서에서 참조 | DBRef 또는 ID 참조 |
| 독립적 수정 필요 | DBRef 또는 ID 참조 |

---

## 5. Repository 규칙

### 5.1 기본 Repository

```kotlin
interface UserDocumentRepository : MongoRepository<UserDocument, String> {

    fun findByIdAndDeleted(id: String, deleted: Boolean): Optional<UserDocument>

    fun findByEmailAndDeleted(email: String, deleted: Boolean): Optional<UserDocument>

    fun findByStatusAndDeleted(status: String, deleted: Boolean): List<UserDocument>

    fun existsByEmailAndDeleted(email: String, deleted: Boolean): Boolean
}
```

### 5.2 DSLTemplate (MongoTemplate 기반)

> PojoFields 객체를 사용하여 하드코드 최소화

```kotlin
@Repository
class UserDocumentQueryDSLTemplate
@Autowired constructor(
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
            criteria = criteria.and(PojoFields.Users.OTHER_CONDITION).`is`(otherCondition)
        }

        return Query.query(criteria)
    }

    fun countUsers(
        userId: Long,
        userTypes: UserTypes,
        otherCondition: String? = null
    ): Long {
        return mongoTemplate.count(
            generateConditions(userId, userTypes, otherCondition),
            Users::class.java
        )
    }

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

## 6. Service 패턴

### 6.1 QueryService

```kotlin
@Service
class UserDocumentQueryService(
    private val repository: UserDocumentRepository
) {
    // ID 조회 (필수) → throw
    fun getById(id: String): UserDocument {
        return repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException("Document not found: $id") }
    }

    // ID 조회 (선택) → nullable
    fun findById(id: String): UserDocument? {
        return repository.findByIdAndDeleted(id, false).orElse(null)
    }

    // 조건 조회 → nullable
    fun findByEmail(email: String): UserDocument? {
        return repository.findByEmailAndDeleted(email, false).orElse(null)
    }

    // 목록 조회 → List
    fun findByStatus(status: String): List<UserDocument> {
        return repository.findByStatusAndDeleted(status, false)
    }
}
```

### 6.2 CommandService

```kotlin
@Service
class UserDocumentCommandService(
    private val repository: UserDocumentRepository
) {
    fun create(request: CreateRequest): UserDocument {
        return repository.save(UserDocument(
            name = request.name,
            email = request.email
        ))
    }

    // ✅ copy + save (data class)
    fun updateName(id: String, name: String): UserDocument {
        val doc = repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException() }
        return repository.save(doc.copy(name = name))
    }

    // ✅ Soft Delete
    fun delete(id: String): UserDocument {
        val doc = repository.findByIdAndDeleted(id, false)
            .orElseThrow { NotFoundException() }
        return repository.save(doc.copy(deleted = true))
    }
}
```

---

## 7. Index 전략

### 7.1 어노테이션 기반

```kotlin
@Document(collection = "users")
@CompoundIndex(def = "{'status': 1, 'deleted': 1}")
data class UserDocument(
    @Indexed
    val status: String,

    @Indexed(unique = true)
    val email: String,

    @Indexed
    val createdAt: LocalDateTime
)
```

### 7.2 Index 유형

| 유형 | 어노테이션 | 용도 |
|------|-----------|------|
| 단일 필드 | `@Indexed` | 단일 필드 검색 |
| 유니크 | `@Indexed(unique = true)` | 중복 방지 |
| 복합 | `@CompoundIndex` | 복수 필드 검색 |
| TTL | `@Indexed(expireAfterSeconds = 3600)` | 자동 만료 |

---

## 8. Soft Delete 정책

```kotlin
// ✅ copy + save
fun softDelete(id: String): UserDocument {
    val doc = repository.findByIdAndDeleted(id, false)
        .orElseThrow { NotFoundException() }
    return repository.save(doc.copy(deleted = true))
}

// ❌ 금지
repository.delete(doc)
repository.deleteById(id)
```

---

## 9. 체크리스트

### Document

- [ ] data class 사용
- [ ] String ID (또는 ObjectId + idHex)
- [ ] Nested vs DBRef 적절히 선택
- [ ] `deleted` 필드 포함
- [ ] 필요한 Index 정의

### Repository

- [ ] 단일 조회는 Optional 반환
- [ ] `deleteBy*` 메서드 없음
- [ ] `deleted` 조건 포함

### Service

- [ ] Query/Command 분리
- [ ] `getById` → throw, `findById` → nullable
- [ ] copy + save 패턴
- [ ] Soft Delete만 사용

---

## 10. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 복잡한 집계 쿼리 | Architect | 쿼리 최적화 검토 |
| 스키마 변경 | User | 마이그레이션 승인 |
| 인덱스 성능 이슈 | Architect | 인덱스 전략 재검토 |
| 샤딩 필요 | User | 인프라 설정 |

---

<!-- pal:convention:workers:backend:document -->
