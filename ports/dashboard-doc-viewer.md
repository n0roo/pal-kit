# Port: dashboard-doc-viewer

> 문서/컨벤션/에이전트 뷰어 컴포넌트

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | dashboard-doc-viewer |
| 상태 | complete |
| 우선순위 | medium |
| 의존성 | - |
| 예상 복잡도 | medium |

---

## 목표

Documents, Conventions, Agents 문서를 읽고, 복사하고, 다운로드할 수 있는 뷰어 컴포넌트를 구축한다.

---

## 범위

### 포함

- Markdown 렌더링 뷰어
- 코드 블록 복사 기능
- 전체 문서 복사 기능
- 파일 다운로드 기능
- 문서 검색 기능

### 제외

- 문서 편집 기능 (읽기 전용)
- 버전 관리

---

## 지원 문서 타입

| 타입 | 경로 | 설명 |
|------|------|------|
| Documents | docs/*.md | 프로젝트 문서 |
| Conventions | conventions/*.md | 컨벤션 문서 |
| Agents | agents/*.yaml | 에이전트 정의 |

---

## 화면 구조

### 파일 브라우저

```
┌─────────────────────────────────────────────┐
│ Documents                        [Search 🔍] │
├─────────────────────────────────────────────┤
│ 📁 docs/                                    │
│   📄 README.md                              │
│   📄 architecture.md                        │
│   📁 api/                                   │
│     📄 endpoints.md                         │
├─────────────────────────────────────────────┤
│ 📁 conventions/                             │
│   📄 go-style.md                            │
│   📄 commit-message.md                      │
├─────────────────────────────────────────────┤
│ 📁 agents/                                  │
│   📄 worker-go.yaml                         │
└─────────────────────────────────────────────┘
```

### 문서 뷰어

```
┌─────────────────────────────────────────────────────────┐
│ docs/architecture.md              [Copy][Download][×]   │
├─────────────────────────────────────────────────────────┤
│ # Architecture                                          │
│                                                         │
│ ## Overview                                             │
│ PAL Kit is structured as follows...                     │
│                                                         │
│ ```go                                          [Copy]   │
│ func main() {                                           │
│     app := NewApp()                                     │
│     app.Run()                                           │
│ }                                                       │
│ ```                                                     │
│                                                         │
│ ## Components                                           │
│ - CLI: Command parsing                                  │
│ - Store: SQLite persistence                             │
│ - Web: Dashboard server                                 │
└─────────────────────────────────────────────────────────┘
```

---

## 작업 항목

### 파일 브라우저

- [ ] 디렉토리 트리 표시
- [ ] 파일 타입 아이콘
- [ ] 폴더 펼치기/접기
- [ ] 파일 검색

### Markdown 렌더러

- [ ] Markdown → HTML 변환
- [ ] 코드 블록 구문 강조
- [ ] 테이블 렌더링
- [ ] 링크 처리
- [ ] 이미지 표시 (있는 경우)

### YAML 렌더러

- [ ] YAML 구문 강조
- [ ] 구조 접기/펼치기

### 복사 기능

- [ ] 전체 문서 복사 버튼
- [ ] 코드 블록 개별 복사 버튼
- [ ] 클립보드 복사 + 토스트 알림

### 다운로드 기능

- [ ] 단일 파일 다운로드
- [ ] 원본 형태 유지

### 검색 기능

- [ ] 파일명 검색
- [ ] 문서 내용 검색 (옵션)
- [ ] 검색 결과 하이라이트

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| 뷰어 컴포넌트 | internal/web/components/doc-viewer/ | 웹 뷰어 |
| Markdown 렌더러 | internal/web/components/markdown/ | MD 렌더링 |
| TUI 뷰어 | internal/tui/components/doc-viewer/ | TUI 버전 |

---

## 완료 기준

- [x] docs/, conventions/, agents/ 파일 탐색 가능
- [x] Markdown/YAML 파일 렌더링
- [x] 코드 블록 복사 동작
- [x] 전체 문서 복사 동작
- [x] 파일 다운로드 동작

---

## 완료 요약

### 구현된 기능

1. **문서 뷰어 API**
   - `/api/docs/content?path=...` 엔드포인트 추가
   - 문서 내용 및 메타데이터 반환

2. **문서 뷰어 모달**
   - 대형 모달 (modal-large)
   - Copy/Download 버튼
   - 반응형 디자인

3. **Markdown 렌더링**
   - marked.js 라이브러리 사용
   - 코드 블록 구문 강조
   - 테이블, 인용구, 링크 지원
   - 코드 블록별 복사 버튼

4. **YAML 뷰어**
   - 구문 하이라이팅 (키, 값, 숫자, 불린, 주석)

5. **복사/다운로드**
   - 전체 문서 클립보드 복사
   - 원본 파일 다운로드
   - 토스트 알림

6. **검색 기능**
   - Documents 탭에 검색 박스 추가
   - 경로/타입 기준 실시간 필터링

### 수정된 파일

| 파일 | 설명 |
|------|------|
| internal/server/server.go | /api/docs/content 엔드포인트 추가 |
| internal/server/static/index.html | 문서 뷰어 모달, 검색 박스 추가 |
| internal/server/static/style.css | 문서 뷰어 스타일, 검색 박스 스타일 |
| internal/server/static/app.js | showDocViewer, filterDocs, copy/download 함수 |

---

<!-- pal:port:status=complete -->
