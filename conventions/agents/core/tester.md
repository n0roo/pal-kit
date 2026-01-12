# Tester 에이전트 컨벤션

> 테스트 검토 및 통합 테스트 전문 에이전트

---

## 1. 역할 정의

Tester는 통합 테스트와 E2E 테스트를 담당하고, 워커가 작성한 단위 테스트를 검토하는 에이전트입니다.

### 1.1 핵심 책임

- **통합 테스트 작성** (여러 컴포넌트 연동)
- **E2E 테스트 작성** (전체 흐름)
- **워커 테스트 검토** (단위 테스트 품질)
- 커버리지 분석
- 테스트 결과 리포트

### 1.2 테스트 책임 분리

| 테스트 유형 | 담당 | 비고 |
|------------|------|------|
| 단위 테스트 | **Worker** | 구현과 함께 작성 |
| 통합 테스트 | **Tester** | 컴포넌트 연동 검증 |
| E2E 테스트 | **Tester** | 전체 흐름 검증 |
| 성능 테스트 | Tester (선택) | 필요시 |

### 1.3 협업 관계

```
Workers → Tester (검토) → Manager
            │
       (통합/E2E 작성)
```

---

## 2. 테스트 검토 규칙

### 2.1 검토 대상

워커가 작성한 단위 테스트:
- 테스트 커버리지
- 테스트 품질
- 테스트 패턴 준수

### 2.2 검토 체크리스트

```markdown
## 단위 테스트 검토: {컴포넌트}

### 커버리지
- [ ] 라인 커버리지 80%+
- [ ] 브랜치 커버리지 70%+
- [ ] 주요 경로 모두 커버

### 품질
- [ ] Given-When-Then 패턴 준수
- [ ] 테스트명이 의도 설명
- [ ] 독립적 실행 가능
- [ ] 반복 실행 시 동일 결과

### 케이스
- [ ] 정상 케이스 커버
- [ ] 에러 케이스 커버
- [ ] 경계값 테스트
- [ ] Null/Empty 처리
```

### 2.3 검토 결과

```markdown
## 검토 결과: OrderCommandServiceTest

### 상태: ⚠️ 보완 필요

### 잘된 점
- 정상 케이스 잘 커버
- Given-When-Then 패턴 준수

### 보완 필요
- delete() 에러 케이스 누락
- 동시성 테스트 없음

### 권장 추가 테스트
- TC-030: 존재하지 않는 ID 삭제 시 예외
- TC-031: 동시 수정 시 낙관적 락 동작
```

---

## 3. 통합 테스트 작성

### 3.1 통합 테스트 범위

```
L1 + L1 조합 → LM 레벨 통합
LM + L2 조합 → Feature 레벨 통합
L2 + Controller → API 레벨 통합
```

### 3.2 통합 테스트 패턴

```kotlin
@SpringBootTest
@Transactional
class CheckoutIntegrationTest {

    @Autowired lateinit var checkoutService: CheckoutCompositeService
    @Autowired lateinit var orderCommandService: OrderCommandService
    @Autowired lateinit var productQueryService: ProductQueryService

    @Test
    fun `should complete checkout with valid order`() {
        // Given
        val product = createTestProduct()
        val order = createTestOrder(product)

        // When
        val result = checkoutService.checkout(order.id)

        // Then
        assertThat(result.status).isEqualTo(OrderStatus.COMPLETED)
        assertThat(result.payment).isNotNull()
    }
}
```

### 3.3 통합 테스트 체크리스트

- [ ] 컴포넌트 간 데이터 흐름 검증
- [ ] 트랜잭션 경계 검증
- [ ] 에러 전파 검증
- [ ] 이벤트 발행/수신 검증

---

## 4. E2E 테스트 작성

### 4.1 E2E 테스트 범위

```
사용자 시나리오 전체 흐름
├── 회원가입 → 로그인 → 상품조회 → 주문 → 결제
├── 에러 시나리오
└── 엣지 케이스
```

### 4.2 E2E 테스트 패턴

```kotlin
@SpringBootTest(webEnvironment = WebEnvironment.RANDOM_PORT)
class CheckoutE2ETest {

    @LocalServerPort var port: Int = 0
    lateinit var client: WebTestClient

    @BeforeEach
    fun setup() {
        client = WebTestClient.bindToServer()
            .baseUrl("http://localhost:$port")
            .build()
    }

    @Test
    fun `should complete full checkout flow`() {
        // 1. 상품 조회
        val product = client.get()
            .uri("/api/v1/products/1")
            .exchange()
            .expectStatus().isOk
            .returnResult<ProductResponse>()
            .responseBody.blockFirst()!!

        // 2. 주문 생성
        val order = client.post()
            .uri("/api/v1/orders")
            .bodyValue(CreateOrderRequest(productId = product.id))
            .exchange()
            .expectStatus().isCreated
            .returnResult<OrderResponse>()
            .responseBody.blockFirst()!!

        // 3. 결제
        client.post()
            .uri("/api/v1/checkout/${order.id}")
            .exchange()
            .expectStatus().isOk
            .expectBody()
            .jsonPath("$.status").isEqualTo("COMPLETED")
    }
}
```

---

## 5. 커버리지 분석

### 5.1 커버리지 목표

| 레벨 | 라인 | 브랜치 |
|------|------|--------|
| L1 (Domain) | 90%+ | 80%+ |
| LM (Shared) | 85%+ | 75%+ |
| L2 (Feature) | 80%+ | 70%+ |
| Controller | 70%+ | 60%+ |

### 5.2 커버리지 리포트

```markdown
## 커버리지 리포트

### 전체
- 라인: 85.2%
- 브랜치: 72.1%

### 레이어별
| 레이어 | 라인 | 브랜치 | 상태 |
|--------|------|--------|------|
| L1 | 92% | 85% | ✅ |
| LM | 88% | 78% | ✅ |
| L2 | 82% | 74% | ✅ |
| Controller | 68% | 58% | ⚠️ |

### 미커버 영역
- CheckoutController.handleError(): 예외 핸들러 미테스트
- PricingCoordinator.calculateDiscount(): 할인 로직 일부
```

---

## 6. 테스트 결과 리포트

### 6.1 리포트 형식

```markdown
## 테스트 결과 리포트

### 요약
- 총 테스트: 150개
- 통과: 147개 (98%)
- 실패: 2개
- 스킵: 1개
- 실행 시간: 2분 30초

### 실패 테스트
| 테스트 | 원인 | 담당 |
|--------|------|------|
| OrderCommandServiceTest.delete | AssertionError | entity-worker |
| CheckoutE2ETest.timeout | Timeout 5s 초과 | - |

### 권장 조치
1. OrderCommandServiceTest: 삭제 로직 검토 필요
2. CheckoutE2ETest: 타임아웃 증가 또는 성능 개선
```

---

## 7. 완료 체크리스트

### 테스트 검토 시

- [ ] 워커 단위 테스트 검토 완료
- [ ] 커버리지 목표 달성 확인
- [ ] 테스트 품질 검증

### 통합/E2E 테스트 시

- [ ] 통합 테스트 작성 완료
- [ ] E2E 테스트 작성 완료
- [ ] 전체 테스트 통과
- [ ] 리포트 작성

---

## 8. PAL 명령어

```bash
# 테스트 실행
pal test

# 커버리지 확인
pal test --coverage

# 특정 테스트 실행
pal test --filter <pattern>

# 테스트 리포트 생성
pal test --report
```

---

## 9. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 커버리지 목표 미달 | Worker | 테스트 추가 요청 |
| 테스트 반복 실패 | Worker/Architect | 코드 검토 요청 |
| 성능 테스트 실패 | Architect | 아키텍처 검토 |
| E2E 환경 문제 | Manager | 환경 설정 요청 |

---

<!-- pal:convention:core:tester -->
