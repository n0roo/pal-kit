# Spec Agent Skills

> 도메인별 명세 작성 스킬

---

## 개요

Spec Agent는 도메인에 따라 적절한 스킬을 로드하여 명세를 작성합니다.
각 스킬은 해당 도메인의 템플릿, 컨벤션, 검증 기준을 제공합니다.

---

## 스킬 목록

| 스킬 | 파일 | 대상 도메인 |
|------|------|------------|
| PA-Layered Go | pa-layered-go.md | Go 백엔드 (PA-Layered) |
| Spring MSA | spring-msa.md | Spring Cloud MSA |
| React Client | react-client.md | React 프론트엔드 |
| Electron | electron.md | Electron 데스크톱 앱 |
| Cloud Infra | cloud-infra.md | IaC, K8s, Cloud |

---

## 스킬 구조

각 스킬은 다음 섹션을 포함합니다:

```markdown
# {Domain} Spec Skill

## 도메인 특성
- 핵심 개념
- 아키텍처 패턴
- 주요 제약 조건

## 템플릿
- 명세 템플릿 종류
- 각 템플릿 용도

## 컨벤션
- 네이밍 규칙
- 구조 규칙
- 의존성 규칙

## 검증 기준
- 필수 체크리스트
- 품질 기준
```

---

## 스킬 로딩

### 자동 로딩 트리거

| 키워드 | 로드되는 스킬 |
|--------|--------------|
| Go, L1, LM, L2, PA-Layered | pa-layered-go.md |
| Spring, MSA, Gateway, Kafka | spring-msa.md |
| React, Redux, Hook, Component | react-client.md |
| Electron, IPC, Main Process | electron.md |
| Terraform, AWS, K8s, Docker | cloud-infra.md |

### 수동 지정

```
"PA-Layered Go 스킬로 명세 작성해줘"
"Spring MSA 명세 템플릿 사용해줘"
```

---

## 다중 도메인

하나의 프로젝트에서 여러 도메인이 필요한 경우:

```
예: Full-stack 프로젝트
- 백엔드: pa-layered-go.md
- 프론트엔드: react-client.md
- 인프라: cloud-infra.md
```

Spec Agent는 각 명세에 적절한 스킬을 로드합니다.

---

## 스킬 확장

새 도메인 스킬 추가 시:

1. `agents/skills/spec/{domain}.md` 파일 생성
2. 표준 구조 따르기
3. `spec.yaml`의 `domain_skills`에 등록
4. 자동 로딩 키워드 추가

---

## 관련 문서

- agents/core/spec.yaml
- agents/core/spec.rules.md
- conventions/CONVENTION-LOADING.md
