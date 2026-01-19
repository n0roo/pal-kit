# Cache Worker 컨벤션

> L1 Domain 레이어 - Redis/캐시 전문 Worker

---

## 1. 역할 정의

Cache Worker는 **L1 Domain 레이어**에서 Redis 기반 캐시 데이터 처리를 담당하는 전문 Worker입니다.

### 1.1 담당 영역

- Cache Service 구현
- Key 네이밍 전략 설계
- TTL 정책 적용
- Stored 객체 정의
- 캐시 무효화 로직 구현

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| 캐시 | Redis, Valkey |
| Sync | Spring Data Redis (RedisTemplate) |
| Reactive | Reactive Redis (CacheGenericOperator) |
| 언어 | Kotlin |

### 1.3 두 가지 구현 방식

| 항목 | Sync (Spring Data Redis) | Reactive (Generic Operator) |
|------|-------------------------|----------------------------|
| 추상화 | 고수준 (RedisTemplate) | 저수준 (Generic Operator) |
| 구조 | service + config | service + stored |
| 함수 | 일반 함수 | suspend 함수 |
| 유연성 | 제한적 | 높음 |

---

## 2. 디렉토리 구조

```
cache/{module}/
├── service/            # Cache Service
├── stored/             # 저장되는 data class (Reactive)
└── config/             # Redis 설정 (Sync)
```

---

## 3. Stored (저장 객체) 규칙

### 3.1 클래스 선언

```kotlin
// ✅ data class 사용
// ✅ 직렬화 가능한 구조
data class CachedUser(
    val id: String,
    val name: String,
    val email: String,
    val permissions: List<String> = emptyList(),
    val cachedAt: LocalDateTime = LocalDateTime.now()
)

data class CachedSession(
    val sessionId: String,
    val userId: Long,
    val metadata: Map<String, Any> = emptyMap(),
    val expiresAt: LocalDateTime
)
```

### 3.2 직렬화 고려사항

- JSON 직렬화 가능한 타입만 사용
- 순환 참조 없음
- LocalDateTime 등 시간 타입은 String 변환 고려

---

## 4. Key 네이밍 규칙

### 4.1 패턴

```
{domain}:{entity}:{identifier}
{domain}:{entity}:{identifier}:{variant}
```

### 4.2 예시

| 패턴 | 예시 | 설명 |
|------|------|------|
| 단일 엔티티 | `user:123` | 사용자 ID 123 |
| 프로필 | `user:123:profile` | 사용자 프로필 |
| 세션 | `session:abc-def-123` | 세션 |
| 목록 | `products:category:electronics` | 전자제품 카테고리 |

```kotlin
// ✅ 좋은 예
"user:123"
"user:123:profile"
"session:abc-def-123"
"cache:product:456"

// ❌ 나쁜 예
"123"                    // 의미 불명
"userProfile123"         // 구분자 없음
```

### 4.3 Key 빌더

```kotlin
companion object {
    private const val PREFIX = "user"
    fun buildKey(userId: String): String = "$PREFIX:$userId"
    fun buildProfileKey(userId: String): String = "$PREFIX:$userId:profile"
}
```

---

## 5. TTL 정책

### 5.1 TTL 상수 정의

```kotlin
companion object {
    val SHORT_TTL: Duration = Duration.ofMinutes(5)
    val DEFAULT_TTL: Duration = Duration.ofHours(1)
    val LONG_TTL: Duration = Duration.ofDays(1)
    val SESSION_TTL: Duration = Duration.ofHours(24)
}
```

### 5.2 TTL 기준

| 데이터 유형 | TTL | 이유 |
|------------|-----|------|
| 상품 정보 | 1시간 | 자주 변경되지 않음 |
| 재고 정보 | 5분 | 실시간성 필요 |
| 사용자 정보 | 1시간 | 자주 변경되지 않음 |
| 세션 | 24시간 | 보안 요구사항 |
| 임시 데이터 | 5분 | 짧은 수명 |

### 5.3 TTL 명시적 전달

```kotlin
// ✅ TTL 명시적 전달
suspend fun cacheWithTTL(key: String, data: Any, ttl: Duration = DEFAULT_TTL) {
    cacheOperator.set(key, data, ttl)
}

// ❌ TTL 없이 저장 (무기한 - 지양)
suspend fun cacheWithoutTTL(key: String, data: Any) {
    cacheOperator.set(key, data)  // 무기한 저장됨
}
```

---

## 6. Reactive Service 패턴 (CacheGenericOperator)

### 6.1 기본 구조

```kotlin
@Service
class UserCacheService(
    private val cacheOperator: CacheGenericOperator
) {
    companion object {
        private const val PREFIX = "user"
        val DEFAULT_TTL: Duration = Duration.ofHours(1)
    }

    private fun buildKey(userId: String): String = "$PREFIX:$userId"

    // 조회
    suspend fun get(userId: String): CachedUser? {
        return cacheOperator.get(buildKey(userId), CachedUser::class.java)
    }

    // 저장
    suspend fun set(userId: String, data: CachedUser, ttl: Duration = DEFAULT_TTL) {
        cacheOperator.set(buildKey(userId), data, ttl)
    }

    // 삭제
    suspend fun delete(userId: String) {
        cacheOperator.delete(buildKey(userId))
    }

    // 존재 여부
    suspend fun exists(userId: String): Boolean {
        return cacheOperator.exists(buildKey(userId))
    }
}
```

### 6.2 조회 패턴

```kotlin
// nullable 반환
suspend fun get(userId: String): CachedUser? {
    return cacheOperator.get(buildKey(userId), CachedUser::class.java)
}

// throw if null
suspend fun getOrThrow(userId: String): CachedUser {
    return cacheOperator.get(buildKey(userId), CachedUser::class.java)
        ?: throw CacheNotFoundException("Cache not found: $userId")
}

// default 반환
suspend fun getOrDefault(userId: String, default: CachedUser): CachedUser {
    return cacheOperator.get(buildKey(userId), CachedUser::class.java) ?: default
}

// 조회 후 없으면 로드
suspend fun getOrLoad(userId: String, loader: suspend () -> CachedUser): CachedUser {
    return get(userId) ?: loader().also { set(userId, it) }
}
```

### 6.3 Atomic 연산

```kotlin
// 카운터
suspend fun increment(key: String): Long
suspend fun decrement(key: String): Long

// 조건부 저장
suspend fun setIfAbsent(key: String, value: T, ttl: Duration): Boolean
```

---

## 7. Sync Service 패턴 (Spring Data Redis)

### 7.1 Cache-Aside 패턴

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

### 7.2 Write-Through 패턴

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

## 8. 에러 처리

```kotlin
// 명시적 예외
class CacheNotFoundException(message: String) : RuntimeException(message)
class CacheOperationException(message: String, cause: Throwable? = null)
    : RuntimeException(message, cause)

// 사용
suspend fun getOrThrow(key: String): CachedData {
    return try {
        cacheOperator.get(key, CachedData::class.java)
            ?: throw CacheNotFoundException("Key not found: $key")
    } catch (e: Exception) {
        throw CacheOperationException("Failed to get cache: $key", e)
    }
}
```

---

## 9. 체크리스트

### Stored
- [ ] data class 사용
- [ ] 직렬화 가능한 타입만
- [ ] 순환 참조 없음

### Service
- [ ] Key 네이밍 규칙 준수 ({domain}:{entity}:{id})
- [ ] TTL 상수 정의
- [ ] TTL 명시적 전달
- [ ] 명확한 에러 처리

### Reactive (CacheGenericOperator)
- [ ] suspend 함수 사용
- [ ] get/set/delete/exists 구현

### Sync (Spring Data Redis)
- [ ] Cache-Aside 또는 Write-Through 패턴
- [ ] evict 메서드 구현

---

## 10. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 캐시 정합성 문제 | Architect | 캐시 전략 재검토 |
| 캐시 일관성 전략 | Architect | Write-Through vs Write-Behind 결정 |
| 분산 락 필요 | Architect | Redisson 도입 검토 |
| Redis 클러스터 설정 | User | 인프라 설정 |
| 캐시 용량 초과 예상 | Architect | 샤딩/클러스터 전략 |

---

<!-- pal:convention:workers:backend:cache -->
