# Port: agent-package-system

> 에이전트 패키지 시스템 설계 및 구현

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | agent-package-system |
| 상태 | complete |
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

- [x] 패키지 YAML 스키마 정의
- [x] 스키마 검증 로직 구현
- [x] 상속(extends) 로직 구현

### 기본 패키지 작성

- [x] PA-Layered Backend 패키지
  - [x] tech 섹션 (Kotlin, Spring, JPA, JOOQ)
  - [x] architecture 섹션 (L1, LM, L2, L3)
  - [x] methodology 섹션 (port_driven, cqs, event_driven)
  - [x] workers 목록 (6종)
  - [x] core_overrides (architect, builder)
- [x] PA-Layered Frontend 패키지
  - [x] tech 섹션 (TypeScript, React, Next.js, MUI, Tailwind)
  - [x] architecture 섹션 (API, Query, Feature, Component)
  - [x] workers 목록 (5종)
- [x] PA-Layered Base 패키지 (상속 기반)

### 패키지 로딩

- [x] 패키지 파일 로딩 (`packages/`)
- [x] 전역 패키지 로딩 (`~/.pal/packages/`)
- [x] 상속 병합 로직
- [x] 순환 참조 감지

### CLI 명령어

- [x] `pal package list` - 사용 가능한 패키지 목록
- [x] `pal package show <id>` - 패키지 상세 정보
- [x] `pal package use <id>` - 프로젝트에 패키지 적용
- [x] `pal package create <id>` - 새 패키지 생성
- [x] `pal package validate [id]` - 패키지 검증
- [x] `pal package workers <id>` - 워커 목록 조회

### 문서화

- [x] 사용자 정의 패키지 가이드 (docs/PACKAGE-GUIDE.md)
- [x] 패키지 작성 템플릿 (가이드 내 포함)

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

- [x] 패키지 스키마 정의 및 검증 동작
- [x] 기본 패키지 3종 로딩 성공 (base, backend, frontend)
- [x] 상속(extends) 동작 확인
- [x] CLI 명령어 6종 동작
- [x] 사용자 정의 패키지 가이드 작성

---

## 완료 요약

### 생성된 파일

| 파일 | 설명 |
|------|------|
| `internal/package/package.go` | 패키지 스키마 및 서비스 |
| `internal/package/package_test.go` | 패키지 테스트 |
| `internal/cli/package.go` | CLI 명령어 |
| `packages/pa-layered-base.yaml` | Base 패키지 |
| `packages/pa-layered-backend.yaml` | Backend 패키지 |
| `packages/pa-layered-frontend.yaml` | Frontend 패키지 |
| `docs/PACKAGE-GUIDE.md` | 사용자 가이드 |

### 수정된 파일

| 파일 | 변경 내용 |
|------|----------|
| `internal/config/paths.go` | GlobalPackagesDir, GlobalPalDir 추가 |

---

<!-- pal:port:status=complete -->
