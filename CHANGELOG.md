# Changelog

All notable changes to PAL Kit will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-01-12

### Added

#### 워크플로우 시스템
- 4가지 워크플로우 타입 지원: `simple`, `single`, `integrate`, `multi`
- 워크플로우별 자동 에이전트 구성
- `.claude/rules/workflow.md` 자동 생성 (세션 시작/종료 시)
- `pal workflow show|set|context|refresh` 명령어

#### 프로젝트 설정
- `.pal/config.yaml` 설정 파일 도입
- `pal config show|init|set|get` 명령어
- 프로젝트별 워크플로우, 에이전트, 설정 관리

#### 에이전트 템플릿 시스템
- 12개 기본 에이전트 템플릿 (embed.FS 기반)
  - Core (7개): collaborator, builder, planner, architect, manager, tester, logger
  - Workers/Backend (3개): go, kotlin, nestjs
  - Workers/Frontend (2개): react, next
- `pal agent templates` - 템플릿 목록 조회
- `pal agent add <template>` - 프로젝트에 에이전트 추가
- `pal install` 시 `~/.pal/agents/`에 템플릿 설치

#### Manifest 시스템
- 파일 변경 추적 (`file_manifests`, `file_changes` 테이블)
- `.pal/manifest.yaml` 파일 기반 관리
- `pal manifest status|sync|add|remove|history` 명령어
- SHA256 해시 기반 변경 감지
- Quick check (mtime 기반 빠른 감지)

#### Claude 설정 플로우
- `pal analyze` - 프로젝트 분석 (기술 스택, 구조, 규모)
- `pal setup` - 대화형 설정 (--auto, --yes 옵션)
- 기술 스택 기반 워커 에이전트 자동 추천
- 설정 완료 후 CLAUDE.md 자동 업데이트

#### Hook 연동
- SessionStart: 워크플로우 컨텍스트 자동 주입
- SessionEnd: rules 파일 자동 정리
- 에이전트/포트 활성화 시 컨텍스트 갱신

### Changed

- CLAUDE.md 템플릿 개선 (설정 플로우 가이드 포함)
- DB 스키마 v5 (manifest 테이블 추가)

### Technical

- 52개 테스트 케이스 추가
- `internal/config/project.go` - 프로젝트 설정 관리
- `internal/workflow/workflow.go` - 워크플로우 서비스
- `internal/manifest/manifest.go` - Manifest 서비스
- `internal/agent/embed.go` - 에이전트 템플릿 embed
- `internal/docs/update.go` - CLAUDE.md 업데이트

---

## [0.3.0] - 2026-01-10

### Added

- 전역 설치 지원 (`pal install`)
- 에이전트 시스템 기본 구조
- 컨벤션 관리
- 파이프라인 실행기
- TUI 대시보드

### Changed

- CLI 구조 개선
- DB 스키마 업데이트

---

## [0.2.0] - 2026-01-08

### Added

- 포트 기반 작업 관리
- 세션 관리
- Lock 시스템
- 에스컬레이션

---

## [0.1.0] - 2026-01-05

### Added

- 초기 릴리스
- CLAUDE.md 컨텍스트 주입
- 기본 CLI 구조
- SQLite DB
