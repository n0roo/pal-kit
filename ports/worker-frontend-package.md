# Port: worker-frontend-package

> PA-Layered Frontend 워커 명세 구현

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | worker-frontend-package |
| 상태 | draft |
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

- [ ] Theme 커스터마이징 규칙
- [ ] sx prop vs styled() 사용 기준
- [ ] 컴포넌트 조합 패턴
- [ ] 반응형 breakpoint 사용

### Tailwind 가이드라인

- [ ] className 작성 규칙
- [ ] 커스텀 유틸리티 최소화
- [ ] Dark Mode 지원
- [ ] MUI와 혼용 시 주의사항

---

## 작업 항목

### Frontend Engineer Worker

- [ ] 워커 명세 작성 (agents/workers/frontend/engineer.yaml)
- [ ] 컨벤션 문서 작성 (conventions/workers/frontend/engineer.md)
  - [ ] 페이지 단위 작업 분해 규칙
  - [ ] 하위 워커 태스크 할당 기준
  - [ ] 작업 순서 결정 가이드
  - [ ] 통합 검증 체크리스트
- [ ] 완료 체크리스트 정의

### Component Model Worker

- [ ] 워커 명세 작성 (agents/workers/frontend/model.yaml)
- [ ] 컨벤션 문서 작성 (conventions/workers/frontend/model.md)
  - [ ] API Service 작성 규칙
  - [ ] TanStack Query Hook 패턴
  - [ ] Custom Hook 작성 규칙
  - [ ] Type 정의 규칙
  - [ ] Error Handling 패턴
- [ ] 완료 체크리스트 정의

### Component UI Worker

- [ ] 워커 명세 작성 (agents/workers/frontend/ui.yaml)
- [ ] 컨벤션 문서 작성 (conventions/workers/frontend/ui.md)
  - [ ] 컴포넌트 구조 규칙
  - [ ] Props Interface 규칙
  - [ ] MUI 사용 가이드라인
  - [ ] Tailwind 사용 가이드라인
  - [ ] 접근성(a11y) 기본 규칙
  - [ ] 반응형 처리 규칙
- [ ] 완료 체크리스트 정의

### E2E Worker

- [ ] 워커 명세 작성 (agents/workers/frontend/e2e.yaml)
- [ ] 컨벤션 문서 작성 (conventions/workers/frontend/e2e.md)
  - [ ] Playwright 설정 가이드
  - [ ] Page Object Model 패턴
  - [ ] 테스트 시나리오 작성 규칙
  - [ ] Fixture 관리 규칙
  - [ ] CI 연동 가이드
- [ ] 완료 체크리스트 정의

### Unit TC Worker

- [ ] 워커 명세 작성 (agents/workers/frontend/unit-tc.yaml)
- [ ] 컨벤션 문서 작성 (conventions/workers/frontend/unit-tc.md)
  - [ ] Vitest/Jest 설정 가이드
  - [ ] Testing Library 사용 규칙
  - [ ] MSW (Mock Service Worker) 설정
  - [ ] Hook 테스트 패턴 (renderHook)
  - [ ] 컴포넌트 테스트 패턴
  - [ ] 커버리지 목표 (80%+)
- [ ] 완료 체크리스트 정의

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

- [ ] 5종 워커 명세 파일 작성 완료
- [ ] 5종 컨벤션 문서 작성 완료
- [ ] MUI/Tailwind 가이드라인 문서 완료
- [ ] 모든 워커에 완료 체크리스트 정의
- [ ] `pal agent list` 에서 워커 표시

---

<!-- pal:port:status=draft -->
