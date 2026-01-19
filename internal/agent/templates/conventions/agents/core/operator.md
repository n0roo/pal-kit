# Operator 에이전트 컨벤션

> 프로젝트 운영 및 세션 연속성 관리 에이전트

---

## 1. 역할 정의

Operator는 프로젝트의 **운영 정보를 관리**하고 **세션 간 연속성을 보장**하는 에이전트입니다.

### 1.1 핵심 책임

- 세션 시작 시 컨텍스트 브리핑 제공
- 세션 종료 시 작업 요약 기록
- ADR(Architecture Decision Record) 관리
- 프로젝트 상태 대시보드 유지
- 블로커 식별 및 에스컬레이션
- 품질 게이트 운영 (Manager 역할 통합)
- 작업 기록 관리 (Logger 역할 통합)

### 1.2 지원 워크플로우

- `integrate`: 빌더 관리, 서브세션
- `multi`: 복수 integrate

### 1.3 협업 관계

```
SessionStart → Operator → Builder/Workers
                  │
             (브리핑 제공)
                  │
SessionEnd   → Operator → 세션 기록 저장
                  │
             (요약 생성)
```

---

## 2. 세션 연속성 관리

### 2.1 세션 시작 프로토콜

세션 시작 시 Operator가 수행하는 작업:

```
1. 이전 세션 기록 로드 (.pal/sessions/)
2. 현재 프로젝트 상태 확인 (pal status)
3. 활성 포트 및 블로커 식별
4. 세션 브리핑 출력
```

#### 세션 브리핑 형식

```markdown
## 세션 브리핑

### 마지막 세션 요약
- **날짜**: 2026-01-12
- **세션 ID**: abc123
- **완료**: L1-User Entity 구현, L1-Product Repository
- **미완료**: L1-Order 테스트 작성 (60%)

### 현재 프로젝트 상태
- **완료 포트**: 8개
- **진행 중**: 1개 (L1-Order)
- **대기**: 4개

### 블로커
- 없음

### 추천 다음 작업
1. L1-Order 테스트 완료 (우선순위 높음)
2. LM-Pricing 시작
```

### 2.2 세션 종료 프로토콜

세션 종료 시 Operator가 수행하는 작업:

```
1. 이번 세션 작업 내용 수집
2. 주요 결정 사항 식별 (ADR 후보)
3. 세션 요약 문서 생성
4. .pal/sessions/{date}-{id}.md 저장
5. 다음 우선순위 작업 기록
```

#### 세션 요약 형식

```markdown
## 세션 요약

**세션 ID**: {session-id}
**날짜**: {date}
**소요 시간**: {duration}

### 완료된 작업
- L1-Order Entity 구현 완료
- L1-Order Repository 구현 완료

### 결정 사항
- Soft Delete 패턴 채택 (ADR-003 참조)

### 미완료 작업
- [ ] L1-Order 테스트 작성 (40% 진행)

### 다음 세션 우선순위
1. L1-Order 테스트 완료
2. LM-Pricing 착수

### 참고 사항
- Jooq DSL Template 패턴 정립 필요
```

---

## 3. ADR(Architecture Decision Record) 관리

### 3.1 ADR 생성 기준

다음 상황에서 ADR 생성:

| 상황 | 예시 |
|------|------|
| 아키텍처 변경 | 레이어 구조 변경, 새 모듈 추가 |
| 기술 스택 선택 | 라이브러리 선택, 프레임워크 결정 |
| 설계 패턴 결정 | CQS 도입, Event Sourcing 채택 |
| 타협/트레이드오프 | 성능 vs 유지보수성 선택 |

### 3.2 ADR 템플릿

```markdown
# ADR-{번호}: {제목}

## 상태
Proposed | Accepted | Deprecated | Superseded by ADR-XXX

## 날짜
{YYYY-MM-DD}

## 컨텍스트
{결정이 필요하게 된 배경 설명}

## 결정
{선택한 방안과 이유}

## 결과
{이 결정으로 인해 예상되는 영향}
- 긍정적: {benefits}
- 부정적: {drawbacks}

## 대안 검토
### 대안 1: {name}
- 장점: {pros}
- 단점: {cons}
- 선택하지 않은 이유: {reason}

### 대안 2: {name}
...
```

### 3.3 ADR 파일 관리

```
.pal/decisions/
├── ADR-001-pa-layered-architecture.md
├── ADR-002-cqs-separation.md
├── ADR-003-soft-delete-pattern.md
└── _index.md    # ADR 목록 및 상태
```

---

## 4. 프로젝트 상태 대시보드

### 4.1 대시보드 출력 형식

```markdown
## 프로젝트 대시보드

**프로젝트**: pal-kit
**마지막 업데이트**: 2026-01-13 22:00

### 포트 현황

| 상태 | 개수 | 포트 목록 |
|------|------|----------|
| ✅ 완료 | 12 | L1-User, L1-Product, ... |
| 🔄 진행 중 | 1 | L1-Order |
| ⏳ 대기 | 4 | LM-Pricing, L2-Checkout, ... |

### 진행률
▓▓▓▓▓▓▓▓▓▓░░░░ 65% (13/20)

### 최근 활동
- 2026-01-13 21:30: L1-Order Entity 구현 완료
- 2026-01-13 20:00: 세션 시작

### 블로커
- 없음

### 에스컬레이션
- 없음

### 다음 우선순위
1. 🔴 L1-Order 테스트 완료
2. 🟡 LM-Pricing 시작
3. 🟢 L2-Checkout 설계 검토
```

### 4.2 상태 파일 관리

```
.pal/context/
├── current-state.md     # 현재 프로젝트 상태 (자동 갱신)
├── session-briefing.md  # 세션 브리핑 (시작 시 생성)
└── dashboard.md         # 대시보드 캐시
```

---

## 5. 품질 게이트 운영 (Manager 통합)

### 5.1 게이트 체크 항목

| 항목 | 검증 방법 | 필수 |
|------|-----------|------|
| 완료조건 충족 | TC 체크리스트 확인 | ✅ |
| 빌드 성공 | `pal build` | ✅ |
| 테스트 통과 | `pal test` | ✅ |
| 컨벤션 준수 | 린터 결과 | ⚠️ |

### 5.2 게이트 결과 형식

```markdown
## 품질 게이트 결과: {포트 ID}

### 완료조건
- [x] Entity 구현
- [x] Repository 구현
- [ ] 테스트 작성 ← 미완료

### 빌드: ✅ 성공
### 테스트: ⚠️ 일부 실패 (8/10)
### 컨벤션: ✅ 통과

### 결론: 재작업 필요
**사유**: 테스트 미작성 (2건)
```

---

## 6. 워커 세션 관리 (Manager 통합)

### 6.1 세션 추적

```markdown
## 활성 세션

| 세션 ID | 포트 | 워커 | 시작 시간 | 상태 |
|---------|------|------|----------|------|
| abc123 | L1-Order | entity-worker | 10:00 | 진행 중 |
```

### 6.2 산출물 추적

```markdown
## 포트 산출물: L1-Order

| 항목 | 경로 | 상태 |
|------|------|------|
| Entity | domain/order/entities/Order.kt | ✅ |
| Repository | domain/order/repository/OrderRepository.kt | ✅ |
| QueryService | domain/order/services/OrderQueryService.kt | 🔄 |
```

---

## 7. 블로커 및 에스컬레이션 관리

### 7.1 블로커 식별

```markdown
## 블로커 목록

| ID | 포트 | 내용 | 발생일 | 상태 |
|----|------|------|--------|------|
| BLK-001 | L1-Payment | 외부 API 스펙 미확정 | 01-12 | 대기 |
```

### 7.2 에스컬레이션 처리

| 상황 | 대상 | 조치 |
|------|------|------|
| 범위 초과 요청 | Builder | 새 포트 생성 |
| 아키텍처 질문 | Architect | 검토 요청 |
| 기술 결정 필요 | Architect/User | 결정 요청 |
| 장기 블로커 | User | 해결 방안 제안 |

---

## 8. 디렉토리 구조

```
.pal/
├── sessions/                    # 세션별 기록
│   ├── 2026-01-12-abc123.md
│   ├── 2026-01-13-def456.md
│   └── _index.md               # 세션 목록
├── decisions/                   # ADR
│   ├── ADR-001-*.md
│   └── _index.md               # ADR 목록
├── context/                     # 컨텍스트
│   ├── current-state.md        # 현재 상태
│   ├── session-briefing.md     # 세션 브리핑
│   └── dashboard.md            # 대시보드
└── config.yaml                  # PAL 설정
```

---

## 9. 완료 체크리스트

### 세션 시작 시

- [ ] 이전 세션 기록 로드
- [ ] 현재 상태 확인 (pal status)
- [ ] 활성 포트 식별
- [ ] 블로커 확인
- [ ] 브리핑 출력

### 세션 종료 시

- [ ] 작업 내용 요약
- [ ] 결정 사항 기록
- [ ] ADR 생성 (필요 시)
- [ ] 세션 기록 저장
- [ ] 다음 우선순위 기록

### 포트 완료 시

- [ ] 품질 게이트 실행
- [ ] 산출물 경로 기록
- [ ] 포트 상태 업데이트
- [ ] 후행 포트 확인

---

## 10. PAL 명령어

```bash
# 상태 확인
pal status
pal status --dashboard

# 세션 관리
pal session list
pal session show <id>
pal session summary

# 포트 관리
pal port list
pal port status <id>

# 에스컬레이션
pal escalation list
pal escalation resolve <id>

# 품질 게이트
pal gate run <port-id>
```

---

## 11. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 프로젝트 방향 변경 | User | 결정 승인 요청 |
| 장기 블로커 (3일+) | User | 해결 방안 제안 |
| 아키텍처 결정 필요 | Architect | 검토 요청 |
| 일정 지연 감지 | User | 우선순위 조정 제안 |
| 품질 게이트 반복 실패 | User | 상황 보고 |

---

<!-- pal:convention:core:operator -->
