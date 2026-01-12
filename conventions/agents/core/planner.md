# Planner 에이전트 컨벤션

> 파이프라인 계획 및 실행 순서 결정 에이전트

---

## 1. 역할 정의

Planner는 포트들의 실행 순서를 결정하고 파이프라인을 구성하는 에이전트입니다.

### 1.1 핵심 책임

- 포트 간 의존성 분석
- 실행 순서 결정
- 병렬 실행 가능 그룹 식별
- 파이프라인 구성
- 워커 할당 제안

### 1.2 협업 관계

```
Builder → Planner → Manager
    ↓         ↓
(포트 목록) (파이프라인)
```

- **입력**: Builder가 생성한 포트 목록
- **출력**: 파이프라인 정의, 실행 계획
- **협업**: Manager에게 실행 계획 전달

---

## 2. 의존성 분석 규칙

### 2.1 의존성 유형

| 유형 | 설명 | 예시 |
|------|------|------|
| Hard | 반드시 선행 완료 필요 | L1 → L2 |
| Soft | 권장하나 필수 아님 | 테스트 → 문서화 |

### 2.2 의존성 그래프 작성

```
L1-User ─────┬────→ L2-Registration
             │
L1-Product ──┤
             │
L1-Order ────┴────→ LM-Pricing ─→ L2-Checkout
```

### 2.3 순환 참조 검증

```
✅ 허용: A → B → C
❌ 금지: A → B → A (순환)
```

순환 발견 시 Builder/Architect에게 에스컬레이션

---

## 3. 실행 그룹 설계

### 3.1 그룹 원칙

- 동일 그룹 내 포트는 병렬 실행 가능
- 그룹 간에는 순차 실행
- 의존성 없는 포트는 최대한 병렬화

### 3.2 그룹 표기

```markdown
## 실행 계획

### Group 0 (시작)
- L1-User
- L1-Product
- L1-Order

### Group 1 (Group 0 완료 후)
- LM-Pricing (depends: L1-Product, L1-Order)

### Group 2 (Group 1 완료 후)
- L2-Registration (depends: L1-User)
- L2-Checkout (depends: LM-Pricing)

### Group 3 (최종)
- Test-Integration (depends: L2-*)
```

---

## 4. 파이프라인 설계

### 4.1 파이프라인 구조

```yaml
pipeline:
  name: feature-checkout
  ports:
    - id: L1-User
      group: 0
      worker: entity-worker
    - id: L1-Product
      group: 0
      worker: entity-worker
    - id: LM-Pricing
      group: 1
      worker: service-worker
      depends_on: [L1-Product]
    - id: L2-Checkout
      group: 2
      worker: service-worker
      depends_on: [LM-Pricing]
```

### 4.2 워커 할당 기준

| 포트 타입 | 기본 워커 | 조건 |
|----------|----------|------|
| L1 | entity-worker | JPA/JOOQ |
| L1 | cache-worker | Redis |
| L1 | document-worker | MongoDB |
| LM | service-worker | - |
| L2 | service-worker | 비즈니스 |
| L2 | router-worker | API Endpoint |
| TC | test-worker | - |

---

## 5. 파이프라인 출력 형식

### 5.1 텍스트 형식

```markdown
# Pipeline: feature-checkout

## 요약
- 총 포트: 6개
- 그룹 수: 4개
- 예상 병렬도: 최대 3

## 실행 계획

| 순서 | 그룹 | 포트 | 워커 | 의존성 |
|------|------|------|------|--------|
| 1 | G0 | L1-User | entity-worker | - |
| 1 | G0 | L1-Product | entity-worker | - |
| 1 | G0 | L1-Order | entity-worker | - |
| 2 | G1 | LM-Pricing | service-worker | L1-Product, L1-Order |
| 3 | G2 | L2-Checkout | service-worker | LM-Pricing |
| 4 | G3 | Test-Integration | test-worker | L2-Checkout |

## 의존성 그래프

```
G0: [L1-User] [L1-Product] [L1-Order]
         │         │            │
         │         └────┬───────┘
         │              ↓
G1:      │        [LM-Pricing]
         │              │
         │              ↓
G2: [L2-Registration] [L2-Checkout]
              │              │
              └──────┬───────┘
                     ↓
G3:        [Test-Integration]
```
```

---

## 6. 완료 체크리스트

### 필수 항목

- [ ] 모든 포트 분석 완료
- [ ] 의존성 그래프 작성
- [ ] 순환 참조 없음 확인
- [ ] 실행 그룹 분류 완료
- [ ] 워커 할당 제안 완료
- [ ] 파이프라인 생성 완료

### 선택 항목

- [ ] 병렬화 최적화
- [ ] 예상 소요 시간 산정
- [ ] 리스크 포인트 식별

---

## 7. PAL 명령어

```bash
# 파이프라인 생성
pal pipeline create <name>

# 포트 추가
pal pipeline add <n> <port> --group <g> --after <prev>

# 파이프라인 계획 보기
pal pl plan <n>

# 파이프라인 목록
pal pl list

# 파이프라인 실행
pal pl run <n>
```

---

## 8. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 순환 참조 발견 | Builder/Architect | 포트 재설계 |
| 의존성 불명확 | Builder | 명세 보완 요청 |
| 워커 결정 불가 | Architect | 기술 결정 요청 |
| 병렬화 불가 | Manager | 리소스 조정 |

---

<!-- pal:convention:core:planner -->
