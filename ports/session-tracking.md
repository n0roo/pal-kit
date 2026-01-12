# Port: session-tracking

> 세션 및 포트 추적 시스템 개선

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | session-tracking |
| 상태 | running |
| 우선순위 | critical |
| 의존성 | - |
| 예상 복잡도 | high |

---

## 목표

세션과 포트의 상태 관리, 토큰/비용 추적을 정확하게 동작하도록 개선한다.

---

## 문제점

### 1. 세션 관련
- 세션 title이 비어있음
- 세션 status가 running으로 고착 (종료 감지 실패)
- 세션 tokens/cost가 항상 0 (수집 미구현)

### 2. 포트 관련
- 포트에 session_id 연결 안됨
- 포트에 tokens/cost/duration 컬럼 없음
- 포트별 에이전트 정보 없음

---

## 작업 항목

### P1: 세션 종료 로직 수정

- [ ] Claude session_id → PAL session 매핑 개선
- [ ] 좀비 세션 정리 로직 추가
- [ ] 세션 종료 시 상태 확실히 complete로 변경

### P1: Usage 수집 구현

- [ ] JSONL 파싱하여 토큰/비용 추출
- [ ] 세션 종료 시 Usage 업데이트

### P2: 포트 스키마 확장

- [ ] ports 테이블에 컬럼 추가
- [ ] DB 마이그레이션 v5 작성

### P2: 세션-포트 연계

- [ ] port-start hook에서 현재 세션 연결
- [ ] port-end hook에서 포트 통계 업데이트

---

## 완료 기준

- [ ] 세션 종료 시 status가 complete로 변경
- [ ] 세션에 토큰/비용 데이터 기록
- [ ] 포트에 세션 연결 및 통계 기록

---

<!-- pal:port:status=running -->
