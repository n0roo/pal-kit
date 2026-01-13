# Cache Worker 컨벤션

> L1 Domain 레이어 - Redis/캐시 전문 Worker

---

## 1. 역할 정의

Cache Worker는 **L1 Domain 레이어**에서 Redis 등 캐시 기반 데이터 처리를 담당하는 전문 Worker입니다.

### 1.1 담당 영역

- 캐시 저장소 구현
- 캐시 정책 적용
- 세션 관리
- 분산 락 구현
- Pub/Sub 메시징

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| 캐시 | Redis, Memcached |
| 클라이언트 | Lettuce, Jedis, Redisson |
| 직렬화 | JSON, Protobuf |

---

## 2. 캐시 패턴

### 2.1 Cache-Aside 패턴

```kotlin
@Service
class ProductCacheService(
    private val redisTemplate: RedisTemplate<String, Product>,
    private val productRepository: ProductRepository
) {
    private val cacheTTL = Duration.ofHours(1)

    fun findById(id: Long): Product? {
        val cacheKey = "product:$id"

        // 1. 캐시에서 조회
        val cached = redisTemplate.opsForValue().get(cacheKey)
        if (cached != null) {
            return cached
        }

        // 2. DB에서 조회
        val product = productRepository.findByIdOrNull(id) ?: return null

        // 3. 캐시에 저장
        redisTemplate.opsForValue().set(cacheKey, product, cacheTTL)

        return product
    }

    fun evict(id: Long) {
        redisTemplate.delete("product:$id")
    }
}
```

### 2.2 Write-Through 패턴

```kotlin
@Service
class ProductCacheService(
    private val redisTemplate: RedisTemplate<String, Product>,
    private val productRepository: ProductRepository
) {
    @Transactional
    fun save(product: Product): Product {
        // 1. DB 저장
        val saved = productRepository.save(product)

        // 2. 캐시 갱신
        redisTemplate.opsForValue().set(
            "product:${saved.id}",
            saved,
            Duration.ofHours(1)
        )

        return saved
    }
}
```

---

## 3. 캐시 키 규칙

### 3.1 키 네이밍 컨벤션

```
{domain}:{entity}:{identifier}:{variant}
```

| 패턴 | 예시 | 설명 |
|------|------|------|
| 단일 엔티티 | `product:123` | 상품 ID 123 |
| 목록 | `products:category:electronics` | 전자제품 카테고리 |
| 사용자별 | `user:1:cart` | 사용자 1의 장바구니 |
| 집계 | `stats:orders:daily:2026-01-12` | 일별 주문 통계 |

### 3.2 키 생성 유틸리티

```kotlin
object CacheKeyGenerator {
    fun product(id: Long) = "product:$id"
    fun productsByCategory(category: String) = "products:category:$category"
    fun userCart(userId: Long) = "user:$userId:cart"
    fun dailyStats(date: LocalDate) = "stats:orders:daily:$date"
}
```

---

## 4. TTL 정책

### 4.1 TTL 기준

| 데이터 유형 | TTL | 이유 |
|------------|-----|------|
| 상품 정보 | 1시간 | 자주 변경되지 않음 |
| 재고 정보 | 5분 | 실시간성 필요 |
| 세션 | 30분 | 보안 요구사항 |
| 통계 데이터 | 24시간 | 집계 데이터 |

### 4.2 TTL 설정 예시

```kotlin
@Configuration
class CacheConfig {

    @Bean
    fun cacheManager(redisConnectionFactory: RedisConnectionFactory): RedisCacheManager {
        val cacheConfigurations = mapOf(
            "products" to RedisCacheConfiguration.defaultCacheConfig()
                .entryTtl(Duration.ofHours(1)),
            "inventory" to RedisCacheConfiguration.defaultCacheConfig()
                .entryTtl(Duration.ofMinutes(5)),
            "sessions" to RedisCacheConfiguration.defaultCacheConfig()
                .entryTtl(Duration.ofMinutes(30))
        )

        return RedisCacheManager.builder(redisConnectionFactory)
            .withInitialCacheConfigurations(cacheConfigurations)
            .build()
    }
}
```

---

## 5. 분산 락

### 5.1 Redisson 분산 락

```kotlin
@Service
class InventoryService(
    private val redissonClient: RedissonClient,
    private val inventoryRepository: InventoryRepository
) {
    fun decreaseStock(productId: Long, quantity: Int) {
        val lock = redissonClient.getLock("lock:inventory:$productId")

        try {
            if (lock.tryLock(5, 10, TimeUnit.SECONDS)) {
                val inventory = inventoryRepository.findByProductId(productId)
                    ?: throw InventoryException.NotFound(productId)

                inventory.decrease(quantity)
                inventoryRepository.save(inventory)
            } else {
                throw InventoryException.LockFailed(productId)
            }
        } finally {
            if (lock.isHeldByCurrentThread) {
                lock.unlock()
            }
        }
    }
}
```

### 5.2 락 키 규칙

```
lock:{domain}:{resource}:{identifier}
```

예시:
- `lock:inventory:123` - 상품 123 재고 락
- `lock:order:456` - 주문 456 처리 락
- `lock:payment:789` - 결제 789 처리 락

---

## 6. Pub/Sub 메시징

### 6.1 이벤트 발행

```kotlin
@Service
class OrderEventPublisher(
    private val redisTemplate: RedisTemplate<String, String>,
    private val objectMapper: ObjectMapper
) {
    fun publishOrderCreated(event: OrderCreatedEvent) {
        val channel = "events:order:created"
        val message = objectMapper.writeValueAsString(event)
        redisTemplate.convertAndSend(channel, message)
    }
}
```

### 6.2 이벤트 수신

```kotlin
@Component
class OrderEventSubscriber(
    private val objectMapper: ObjectMapper
) {
    @RedisListener(channels = ["events:order:created"])
    fun handleOrderCreated(message: String) {
        val event = objectMapper.readValue<OrderCreatedEvent>(message)
        // 이벤트 처리
    }
}
```

---

## 7. 세션 관리

### 7.1 세션 저장소

```kotlin
@Service
class SessionService(
    private val redisTemplate: RedisTemplate<String, UserSession>
) {
    private val sessionTTL = Duration.ofMinutes(30)

    fun createSession(userId: Long): String {
        val sessionId = UUID.randomUUID().toString()
        val session = UserSession(
            id = sessionId,
            userId = userId,
            createdAt = Instant.now()
        )

        redisTemplate.opsForValue().set(
            "session:$sessionId",
            session,
            sessionTTL
        )

        return sessionId
    }

    fun getSession(sessionId: String): UserSession? {
        return redisTemplate.opsForValue().get("session:$sessionId")
    }

    fun refreshSession(sessionId: String) {
        redisTemplate.expire("session:$sessionId", sessionTTL)
    }

    fun invalidateSession(sessionId: String) {
        redisTemplate.delete("session:$sessionId")
    }
}
```

---

## 8. 단위 테스트

### 8.1 Embedded Redis 테스트

```kotlin
@SpringBootTest
@TestConfiguration
class CacheServiceTest {

    @Autowired
    lateinit var productCacheService: ProductCacheService

    @Test
    fun `should cache product on first access`() {
        // Given
        val productId = 1L

        // When - 첫 번째 조회 (DB에서)
        val first = productCacheService.findById(productId)

        // When - 두 번째 조회 (캐시에서)
        val second = productCacheService.findById(productId)

        // Then
        assertThat(first).isEqualTo(second)
        // 캐시 히트 확인 (metrics 또는 로그로)
    }

    @Test
    fun `should evict cache on update`() {
        // Given
        val productId = 1L
        productCacheService.findById(productId) // 캐시 적재

        // When
        productCacheService.evict(productId)

        // Then - 캐시 미스 확인
        // (다음 조회 시 DB에서 가져옴)
    }
}
```

---

## 9. 파일 구조

```
domain/
└── cache/
    ├── config/
    │   ├── RedisConfig.kt
    │   └── CacheConfig.kt
    ├── service/
    │   ├── ProductCacheService.kt
    │   └── SessionService.kt
    ├── lock/
    │   └── DistributedLockService.kt
    └── pubsub/
        ├── EventPublisher.kt
        └── EventSubscriber.kt
```

---

## 10. 완료 체크리스트

- [ ] Redis 설정 구성
- [ ] 캐시 서비스 구현
- [ ] 캐시 키 규칙 정의
- [ ] TTL 정책 적용
- [ ] 분산 락 구현 (필요시)
- [ ] Pub/Sub 구현 (필요시)
- [ ] 단위 테스트 작성
- [ ] 캐시 무효화 전략 구현

---

## 11. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 캐시 정합성 문제 | Architect | 캐시 전략 재검토 |
| 성능 이슈 | Architect | 캐시 구조 최적화 |
| Redis 클러스터 설정 | User | 인프라 설정 |
| 데이터 직렬화 문제 | Architect | 직렬화 방식 결정 |

---

<!-- pal:convention:workers:backend:cache -->
