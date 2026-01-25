# Port: L2-agent-docs

> Docs 에이전트 - 문서화 전담 에이전트

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | L2-agent-docs |
| 타입 | atomic |
| 레이어 | L2 (Agent) |
| 상태 | complete |
| 우선순위 | medium |
| 의존성 | - |
| 예상 토큰 | 5,000 |

---

## 목표

API 문서, 사용자 가이드, README 등 문서화를 담당하는 Docs 에이전트를 추가한다.

---

## 범위

### 포함

- Docs 에이전트 템플릿 (YAML)
- Docs 컨벤션 (MD)
- Docs Rules 파일 (MD)
- 문서 템플릿 제공

### 제외

- 자동 문서 생성 도구 연동 (godoc, typedoc 등)
- 문서 배포 시스템

---

## 작업 항목

### 1. 에이전트 템플릿

- [ ] `internal/agent/templates/agents/core/docs.yaml`
  ```yaml
  agent:
    id: docs
    name: Docs Writer
    type: core
    workflow: [single, integrate, multi]
    
    description: |
      문서화를 담당하는 에이전트.
      API 문서, 사용자 가이드, README를 작성합니다.
    
    responsibilities:
      - API 문서 작성/업데이트
      - 사용자 가이드 작성
      - README 업데이트
      - 코드 주석 검토/개선
      - CHANGELOG 작성
    
    inputs:
      - source-code (소스 코드)
      - port-spec (포트 명세)
      - existing-docs (기존 문서)
    
    outputs:
      - api-docs (API 문서)
      - user-guide (사용자 가이드)
      - readme (README 파일)
      - changelog (변경 이력)
    
    conventions_ref: conventions/agents/core/docs.md
    rules_ref: agents/core/docs.rules.md
    
    completion:
      checklist:
        - API 문서 완성
        - 코드 예제 포함
        - 문법/맞춤법 검토
        - 링크 유효성 확인
      required: true
    
    escalation:
      - condition: API 변경 사항 불명확
        target: worker
        action: 명확화 요청
      - condition: 사용자 시나리오 불명확
        target: user
        action: 요구사항 확인
    
    commands:
      - pal docs lint
      - pal docs snapshot
      - pal port show <id>
    
    prompt: |
      # Docs Writer Agent
      
      당신은 Docs Writer 에이전트입니다.
      코드와 기능에 대한 문서를 작성하고 유지합니다.
      
      ## 문서화 원칙
      
      1. **명확성**: 모호하지 않은 표현
      2. **완전성**: 필요한 정보 누락 없이
      3. **일관성**: 용어와 형식 통일
      4. **예제 중심**: 코드 예제로 이해 돕기
      
      ## 문서 유형별 가이드
      
      ### API 문서
      - 엔드포인트/함수 시그니처
      - 파라미터 설명
      - 반환값 설명
      - 에러 케이스
      - 사용 예제
      
      ### 사용자 가이드
      - 설치 방법
      - 빠른 시작 가이드
      - 상세 사용법
      - FAQ
      - 트러블슈팅
      
      ### README
      - 프로젝트 소개 (1-2문장)
      - 주요 기능
      - 설치 방법
      - 빠른 시작
      - 문서 링크
      - 라이선스
      
      ## 출력 형식
      
      - Markdown 사용
      - 코드 블록에 언어 명시
      - 적절한 헤딩 계층
      - 테이블로 구조화된 정보
  ```

### 2. 컨벤션 파일

- [ ] `internal/agent/templates/conventions/agents/core/docs.md`
  ```markdown
  # Docs Writer 에이전트 컨벤션
  
  ## 1. 문서화 원칙
  
  ### 명확성
  - 전문 용어는 처음 사용 시 설명
  - 약어는 풀네임과 함께 표기
  - 모호한 표현 ("등", "여러 가지") 지양
  
  ### 완전성
  - 모든 공개 API 문서화
  - 에러 케이스 명시
  - 버전 정보 포함
  
  ### 일관성
  - 용어 통일 (glossary 참조)
  - 문서 구조 통일
  - 코드 스타일 통일
  
  ## 2. 파일별 가이드
  
  ### README.md
  ```
  # 프로젝트명
  
  > 한 줄 설명
  
  ## 주요 기능
  - 기능 1
  - 기능 2
  
  ## 설치
  ```bash
  npm install ...
  ```
  
  ## 빠른 시작
  ...
  
  ## 문서
  - [API 문서](./docs/api.md)
  - [사용자 가이드](./docs/guide.md)
  
  ## 라이선스
  MIT
  ```
  
  ### API 문서
  
  ```
  # API 이름
  
  ## 개요
  설명
  
  ## 시그니처
  ```go
  func Name(param Type) ReturnType
  ```
  
  ## 파라미터
  | 이름 | 타입 | 필수 | 설명 |
  |------|------|------|------|
  | param | Type | Y | 설명 |
  
  ## 반환값
  설명
  
  ## 예제
  ```go
  result := Name(value)
  ```
  
  ## 에러
  - `ErrXxx`: 발생 조건
  ```
  
  ## 3. 체크리스트
  
  ### 문서 작성 전
  - [ ] 대상 코드/기능 파악
  - [ ] 기존 문서 확인
  - [ ] 변경 사항 식별
  
  ### 문서 작성 후
  - [ ] 문법/맞춤법 검토
  - [ ] 코드 예제 실행 확인
  - [ ] 링크 유효성 확인
  - [ ] 일관성 검토
  ```

### 3. Rules 파일

- [ ] `internal/agent/templates/agents/core/docs.rules.md`
  ```markdown
  ---
  description: Docs Writer 에이전트 규칙
  globs:
    - "**/*.md"
    - "**/README*"
    - "**/docs/**"
  alwaysApply: false
  ---
  
  # Docs Writer 규칙
  
  ## 문서 작성 시작
  
  1. 대상 코드/기능 확인
  2. 기존 문서 확인: `pal docs lint`
  3. 변경 필요 사항 식별
  
  ## 문서 작성 중
  
  1. 템플릿 활용
  2. 예제 코드 작성 및 테스트
  3. 링크 확인
  
  ## 문서 완료
  
  1. `pal docs lint` 실행
  2. 스펠 체크
  3. 스냅샷 생성: `pal docs snapshot`
  
  ## PAL 명령어
  
  - `pal docs lint` - 문서 검사
  - `pal docs snapshot` - 현재 상태 스냅샷
  - `pal port show <id>` - 관련 포트 확인
  ```

### 4. 문서 템플릿

- [ ] `internal/docs/templates/readme.md.tmpl`
- [ ] `internal/docs/templates/api.md.tmpl`
- [ ] `internal/docs/templates/guide.md.tmpl`
- [ ] `internal/docs/templates/changelog.md.tmpl`

---

## 완료 기준

- [x] `docs.yaml` 템플릿 생성
- [x] `docs.md` 컨벤션 생성
- [x] `docs.rules.md` 생성
- [x] 문서 템플릿 4종 생성
- [ ] `pal agent list`에 docs 표시

---

## 참조

- `specs/agent-improvement-proposal.md` - Docs 에이전트 제안
- `internal/agent/templates/agents/core/` - 기존 Core 에이전트
- `internal/docs/` - 기존 docs 패키지

---

<!-- pal:port:L2-agent-docs -->
