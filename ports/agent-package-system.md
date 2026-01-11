# Port: agent-package-system

> 에이전트 패키지 시스템 설계 및 구현

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | agent-package-system |
| 상태 | draft |
| 우선순위 | high |
| 의존성 | agent-convention-split |
| 예상 복잡도 | high |

---

## 목표

에이전트를 묶는 상위 구조인 **패키지(Package)** 시스템을 설계하고 구현한다.
패키지는 기술 스택, 아키텍처, 컨벤션, 개발 방법론을 통합 관리한다.

---

## 범위

### 포함

- 패키지 스키마 정의
- 기본 패키지 작성 (PA-Layered Backend/Frontend)
- 패키지 로딩/파싱 로직 구현
- 패키지 CLI 명령어 추가
- 사용자 정의 패키지 가이드

### 제외

- 개별 워커 명세 구현 (별도 포트)
- Claude 연계 로직 (별도 포트)

---

## 패키지 스키마

```yaml
package:
  # 필수 필드
  id: string                    # 고유 ID
  name: string                  # 표시 이름
  version: string               # 시맨틱 버전

  # 상속 (선택)
  extends: string               # 부모 패키지 ID

  # 기술 스택
  tech:
    language: string            # 주 언어
    frameworks: string[]        # 프레임워크 목록
    build_tool: string          # 빌드 도구

  # 아키텍처
  architecture:
    name: string                # 아키텍처 이름
    layers: string[]            # 레이어 목록
    conventions_ref: string     # 컨벤션 문서 경로

  # 개발 방법론
  methodology:
    port_driven: boolean        # 포트 명세 기반
    cqs: boolean                # Command/Query 분리
    event_driven: boolean       # 이벤트 기반 통신

  # 포함 워커
  workers: string[]             # 워커 ID 목록

  # Core 에이전트 오버라이드
  core_overrides:
    architect:
      conventions_ref: string
    builder:
      port_templates: string[]
```

---

## 작업 항목

### 스키마 정의

- [ ] 패키지 YAML 스키마 정의
- [ ] 스키마 검증 로직 구현
- [ ] 상속(extends) 로직 구현

### 기본 패키지 작성

- [ ] PA-Layered Backend 패키지
  - [ ] tech 섹션 (Kotlin, Spring, JPA, JOOQ)
  - [ ] architecture 섹션 (L1, LM, L2, L3)
  - [ ] methodology 섹션 (port_driven, cqs, event_driven)
  - [ ] workers 목록 (6종)
  - [ ] core_overrides (architect, builder)
- [ ] PA-Layered Frontend 패키지
  - [ ] tech 섹션 (TypeScript, React, Next.js, MUI, Tailwind)
  - [ ] architecture 섹션 (API, Query, Feature, Component)
  - [ ] workers 목록 (5종)

### 패키지 로딩

- [ ] 패키지 파일 로딩 (`.pal/packages/`)
- [ ] 전역 패키지 로딩 (`~/.pal/packages/`)
- [ ] 상속 병합 로직
- [ ] 패키지 캐싱

### CLI 명령어

- [ ] `pal package list` - 사용 가능한 패키지 목록
- [ ] `pal package show <id>` - 패키지 상세 정보
- [ ] `pal package use <id>` - 프로젝트에 패키지 적용
- [ ] `pal package create <id>` - 새 패키지 생성

### 문서화

- [ ] 사용자 정의 패키지 가이드
- [ ] 패키지 작성 템플릿

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| 패키지 스키마 | internal/package/schema.go | 스키마 정의 |
| 패키지 서비스 | internal/package/service.go | 로딩/파싱 |
| Backend 패키지 | packages/pa-layered-backend.yaml | 기본 패키지 |
| Frontend 패키지 | packages/pa-layered-frontend.yaml | 기본 패키지 |
| CLI 명령어 | internal/cli/package.go | CLI |

---

## 완료 기준

- [ ] 패키지 스키마 정의 및 검증 동작
- [ ] 기본 패키지 2종 로딩 성공
- [ ] 상속(extends) 동작 확인
- [ ] CLI 명령어 4종 동작
- [ ] 사용자 정의 패키지 가이드 작성

---

<!-- pal:port:status=draft -->
