# Port: worker-frontend-package

> PA-Layered Frontend 워커 명세 구현

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | worker-frontend-package |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | agent-package-system |
| 예상 복잡도 | high |

---

## 목표

PA-Layered Frontend 패키지에 포함되는 5종의 워커 명세를 구현한다.
각 워커는 프론트엔드 개발 흐름에 특화되어 동작한다.

---

## 범위

### 포함

- Frontend Engineer Worker (Orchestration)
- Component Model Worker (Logic Layer)
- Component UI Worker (View Layer)
- E2E Worker (Playwright)
- Unit TC Worker (Unit Test)

### 제외

- 워커 실행 로직 (별도 구현)
- Claude 프롬프트 통합 (별도 포트)

---

## 워커 목록 및 레이어 매핑

| 워커 ID | 이름 | 레이어 | 포트 타입 | 기술 |
|---------|------|--------|-----------|------|
| frontend-engineer-worker | Frontend Engineer | Orchestration | tpl-client-feature | - |
| component-model-worker | Component Model | Logic | tpl-client-api-port, tpl-client-query | TanStack Query |
| component-ui-worker | Component UI | View | tpl-client-component-port | MUI, Tailwind |
| e2e-worker | E2E Test | Test | E2E 시나리오 | Playwright |
| unit-tc-worker | Unit TC | Test | TC 체크리스트 | Vitest, Testing Library |

---

## UI 가이드라인 요구사항

### MUI 가이드라인

- [x] Theme 커스터마이징 규칙
- [x] sx prop vs styled() 사용 기준
- [x] 컴포넌트 조합 패턴
- [x] 반응형 breakpoint 사용

### Tailwind 가이드라인

- [x] className 작성 규칙
- [x] 커스텀 유틸리티 최소화
- [x] Dark Mode 지원
- [x] MUI와 혼용 시 주의사항

---

## 작업 항목

### Frontend Engineer Worker

- [x] 워커 명세 작성 (agents/workers/frontend/engineer.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/frontend/engineer.md)
  - [x] 페이지 단위 작업 분해 규칙
  - [x] 하위 워커 태스크 할당 기준
  - [x] 작업 순서 결정 가이드
  - [x] 통합 검증 체크리스트
- [x] 완료 체크리스트 정의

### Component Model Worker

- [x] 워커 명세 작성 (agents/workers/frontend/model.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/frontend/model.md)
  - [x] API Service 작성 규칙
  - [x] TanStack Query Hook 패턴
  - [x] Custom Hook 작성 규칙
  - [x] Type 정의 규칙
  - [x] Error Handling 패턴
- [x] 완료 체크리스트 정의

### Component UI Worker

- [x] 워커 명세 작성 (agents/workers/frontend/ui.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/frontend/ui.md)
  - [x] 컴포넌트 구조 규칙
  - [x] Props Interface 규칙
  - [x] MUI 사용 가이드라인
  - [x] Tailwind 사용 가이드라인
  - [x] 접근성(a11y) 기본 규칙
  - [x] 반응형 처리 규칙
- [x] 완료 체크리스트 정의

### E2E Worker

- [x] 워커 명세 작성 (agents/workers/frontend/e2e.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/frontend/e2e.md)
  - [x] Playwright 설정 가이드
  - [x] Page Object Model 패턴
  - [x] 테스트 시나리오 작성 규칙
  - [x] Fixture 관리 규칙
  - [x] CI 연동 가이드
- [x] 완료 체크리스트 정의

### Unit TC Worker

- [x] 워커 명세 작성 (agents/workers/frontend/unit-tc.yaml)
- [x] 컨벤션 문서 작성 (conventions/workers/frontend/unit-tc.md)
  - [x] Vitest/Jest 설정 가이드
  - [x] Testing Library 사용 규칙
  - [x] MSW (Mock Service Worker) 설정
  - [x] Hook 테스트 패턴 (renderHook)
  - [x] 컴포넌트 테스트 패턴
  - [x] 커버리지 목표 (80%+)
- [x] 완료 체크리스트 정의

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| Engineer Worker | agents/workers/frontend/engineer.yaml | 명세 |
| Model Worker | agents/workers/frontend/model.yaml | 명세 |
| UI Worker | agents/workers/frontend/ui.yaml | 명세 |
| E2E Worker | agents/workers/frontend/e2e.yaml | 명세 |
| Unit TC Worker | agents/workers/frontend/unit-tc.yaml | 명세 |
| 컨벤션 문서 | conventions/workers/frontend/*.md | 5종 |
| MUI 가이드 | conventions/ui/mui.md | 스타일 가이드 |
| Tailwind 가이드 | conventions/ui/tailwind.md | 스타일 가이드 |

---

## 완료 기준

- [x] 5종 워커 명세 파일 작성 완료
- [x] 5종 컨벤션 문서 작성 완료
- [x] MUI/Tailwind 가이드라인 문서 완료
- [x] 모든 워커에 완료 체크리스트 정의
- [x] 워커 YAML에 prompt 포함 (Claude 연계용)

---

## 완료 요약

### 생성된 워커 명세 (YAML)

| 워커 | 파일 | 레이어 | 기술 |
|------|------|--------|------|
| Frontend Engineer | `agents/workers/frontend/engineer.yaml` | Orchestration | React, Next.js |
| Component Model | `agents/workers/frontend/model.yaml` | Logic | TanStack Query, Zustand |
| Component UI | `agents/workers/frontend/ui.yaml` | View | MUI, Tailwind |
| E2E Test | `agents/workers/frontend/e2e.yaml` | Test | Playwright |
| Unit TC | `agents/workers/frontend/unit-tc.yaml` | Test | Vitest, Testing Library |

### 컨벤션 문서 (MD)

| 워커 | 파일 | 내용 |
|------|------|------|
| Frontend Engineer | `conventions/agents/workers/frontend/engineer.md` | 페이지 구성, 라우팅, 레이아웃 |
| Component Model | `conventions/agents/workers/frontend/model.md` | 상태 관리, Hooks, Form |
| Component UI | `conventions/agents/workers/frontend/ui.md` | 컴포넌트, 스타일링, a11y |
| E2E Test | `conventions/agents/workers/frontend/e2e.md` | Playwright, POM |
| Unit TC | `conventions/agents/workers/frontend/unit-tc.md` | Vitest, MSW |

### UI 가이드라인 (MD)

| 문서 | 파일 | 내용 |
|------|------|------|
| MUI 가이드 | `conventions/ui/mui.md` | Theme, sx prop, 컴포넌트 패턴 |
| Tailwind 가이드 | `conventions/ui/tailwind.md` | 클래스 규칙, 반응형, 다크 모드 |

---

<!-- pal:port:status=complete -->
