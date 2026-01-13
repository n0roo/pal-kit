# Port: pal-marker

> PAL (Port-Adapter Layered) 아키텍처 코드 문서화 표준

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | pal-marker |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | - |
| 예상 복잡도 | medium |

---

## 목표

PAL 마커는 코드에 아키텍처 메타데이터를 부여하여:
- **인덱싱**: PAL 도구가 코드베이스를 분석하고 의존성 그래프 생성
- **검색**: Port/Layer/Domain 기반 빠른 코드 탐색
- **일관성**: Claude 생성 코드 추적 및 품질 관리

**범위 제한**: 커밋 연동은 제외. 워크플로우 안정화 전까지 커밋은 수동 진행.

---

## 표준 태그

### 필수 태그

| 태그 | 설명 | 형식 | 예시 |
|------|------|------|------|
| `@pal-port` | Port 식별자 | `{Layer}-{PortName}` | `L1-InventoryCommandService` |
| `@pal-layer` | 레이어 구분 | `l1`, `lm`, `l2` | `l1` |
| `@pal-domain` | 도메인 영역 | 소문자 복수형 | `inventories`, `units` |

### 선택 태그

| 태그 | 설명 | 형식 | 예시 |
|------|------|------|------|
| `@see` | 관련 클래스 참조 | KDoc 표준 | `@see InventoryCommandTrigger 이벤트 클래스` |
| `@pal-adapter` | 어댑터 구현체 | 어댑터 이름 | `JpaInventoryRepository` |
| `@pal-depends` | 의존 Port | Port 식별자 목록 | `L1-BaseUnitQueryService` |

### Claude 생성 코드 태그

| 태그 | 설명 | 위치 |
|------|------|------|
| `@pal-generated` | Claude 생성 코드 마커 | KDoc 내부 |
| `@Comment` | 기능 상세 설명 | KDoc 내부 |

---

## 레이어별 Port 명명 규칙

### L1 (Domain Layer)

```kotlin
/**
 * 인벤토리 커맨드 서비스
 *
 * 인벤토리 CRUD 작업을 처리하고, 작업 완료 후 이벤트를 발행합니다.
 *
 * @pal-port L1-InventoryCommandService
 * @pal-layer l1
 * @pal-domain inventories
 * @pal-generated
 *
 * @see InventoryCommandTrigger 커맨드 이벤트
 * @see InventoriesRepository JPA 저장소
 */
@Service
class InventoryCommandService { ... }
```

### L1 Query Service

```kotlin
/**
 * 기본 단위 조회 서비스
 *
 * BaseUnits 엔티티 조회 및 DSL Template 활용
 *
 * @pal-port L1-BaseUnitQueryService
 * @pal-layer l1
 * @pal-domain units
 * @pal-generated
 *
 * @see BaseUnitQueryDSLTemplate Jooq DSL 템플릿
 */
@Service
class BaseUnitQueryService { ... }
```

### L1 DSL Template

```kotlin
/**
 * 기본 단위 조회 DSL 템플릿
 *
 * Jooq를 활용한 동적 쿼리 생성
 *
 * @pal-port L1-BaseUnitQueryDSLTemplate
 * @pal-layer l1
 * @pal-domain units
 * @pal-adapter jooq
 * @pal-generated
 *
 * @Comment Optional 필터 조건을 Jooq DSL로 처리
 */
@Repository
class BaseUnitQueryDSLTemplate { ... }
```

### LM (Shared Composition Layer)

```kotlin
/**
 * 배출 계산 코디네이터
 *
 * @pal-port LM-EmissionCalculationCoordinator
 * @pal-layer lm
 * @pal-domain emissions
 * @pal-depends L1-FormulaQueryService, L1-BaseUnitQueryService
 */
@Service
class EmissionCalculationCoordinator { ... }
```

### L2 (Feature Composition Layer)

```kotlin
/**
 * Admin Structure 조회 Composite 서비스
 *
 * @pal-port L2-AdminStructureQueryCompositeService
 * @pal-layer l2
 * @pal-domain admin
 * @pal-depends L1-FormulaBaseStructureQueryService
 * @pal-generated
 *
 * @Comment Backoffice Structure 목록/상세 조회 API 구현
 */
@Service
class AdminStructureQueryCompositeService { ... }
```

---

## 도메인 목록

| 도메인 | 설명 | 예시 Port |
|--------|------|-----------|
| `units` | 단위 관리 | L1-BaseUnitQueryService |
| `formulas` | 수식/규칙 관리 | L1-FormulaBaseStructureQueryService |
| `inventories` | 인벤토리 관리 | L1-InventoryCommandService |
| `emissions` | 배출량 계산 | LM-EmissionCalculationCoordinator |
| `admin` | Backoffice 기능 | L2-AdminStructureQueryCompositeService |
| `projects` | 프로젝트 관리 | L2-ProjectCompositeService |

---

## 지원 언어

| 언어 | 주석 형식 | 태그 위치 |
|------|----------|----------|
| Kotlin | `/** ... */` (KDoc) | KDoc 내부 |
| Go | `// ...` | 함수/타입 상단 |
| TypeScript | `/** ... */` (JSDoc) | JSDoc 내부 |
| Python | `"""..."""` (docstring) | docstring 내부 |

---

## CLI 명령어

```bash
# 마커 목록
pal marker list                    # 전체 마커
pal marker list --port L1-*        # L1 레이어 마커
pal marker list --domain units     # 특정 도메인

# 마커 검증
pal marker check                   # 유효하지 않은 마커 검출
pal marker check --strict          # 필수 태그 누락 검출

# Port 검색
pal marker files L1-InventoryCommandService  # 포트가 마킹된 파일

# 의존성 그래프
pal marker deps L2-AdminStructureQueryCompositeService  # 의존성 트리

# Claude 생성 코드 검색
pal marker generated               # @pal-generated 마커 파일 목록
```

---

## PAL 인덱싱 지원

### 검색 패턴

```bash
# Port 검색
grep -r "@pal-port L1-" --include="*.kt"

# 레이어별 검색
grep -r "@pal-layer l2" --include="*.kt"

# 도메인별 검색
grep -r "@pal-domain inventories" --include="*.kt"

# Claude 생성 코드 검색
grep -r "@pal-generated" --include="*.kt"
```

### 의존성 그래프 생성

```
L2-AdminStructureQueryCompositeService
    └── @pal-depends L1-FormulaBaseStructureQueryService
                         └── @pal-adapter FormulaBaseStructureQueryDSLTemplate
```

---

## Kotlin Idiom 가이드

### Null 처리

```kotlin
// ✅ Kotlin idiom - let 사용
typeCode?.let { condition = condition.and(DB.table.TYPE_CODE.eq(it)) }

// ❌ Java 스타일
if (typeCode != null) {
    condition = condition.and(DB.table.TYPE_CODE.eq(typeCode))
}
```

### 다중 Optional 필터

```kotlin
// ✅ 권장 - 각 필터를 let으로 처리
fun fetchWithFilters(
    scope: Scope? = null,
    tier: Tier? = null,
    activated: Boolean? = null
): List<Long> {
    var condition = DB.table.DELETED.eq(false)

    scope?.let { condition = condition.and(DB.table.SCOPE.eq(it.name)) }
    tier?.let { condition = condition.and(DB.table.TIER.eq(it.name)) }
    activated?.let { condition = condition.and(DB.table.ACTIVATED.eq(it)) }

    return dslContext.select(DB.table.ID)
        .from(DB.table)
        .where(condition)
        .fetchInto(Long::class.java)
}
```

### Elvis 연산자 활용

```kotlin
// ✅ Elvis 연산자
val page = params.page ?: 1
val size = params.size ?: 20

// ✅ takeIf 활용
val activeOnly = params.activated?.takeIf { it }
```

---

## 마이그레이션 가이드

### 기존 코드 → PAL 마커 적용

**Before:**
```kotlin
/**
 * 서비스 설명
 * pal-marker: claude-generated
 * @Comment 상세 설명
 */
@Service
class SomeService { ... }
```

**After:**
```kotlin
/**
 * 서비스 설명
 *
 * 상세 설명
 *
 * @pal-port L1-SomeService
 * @pal-layer l1
 * @pal-domain some-domain
 * @pal-generated
 *
 * @Comment 상세 설명
 * @see RelatedClass 관련 클래스
 */
@Service
class SomeService { ... }
```

---

## 구현 가이드

### P1: 마커 파싱

- [x] 정규식 기반 마커 추출 (KDoc, JSDoc, docstring)
- [x] 다중 언어 주석 형식 지원
- [x] 마커 정보 구조체 정의

### P2: CLI 명령어

- [x] `pal marker list` 구현
- [x] `pal marker check` 구현
- [x] `pal marker files` 구현
- [x] `pal marker deps` 구현
- [x] `pal marker index` 구현
- [x] `pal marker stats` 구현
- [x] `pal marker graph` 구현

### P3: 인덱싱 연동

- [x] code_markers 테이블에 마커 정보 저장
- [x] 의존성 그래프 생성 (code_marker_deps 테이블)
- [x] 포트 명세와 코드 연결 (marker_port_links 테이블)

---

## 체크리스트 (새 코드 작성 시)

- [ ] 클래스에 `@pal-port` 태그 추가
- [ ] `@pal-layer` (l1/lm/l2) 명시
- [ ] `@pal-domain` 도메인 영역 명시
- [ ] Claude 생성 시 `@pal-generated` 추가
- [ ] 관련 클래스 `@see` 참조 추가
- [ ] Kotlin idiom 준수 (let, takeIf, Elvis)
- [ ] Import 간소화 (파라미터에 패키지 경로 노출 금지)

---

## 완료 기준

- [x] 코드에서 `@pal-port`, `@pal-layer`, `@pal-domain` 마커 추출 가능
- [x] Port ID로 관련 코드 파일 검색 가능
- [x] 의존성 그래프 생성 가능
- [x] Claude 생성 코드 추적 가능

---

<!-- pal:port:status=complete -->
