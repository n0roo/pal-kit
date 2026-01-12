# 에이전트 명세 분석 보고서

> 작성일: 2026-01-12 | 포트: agent-spec-review

---

## 1. 현황 요약

### 1.1 에이전트 인벤토리

| 구분 | 에이전트 | 수량 |
|------|---------|------|
| Core | builder, planner, architect, manager, tester, logger, collaborator | 7 |
| Worker (Backend) | go, kotlin, nestjs | 3 |
| Worker (Frontend) | react, next | 2 |
| **합계** | | **12** |

### 1.2 워크플로우별 지원

| 에이전트 | simple | single | integrate | multi |
|---------|--------|--------|-----------|-------|
| builder | - | O | O | O |
| planner | - | O | O | O |
| architect | - | O | O | O |
| manager | - | - | O | O |
| tester | - | O | O | O |
| logger | - | O | O | O |
| collaborator | O | - | - | - |
| workers (전체) | - | O | O | O |

---

## 2. Core 에이전트 상세 분석

### 2.1 Builder

| 항목 | 내용 |
|------|------|
| **역할** | 요구사항 분석, 포트 분해 |
| **책임** | 요구사항 명확화, 포트 단위 분해, 의존성 파악, 포트 명세 작성 |
| **출력물** | ports/*.md (포트 명세) |
| **명령어** | `pal port create`, `pal port list`, `pal status` |

**문제점:**
- 완료 체크리스트 없음
- 컨벤션 섹션 누락
- 에스컬레이션 기준 미정의

---

### 2.2 Planner

| 항목 | 내용 |
|------|------|
| **역할** | 파이프라인 계획 수립 |
| **책임** | 의존성 분석, 실행 순서 결정, 에이전트 할당 제안, 병렬 그룹 식별 |
| **출력물** | 파이프라인 정의, 실행 계획, 에이전트 할당표 |
| **명령어** | `pal pipeline create`, `pal pipeline add`, `pal pl plan` |

**문제점:**
- builder와 의존성 분석 역할 일부 중복
- 완료 체크리스트 없음

---

### 2.3 Architect

| 항목 | 내용 |
|------|------|
| **역할** | 아키텍처 설계, 기술 결정 |
| **책임** | 기술 스택 선택, 아키텍처 패턴 결정, 디렉토리 구조 설계, 워커 에이전트 유형 결정 |
| **출력물** | 아키텍처 문서, ADR, 디렉토리 구조 |
| **명령어** | 없음 |

**문제점:**
- Worker와 디렉토리 구조 설계 역할 중복
- 명령어 섹션 누락
- 컨벤션 섹션 누락

---

### 2.4 Manager

| 항목 | 내용 |
|------|------|
| **역할** | 워커 세션 관리, 품질 게이트 |
| **책임** | 세션 생성/관리, 완료 평가, 품질 체크, 산출물 전달, 에스컬레이션 |
| **출력물** | 없음 (관리 역할) |
| **명령어** | `pal session list`, `pal port status`, `pal escalation list` |

**문제점:**
- integrate/multi 전용으로 single에서 품질 게이트 공백
- 완료 체크리스트가 있으나 강제 적용 메커니즘 없음

---

### 2.5 Tester

| 항목 | 내용 |
|------|------|
| **역할** | 테스트 작성 및 실행 |
| **책임** | 테스트 케이스 작성, 실행, 커버리지 확인, 버그 리포트 |
| **출력물** | 테스트 코드, 테스트 결과 리포트, 커버리지 리포트 |
| **명령어** | 없음 |

**문제점:**
- Worker 에이전트도 테스트 작성 책임 보유 → 역할 중복
- 명령어 섹션 누락
- 기술 스택별 테스트 도구 미정의

---

### 2.6 Logger

| 항목 | 내용 |
|------|------|
| **역할** | 이력 관리, 문서화 |
| **책임** | 작업 이력 기록, 변경사항 문서화, 커밋 메시지 작성, CHANGELOG 관리 |
| **출력물** | CHANGELOG.md, 커밋 메시지, 릴리즈 노트 |
| **명령어** | 없음 |

**문제점:**
- 명령어 섹션 누락
- git 명령어 가이드 필요
- 완료 체크리스트 없음

---

### 2.7 Collaborator

| 항목 | 내용 |
|------|------|
| **역할** | 종합 협업 에이전트 (simple 워크플로우 전용) |
| **책임** | 요구사항 이해, 코드 작성/수정, 테스트 작성, 리뷰, 문서화 |
| **출력물** | 없음 (대화형) |
| **명령어** | 없음 |

**문제점:**
- simple 전용이라 다른 워크플로우에서 활용 불가
- 역할 범위가 너무 넓음 (모든 것을 함)
- 완료 기준 모호

---

## 3. Worker 에이전트 상세 분석

### 3.1 구조 비교

| 필드 | go | kotlin | nestjs | react | next |
|------|-----|--------|--------|-------|------|
| id | O | O | O | O | O |
| name | O | O | O | O | O |
| type | O | O | O | O | O |
| tech | O | O | O | O | O |
| workflow | O | O | O | O | O |
| description | O | O | O | O | O |
| responsibilities | O | O | O | O | O |
| **conventions** | O | O | O | O | O |
| tools | O | O | O | O | O |
| prompt | O | O | O | O | O |
| **outputs** | - | - | - | - | - |
| **commands** | - | - | - | - | - |

### 3.2 공통 구조

```yaml
agent:
  id: worker-{tech}
  name: {Tech} Worker
  type: worker
  tech: {tech}
  workflow: [single, integrate, multi]
  description: |
    {tech} 코드 작성을 담당하는 워커 에이전트.
  responsibilities:
    - 코드 작성
    - 패키지/모듈 구조 설계
    - 에러 처리
    - 테스트 작성
  conventions:
    - (기술별 상이)
  tools:
    - (기술별 상이)
  prompt: |
    # {Tech} Worker
    ...
```

### 3.3 Worker 공통 문제점

1. **outputs 필드 누락**: 산출물 정의 없음
2. **commands 필드 누락**: PAL 명령어 가이드 없음
3. **완료 체크리스트 미표준화**: 각 Worker마다 작업 규칙이 다름
4. **테스트 작성 책임 중복**: Core Tester와 역할 중복

---

## 4. 역할 중복/누락 매트릭스

### 4.1 역할 매트릭스

| 역할 | 담당 에이전트 | 비고 |
|------|--------------|------|
| 요구사항 분석 | builder, collaborator | 중복 |
| 포트 분해 | builder | - |
| 의존성 분석 | builder, planner | 부분 중복 |
| 파이프라인 계획 | planner | - |
| 아키텍처 설계 | architect | - |
| 디렉토리 구조 | architect, workers | 중복 |
| 코드 작성 | workers, collaborator | 역할 분리됨 |
| 테스트 작성 | tester, workers | **중복 (문제)** |
| 테스트 실행 | tester | - |
| 품질 게이트 | manager | integrate/multi만 |
| 코드 리뷰 | - | **누락** |
| 커밋/이력 | logger | - |
| 문서화 | logger, collaborator | 부분 중복 |

### 4.2 누락된 역할

| 역할 | 설명 | 권장 |
|------|------|------|
| **Reviewer** | 코드 리뷰 전담 | Core 에이전트 추가 |
| **Docs Writer** | API 문서, 사용자 가이드 | Core 에이전트 추가 또는 Logger 확장 |
| **Security** | 보안 검토 | Worker 또는 Core 에이전트 추가 |

### 4.3 코드에 정의된 미사용 타입

```go
// internal/agent/agent.go:222-231
func GetAgentTypes() []string {
    return []string{
        "builder",  // 사용 중
        "worker",   // 사용 중
        "reviewer", // 템플릿 없음 ← 누락
        "planner",  // 사용 중
        "tester",   // 사용 중
        "docs",     // 템플릿 없음 ← 누락
        "custom",   // 템플릿 없음
    }
}
```

---

## 5. 구조적 문제점 요약

### 5.1 필드 불일치

| 구분 | Core | Worker |
|------|------|--------|
| conventions | 일부만 | 모두 있음 |
| outputs | 일부만 | 없음 |
| commands | 일부만 | 없음 |
| tools | 없음 | 모두 있음 |
| tech | 없음 | 모두 있음 |

### 5.2 완료 체크리스트 현황

| 에이전트 | 체크리스트 존재 | 강제 메커니즘 |
|---------|----------------|---------------|
| builder | X | - |
| planner | X | - |
| architect | X | - |
| manager | O (품질) | X |
| tester | X | - |
| logger | X | - |
| collaborator | X | - |
| workers | X | - |

### 5.3 워크플로우별 품질 게이트 공백

```
simple:     collaborator (체크리스트 없음)
single:     품질 게이트 담당자 없음 ← 문제
integrate:  manager (체크리스트 있음, 강제 없음)
multi:      manager (체크리스트 있음, 강제 없음)
```

---

## 6. 결론

### 6.1 주요 발견사항

1. **역할 중복**: 테스트 작성(Tester vs Workers), 디렉토리 구조(Architect vs Workers)
2. **역할 누락**: Reviewer, Docs Writer 에이전트 미구현
3. **구조 불일치**: Core와 Worker 간 필드 구조 상이
4. **완료 보장 미흡**: 체크리스트 미표준화, 강제 메커니즘 부재
5. **워크플로우 공백**: single 워크플로우에 품질 게이트 없음

### 6.2 개선 필요 영역

1. 에이전트 명세 구조 표준화
2. 역할 책임 명확화 및 중복 제거
3. 완료 체크리스트 표준화 및 강제 메커니즘
4. 누락 에이전트 추가 (Reviewer, Docs)
5. 컨벤션 분리 구조화

---

<!-- pal:port:agent-spec-review -->
