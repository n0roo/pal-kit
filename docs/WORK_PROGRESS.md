# PAL Kit 작업 명세서
> 작성일: 2026-01-23
> 세션: DuckDB 전환 및 GUI 수정

---

## 1. 완료된 작업

### 1.1 DuckDB 통합 (90% 완료)

#### 생성된 파일
| 파일 | 상태 | 설명 |
|------|------|------|
| `internal/db/duckdb.go` | ✅ 완료 | DuckDB 드라이버, 스키마 (now() 문법 적용) |
| `internal/db/migrate.go` | ✅ 완료 | SQLite → DuckDB 마이그레이션 도구 |
| `internal/db/interface.go` | ✅ 완료 | 통합 인터페이스 (`Database`) |
| `internal/cli/migrate.go` | ✅ 완료 | CLI 마이그레이션 명령어 |

#### CLI 명령어
```bash
# 상태 확인
pal migrate status

# SQLite → DuckDB 마이그레이션
pal migrate to-duckdb [--force] [--source <path>]

# DuckDB 사용 활성화
export PAL_DB_TYPE=duckdb
```

#### 스키마 특징
- SQLite v10 스키마와 완전 호환
- `CURRENT_TIMESTAMP` → `now()` 변환 완료
- 분석 쿼리 성능 향상
- 32개 테이블 마이그레이션 지원

---

### 1.2 Electron GUI 수정 (완료)

#### 수정된 파일
| 파일 | 변경 내용 |
|------|----------|
| `electron-gui/src/pages/Sessions.tsx` | `normalizeSession()` 함수 추가 - API 응답 구조 정규화 |
| `electron-gui/src/pages/Attention.tsx` | `normalizeSession()` 함수 추가 - API 응답 구조 정규화 |
| `electron-gui/src/hooks/useApi.ts` | 웹/Electron 양쪽 지원 (프록시 모드) |
| `electron-gui/postcss.config.js` | 손상 복구 (`npm run dev` 문자열 제거) |
| `electron-gui/package.json` | 원복 완료 |
| `electron-gui/vite.config.ts` | 원복 완료 |

#### 무한루프 원인 해결
- **원인**: 9000번 포트에서 `pal serve`가 이미 실행 중일 때 Electron이 서버 시작 실패 → 재시도 무한 루프
- **해결**: Electron 실행 전 `pkill -f "pal serve"` 또는 기존 서버 사용

---

### 1.3 이전 세션 작업 (완료)

| 작업 | 파일 | 설명 |
|------|------|------|
| API 수정 | `internal/session/hierarchy.go` | `GetRootHierarchicalSessions()` 함수 추가 |
| NULL 처리 | `internal/session/hierarchy.go` | `COALESCE()` 적용 |
| API 연결 | `internal/server/api_v2.go` | `handleSessionHierarchy` 수정 |
| 테스트 데이터 | `scripts/test-hierarchy.sh` | 계층 구조 샘플 데이터 생성 스크립트 |

---

## 2. 잔여 작업

### 2.1 DuckDB 마이그레이션 완료 (즉시)

```bash
# 1. 빌드
cd ~/playground/CodeSpace/pal-kit
go build -o pal ./cmd/pal

# 2. 마이그레이션 실행
./pal migrate to-duckdb --force

# 3. 결과 확인
./pal migrate status

# 4. DuckDB 활성화 (선택)
export PAL_DB_TYPE=duckdb
```

**예상 결과:**
- 32개 테이블 마이그레이션
- `~/.pal/pal.duckdb` 파일 생성
- SQLite 백업 유지

---

### 2.2 GUI 데이터 연동 확인 (즉시)

```bash
# 1. PAL 서버 실행 (한 터미널에서)
cd ~/playground/CodeSpace/pal-kit
./pal serve

# 2. GUI 실행 (다른 터미널에서)
cd ~/playground/CodeSpace/pal-kit/electron-gui
npm run dev

# 브라우저에서 http://localhost:5173 접속
```

**확인 포인트:**
- [ ] Dashboard: 상태 카드 표시
- [ ] Sessions: 계층 트리 표시
- [ ] Attention: 토큰 게이지 표시
- [ ] Orchestrations: 진행률 표시
- [ ] Agents: 에이전트 목록 표시

---

### 2.3 v1.0 재설계 추가 기능 (우선순위별)

#### Phase 1: Core 강화 (높음)
| 기능 | 설명 | 예상 시간 |
|------|------|----------|
| Compact 이벤트 추적 | `compact_events` 테이블 자동 기록 | 2h |
| 체크포인트 자동 저장 | 80% 토큰 도달 시 스냅샷 | 3h |
| 세션 상태 머신 | running → blocked → complete | 2h |

#### Phase 2: Worker 개선 (중간)
| 기능 | 설명 | 예상 시간 |
|------|------|----------|
| Worker-Test 페어링 | 구현↔테스트 자동 연결 | 3h |
| 의존성 그래프 | 포트 간 의존성 시각화 | 4h |
| Handoff 프로토콜 | 세션 간 컨텍스트 전달 | 4h |

#### Phase 3: 실시간 기능 (중간)
| 기능 | 설명 | 예상 시간 |
|------|------|----------|
| SSE 이벤트 스트림 | `/api/v2/events/stream` | 3h |
| Compact Alert | 컴팩션 발생 즉시 알림 | 2h |
| 진행률 업데이트 | 오케스트레이션 진행 상황 | 2h |

#### Phase 4: GUI 고도화 (낮음)
| 기능 | 설명 | 예상 시간 |
|------|------|----------|
| SessionTree 개선 | 드래그앤드롭 | 4h |
| AttentionGauge 개선 | 실시간 업데이트 | 2h |
| CompactAlert 컴포넌트 | 경고 팝업 | 2h |
| DependencyGraph | D3.js 시각화 | 6h |

---

## 3. 현재 프로젝트 상태

### 파일 구조
```
pal-kit/
├── cmd/pal/main.go
├── internal/
│   ├── db/
│   │   ├── db.go           # SQLite (기존)
│   │   ├── duckdb.go       # DuckDB (신규) ✅
│   │   ├── migrate.go      # 마이그레이션 (신규) ✅
│   │   └── interface.go    # 통합 인터페이스 (신규) ✅
│   ├── cli/
│   │   └── migrate.go      # CLI 명령어 (신규) ✅
│   ├── session/
│   │   └── hierarchy.go    # 계층 API ✅
│   ├── agentv2/
│   │   └── agent.go        # 에이전트 v2 ✅
│   └── ...
├── electron-gui/
│   ├── src/
│   │   ├── pages/
│   │   │   ├── Sessions.tsx   # 수정됨 ✅
│   │   │   └── Attention.tsx  # 수정됨 ✅
│   │   └── hooks/
│   │       └── useApi.ts      # 수정됨 ✅
│   └── ...
└── scripts/
    └── test-hierarchy.sh   # 테스트 데이터 ✅
```

### 의존성
```
go.mod:
  github.com/marcboeker/go-duckdb/v2  # DuckDB 드라이버 (이미 추가됨)
  github.com/mattn/go-sqlite3         # SQLite (기존)
```

### 환경 변수
```bash
PAL_DB_TYPE=duckdb    # DuckDB 사용 시
PAL_DB_TYPE=sqlite    # SQLite 사용 시 (기본값)
```

---

## 4. 트러블슈팅

### DuckDB 마이그레이션 오류
- **증상**: `CURRENT_TIMESTAMP` 컬럼 오류
- **원인**: DuckDB는 `now()` 사용
- **해결**: `internal/db/duckdb.go`에서 `DEFAULT now()` 적용 완료

### Electron 무한 루프
- **증상**: `npm run dev` 시 무한 리빌드
- **원인**: 9000번 포트 충돌
- **해결**: `pkill -f "pal serve"` 후 실행

### API 데이터 구조 불일치
- **증상**: `Cannot read properties of undefined (reading 'title')`
- **원인**: API 응답이 `{session: {...}}` 또는 `{id, title, ...}` 두 형태
- **해결**: `normalizeSession()` 함수로 정규화

---

## 5. 다음 세션 시작 명령어

```bash
# 1. 프로젝트 이동
cd ~/playground/CodeSpace/pal-kit

# 2. 빌드
go build -o pal ./cmd/pal

# 3. 마이그레이션 완료 (아직 안했다면)
./pal migrate to-duckdb --force

# 4. 상태 확인
./pal migrate status

# 5. GUI 테스트
./pal serve &
cd electron-gui && npm run dev
```

---

## 6. 참고 문서

- 이전 트랜스크립트: `/mnt/transcripts/2026-01-22-15-34-45-pal-kit-hierarchy-api-fix.txt`
- 프로젝트 README: `~/playground/CodeSpace/pal-kit/README.md`
- API 명세: `~/playground/CodeSpace/pal-kit/docs/api-v2.md`

---

*작성: Claude (Opus 4.5)*
*마지막 업데이트: 2026-01-23*
