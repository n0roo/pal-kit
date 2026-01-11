# PAL Kit

> Personal Agentic Layer - Claude Code 에이전트 오케스트레이션 CLI 도구

## 프로젝트 개요

PAL Kit은 Claude Code와 함께 사용하여 복잡한 작업을 체계적으로 관리하는 CLI 도구입니다.

- **버전**: v0.3.0
- **언어**: Go 1.22
- **DB**: SQLite (`~/.pal/pal.db`)
- **CLI**: Cobra

## 핵심 기능

| 기능 | 명령어 |
|------|--------|
| 전역 설치 | `pal install` |
| 설치 확인 | `pal doctor` |
| 프로젝트 초기화 | `pal init` |
| 세션 관리 | `pal session` |
| 포트 관리 | `pal port` |
| 파이프라인 | `pal pipeline` |
| 대시보드 | `pal serve` |
| Hook | `pal hook` |

## 디렉토리 구조

```
.
├── cmd/pal/           # 메인 엔트리포인트
├── internal/
│   ├── cli/           # CLI 명령어 (Cobra)
│   ├── config/        # 경로 설정
│   ├── db/            # DB 스키마 (v4)
│   ├── session/       # 세션 서비스
│   ├── port/          # 포트 서비스
│   ├── pipeline/      # 파이프라인 서비스
│   ├── server/        # WebUI API
│   ├── usage/         # 토큰 추적
│   └── ...
├── agents/            # 프로젝트 에이전트
├── conventions/       # 프로젝트 컨벤션
├── ports/             # 포트 명세
└── .claude/           # Claude Code 설정
    └── settings.json  # Hook 설정
```

## 빌드 & 테스트

```bash
# 빌드
go build -o pal ./cmd/pal

# 테스트
go test ./...

# 도움말
./pal --help
```

## 개발 규칙

### CLI 명령어 추가
1. `internal/cli/{command}.go` 생성
2. Cobra 명령어 정의
3. `rootCmd.AddCommand()` 등록
4. `--json` 플래그 지원

### 서비스 추가
1. `internal/{domain}/{domain}.go` 생성
2. `Service` 구조체 + `NewService` 생성자
3. DB 의존성 주입

### DB 스키마 변경
1. `internal/db/db.go` 수정
2. `schemaVersion` 상수 증가
3. `schemaVX` 상수로 새 테이블 정의
4. `migrate()` 함수에 ALTER 추가

## 컨벤션

- [Go 스타일](conventions/go-style.yaml)
- [PAL Kit 패턴](conventions/pal-kit.yaml)

## 에이전트

- [PAL Developer](agents/pal-developer.yaml) - 개발 작업용

## 자주 쓰는 명령어

```bash
./pal doctor              # 설치 상태 확인
./pal status              # 통합 상태 조회
./pal serve               # 대시보드 (localhost:8080)
./pal session list        # 세션 목록
./pal port list           # 포트 목록
```

<!-- pal:context:start -->
<!-- PAL Kit이 자동으로 업데이트합니다 -->
<!-- pal:context:end -->
