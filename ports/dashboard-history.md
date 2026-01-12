# Port: dashboard-history

> History 화면 고도화

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | dashboard-history |
| 상태 | complete |
| 우선순위 | medium |
| 의존성 | dashboard-projects |
| 예상 복잡도 | medium |
| 완료일 | 2026-01-12 |

---

## 목표

프로젝트, 워크플로우, 실행시간, 명령어, 상태 등의 로그를 완결성 있게 표현하는 History 화면을 구축한다.

---

## 범위

### 포함

- 통합 히스토리 로그 뷰
- 필터링 (프로젝트, 이벤트 타입, 상태, 기간, 검색)
- 로그 상세 보기
- 내보내기 기능 (JSON/CSV)

### 제외

- 실시간 스트리밍 로그
- 로그 삭제 기능

---

## 구현 완료 내용

### 1. History 서비스 (`internal/history/history.go`)

```go
// 주요 구조체
type Event struct {
    ID, SessionID, EventType, EventData, CreatedAt
    ProjectRoot, ProjectName (조인 필드)
}

type EventDetail struct {
    Event
    ParsedData map[string]interface{}
    Status     string  // success, error, warning, info
}

type Filter struct {
    SessionID, EventType, ProjectRoot, Status
    StartDate, EndDate, Search
    Limit, Offset
}

// 주요 메서드
func (s *Service) List(filter Filter) ([]EventDetail, int, error)
func (s *Service) GetEventTypes() ([]string, error)
func (s *Service) GetProjects() ([]string, error)
func (s *Service) GetStats() (map[string]interface{}, error)
func (s *Service) ExportJSON(filter Filter) ([]byte, error)
func (s *Service) ExportCSV(filter Filter) (string, error)
```

### 2. History API (`internal/server/server.go`)

| 엔드포인트 | 메서드 | 설명 |
|-----------|--------|------|
| `/api/history/events` | GET | 이벤트 목록 (필터/페이지네이션) |
| `/api/history/types` | GET | 고유 이벤트 타입 목록 |
| `/api/history/projects` | GET | 고유 프로젝트 목록 |
| `/api/history/stats` | GET | 이벤트 통계 |
| `/api/history/export` | GET | JSON/CSV 내보내기 |

### 3. History UI (`internal/server/static/`)

**index.html 변경:**
- History 탭 전체 리디자인
- 필터 UI (Event Type, Project, Date Range, Search)
- 이벤트 테이블 (Status, Time, Event Type, Session, Project, Details)
- 페이지네이션
- Export 버튼 (JSON/CSV)

**app.js 추가 함수:**
- `initHistoryFilters()` - 필터 드롭다운 초기화
- `loadEventHistory()` - 이벤트 목록 로드
- `eventStatusBadge()`, `eventTypeIcon()` - UI 헬퍼
- `updateHistoryPagination()` - 페이지네이션 갱신
- `historyPrevPage()`, `historyNextPage()` - 페이지 이동
- `debounceHistorySearch()` - 검색 디바운스
- `exportHistory(format)` - 내보내기 실행

**style.css 추가:**
- `.filters-row`, `.filter-group` - 필터 레이아웃
- `.history-stats` - 통계 표시
- `.pagination` - 페이지네이션
- `.event-detail` - 이벤트 상세 셀
- `.toast` - 알림 토스트

---

## 작업 항목

### 로그 수집 (기존 session_events 테이블 활용)

- [x] 기존 session_events 스키마 활용
- [x] session_start, session_end, compact 이벤트 기록
- [x] 세션-프로젝트 조인으로 프로젝트 정보 포함

### 히스토리 화면

- [x] 이벤트 테이블 컴포넌트
- [x] 페이지네이션 (50건 단위)
- [x] 시간순 정렬 (최신순)
- [x] 상태별 아이콘/뱃지

### 필터링

- [x] 이벤트 타입 필터
- [x] 프로젝트 필터 (project_name, project_root 모두 지원)
- [x] 날짜 범위 필터 (Today, 7d, 30d, All)
- [x] 텍스트 검색 (디바운스 적용)

### 내보내기

- [x] JSON 내보내기
- [x] CSV 내보내기
- [x] 필터 적용 상태로 내보내기

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| History 서비스 | `internal/history/history.go` | 히스토리 조회/내보내기 서비스 |
| History API | `internal/server/server.go` | 5개 API 엔드포인트 |
| History UI | `internal/server/static/index.html` | History 탭 UI |
| History JS | `internal/server/static/app.js` | 필터/페이지네이션/내보내기 |
| History CSS | `internal/server/static/style.css` | 필터/페이지네이션 스타일 |

---

## 테스트 결과

### 필터링 테스트

| 필터 | 조건 | 결과 |
|------|------|------|
| Event Type | `session_start` | 9건 |
| Event Type | `compact` | 6건 |
| Project | `setup-test` | 2건 |
| Project | `pal-kit` | 15건 |
| Search | `prompt_input` | 2건 |
| 복합 | `pal-kit` + `session_start` | 8건 |
| 날짜 | 오늘 | 8건 |

### Export 테스트

- JSON Export: 17건 정상 출력
- CSV Export: 헤더 + 데이터 정상 출력
- 필터 적용 Export: 정상 동작

---

## 완료 기준

- [x] session_events 기반 이벤트 로그 조회
- [x] 5가지 필터 모두 동작 (타입, 프로젝트, 날짜, 검색, 복합)
- [x] 페이지네이션 동작
- [x] JSON/CSV 내보내기 동작
- [x] 필터 적용 상태로 내보내기 동작

---

<!-- pal:port:status=complete -->
