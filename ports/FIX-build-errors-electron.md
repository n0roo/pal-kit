# Port: FIX-build-errors-electron

> 빌드 오류 수정 및 Electron GUI 안정화

---

## 메타데이터

| 항목 | 값 |
|------|-----|
| ID | FIX-build-errors-electron |
| 타입 | atomic |
| 레이어 | FIX (Hotfix) |
| 상태 | complete |
| 우선순위 | critical |
| 의존성 | - |
| 예상 토큰 | 8,000 |

---

## 목표

LM-sse-stream 구현 후 발생한 빌드 오류를 수정하고, Electron GUI의 안정적인 동작을 보장한다.

---

## 범위

### Go 빌드 오류

1. **internal/context/injection.go**
   - `e.Data` undefined - SessionEvent.EventData (string) 사용

2. **internal/recovery/service.go**
   - `port.Port` vs `*port.Port` 타입 불일치
   - `e.Data` undefined - SessionEvent.EventData 사용
   - `LogEvent` 파라미터 타입 불일치 (map → JSON string)

3. **internal/server/websocket.go**
   - `SSEHub` 중복 선언 - 이미 수정됨 (sse.go만 사용)

### Electron GUI 오류

4. **DevTools Fetch 에러**
   - `Failed to fetch` 에러 로깅 (무시 가능 - DevTools 내부 문제)

5. **서버 포트 불일치**
   - useSSE.ts에서 동적 포트 해결 로직 추가
   - Electron main process에서 포트를 가져와 사용

6. **툴바 겹침**
   - App.tsx에서 macOS일 때 상단 여백(pt-7) 추가
   - platform 감지하여 조건부 스타일링

7. **연결 상태 불일치**
   - StatusBar.tsx 언어 통일 (영어 → 한국어)
   - 상단/하단 모두 "연결됨" / "연결 끊김" 표시

---

## 작업 항목

### 1. Go 빌드 오류 수정

- [x] `internal/context/injection.go` - SessionEvent.EventData 사용 + JSON 파싱
- [x] `internal/recovery/service.go` - 타입 불일치 수정 (&p 사용)
- [x] `internal/recovery/service.go` - LogEvent JSON string 파라미터
- [x] `internal/server/websocket.go` - SSEHub 중복 없음 확인

### 2. Electron GUI 수정

- [x] `useSSE.ts` - 동적 포트 해결 (Electron/Web 모두 지원)
- [x] `App.tsx` - macOS 타이틀바 여백 추가 (isMac 조건)
- [x] `StatusBar.tsx` - 연결 상태 표기 한국어 통일

---

## 완료 기준

- [x] Go 빌드 오류 수정 (타입, 필드 접근)
- [x] Electron SSE 포트 동적 해결
- [x] macOS 툴바 겹침 해결
- [x] 연결 상태 표기 일관성

---

## 수정된 파일

| 파일 | 변경 내용 |
|------|----------|
| internal/context/injection.go | e.EventData 사용, JSON 파싱 추가 |
| internal/recovery/service.go | &p 포인터, JSON string LogEvent |
| electron-gui/src/hooks/useSSE.ts | 동적 baseUrl 해결 로직 |
| electron-gui/src/App.tsx | isMac 조건부 패딩 |
| electron-gui/src/components/StatusBar.tsx | 한국어 통일 |

---

<!-- pal:port:FIX-build-errors-electron -->
