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

## 2. Document 모델 규칙

### 2.1 Document 클래스 구조

```kotlin
@Document(collection = "products")
data class ProductDocument(
    @Id
    val id: String = ObjectId.get().toHexString(),

    @Field("name")
    val name: String,

    @Field("price")
    val price: Money,

    @Field("category")
    val category: String,

    @Field("attributes")
    val attributes: Map<String, Any> = emptyMap(),

    @Field("tags")
    val tags: List<String> = emptyList(),

    @Field("created_at")
    val createdAt: Instant = Instant.now(),

    @Field("updated_at")
    var updatedAt: Instant = Instant.now()
) {
    companion object {
        fun create(name: String, price: Money, category: String): ProductDocument {
            return ProductDocument(
                name = name,
                price = price,
                category = category
            )
        }
    }

    fun updatePrice(newPrice: Money): ProductDocument {
        return copy(price = newPrice, updatedAt = Instant.now())
    }
}
```

### 2.2 Embedded Document

```kotlin
data class Money(
    val amount: BigDecimal,
    val currency: String = "KRW"
)

data class Address(
    val street: String,
    val city: String,
    val zipCode: String,
    val country: String = "KR"
)
```

### 2.3 Document 설계 원칙

| 원칙 | 설명 |
|------|------|
| 비정규화 | 조회 패턴에 맞게 데이터 중복 허용 |
| Embedded | 1:1, 1:Few 관계는 임베딩 |
| Reference | 1:Many, Many:Many는 참조 |
| 스키마 유연성 | attributes로 동적 필드 지원 |

---

## 3. Repository 규칙

### 3.1 기본 Repository

```kotlin
interface ProductRepository : MongoRepository<ProductDocument, String> {
    fun findByCategory(category: String): List<ProductDocument>
    fun findByTagsContaining(tag: String): List<ProductDocument>
    fun findByPriceAmountBetween(min: BigDecimal, max: BigDecimal): List<ProductDocument>
}
```

### 3.2 Custom Repository

```kotlin
interface ProductRepositoryCustom {
    fun searchWithFilters(filters: ProductFilters): List<ProductDocument>
    fun aggregateByCategory(): List<CategoryStats>
}

class ProductRepositoryImpl(
    private val mongoTemplate: MongoTemplate
) : ProductRepositoryCustom {

    override fun searchWithFilters(filters: ProductFilters): List<ProductDocument> {
        val query = Query()

        filters.category?.let {
            query.addCriteria(Criteria.where("category").`is`(it))
        }

        filters.minPrice?.let {
            query.addCriteria(Criteria.where("price.amount").gte(it))
        }

        filters.tags?.let {
            query.addCriteria(Criteria.where("tags").all(it))
        }

        return mongoTemplate.find(query, ProductDocument::class.java)
    }
}
```

---

## 4. Aggregation Pipeline

### 4.1 집계 쿼리

```kotlin
fun aggregateByCategory(): List<CategoryStats> {
    val aggregation = Aggregation.newAggregation(
        Aggregation.group("category")
            .count().`as`("count")
            .avg("price.amount").`as`("avgPrice")
            .sum("price.amount").`as`("totalValue"),
        Aggregation.project()
            .and("_id").`as`("category")
            .andInclude("count", "avgPrice", "totalValue"),
        Aggregation.sort(Sort.Direction.DESC, "count")
    )

    return mongoTemplate.aggregate(
        aggregation,
        "products",
        CategoryStats::class.java
    ).mappedResults
}
```

### 4.2 집계 결과 DTO

```kotlin
data class CategoryStats(
    val category: String,
    val count: Long,
    val avgPrice: BigDecimal,
    val totalValue: BigDecimal
)
```

---

## 5. 인덱스 설계

### 5.1 인덱스 어노테이션

```kotlin
@Document(collection = "products")
@CompoundIndexes(
    CompoundIndex(
        name = "category_price_idx",
        def = "{'category': 1, 'price.amount': -1}"
    )
)
data class ProductDocument(
    @Indexed
    val category: String,

    @Indexed
    val createdAt: Instant,

    @TextIndexed
    val name: String,

    // ...
)
```

### 5.2 인덱스 전략

| 쿼리 패턴 | 인덱스 |
|----------|--------|
| 카테고리 필터 | `{ category: 1 }` |
| 가격 범위 + 정렬 | `{ price.amount: 1 }` |
| 텍스트 검색 | Text Index on `name` |
| 복합 필터 | `{ category: 1, price.amount: -1 }` |

---

## 6. Command/Query Service

### 6.1 CommandService

```kotlin
@Service
class ProductCommandService(
    private val productRepository: ProductRepository
) {
    fun create(request: CreateProductRequest): String {
        val product = ProductDocument.create(
            name = request.name,
            price = request.price,
            category = request.category
        )
        return productRepository.save(product).id
    }

    fun updatePrice(id: String, newPrice: Money) {
        val product = productRepository.findByIdOrNull(id)
            ?: throw ProductException.NotFound(id)

        val updated = product.updatePrice(newPrice)
        productRepository.save(updated)
    }

    fun delete(id: String) {
        productRepository.deleteById(id)
    }
}
```

### 6.2 QueryService

```kotlin
@Service
class ProductQueryService(
    private val productRepository: ProductRepository
) {
    fun findById(id: String): ProductDocument? {
        return productRepository.findByIdOrNull(id)
    }

    fun search(filters: ProductFilters): List<ProductDocument> {
        return productRepository.searchWithFilters(filters)
    }

    fun getCategoryStats(): List<CategoryStats> {
        return productRepository.aggregateByCategory()
    }
}
```

---

## 7. 스키마 마이그레이션

### 7.1 마이그레이션 스크립트

```kotlin
@Component
class ProductMigration(
    private val mongoTemplate: MongoTemplate
) {
    /**
     * v1 -> v2: price를 embedded document로 변경
     */
    fun migrateV1ToV2() {
        val query = Query(Criteria.where("price").type(JsonSchemaObject.Type.NUMBER))

        mongoTemplate.updateMulti(
            query,
            Update()
                .set("price", mapOf("amount" to "\$price", "currency" to "KRW"))
                .set("schema_version", 2),
            "products"
        )
    }
}
```

### 7.2 버전 관리

```kotlin
@Document(collection = "products")
data class ProductDocument(
    // ...

    @Field("schema_version")
    val schemaVersion: Int = CURRENT_VERSION
) {
    companion object {
        const val CURRENT_VERSION = 2
    }
}
```

---

## 8. 단위 테스트

### 8.1 Embedded MongoDB 테스트

```kotlin
@DataMongoTest
class ProductRepositoryTest {

    @Autowired
    lateinit var productRepository: ProductRepository

    @Test
    fun `should find products by category`() {
        // Given
        val product = ProductDocument.create(
            name = "Test Product",
            price = Money(BigDecimal("10000")),
            category = "electronics"
        )
        productRepository.save(product)

        // When
        val found = productRepository.findByCategory("electronics")

        // Then
        assertThat(found).hasSize(1)
        assertThat(found[0].name).isEqualTo("Test Product")
    }

    @Test
    fun `should aggregate by category`() {
        // Given
        repeat(5) {
            productRepository.save(
                ProductDocument.create("Product $it", Money(BigDecimal("1000")), "electronics")
            )
        }

        // When
        val stats = productRepository.aggregateByCategory()

        // Then
        assertThat(stats).hasSize(1)
        assertThat(stats[0].count).isEqualTo(5)
    }
}
```

---

## 9. 파일 구조

```
domain/
└── catalog/
    ├── model/
    │   ├── ProductDocument.kt
    │   ├── Money.kt
    │   └── ProductRepository.kt
    ├── command/
    │   ├── ProductCommandService.kt
    │   └── CreateProductRequest.kt
    ├── query/
    │   ├── ProductQueryService.kt
    │   ├── ProductFilters.kt
    │   └── CategoryStats.kt
    └── migration/
        └── ProductMigration.kt
```

---

## 10. 완료 체크리스트

- [ ] Document 모델 구현
- [ ] Embedded Document 정의
- [ ] Repository 구현
- [ ] 인덱스 설계 및 적용
- [ ] Aggregation Pipeline 구현 (필요시)
- [ ] CommandService 구현
- [ ] QueryService 구현
- [ ] 마이그레이션 스크립트 (필요시)
- [ ] 단위 테스트 작성

---

## 11. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 복잡한 집계 쿼리 | Architect | 쿼리 최적화 검토 |
| 스키마 변경 | User | 마이그레이션 승인 |
| 인덱스 성능 이슈 | Architect | 인덱스 전략 재검토 |
| 샤딩 필요 | User | 인프라 설정 |

---

<!-- pal:convention:workers:backend:document -->
