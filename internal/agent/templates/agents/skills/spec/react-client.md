# React Client Spec Skill

> React 프론트엔드 명세 스킬

---

## 도메인 특성

### 핵심 개념

컴포넌트 기반 Feature Composition 아키텍처입니다.

```
Feature (L2)
    ↓
├── API Adapter
├── State Management
└── UI Components
```

### 레이어 구조

| 레이어 | 역할 | 예시 |
|--------|------|------|
| Feature | 유스케이스/페이지 조합 | CheckoutFeature |
| Component | 재사용 UI 단위 | ProductCard, Button |
| API Adapter | 서버 통신 | useProductsQuery |
| State | 상태 관리 | Redux Slice, Zustand |

### 상태 관리 패턴

| 유형 | 도구 | 용도 |
|------|------|------|
| Server State | React Query, SWR | API 데이터 캐싱 |
| Client State | Zustand, Jotai | UI 상태 |
| Form State | React Hook Form | 폼 처리 |

---

## 템플릿

### Feature Port

```yaml
---
type: port
layer: feature
domain: {domain}
title: "{Feature} Feature"
priority: {priority}
dependencies: [api-xxx, component-yyy]
---

# {Feature} Feature Port

## 목표
{Feature} 페이지/기능 구현

## 화면 구조
- Header / Navigation
- Main Content Area
- Sidebar (선택)
- Footer

## 컴포넌트 구성

| 컴포넌트 | 역할 | Props |
|----------|------|-------|
| {Feature}Page | 페이지 컨테이너 | - |
| {Feature}Content | 주요 콘텐츠 | data: T[] |

## 데이터 흐름

### API Hooks
- use{Feature}Query: 데이터 조회
- use{Feature}Mutation: 데이터 변경

### State
- filters: 필터 상태
- selectedId: 선택 상태

## 라우팅

| Path | 컴포넌트 | 설명 |
|------|----------|------|
| /{feature} | {Feature}Page | 목록 |
| /{feature}/:id | {Feature}DetailPage | 상세 |

## 검증 규칙
- [ ] 로딩 상태 처리
- [ ] 에러 상태 처리
- [ ] Empty 상태 처리
- [ ] 반응형 레이아웃
```

### Component Port

```yaml
---
type: port
layer: component
domain: ui
title: "{Component} Component"
priority: {priority}
---

# {Component} Component Port

## Props Interface
- value: T (필수)
- onChange: (value: T) => void
- disabled?: boolean
- size?: 'sm' | 'md' | 'lg'
- variant?: 'primary' | 'secondary'

## 변형 (Variants)
- primary: 주요 액션
- secondary: 보조 액션
- outline: 경계선만

## 접근성
- [ ] ARIA 레이블
- [ ] 키보드 네비게이션
- [ ] 포커스 관리
```

### API Adapter Port

```yaml
---
type: port
layer: api-adapter
domain: {domain}
title: "{Domain} API Adapter"
priority: {priority}
---

# {Domain} API Adapter Port

## API 엔드포인트

| Method | Path | 설명 |
|--------|------|------|
| GET | /api/v1/{domain} | 목록 |
| GET | /api/v1/{domain}/:id | 상세 |
| POST | /api/v1/{domain} | 생성 |

## Hooks
- use{Entity}List: 목록 조회
- use{Entity}Detail: 상세 조회
- useCreate{Entity}: 생성
- useUpdate{Entity}: 수정

## 에러 처리
- 401: 인증 필요
- 403: 권한 없음
- 404: 리소스 없음
```

---

## 컨벤션

### 프로젝트 구조

```
src/
├── features/           # Feature (L2)
│   └── {feature}/
│       ├── {Feature}Page.tsx
│       ├── components/
│       └── hooks/
├── components/         # 공통 컴포넌트
├── api/                # API Adapter
└── store/              # 전역 상태
```

### 네이밍 규칙

| 구성요소 | 패턴 | 예시 |
|----------|------|------|
| Feature Page | {Feature}Page | ProductListPage |
| Component | {Component} | ProductCard |
| Hook | use{Action} | useProducts |
| Store | use{Store}Store | useCartStore |

---

## 검증 기준

- [ ] Props 타입 정의
- [ ] Storybook 스토리
- [ ] 접근성 검사
- [ ] 에러 바운더리
