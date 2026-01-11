# Port: tui-migration-plan

> TUI 점진적 마이그레이션 계획

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | tui-migration-plan |
| 상태 | draft |
| 우선순위 | medium |
| 의존성 | dashboard-overview, dashboard-projects, dashboard-history, dashboard-doc-viewer |
| 예상 복잡도 | high |

---

## 목표

lazygit 수준의 완성도를 갖춘 TUI를 구축하여 Web GUI를 대체할 수 있도록 점진적으로 마이그레이션한다.

---

## 범위

### 포함

- TUI 아키텍처 설계
- Web GUI 기능의 TUI 구현
- 키보드 중심 인터랙션
- 점진적 마이그레이션 로드맵

### 제외

- Web GUI 제거 (공존 유지)
- 모바일 UI

---

## 참고: lazygit 스타일

```
┌─ Status ─────────────────────────────────────────────────────┐
│ PAL Kit | integrate | 3 sessions active                      │
├─ Sessions ───────────────────┬─ Details ─────────────────────┤
│ > 4bd7da58 (active) 5m       │ Session: 4bd7da58             │
│   a1b2c3d4 (done) 2h         │ Duration: 5m30s               │
│   e5f6g7h8 (done) 1d         │ Tokens: 12.5K                 │
│                              │ Ports: 2                      │
│                              │ Status: active                │
├─ Ports ──────────────────────┼───────────────────────────────┤
│ > dashboard-overview (draft) │ # dashboard-overview          │
│   agent-spec-review (draft)  │                               │
│   tui-migration (draft)      │ Overview 화면 리디자인         │
│                              │ Status: draft                 │
├─ Commands ───────────────────┴───────────────────────────────┤
│ [j/k] Navigate  [Enter] Select  [?] Help  [q] Quit           │
└──────────────────────────────────────────────────────────────┘
```

---

## 마이그레이션 로드맵

### Phase 1: 기반 구축

| 순서 | 기능 | 설명 | 우선순위 |
|------|------|------|----------|
| 1-1 | TUI 프레임워크 선정 | Bubble Tea / tview 비교 | P0 |
| 1-2 | 레이아웃 시스템 | 패널 분할, 리사이즈 | P0 |
| 1-3 | 키맵 시스템 | 단축키 바인딩 | P0 |
| 1-4 | 테마 시스템 | 색상, 스타일 | P1 |

### Phase 2: 핵심 화면

| 순서 | 기능 | Web 대응 | 우선순위 |
|------|------|----------|----------|
| 2-1 | Status View | Overview | P0 |
| 2-2 | Sessions View | Sessions 탭 | P0 |
| 2-3 | Ports View | Ports 탭 | P0 |
| 2-4 | Workflows View | Pipelines 탭 | P1 |

### Phase 3: 확장 화면

| 순서 | 기능 | Web 대응 | 우선순위 |
|------|------|----------|----------|
| 3-1 | Projects View | Projects 화면 | P1 |
| 3-2 | History View | History 화면 | P1 |
| 3-3 | Document Viewer | Doc Viewer | P2 |
| 3-4 | Agent Manager | Agents 화면 | P2 |

### Phase 4: 고급 기능

| 순서 | 기능 | 설명 | 우선순위 |
|------|------|------|----------|
| 4-1 | 실시간 업데이트 | WebSocket 연동 | P2 |
| 4-2 | 명령어 팔레트 | fuzzy finder | P2 |
| 4-3 | 분할 뷰 | 다중 패널 | P2 |
| 4-4 | 설정 화면 | TUI 내 설정 | P3 |

---

## 작업 항목

### Phase 1 상세

- [ ] TUI 프레임워크 비교 분석
  - [ ] Bubble Tea 장단점
  - [ ] tview 장단점
  - [ ] 선정 및 문서화
- [ ] 기본 레이아웃 구현
  - [ ] 메인 레이아웃 (3-패널)
  - [ ] 패널 포커스 전환
  - [ ] 패널 크기 조정
- [ ] 키맵 시스템
  - [ ] 전역 키맵
  - [ ] 뷰별 키맵
  - [ ] 키맵 도움말 (?)
- [ ] 테마 시스템
  - [ ] 기본 테마
  - [ ] 다크/라이트 모드

### Phase 2 상세

- [ ] Status View
  - [ ] 세션 요약
  - [ ] 포트 요약
  - [ ] 메트릭 표시
- [ ] Sessions View
  - [ ] 세션 목록
  - [ ] 세션 상세
  - [ ] 세션 작업 (시작/종료)
- [ ] Ports View
  - [ ] 포트 목록
  - [ ] 포트 상세 (Markdown 렌더링)
  - [ ] 포트 작업 (생성/상태변경)
- [ ] Workflows View
  - [ ] 파이프라인 목록
  - [ ] 파이프라인 실행

---

## 기술 스택 비교

### Bubble Tea

| 장점 | 단점 |
|------|------|
| 모던 Go 패턴 (Elm 아키텍처) | 학습 곡선 |
| 활발한 커뮤니티 | 복잡한 레이아웃 어려움 |
| 컴포넌트 재사용 용이 | - |
| Lip Gloss 스타일링 | - |

### tview

| 장점 | 단점 |
|------|------|
| 풍부한 위젯 | 레거시 패턴 |
| 복잡한 레이아웃 지원 | 커스터마이징 어려움 |
| 문서화 잘 됨 | - |

**권장**: Bubble Tea (모던 패턴, lazygit도 Bubble Tea 기반)

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| TUI 설계 문서 | docs/tui-design.md | 아키텍처 설계 |
| TUI 패키지 | internal/tui/ | TUI 구현 |
| 키맵 문서 | docs/tui-keymap.md | 단축키 가이드 |

---

## 마일스톤

| 마일스톤 | 포함 기능 | 목표 |
|----------|----------|------|
| v0.1-tui | Phase 1 + Status View | 기본 동작 |
| v0.2-tui | Sessions, Ports View | 핵심 기능 |
| v0.3-tui | Projects, History | 확장 기능 |
| v1.0-tui | 전체 기능 | Web 대체 가능 |

---

## 완료 기준

- [ ] Phase 1 완료: 기본 프레임워크 동작
- [ ] Phase 2 완료: 핵심 화면 동작
- [ ] Phase 3 완료: 확장 화면 동작
- [ ] Phase 4 완료: 고급 기능 동작
- [ ] Web GUI의 모든 기능 TUI에서 사용 가능

---

<!-- pal:port:status=draft -->
