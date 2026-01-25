# Electron Spec Skill

> Electron 데스크톱 애플리케이션 명세 스킬

---

## 도메인 특성

### 핵심 개념

Electron은 Main Process와 Renderer Process로 분리된 아키텍처입니다.

```
┌─────────────────────────────────────┐
│           Main Process              │
│  - Node.js 환경                      │
│  - 시스템 API 접근                    │
│  - 창 관리, 메뉴, 트레이              │
└─────────────────────────────────────┘
              ↓ IPC
┌─────────────────────────────────────┐
│         Renderer Process            │
│  - Chromium 환경                     │
│  - UI 렌더링 (React/Vue)             │
│  - 브라우저 API                       │
└─────────────────────────────────────┘
```

### 보안 원칙

| 원칙 | 설명 |
|------|------|
| Context Isolation | Renderer에서 Node.js 직접 접근 차단 |
| Preload Script | 안전한 API만 노출 |
| IPC 통신 | Main-Renderer 간 메시지 기반 통신 |

### IPC 패턴

| 패턴 | 용도 | 방향 |
|------|------|------|
| invoke/handle | 요청-응답 | Renderer → Main |
| send/on | 단방향 메시지 | 양방향 |
| sendSync | 동기 요청 (피해야 함) | Renderer → Main |

---

## 템플릿

### Main Process Port

```yaml
---
type: port
layer: main-process
domain: {domain}
title: "{Feature} Main Process"
priority: {priority}
dependencies: []
---

# {Feature} Main Process Port

## 목표
{Feature} 기능의 Main Process 구현

## 범위
- IPC 핸들러
- 시스템 API 연동
- 파일 시스템 접근

## IPC 핸들러

### invoke/handle

| Channel | 입력 | 출력 | 설명 |
|---------|------|------|------|
| {feature}:get | id: string | Data | 데이터 조회 |
| {feature}:save | data: Data | Result | 데이터 저장 |

### send/on (Main → Renderer)

| Channel | 페이로드 | 설명 |
|---------|----------|------|
| {feature}:updated | data: Data | 데이터 변경 알림 |

## 구현

```typescript
// main/{feature}.ts
import { ipcMain } from 'electron';

export function register{Feature}Handlers() {
  ipcMain.handle('{feature}:get', async (event, id: string) => {
    // 구현
  });

  ipcMain.handle('{feature}:save', async (event, data: Data) => {
    // 구현
  });
}
```

## 시스템 연동

| API | 용도 |
|-----|------|
| fs | 파일 읽기/쓰기 |
| dialog | 파일 선택 다이얼로그 |
| shell | 외부 앱 실행 |

## 검증 규칙
- [ ] 에러 핸들링
- [ ] 입력 검증
- [ ] 경로 정규화 (보안)
```

### Preload Port

```yaml
---
type: port
layer: preload
title: "Preload Script"
priority: critical
dependencies: []
---

# Preload Script Port

## 목표
Renderer에 노출할 안전한 API 정의

## 범위
- Context Bridge API
- 타입 정의
- 입력 검증

## API 정의

```typescript
// preload.ts
import { contextBridge, ipcRenderer } from 'electron';

const api = {
  {feature}: {
    get: (id: string) => ipcRenderer.invoke('{feature}:get', id),
    save: (data: Data) => ipcRenderer.invoke('{feature}:save', data),
    onUpdated: (callback: (data: Data) => void) => {
      ipcRenderer.on('{feature}:updated', (_, data) => callback(data));
    },
  },
};

contextBridge.exposeInMainWorld('electronAPI', api);
```

## 타입 정의

```typescript
// types/electron.d.ts
interface ElectronAPI {
  {feature}: {
    get: (id: string) => Promise<Data>;
    save: (data: Data) => Promise<Result>;
    onUpdated: (callback: (data: Data) => void) => void;
  };
}

declare global {
  interface Window {
    electronAPI: ElectronAPI;
  }
}
```

## 보안 규칙
- [ ] 최소 권한 원칙
- [ ] 입력 검증
- [ ] 경로 접근 제한
```

### Renderer Port

```yaml
---
type: port
layer: renderer
domain: {domain}
title: "{Feature} Renderer"
priority: {priority}
dependencies: [preload]
---

# {Feature} Renderer Port

## 목표
{Feature} UI 구현 (React 기반)

## 범위
- 페이지/컴포넌트
- IPC 호출 Hook
- 상태 관리

## IPC Hook

```typescript
// hooks/use{Feature}.ts
export function use{Feature}() {
  const [data, setData] = useState<Data | null>(null);
  const [loading, setLoading] = useState(false);

  const fetch = async (id: string) => {
    setLoading(true);
    try {
      const result = await window.electronAPI.{feature}.get(id);
      setData(result);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    window.electronAPI.{feature}.onUpdated((newData) => {
      setData(newData);
    });
  }, []);

  return { data, loading, fetch };
}
```

## 페이지 구성

| 페이지 | 설명 |
|--------|------|
| {Feature}Page | 메인 페이지 |
| {Feature}Settings | 설정 |

## 검증 규칙
- [ ] window.electronAPI 타입 체크
- [ ] 로딩/에러 상태 처리
- [ ] 이벤트 리스너 정리
```

---

## 컨벤션

### 프로젝트 구조

```
electron-app/
├── electron/
│   ├── main.ts           # Main Process 진입점
│   ├── preload.ts        # Preload Script
│   └── handlers/         # IPC 핸들러
│       └── {feature}.ts
│
├── src/                  # Renderer (React)
│   ├── App.tsx
│   ├── pages/
│   ├── components/
│   └── hooks/
│       └── use{Feature}.ts
│
└── types/
    └── electron.d.ts     # Electron API 타입
```

### IPC 채널 네이밍

```
{도메인}:{액션}

예시:
- file:open
- file:save
- settings:get
- settings:set
- app:quit
```

### 보안 체크리스트

- [ ] contextIsolation: true
- [ ] nodeIntegration: false
- [ ] webSecurity: true
- [ ] allowRunningInsecureContent: false

---

## 검증 기준

### Main Process
- [ ] IPC 핸들러 등록
- [ ] 에러 전파
- [ ] 입력 검증

### Preload
- [ ] 최소 API 노출
- [ ] 타입 정의

### Renderer
- [ ] API 호출 추상화
- [ ] 이벤트 정리
