# Cache Worker Rules

> Claude 참조용 L1 Redis Cache 규칙

---

## Quick Reference

```
Layer: L1 Domain
Tech: Redis / Valkey (Kotlin)
Modes: Reactive (CacheGenericOperator) | Sync (Spring Data Redis)
Pattern: Cache-Aside, Write-Through
TTL: Always explicit
```

---

## Directory Structure

```
cache/{module}/
├── service/      # Cache Service
├── stored/       # data class (Reactive)
└── config/       # Redis config (Sync)
```

---

## Stored (Cache Object) Rules

### Declaration
- **MUST** use `data class`
- **MUST** be JSON serializable
- **MUST NOT** have circular references

### Example
```kotlin
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

---

## Key Naming Rules

### Pattern
```
{domain}:{entity}:{id}
{domain}:{entity}:{id}:{variant}
```

### Examples
| Pattern | Example | Description |
|---------|---------|-------------|
| Single entity | `user:123` | User ID 123 |
| Profile | `user:123:profile` | User profile |
| Session | `session:abc-def` | Session |
| Category list | `products:category:electronics` | Electronics |

### DO / DON'T
```kotlin
// ✅ Good
"user:123"
"user:123:profile"
"session:abc-def-123"

// ❌ Bad
"123"                // Unclear
"userProfile123"     // No separator
```

### Key Builder
```kotlin
companion object {
    private const val PREFIX = "user"
    fun buildKey(userId: String): String = "$PREFIX:$userId"
}
```

---

## TTL Policy

### TTL Constants (Required)
```kotlin
companion object {
    val SHORT_TTL: Duration = Duration.ofMinutes(5)
    val DEFAULT_TTL: Duration = Duration.ofHours(1)
    val LONG_TTL: Duration = Duration.ofDays(1)
    val SESSION_TTL: Duration = Duration.ofHours(24)
}
```

### TTL Guidelines
| Data Type | TTL | Reason |
|-----------|-----|--------|
| Product info | 1 hour | Rarely changes |
| Inventory | 5 min | Real-time needed |
| User info | 1 hour | Rarely changes |
| Session | 24 hours | Security |
| Temp data | 5 min | Short-lived |

### Always Pass TTL
```kotlin
// ✅ DO - Explicit TTL
cacheOperator.set(key, data, DEFAULT_TTL)

// ❌ DON'T - No TTL (infinite)
cacheOperator.set(key, data)
```

---

## Reactive Pattern (CacheGenericOperator)

### Basic Service Structure
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

    suspend fun get(userId: String): CachedUser? =
        cacheOperator.get(buildKey(userId), CachedUser::class.java)

    suspend fun set(userId: String, data: CachedUser, ttl: Duration = DEFAULT_TTL) =
        cacheOperator.set(buildKey(userId), data, ttl)

    suspend fun delete(userId: String) =
        cacheOperator.delete(buildKey(userId))

    suspend fun exists(userId: String): Boolean =
        cacheOperator.exists(buildKey(userId))
}
```

### Query Patterns
```kotlin
// nullable
suspend fun get(userId: String): CachedUser?

// throw if null
suspend fun getOrThrow(userId: String): CachedUser =
    get(userId) ?: throw CacheNotFoundException("Not found: $userId")

// default
suspend fun getOrDefault(userId: String, default: CachedUser): CachedUser =
    get(userId) ?: default

// load if missing
suspend fun getOrLoad(userId: String, loader: suspend () -> CachedUser): CachedUser =
    get(userId) ?: loader().also { set(userId, it) }
```

### Atomic Operations
```kotlin
suspend fun increment(key: String): Long
suspend fun decrement(key: String): Long
suspend fun setIfAbsent(key: String, value: T, ttl: Duration): Boolean
```

---

## Sync Pattern (Spring Data Redis)

### Cache-Aside Pattern
```kotlin
@Service
class ProductCacheService(
    private val redisTemplate: RedisTemplate<String, Product>,
    private val repository: ProductRepository
) {
    fun findById(id: Long): Product? {
        val key = "product:$id"

        // 1. Cache lookup
        redisTemplate.opsForValue().get(key)?.let { return it }

        // 2. DB lookup
        val product = repository.findByIdOrNull(id) ?: return null

        // 3. Cache store
        redisTemplate.opsForValue().set(key, product, Duration.ofHours(1))
        return product
    }

    fun evict(id: Long) = redisTemplate.delete("product:$id")
}
```

### Write-Through Pattern
```kotlin
@Transactional
fun save(product: Product): Product {
    // 1. DB save
    val saved = repository.save(product)

    // 2. Cache update
    redisTemplate.opsForValue().set("product:${saved.id}", saved, Duration.ofHours(1))

    return saved
}
```

---

## Error Handling

### Exception Classes
```kotlin
class CacheNotFoundException(message: String) : RuntimeException(message)
class CacheOperationException(message: String, cause: Throwable? = null)
    : RuntimeException(message, cause)
```

### Usage
```kotlin
suspend fun getOrThrow(key: String): CachedData {
    return try {
        cacheOperator.get(key, CachedData::class.java)
            ?: throw CacheNotFoundException("Key not found: $key")
    } catch (e: Exception) {
        throw CacheOperationException("Failed to get: $key", e)
    }
}
```

---

## Checklist

### Stored
- [ ] `data class` declaration
- [ ] JSON serializable types only
- [ ] No circular references

### Service
- [ ] Key naming: `{domain}:{entity}:{id}`
- [ ] TTL constants defined
- [ ] TTL explicitly passed
- [ ] Clear error handling

### Reactive
- [ ] `suspend` functions
- [ ] get/set/delete/exists implemented

### Sync
- [ ] Cache-Aside or Write-Through pattern
- [ ] evict method implemented

---

## Escalation

| Situation | Target | Action |
|-----------|--------|--------|
| Cache consistency issue | Architect | Strategy review |
| Write-Through vs Write-Behind | Architect | Pattern decision |
| Distributed lock needed | Architect | Redisson review |
| Redis cluster setup | User | Infrastructure |
| Capacity exceeded | Architect | Sharding strategy |

---

<!-- pal:rules:workers:backend:cache -->
