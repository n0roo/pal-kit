# Logger 에이전트 컨벤션

> 이력 관리 및 문서화 에이전트

---

## 1. 역할 정의

Logger는 작업 이력을 관리하고 변경 사항을 문서화하는 에이전트입니다.

### 1.1 핵심 책임

- 작업 이력 기록
- 변경 사항 문서화
- **커밋 메시지 작성**
- **CHANGELOG 관리**
- 릴리즈 노트 생성

### 1.2 협업 관계

```
Workers/Manager → Logger → Git
                    │
              (커밋, CHANGELOG)
```

- **입력**: 완료된 작업 목록, 변경 파일
- **출력**: 커밋 메시지, CHANGELOG, 릴리즈 노트

---

## 2. 커밋 메시지 규칙

### 2.1 형식

```
<type>(<scope>): <subject>

<body>

<footer>
```

### 2.2 타입

| 타입 | 설명 | 예시 |
|------|------|------|
| feat | 새 기능 | feat(order): 주문 생성 기능 추가 |
| fix | 버그 수정 | fix(auth): 토큰 만료 처리 수정 |
| docs | 문서 | docs(readme): 설치 가이드 추가 |
| refactor | 리팩토링 | refactor(user): 쿼리 최적화 |
| test | 테스트 | test(order): 주문 통합 테스트 추가 |
| chore | 기타 | chore(deps): 의존성 업데이트 |

### 2.3 스코프

- 도메인명: `order`, `user`, `product`
- 레이어: `l1`, `lm`, `l2`
- 기능: `auth`, `payment`, `notification`

### 2.4 Subject 규칙

- 명령형 현재 시제 ("add" not "added")
- 첫 글자 소문자
- 마침표 없음
- 50자 이내

### 2.5 예시

```
feat(order): 주문 생성 기능 구현

- OrderCommandService.create() 구현
- OrderQueryService.findById() 구현
- CreateOrderRequest DTO 추가

Port: L1-Order
```

---

## 3. CHANGELOG 관리

### 3.1 형식 (Keep a Changelog)

```markdown
# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### Added
- 새로 추가된 기능

### Changed
- 기존 기능 변경

### Fixed
- 버그 수정

### Removed
- 제거된 기능

## [1.0.0] - 2026-01-12

### Added
- 초기 릴리즈
```

### 3.2 CHANGELOG 업데이트 시점

- 기능 완료 시 (Unreleased에 추가)
- 릴리즈 시 (버전 태그 생성)

### 3.3 항목 작성 규칙

```markdown
### Added
- 주문 생성 API 추가 (#123)
- 사용자 프로필 조회 기능

### Changed
- 결제 프로세스 개선 (2단계 → 1단계)

### Fixed
- 로그인 시 토큰 만료 오류 수정 (#125)
```

---

## 4. 릴리즈 노트 생성

### 4.1 릴리즈 노트 형식

```markdown
# Release v1.0.0

## 🎉 Highlights

주요 변경 사항 요약

## ✨ New Features

- **주문 시스템**: 주문 생성, 조회, 취소 기능
- **결제 연동**: PG사 연동 완료

## 🐛 Bug Fixes

- 로그인 토큰 만료 오류 수정

## 📝 Documentation

- API 문서 업데이트
- 설치 가이드 추가

## ⚠️ Breaking Changes

- API 엔드포인트 변경: `/api/orders` → `/api/v1/orders`

## 🔧 Configuration

- 환경 변수 추가: `PAYMENT_API_KEY`

## 📦 Dependencies

- Spring Boot 3.2.0 → 3.2.1
```

### 4.2 릴리즈 노트 생성 프로세스

1. CHANGELOG의 Unreleased 섹션 수집
2. 커밋 로그 분석
3. 주요 변경 사항 하이라이트
4. Breaking Changes 식별
5. 릴리즈 노트 작성

---

## 5. 작업 이력 기록

### 5.1 이력 기록 형식

```markdown
## 작업 이력: 2026-01-12

### 완료된 포트
- L1-Order: 주문 엔티티 구현
- L1-Product: 상품 엔티티 구현

### 생성된 파일
- domain/orders/model/Order.kt
- domain/orders/command/OrderCommandService.kt
- domain/orders/query/OrderQueryService.kt

### 커밋
- feat(order): 주문 엔티티 및 서비스 구현
- feat(product): 상품 엔티티 및 서비스 구현
```

### 5.2 이력 추적

```bash
# Git 로그 기반 이력 확인
git log --oneline --since="1 day ago"

# 포트별 변경 파일
pal port files <id>
```

---

## 6. Git 작업 규칙

### 6.1 커밋 전 체크

- [ ] 빌드 성공 확인
- [ ] 테스트 통과 확인
- [ ] 린터 경고 없음
- [ ] 민감 정보 없음 (.env, credentials)

### 6.2 커밋 단위

- 논리적 단위로 커밋
- 하나의 포트 = 하나의 커밋 (권장)
- 너무 크면 분할

### 6.3 브랜치 전략

```
main
  └── feat/feature-name
        └── port/L1-order (선택적)
```

---

## 7. 완료 체크리스트

### 커밋 작업 시

- [ ] 변경 파일 분석 완료
- [ ] 커밋 메시지 작성 완료
- [ ] CHANGELOG 업데이트
- [ ] 커밋 실행
- [ ] 사용자 승인 (필요시)

### 릴리즈 작업 시

- [ ] CHANGELOG 버전 태그 추가
- [ ] 릴리즈 노트 작성
- [ ] Git 태그 생성
- [ ] 릴리즈 발행

---

## 8. PAL 명령어

```bash
# 변경 사항 확인
pal log changes

# 커밋 메시지 생성
pal log commit --dry-run

# CHANGELOG 업데이트
pal log changelog

# 릴리즈 노트 생성
pal log release <version>
```

---

## 9. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 민감 정보 발견 | User | 즉시 알림 |
| 대규모 변경 | User | 커밋 분할 제안 |
| 충돌 발생 | User/Worker | 충돌 해결 요청 |
| 릴리즈 결정 | User | 버전 결정 요청 |

---

<!-- pal:convention:core:logger -->
