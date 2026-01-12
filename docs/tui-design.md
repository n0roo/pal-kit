# PAL Kit TUI 설계 문서

> lazygit 수준의 완성도를 갖춘 TUI 마이그레이션 계획

---

## 1. 현황 분석

### 1.1 기술 스택 (확정)

| 항목 | 선택 | 근거 |
|------|------|------|
| Framework | **Bubble Tea** | 이미 사용 중, 모던 Go 패턴, lazygit 참고 |
| Styling | **Lip Gloss** | Bubble Tea 생태계, 이미 사용 중 |
| Components | **Bubbles** | spinner, viewport 등 표준 컴포넌트 |

### 1.2 현재 구현 상태

```
internal/tui/
├── tui.go      # 475 lines - 메인 모델, 5개 탭
└── styles.go   # 145 lines - 스타일 정의
```

**구현된 기능:**

| 기능 | 상태 | 비고 |
|------|------|------|
| Tab Navigation | O | 1-5, Tab/Shift+Tab |
| Status View | O | Sessions, Workflows, Docs, Conventions 요약 |
| Sessions List | O | 기본 목록 표시 |
| Workflows List | O | 기본 목록 표시 |
| Docs List | O | 타입별 그룹핑 |
| Conventions List | O | 기본 목록 표시 |
| Auto Refresh | O | 5초 간격 |
| Styling | O | 색상, 박스, 상태 아이콘 |

**미구현 기능:**

| 기능 | Web GUI | 우선순위 |
|------|---------|----------|
| Ports View | Ports 탭 | P0 |
| Projects View | Projects 탭 | P1 |
| History View | History 탭 | P1 |
| Agents View | Agents 탭 | P2 |
| Session Detail | 모달 | P0 |
| Port Detail | 인라인 확장 | P0 |
| List Selection | - | P0 |
| Scrollable Lists | - | P0 |
| Interactive Actions | 생성/수정/삭제 | P1 |
| Command Palette | - | P2 |

---

## 2. 아키텍처 설계

### 2.1 컴포넌트 구조

```
internal/tui/
├── tui.go              # 메인 모델, 라우팅
├── styles.go           # 전역 스타일
├── keymap.go           # 키맵 정의
├── components/
│   ├── list.go         # 선택 가능한 리스트
│   ├── table.go        # 테이블 컴포넌트
│   ├── detail.go       # 상세 뷰 패널
│   ├── modal.go        # 모달 컴포넌트
│   └── help.go         # 도움말 오버레이
└── views/
    ├── status.go       # Overview 뷰
    ├── sessions.go     # Sessions 뷰
    ├── ports.go        # Ports 뷰
    ├── projects.go     # Projects 뷰
    ├── history.go      # History 뷰
    ├── workflows.go    # Workflows 뷰
    ├── docs.go         # Documents 뷰
    ├── conventions.go  # Conventions 뷰
    └── agents.go       # Agents 뷰
```

### 2.2 레이아웃 시스템

**lazygit 스타일 3-패널 레이아웃:**

```
┌─ Header ────────────────────────────────────────────────────────┐
│ PAL Kit | integrate | 3 sessions active                          │
├─ List Panel ─────────────┬─ Detail Panel ────────────────────────┤
│ > item 1 (selected)      │ Detail view for selected item         │
│   item 2                 │                                       │
│   item 3                 │ - Field 1: value                      │
│   item 4                 │ - Field 2: value                      │
│                          │ - Field 3: value                      │
│                          │                                       │
├─ Help ───────────────────┴───────────────────────────────────────┤
│ [j/k] Navigate  [Enter] Select  [?] Help  [q] Quit               │
└──────────────────────────────────────────────────────────────────┘
```

**레이아웃 비율:**
- List Panel: 40%
- Detail Panel: 60%
- 최소 너비: 80자
- 최소 높이: 24줄

### 2.3 상태 관리

```go
type Model struct {
    // Navigation
    currentTab    Tab
    currentView   View
    focusedPanel  Panel  // list, detail

    // Selection
    cursor        int
    selectedID    string

    // Window
    width, height int
    ready         bool

    // Data (캐시)
    sessions    []Session
    ports       []Port
    projects    []Project
    // ...

    // Sub-models
    listModel   list.Model
    detailModel detail.Model
    helpModel   help.Model
}
```

---

## 3. 키맵 설계

### 3.1 전역 키맵

| 키 | 동작 | 비고 |
|----|------|------|
| `q` | 종료 | Quit |
| `?` | 도움말 토글 | Help overlay |
| `1-9` | 탭 전환 | Direct tab switch |
| `Tab` | 다음 탭 | Next tab |
| `Shift+Tab` | 이전 탭 | Previous tab |
| `r` | 새로고침 | Refresh data |
| `Ctrl+C` | 종료 | Force quit |

### 3.2 리스트 키맵

| 키 | 동작 | 비고 |
|----|------|------|
| `j` / `Down` | 다음 항목 | vim-style |
| `k` / `Up` | 이전 항목 | vim-style |
| `g` | 첫 항목 | Go to top |
| `G` | 마지막 항목 | Go to bottom |
| `Enter` | 선택/상세 | Select item |
| `/` | 검색 | Filter list |
| `Esc` | 검색 취소 | Clear filter |

### 3.3 뷰별 키맵

**Sessions View:**
| 키 | 동작 |
|----|------|
| `n` | 새 세션 |
| `d` | 세션 종료 |
| `e` | 이벤트 보기 |

**Ports View:**
| 키 | 동작 |
|----|------|
| `n` | 새 포트 생성 |
| `s` | 포트 시작 |
| `c` | 포트 완료 |
| `e` | 포트 편집 |

**History View:**
| 키 | 동작 |
|----|------|
| `f` | 필터 설정 |
| `x` | JSON 내보내기 |
| `X` | CSV 내보내기 |

---

## 4. 마이그레이션 로드맵

### Phase 1: 기반 강화 (v0.4-tui)

**목표:** 리스트 선택, 상세 뷰, 기본 인터랙션

| 작업 | 파일 | 설명 |
|------|------|------|
| 선택 가능 리스트 | components/list.go | cursor, selection |
| 상세 뷰 패널 | components/detail.go | 선택 항목 상세 |
| 키맵 시스템 | keymap.go | 통합 키맵 관리 |
| 도움말 오버레이 | components/help.go | ? 키로 토글 |
| 패널 포커스 | tui.go | List/Detail 전환 |

**완료 기준:**
- [ ] j/k로 리스트 탐색
- [ ] Enter로 상세 보기
- [ ] Tab으로 패널 전환
- [ ] ?로 도움말 표시

### Phase 2: 핵심 뷰 완성 (v0.5-tui)

**목표:** Ports, Sessions 뷰 완성

| 작업 | 파일 | 설명 |
|------|------|------|
| Ports View | views/ports.go | 포트 목록/상세/작업 |
| Sessions View 개선 | views/sessions.go | 상세, 이벤트 타임라인 |
| Status View 개선 | views/status.go | 클릭 가능한 카드 |

**완료 기준:**
- [ ] Ports 탭에서 포트 목록/상세 확인
- [ ] 포트 시작/완료 키보드 작업
- [ ] 세션 상세에서 이벤트 타임라인

### Phase 3: 확장 뷰 (v0.6-tui)

**목표:** Projects, History, Agents 뷰

| 작업 | 파일 | 설명 |
|------|------|------|
| Projects View | views/projects.go | 프로젝트 목록/상세 |
| History View | views/history.go | 이벤트 로그, 필터 |
| Agents View | views/agents.go | 에이전트 목록 |

**완료 기준:**
- [ ] 프로젝트별 세션/포트 확인
- [ ] 히스토리 필터링
- [ ] 에이전트 목록 표시

### Phase 4: 고급 기능 (v1.0-tui)

**목표:** Web GUI 완전 대체

| 작업 | 설명 |
|------|------|
| Command Palette | `/` 또는 `Ctrl+P` fuzzy finder |
| 분할 뷰 | 다중 패널 동시 표시 |
| 실시간 업데이트 | WebSocket 또는 polling 최적화 |
| 테마 커스터마이징 | 사용자 정의 색상 |

---

## 5. 기술 결정 사항

### 5.1 Bubble Tea 선택 이유

| 기준 | Bubble Tea | tview |
|------|------------|-------|
| 아키텍처 | Elm (MVU) | 전통적 이벤트 |
| 학습 곡선 | 중간 | 낮음 |
| 커스터마이징 | 높음 | 중간 |
| 커뮤니티 | 활발 | 안정 |
| lazygit 참고 | O | X |

**결론:** 이미 Bubble Tea 사용 중이며, 모던 패턴과 lazygit 참고 가능

### 5.2 List 컴포넌트 선택

- **bubbles/list**: 기본 제공, 필터링 지원
- **커스텀 구현**: 더 세밀한 제어 필요시

**권장:** bubbles/list 우선 사용, 필요시 확장

### 5.3 데이터 갱신 전략

- **현재:** 5초 간격 polling
- **개선:** 변경 감지 시에만 갱신 (file watcher 또는 DB trigger)

---

## 6. 구현 가이드

### 6.1 새 뷰 추가 절차

1. `views/` 디렉토리에 새 파일 생성
2. View 모델 정의 (Init, Update, View)
3. Tab enum에 추가
4. 메인 Model의 라우팅에 추가
5. 키맵 추가

### 6.2 스타일 가이드

```go
// 색상 팔레트 (styles.go)
primaryColor   = "#7C3AED"  // Purple - 주요 강조
secondaryColor = "#10B981"  // Green - 성공/활성
warningColor   = "#F59E0B"  // Yellow - 경고
errorColor     = "#EF4444"  // Red - 오류
mutedColor     = "#6B7280"  // Gray - 비활성
```

### 6.3 테스트 전략

- 각 뷰의 렌더링 결과 스냅샷 테스트
- 키 입력에 따른 상태 변화 유닛 테스트
- 통합 테스트: 전체 흐름 시나리오

---

## 7. 참고 자료

- [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- [Bubbles](https://github.com/charmbracelet/bubbles)
- [lazygit](https://github.com/jesseduffield/lazygit) - UI 참고
- [glow](https://github.com/charmbracelet/glow) - 마크다운 렌더링 참고

---

<!-- Last updated: 2026-01-12 -->
