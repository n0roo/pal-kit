# PAL Kit

> Personal Agentic Layer - Claude Code 에이전트 오케스트레이션 도구

## 개요

PAL Kit은 Claude Code와 함께 사용하여 복잡한 작업을 체계적으로 관리하는 CLI 도구입니다.

## 주요 기능

| 기능 | 설명 |
|------|------|
| **세션 관리** | 에이전트 세션 추적, 계층 구조 (builder → sub) |
| **포트 관리** | 작업 단위 정의, 상태 추적, 배타적 소유권 |
| **파이프라인** | 포트 의존성 관리, 실행 그룹화 |
| **Lock** | 리소스 충돌 방지, 동시성 제어 |
| **Rules 연동** | `.claude/rules/` 동적 생성으로 조건부 컨텍스트 |
| **에스컬레이션** | 상위 에이전트로 이슈 전달 |
| **컨텍스트 관리** | CLAUDE.md 동적 업데이트 |

## 설치

```bash
cd ~/playground/CodeSpace/pal-kit
go build -o pal ./cmd/pal

# 전역 사용 (선택)
sudo ln -s $(pwd)/pal /usr/local/bin/pal
```

## 빠른 시작

```bash
# 프로젝트 초기화
cd your-project
pal init

# 포트 생성
pal port create port-001 --title "Order Entity"
pal port create port-002 --title "Order Service"

# 파이프라인 구성
pal pipeline create feature-order "Order Feature"
pal pl add feature-order port-001 --group 0
pal pl add feature-order port-002 --group 1 --after port-001

# 작업 시작
pal port activate port-001
pal lock acquire domain
pal port status port-001 running

# 상태 확인
pal status
```

## 커맨드 레퍼런스

### 초기화

```bash
pal init [--force]     # 프로젝트 초기화
```

### 세션 관리

```bash
pal session start [--type TYPE] [--parent ID] [--port PORT] [--title TITLE]
pal session end [ID]
pal session list [--active] [--limit N]
pal session show <ID>
pal session tree [ID]    # 세션 계층 트리 조회
```

**세션 유형:**
- `single` - 단일 세션 (기본)
- `multi` - 멀티 세션 (병렬 독립)
- `sub` - 서브 세션 (상위에서 spawn)
- `builder` - 빌더 세션 (파이프라인 관리)

### 포트 관리

```bash
pal port create <ID> [--title TITLE] [--file PATH]
pal port list [--status STATUS] [--limit N]
pal port show <ID>
pal port status <ID> <STATUS>   # pending|running|complete|failed|blocked
pal port delete <ID>
pal port summary

# Rules 연동
pal port activate <ID> [--path PATTERN...]   # .claude/rules/ 생성
pal port deactivate <ID>                     # .claude/rules/ 삭제
pal port rules                               # 활성 규칙 목록
```

### 파이프라인

```bash
pal pipeline create <ID> [NAME]
pal pl add <PIPELINE> <PORT> [--group N] [--after PORT]
pal pl list [--status STATUS]
pal pl show <ID>           # 트리뷰
pal pl status <ID> [STATUS]
pal pl delete <ID>
```

### Lock

```bash
pal lock acquire <RESOURCE> [--session ID]
pal lock release <RESOURCE>
pal lock list
pal lock check <RESOURCE>
```

### 에스컬레이션

```bash
pal esc create --issue "문제 설명" [--session ID] [--port ID]
pal esc list [--status STATUS]
pal esc show <ID>
pal esc resolve <ID>
pal esc dismiss <ID>
pal esc summary
```

### 컨텍스트

```bash
pal ctx show              # 현재 컨텍스트 출력
pal ctx inject [--file]   # CLAUDE.md에 컨텍스트 주입
pal ctx for-port <ID>     # 포트 기반 컨텍스트 생성
```

### 통합 상태

```bash
pal status   # 대시보드 (세션, 포트, 파이프라인, Lock, 에스컬레이션)
```

### 템플릿

```bash
pal template list
pal template create <TYPE> --id <ID> [--title TITLE]
pal template show <TYPE>
```

### 사용량

```bash
pal usage [--session ID]
```

## 디렉토리 구조

```
your-project/
├── CLAUDE.md              # 프로젝트 컨텍스트 (동적 섹션 포함)
├── .claude/
│   ├── pal.db            # SQLite 데이터베이스
│   ├── agents/           # 에이전트 프롬프트
│   ├── rules/            # 조건부 규칙 (동적 생성)
│   ├── hooks/            # Claude Code Hook 스크립트
│   └── state/            # 상태 디렉토리
└── ports/                 # 포트 명세 문서
    ├── port-001.md
    └── port-002.md
```

## 워크플로우 예시

### 기본 워크플로우

```bash
# 1. 초기화
pal init

# 2. 포트 정의
pal port create entity-order --title "Order Entity 구현"

# 3. 작업 시작
pal port activate entity-order
pal lock acquire domain
pal port status entity-order running

# 4. 작업 수행 (Claude Code에서)
# ...

# 5. 완료
pal port status entity-order complete
pal lock release domain
pal port deactivate entity-order
```

### 파이프라인 워크플로우

```bash
# 빌더 세션에서 파이프라인 구성
pal session start --type builder --title "Feature X"
pal pipeline create feature-x

# 포트 추가 (의존성 포함)
pal pl add feature-x port-001 --group 0
pal pl add feature-x port-002 --group 1 --after port-001
pal pl add feature-x port-003 --group 1 --after port-001
pal pl add feature-x port-004 --group 2 --after port-002 --after port-003

# 상태 확인
pal pl show feature-x
```

### 계층적 세션

```bash
# 빌더 세션
pal session start --type builder --title "Feature Builder"
# → ID: builder-123

# 서브 세션들
pal session start --type sub --parent builder-123 --port port-001 --title "Entity Work"
pal session start --type sub --parent builder-123 --port port-002 --title "Service Work"

# 트리 조회
pal session tree
```

## Hook 연동

`.claude/hooks/session-start` 예시:

```bash
#!/bin/bash
pal hook session-start
pal ctx inject
```

## 환경 변수

| 변수 | 설명 |
|------|------|
| `CLAUDE_SESSION_ID` | 현재 Claude Code 세션 ID |
| `CLAUDE_PROJECT_DIR` | 프로젝트 루트 디렉토리 |

## 라이선스

MIT
