# Port: dashboard-history

> History 화면 고도화

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | dashboard-history |
| 상태 | draft |
| 우선순위 | medium |
| 의존성 | dashboard-projects |
| 예상 복잡도 | medium |

---

## 목표

프로젝트, 워크플로우, 실행시간, 명령어, 상태 등의 로그를 완결성 있게 표현하는 History 화면을 구축한다.

---

## 범위

### 포함

- 통합 히스토리 로그 뷰
- 필터링 (프로젝트, 워크플로우, 상태, 기간)
- 로그 상세 보기
- 내보내기 기능

### 제외

- 실시간 스트리밍 로그
- 로그 삭제 기능

---

## 로그 필드 정의

| 필드 | 설명 | 예시 |
|------|------|------|
| timestamp | 실행 시간 | 2026-01-12 14:30:00 |
| project | 프로젝트 ID | pal-kit |
| workflow | 워크플로우 타입 | integrate |
| session_id | 세션 ID | 4bd7da58 |
| command | 실행 명령어 | pal port create |
| status | 상태 | success/error/warning |
| duration | 소요 시간 | 1.2s |
| details | 추가 정보 | JSON 형태 |

---

## 화면 구조

```
┌─────────────────────────────────────────────────────────────┐
│ History                                      [Export] [↻]   │
├─────────────────────────────────────────────────────────────┤
│ Filters:                                                    │
│ [Project ▼] [Workflow ▼] [Status ▼] [Date Range ▼] [Search] │
├─────────────────────────────────────────────────────────────┤
│ Timestamp         │ Project  │ Command          │ Status    │
├───────────────────┼──────────┼──────────────────┼───────────┤
│ 14:30:00          │ pal-kit  │ pal port create  │ ✓ success │
│ 14:28:15          │ pal-kit  │ pal agent add    │ ✓ success │
│ 14:25:00          │ k-esg    │ pal hook start   │ ⚠ warning │
│ 14:20:30          │ pal-kit  │ pal config set   │ ✓ success │
└─────────────────────────────────────────────────────────────┘
```

### 상세 모달

```
┌─────────────────────────────────────────────┐
│ Log Detail                            [×]   │
├─────────────────────────────────────────────┤
│ Timestamp: 2026-01-12 14:30:00              │
│ Project:   pal-kit                          │
│ Workflow:  integrate                        │
│ Session:   4bd7da58                         │
│ Command:   pal port create dashboard-overview │
│ Status:    success                          │
│ Duration:  1.2s                             │
├─────────────────────────────────────────────┤
│ Details:                                    │
│ {                                           │
│   "port_id": "dashboard-overview",          │
│   "file": "ports/dashboard-overview.md"     │
│ }                                           │
└─────────────────────────────────────────────┘
```

---

## 작업 항목

### 로그 수집

- [ ] 로그 스키마 정의 (SQLite)
- [ ] 명령어 실행 시 로그 기록
- [ ] Hook 실행 시 로그 기록
- [ ] 에러/경고 캡처

### 히스토리 화면

- [ ] 로그 테이블 컴포넌트
- [ ] 페이지네이션
- [ ] 정렬 (시간순, 역순)
- [ ] 상세 모달

### 필터링

- [ ] 프로젝트 필터
- [ ] 워크플로우 필터
- [ ] 상태 필터 (success/error/warning)
- [ ] 날짜 범위 필터
- [ ] 텍스트 검색

### 내보내기

- [ ] JSON 내보내기
- [ ] CSV 내보내기
- [ ] 필터 적용 상태로 내보내기

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| 로그 저장소 | internal/store/history.go | 히스토리 로그 저장 |
| History API | internal/api/history.go | 히스토리 조회 API |
| History 화면 | internal/web/views/history/ | 웹 화면 |
| TUI History | internal/tui/views/history/ | TUI 버전 |

---

## 완료 기준

- [ ] 모든 PAL 명령어 실행 로그 기록
- [ ] 5가지 필터 모두 동작
- [ ] 로그 상세 모달에서 전체 정보 표시
- [ ] JSON/CSV 내보내기 동작

---

<!-- pal:port:status=draft -->
