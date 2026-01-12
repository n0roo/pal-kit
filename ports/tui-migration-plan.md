# Port: tui-migration-plan

> TUI 점진적 마이그레이션 계획

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | tui-migration-plan |
| 상태 | complete |
| 우선순위 | medium |
| 의존성 | dashboard-overview, dashboard-projects, dashboard-history, dashboard-doc-viewer |
| 예상 복잡도 | high |
| 완료일 | 2026-01-12 |

---

## 목표

lazygit 수준의 완성도를 갖춘 TUI를 구축하여 Web GUI를 대체할 수 있도록 점진적으로 마이그레이션한다.

---

## 범위

### 포함

- TUI 아키텍처 설계
- Web GUI 기능의 TUI 구현 로드맵
- 키보드 중심 인터랙션 설계
- 점진적 마이그레이션 로드맵

### 제외

- Web GUI 제거 (공존 유지)
- 모바일 UI
- 실제 구현 (계획 문서만)

---

## 현황 분석 결과

### 기술 스택 (확정)

| 항목 | 선택 | 상태 |
|------|------|------|
| Framework | **Bubble Tea** | 이미 사용 중 |
| Styling | **Lip Gloss** | 이미 사용 중 |
| Components | **Bubbles** | 이미 사용 중 |

### 기존 구현 상태

```
internal/tui/
├── tui.go      # 475 lines - 5개 탭 구현
└── styles.go   # 145 lines - 스타일 정의
```

**구현 완료:**
- Tab Navigation (1-5, Tab/Shift+Tab)
- Status, Sessions, Workflows, Docs, Conventions 뷰
- Auto Refresh (5초)
- 기본 스타일링

**미구현 (Gap):**
- Ports View
- Projects View
- History View
- Agents View
- 리스트 선택/스크롤
- 상세 뷰 패널
- Interactive Actions

---

## 마이그레이션 로드맵

### Phase 1: 기반 강화 (v0.4-tui)

| 작업 | 설명 | 우선순위 |
|------|------|----------|
| 선택 가능 리스트 | cursor, selection | P0 |
| 상세 뷰 패널 | 선택 항목 상세 표시 | P0 |
| 키맵 시스템 | 통합 키맵 관리 | P0 |
| 도움말 오버레이 | ? 키로 토글 | P0 |
| 패널 포커스 | List/Detail 전환 | P0 |

### Phase 2: 핵심 뷰 완성 (v0.5-tui)

| 작업 | Web 대응 | 우선순위 |
|------|----------|----------|
| Ports View | Ports 탭 | P0 |
| Sessions View 개선 | 상세, 이벤트 | P0 |
| Status View 개선 | 클릭 가능 카드 | P1 |

### Phase 3: 확장 뷰 (v0.6-tui)

| 작업 | Web 대응 | 우선순위 |
|------|----------|----------|
| Projects View | Projects 탭 | P1 |
| History View | History 탭 | P1 |
| Agents View | Agents 탭 | P2 |

### Phase 4: 고급 기능 (v1.0-tui)

| 작업 | 설명 | 우선순위 |
|------|------|----------|
| Command Palette | fuzzy finder | P2 |
| 분할 뷰 | 다중 패널 | P2 |
| 실시간 업데이트 | 최적화 | P2 |
| 테마 커스터마이징 | 사용자 정의 | P3 |

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| TUI 설계 문서 | `docs/tui-design.md` | 아키텍처, Gap 분석, 로드맵 |
| 키맵 문서 | `docs/tui-keymap.md` | vim-style 단축키 가이드 |

---

## 작업 항목

### 분석 및 계획

- [x] TUI 프레임워크 비교 분석
  - [x] Bubble Tea 장단점 (선택됨)
  - [x] tview 장단점
  - [x] 선정 및 문서화
- [x] 현재 구현 상태 분석
- [x] Web GUI와의 Gap 식별

### 설계 문서

- [x] 아키텍처 설계 (컴포넌트 구조)
- [x] 레이아웃 시스템 설계
- [x] 상태 관리 설계
- [x] 키맵 설계

### 로드맵 문서

- [x] Phase 1 상세 계획
- [x] Phase 2 상세 계획
- [x] Phase 3 상세 계획
- [x] Phase 4 상세 계획
- [x] 마일스톤 정의

---

## 마일스톤

| 마일스톤 | 포함 기능 | 목표 |
|----------|----------|------|
| v0.4-tui | Phase 1 (기반 강화) | 리스트 선택, 상세 뷰 |
| v0.5-tui | Phase 2 (핵심 뷰) | Ports, Sessions 완성 |
| v0.6-tui | Phase 3 (확장 뷰) | Projects, History, Agents |
| v1.0-tui | Phase 4 (고급 기능) | Web GUI 완전 대체 |

---

## 완료 기준

- [x] TUI 설계 문서 작성 (`docs/tui-design.md`)
- [x] 키맵 문서 작성 (`docs/tui-keymap.md`)
- [x] 현황 분석 및 Gap 식별
- [x] 4단계 마이그레이션 로드맵 수립
- [x] 마일스톤 정의

---

<!-- pal:port:status=complete -->
