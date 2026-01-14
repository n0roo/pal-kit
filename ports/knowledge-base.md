# Port: knowledge-base

> Knowledge Base 구조 관리 시스템

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | knowledge-base |
| 상태 | draft |
| 우선순위 | high |
| 의존성 | pal-reliability |
| 예상 복잡도 | high |

---

## 배경

### 문제점

1. Vault 문서 구조가 체계적이지 않음
2. 목차 관리가 수동으로 이루어짐
3. 검색이 파일명/내용 기반으로만 가능 (분류체계 없음)
4. 프로젝트 문서와 Vault 문서 간 동기화 부재
5. 옵시디언 링크/태그 활용도 낮음

### 참고

- [Semantic Search without Embeddings](https://softwaredoug.com/blog/2026/01/08/semantic-search-without-embeddings.html)
- 분류체계(Taxonomy) 기반 검색 접근법

---

## 목표

1. 체계적인 Knowledge Base 디렉토리 구조 정의
2. 자동 목차(TOC) 생성/갱신 기능
3. 분류체계(Taxonomy) 기반 색인 및 검색
4. 옵시디언 친화적 링크/태그 관리
5. 프로젝트 ↔ Vault 문서 동기화

---

## 디렉토리 구조

```
vault/
├── _index.md                       # 전체 목차 (자동 생성)
├── _taxonomy/                      # 분류체계 정의
│   ├── domains.yaml                # 도메인 정의
│   ├── doc-types.yaml              # 문서 타입 정의
│   ├── tags.yaml                   # 태그 계층
│   └── templates/                  # 문서 템플릿
│
├── 00-System/                      # 시스템/메타 문서
│   ├── _toc.md
│   ├── taxonomy/
│   ├── templates/
│   ├── guides/
│   └── conventions/
│
├── 10-Domains/                     # 도메인 지식
│   ├── _toc.md
│   └── {domain}/
│       ├── _index.md
│       ├── concepts/
│       ├── rules/
│       └── decisions/
│
├── 20-Projects/                    # 프로젝트별 문서
│   ├── _toc.md
│   └── {project}/
│       ├── _index.md
│       ├── ports/
│       ├── architecture/
│       ├── decisions/
│       └── sessions/
│
├── 30-References/                  # 참조 문서
│   ├── _toc.md
│   ├── patterns/
│   ├── libraries/
│   └── standards/
│
├── 40-Archive/                     # 아카이브
│   ├── _toc.md
│   ├── completed-ports/
│   └── deprecated/
│
└── .pal-kb/                        # PAL Kit KB 메타데이터
    ├── index.db                    # 검색 색인 (SQLite)
    ├── link-graph.json             # 링크 그래프
    ├── toc-cache.json              # 목차 캐시
    └── sync-state.yaml             # 프로젝트 동기화 상태
```

---

## 작업 항목

### Phase 1: 기반 구조 (P1)

- [ ] `pal kb init` - KB 초기화
  - [ ] 디렉토리 구조 생성
  - [ ] _taxonomy/ 기본 파일 생성
  - [ ] _index.md 템플릿 생성
- [ ] `pal kb status` - KB 상태 확인
- [ ] Taxonomy YAML 스키마 정의
  - [ ] domains.yaml
  - [ ] doc-types.yaml
  - [ ] tags.yaml

### Phase 2: 목차 관리 (P2)

- [ ] `pal kb toc generate` - 목차 생성
  - [ ] n-depth 지원 (설정 가능)
  - [ ] 정렬 옵션 (alphabetical, date, custom)
  - [ ] 통계 포함 (문서 수, 최근 수정)
- [ ] `pal kb toc update` - 목차 갱신
- [ ] `pal kb toc check` - 목차 무결성 검사
- [ ] _toc.md frontmatter 스키마 정의

### Phase 3: 색인 및 검색 (P3)

- [ ] `pal kb index` - 색인 구축/갱신
  - [ ] 제목 색인
  - [ ] 요약(summary) 색인
  - [ ] 태그 색인
  - [ ] 별칭(aliases) 색인
- [ ] `pal kb search <query>` - 검색
  - [ ] 분류체계 기반 필터링
  - [ ] 토큰 예산 지원
- [ ] index.db 스키마 설계 (SQLite)

### Phase 4: 링크/태그 관리 (P4)

- [ ] `pal kb link check` - 깨진 링크 검사
- [ ] `pal kb link graph` - 링크 그래프 생성
- [ ] `pal kb tag list` - 태그 목록
- [ ] `pal kb tag orphan` - 미사용 태그 검출
- [ ] 옵시디언 [[wikilink]] 파싱

### Phase 5: 프로젝트 동기화 (P5)

- [ ] `pal kb sync <project>` - 프로젝트 동기화
  - [ ] ports/ → 20-Projects/{project}/ports/
  - [ ] .pal/decisions/ → 20-Projects/{project}/decisions/
  - [ ] .pal/sessions/ → 20-Projects/{project}/sessions/
- [ ] sync-state.yaml 관리
- [ ] 양방향 vs 단방향 동기화 정책

### Phase 6: Supporter 통합 (P6)

- [ ] Support Agent에 KB 검색 도구 추가
- [ ] 분류 추천 기능
- [ ] 문서 품질 체크 기능

---

## 문서 스키마

### Frontmatter 필수 필드

```yaml
---
type: <doc-type>          # port, adr, concept, guide, etc.
title: <제목>
aliases: [<별칭들>]        # 검색용
tags: [<태그들>]           # 분류용
status: <상태>             # draft, active, archived
created: <생성일>
updated: <수정일>
---
```

### 문서 타입별 추가 필드

```yaml
# port
domain: <도메인>
priority: <우선순위>
dependencies: [<의존성>]

# adr
decision_date: <결정일>
decision_makers: [<결정자>]
supersedes: <대체하는 ADR>

# concept
domain: <도메인>
related: [<관련 개념>]
```

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| KB 서비스 | internal/kb/service.go | KB 핵심 로직 |
| KB CLI | internal/cli/kb.go | pal kb 커맨드 |
| TOC 생성기 | internal/kb/toc.go | 목차 생성/갱신 |
| 색인 서비스 | internal/kb/index.go | 검색 색인 |
| 링크 분석기 | internal/kb/link.go | 링크/태그 관리 |
| 동기화 서비스 | internal/kb/sync.go | 프로젝트 동기화 |
| Taxonomy 파서 | internal/kb/taxonomy.go | 분류체계 처리 |
| DB 스키마 | internal/db/kb_schema.sql | 색인 DB |

---

## 완료 기준

- [ ] `pal kb init`으로 KB 구조 생성 가능
- [ ] `pal kb toc generate`로 n-depth 목차 자동 생성
- [ ] `pal kb index && pal kb search`로 분류체계 기반 검색
- [ ] `pal kb link check`로 깨진 링크 검출
- [ ] `pal kb sync`로 프로젝트 문서 동기화
- [ ] 옵시디언에서 링크/태그 정상 동작
- [ ] Supporter가 KB 검색 활용 가능

---

## 기술 결정

### 색인 저장소: SQLite

- 이유: 단일 파일, 설치 불필요, 빠른 조회
- 대안: JSON (느림), 외부 DB (복잡)

### 목차 형식: Markdown + Frontmatter

- 이유: 옵시디언 호환, 사람 읽기 가능
- 자동 생성 영역 마커 사용

### 링크 형식: [[wikilink]]

- 이유: 옵시디언 네이티브
- 경로 기반과 별칭 기반 모두 지원

### 언어 정책

- 메타데이터 키: 영문
- 태그: 영문 (예: domain/auth)
- 내용: 한글 자유
- 별칭: 한글+영문 (검색용)

---

## 구현 우선순위

```
P1: 기반 구조     ← 필수 선행
P2: 목차 관리     ← 핵심 기능
P3: 색인/검색     ← 핵심 기능
P4: 링크/태그     ← 옵시디언 연동
P5: 프로젝트 동기화 ← 통합
P6: Supporter     ← 확장
```

---

<!-- pal:port:status=draft -->
