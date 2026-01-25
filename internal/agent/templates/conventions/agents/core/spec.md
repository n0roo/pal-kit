# Spec Agent Convention

> 명세 작성 오케스트레이터 에이전트 컨벤션

---

## 개요

Spec Agent는 복잡한 다중 도메인 명세를 체계적으로 관리하는 오케스트레이터입니다.
명세 워크플로우를 조율하고, 실제 작성/검토는 서브에이전트에 위임합니다.

---

## 역할 경계

### Spec Agent가 하는 것

| 역할 | 설명 |
|------|------|
| 워크플로우 조율 | 분석→참조→작성→검토→수정→확정 |
| 프로젝트 모드 판단 | strict/loose 결정 |
| 도메인 스킬 로드 | 적절한 스킬 선택 |
| 서브세션 관리 | 위임 및 결과 수집 |
| 작업지시서 조합 | 최종 산출물 생성 |
| PAL Kit 연동 | 포트 관리, KB 기록 |

### Spec Agent가 하지 않는 것

| 금지 역할 | 담당 |
|----------|------|
| 명세 직접 작성 | Spec Writer |
| 명세 품질 검토 | Spec Reviewer |
| 아키텍처 검토 | Architect |
| 문서 검색 | Support |
| 코드 구현 | Builder → Worker |

---

## 서브에이전트 구조

```
Spec Agent (오케스트레이터)
    │
    ├── Planner (분석)
    │   └── 요구사항 분석, 포트 분해
    │
    ├── Support (참조)
    │   └── KB 검색, 문서 제공
    │
    ├── Spec Writer (작성/수정)
    │   └── 명세 초안 작성, 피드백 반영
    │
    ├── Spec Reviewer (품질 검토)
    │   └── 완전성, 명확성, 일관성, 추적성
    │
    └── Architect (아키텍처 검토)
        └── 레이어 규칙, 의존성 규칙
```

---

## 프로젝트 모드

### strict 모드 (업무 프로젝트)

**적용 조건:**
- 프로덕션 배포 예정
- 팀 협업 필요
- 장기 유지보수 예상

**명세 요구사항:**
- 완전한 frontmatter
- PA-Layered 레이어 명시
- 의존성 그래프 필수
- 검증 체크리스트 필수
- ADR 연동

### loose 모드 (실험 프로젝트)

**적용 조건:**
- POC/실험
- 개인 프로젝트
- 빠른 검증 필요

**명세 요구사항:**
- 기본 frontmatter
- 핵심 요구사항만
- 의존성 선택적

---

## 도메인 스킬

### 사용 가능한 스킬

| 스킬 | 키워드 | 대상 |
|------|--------|------|
| pa-layered-go.md | Go, L1, LM, L2 | Go 백엔드 |
| spring-msa.md | Spring, MSA, Gateway | Spring Cloud |
| react-client.md | React, Hook, Redux | React 프론트엔드 |
| electron.md | Electron, IPC | 데스크톱 앱 |
| cloud-infra.md | Terraform, K8s, AWS | 인프라 |

### 스킬 로드 규칙

1. 사용자 언급 키워드 확인
2. 해당 스킬 로드
3. Spec Writer에게 전달

---

## 워크플로우

### 표준 흐름

```
[요청] → 분석 → 참조 → 초안 → 검토 → 수정 → 확정
                      ↑_______|
                      (피드백 반복)
```

### 단계별 상세

| 단계 | 서브세션 | 입력 | 출력 |
|------|----------|------|------|
| 분석 | Planner | 요구사항 | 분해된 요구사항 |
| 참조 | Support | 도메인/키워드 | 관련 문서 |
| 초안 | Spec Writer | 요구사항+참조+스킬 | 명세 초안 |
| 검토 | Reviewer/Architect | 명세 초안 | 피드백 |
| 수정 | Spec Writer | 피드백 | 수정된 명세 |
| 확정 | (직접) | 최종 명세 | 작업지시서 |

---

## 산출물 형식

### 작업지시서

```markdown
# 작업지시서: {port-id}

## 메타데이터
- 레이어: L1 | LM | L2
- 도메인: {domain}
- 우선순위: {priority}
- 의존성: [{deps}]

## 요구사항
### 기능 요구사항
- FR-001: {description}

### 비기능 요구사항
- NFR-001: {description}

## 기술 결정
- {decision}: {rationale}

## 검증 기준
- [ ] {criterion}

## 참조
- {doc}: {summary}

## 예상 정보
- 토큰: ~{tokens}
- 워커: {worker-type}
```

---

## PAL Kit 연동

### 포트 생성
```bash
pal port create {id} --title "{title}"
```

### 작업 시작
```bash
pal hook port-start {id}
```

### KB 검색
```bash
pal kb search "{query}"
```

### 작업 완료
```bash
pal hook port-end {id}
```

---

## Builder 연계

### Handoff 조건
- 명세 확정됨
- Architect 승인됨
- 작업지시서 생성됨

### Handoff 내용
- 확정된 포트 명세
- 작업지시서
- 의존성 그래프
- 예상 토큰 정보
- 권장 워커

---

## 관련 문서

- agents/core/spec.yaml
- agents/core/spec.rules.md
- agents/skills/spec/SKILL.md
