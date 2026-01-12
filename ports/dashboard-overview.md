# Port: dashboard-overview

> Overview 화면 리디자인

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | dashboard-overview |
| 상태 | complete |
| 우선순위 | high |
| 의존성 | - |
| 예상 복잡도 | medium |

---

## 목표

대시보드 Overview 화면의 카드 UI를 개선하고, 상세 정보 접근성을 높인다.

---

## 범위

### 포함

- 세션 카드 리디자인 (Active/Completed 분리)
- Ports 카드 리디자인
- Pipelines → Workflows 명칭 변경
- 메트릭 카드 분리 (Tokens, Cost, Time, Escalations)
- 모달 상세 뷰 구현
- 탭 이동 + 필터 적용 기능

### 제외

- 백엔드 API 변경
- 새로운 메트릭 추가

---

## 현재 상태 분석

```
현재 Overview 구조:
┌─────────────────────────────────────┐
│ Sessions: X active / Y total        │
│ Ports: (없음)                       │
│ Pipelines: X active / Y total       │
│ Agents: X 등록됨                    │
│ Locks: X active                     │
│ Escalations: 없음                   │
└─────────────────────────────────────┘
```

---

## 목표 UI 구조

```
개선된 Overview 구조:
┌──────────────────┬──────────────────┐
│ Active Sessions  │ Completed        │
│ [3] ▶ 상세       │ [12] ▶ 상세      │
├──────────────────┼──────────────────┤
│ Ports            │ Workflows        │
│ [5] ▶ 상세       │ [2] ▶ 상세       │
├──────────────────┴──────────────────┤
│ Metrics                             │
│ ┌────────┬────────┬────────┬──────┐ │
│ │Tokens  │ Cost   │ Time   │Escal.│ │
│ │ 50.2K  │ $1.23  │ 2h30m  │  0   │ │
│ └────────┴────────┴────────┴──────┘ │
└─────────────────────────────────────┘
```

---

## 작업 항목

### 카드 UI 개선

- [ ] ActiveSessions 카드
  - [ ] 개수 표시 + 클릭 시 모달
  - [ ] 모달에서 세션 목록 표시
  - [ ] "Sessions 탭으로 이동" 버튼
  - [ ] 필터 적용 후 이동 기능
- [ ] CompletedSessions 카드
  - [ ] 위와 동일한 동작
- [ ] Ports 카드
  - [ ] 개수 표시 + 클릭 시 모달
  - [ ] 상태별 필터 (draft/active/done)
- [ ] Workflows 카드 (Pipelines 명칭 변경)
  - [ ] 개수 표시 + 클릭 시 모달

### 메트릭 카드 분리

- [ ] Tokens 카드
  - [ ] 총 사용량 표시
  - [ ] 세션별 breakdown 모달
- [ ] Cost 카드
  - [ ] 예상 비용 표시
  - [ ] 기간별 추이 (옵션)
- [ ] Time 카드
  - [ ] 총 작업 시간
  - [ ] 평균 세션 시간
- [ ] Escalations 카드
  - [ ] 미해결 에스컬레이션 수
  - [ ] 클릭 시 상세 목록

### 모달 컴포넌트

- [ ] 공통 모달 레이아웃
- [ ] 목록 표시 컴포넌트
- [ ] 필터 UI
- [ ] 탭 이동 액션

---

## 산출물

| 산출물 | 경로 | 설명 |
|--------|------|------|
| Overview 컴포넌트 | internal/web/components/overview/ | 리디자인된 컴포넌트 |
| 모달 컴포넌트 | internal/web/components/modal/ | 공통 모달 |
| TUI Overview | internal/tui/views/overview/ | TUI 버전 |

---

## 완료 기준

- [x] 모든 카드가 클릭 시 모달 표시
- [x] 모달에서 상세 탭으로 이동 + 필터 적용 동작
- [x] Pipelines → Workflows 명칭 변경 완료
- [x] 메트릭 카드 4개 분리 완료

---

## 완료 요약

### 구현된 기능

1. **Overview 카드 리디자인**
   - Active Sessions / Completed Sessions 분리
   - Ports 카드 추가 (상태별 breakdown)
   - Workflows 카드 (Pipelines에서 명칭 변경)

2. **Metrics 카드 분리**
   - Tokens 카드 (총 토큰 사용량)
   - Cost 카드 (예상 비용, 하이라이트 표시)
   - Time 카드 (총 작업 시간)
   - Escalations 카드 (미해결 건수, 경고 표시)

3. **showOverviewModal 구현**
   - 각 카드 클릭 시 모달 팝업
   - 유형별 상세 정보 표시 (세션 목록, 포트 상태별 그룹, 토큰 breakdown 등)
   - 모달에서 세션 상세 조회 연결

4. **TUI 업데이트**
   - Pipelines → Workflows 명칭 변경
   - Status 탭의 Workflows 박스 표시

### 수정된 파일

| 파일 | 설명 |
|------|------|
| internal/server/static/index.html | Overview 레이아웃 리디자인 |
| internal/server/static/style.css | 새 CSS 클래스 추가 |
| internal/server/static/app.js | showOverviewModal, loadWorkflows |
| internal/tui/tui.go | Workflows 명칭 변경 |

---

<!-- pal:port:status=complete -->
