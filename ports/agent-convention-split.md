# Port: agent-convention-split

> 코어/워커 에이전트 컨벤션 분리 및 고도화

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | agent-convention-split |
| 상태 | draft |
| 우선순위 | high |
| 의존성 | agent-spec-review |
| 예상 복잡도 | high |

---

## 목표

Core 에이전트와 Worker 에이전트의 특화된 컨벤션을 분리하여 작업 완료성을 확보한다.

---

## 범위

### 포함

- Core 에이전트 공통 컨벤션 정의
- Core 에이전트 개별 컨벤션 분리
- Worker 에이전트 공통 컨벤션 정의
- Worker 에이전트 기술 스택별 컨벤션 분리
- 컨벤션 로딩 메커니즘 구현/개선
- 작업 완료 체크리스트 표준화

### 제외

- 에이전트 실행 로직 변경
- 새 에이전트 타입 추가

---

## 작업 항목

### Core 컨벤션 분리

- [ ] 공통 컨벤션 정의 (`conventions/agents/core/_common.md`)
  - [ ] 역할 전환 규칙
  - [ ] 에스컬레이션 기준
  - [ ] 완료 체크리스트 표준
- [ ] 개별 컨벤션 분리
  - [ ] builder 컨벤션
  - [ ] planner 컨벤션
  - [ ] architect 컨벤션
  - [ ] manager 컨벤션
  - [ ] tester 컨벤션
  - [ ] logger 컨벤션

### Worker 컨벤션 분리

- [ ] 공통 컨벤션 정의 (`conventions/agents/workers/_common.md`)
  - [ ] 코드 품질 기준
  - [ ] 테스트 요구사항
  - [ ] 완료 체크리스트 표준
- [ ] 기술 스택별 컨벤션
  - [ ] Go 컨벤션 (`conventions/agents/workers/go.md`)
  - [ ] React 컨벤션 (템플릿)
  - [ ] Python 컨벤션 (템플릿)

### 메커니즘 구현

- [ ] 컨벤션 로딩 순서 정의 (공통 → 개별)
- [ ] 컨벤션 병합 로직 구현
- [ ] 에이전트 생성 시 컨벤션 자동 연결

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| Core 공통 컨벤션 | conventions/agents/core/_common.md | 코어 에이전트 공통 규칙 |
| Core 개별 컨벤션 | conventions/agents/core/*.md | 에이전트별 특화 규칙 |
| Worker 공통 컨벤션 | conventions/agents/workers/_common.md | 워커 에이전트 공통 규칙 |
| Worker 개별 컨벤션 | conventions/agents/workers/*.md | 기술 스택별 규칙 |

---

## 완료 기준

- [ ] 모든 Core 에이전트에 개별 컨벤션 적용
- [ ] Worker 에이전트 생성 시 기술 스택 컨벤션 자동 로드
- [ ] 작업 완료 체크리스트가 모든 에이전트에 표준화

---

<!-- pal:port:status=draft -->
