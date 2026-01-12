# PAL Kit 백업 및 동기화 가이드

> 다중 환경에서 PAL Kit 데이터를 안전하게 백업하고 동기화하는 방법

---

## 개요

PAL Kit은 Git 기반 동기화 시스템을 제공하여 여러 환경(집, 회사 등)에서 작업 내역을 공유할 수 있습니다.

### 동기화 대상 데이터

| 데이터 | 동기화 | 비고 |
|--------|--------|------|
| 포트 (Ports) | O | 작업 명세 및 상태 |
| 세션 메타 (Sessions) | O | 토큰, 비용, 상태 등 |
| 에스컬레이션 (Escalations) | O | 이슈 추적 |
| 파이프라인 (Pipelines) | O | 워크플로우 |
| 프로젝트 (Projects) | O | 논리 경로로 저장 |
| Claude 로컬 파일 | X | jsonl, transcript (환경별 독립) |

---

## 빠른 시작

### 1. 환경 설정

```bash
# 현재 환경 등록
pal env setup --name home-mac

# 환경 확인
pal env current
```

### 2. 동기화 저장소 초기화

```bash
# Git 저장소 초기화 (원격 없이)
pal sync init

# 또는 원격 저장소와 함께
pal sync init git@github.com:username/pal-sync.git
```

### 3. 데이터 백업 (Push)

```bash
# 현재 데이터를 동기화 저장소에 저장
pal sync push

# 커밋 메시지 지정
pal sync push -m "작업 완료 후 백업"
```

### 4. 데이터 복원 (Pull)

```bash
# 동기화 저장소에서 데이터 가져오기
pal sync pull
```

---

## 상세 사용법

### 환경 관리

```bash
# 환경 목록 확인
pal env list

# 환경 전환
pal env switch office-mac

# 환경 자동 감지
pal env detect

# 환경 삭제
pal env delete old-env
```

### 동기화 상태 확인

```bash
# 전체 동기화 상태
pal sync status

# 출력 예시:
# 현재 환경: home-mac (6386c44f)
#
# Git 저장소:
#   디렉토리: /Users/n0roo/.pal/sync
#   원격: git@github.com:user/pal-sync.git
#   브랜치: master
```

### 수동 Export/Import

Git 없이 파일로 직접 백업할 수도 있습니다.

```bash
# YAML 파일로 내보내기
pal sync export backup-2026-01-12.yaml

# YAML 파일에서 가져오기
pal sync import backup-2026-01-12.yaml

# Dry-run (실제 변경 없이 미리보기)
pal sync import backup.yaml --dry-run
```

---

## 다중 환경 설정

### 새 환경 추가 (예: 회사 PC)

```bash
# 1. PAL Kit 설치 후 환경 등록
pal env setup --name office-mac

# 2. 동기화 저장소 연결
pal sync init git@github.com:username/pal-sync.git

# 3. 기존 데이터 가져오기
pal sync pull
```

### 일상 워크플로우

```bash
# 작업 시작 전 - 최신 데이터 동기화
pal sync pull

# 작업 중...

# 작업 종료 후 - 변경사항 백업
pal sync push
```

---

## 충돌 해결

동일한 데이터가 여러 환경에서 수정된 경우 충돌이 발생할 수 있습니다.

### 충돌 확인

```bash
# 충돌 목록 확인
pal sync conflicts

# 로컬과 원격 비교
pal sync diff
```

### 충돌 해결

```bash
# 모든 충돌을 로컬 데이터로 해결
pal sync resolve --all --keep-local

# 모든 충돌을 원격 데이터로 해결
pal sync resolve --all --keep-remote

# 특정 충돌만 해결
pal sync resolve <id> --keep-local

# 충돌 건너뛰기
pal sync resolve --all --skip
```

### Merge 전략

Import 시 충돌 처리 전략을 지정할 수 있습니다.

```bash
# 최신 데이터 우선 (기본값)
pal sync import data.yaml --strategy last_write_wins

# 로컬 데이터 유지
pal sync import data.yaml --strategy keep_local

# 원격 데이터 우선
pal sync import data.yaml --strategy keep_remote

# 충돌 시 건너뛰기
pal sync import data.yaml --skip-conflicts
```

---

## 경로 관리

PAL Kit은 환경 간 경로 호환성을 위해 논리 경로를 사용합니다.

### 경로 변수

| 변수 | 설명 | 예시 |
|------|------|------|
| `$workspace` | 작업 디렉토리 | `/Users/n0roo/playground` |
| `$claude_data` | Claude 데이터 | `/Users/n0roo/.claude` |
| `$home` | 홈 디렉토리 | `/Users/n0roo` |

### 경로 변환

```bash
# 절대 경로 → 논리 경로
pal path to-logical /Users/n0roo/playground/project
# 출력: $workspace/project

# 논리 경로 → 절대 경로
pal path to-absolute '$workspace/project'
# 출력: /Users/n0roo/playground/project

# 경로 분석
pal path analyze /Users/n0roo/playground/project
```

### 기존 데이터 마이그레이션

```bash
# DB의 절대 경로를 논리 경로로 변환
pal path migrate

# Dry-run
pal path migrate --dry-run
```

---

## 백업 위치

### 기본 디렉토리 구조

```
~/.pal/
├── pal.db                    # 로컬 SQLite DB
├── environments.yaml         # 환경 설정 (로컬 전용)
└── sync/                     # 동기화 저장소
    ├── .git/                 # Git 저장소
    ├── pal-data.yaml         # 동기화 데이터
    └── conflicts.yaml        # 충돌 정보 (있는 경우)
```

### 수동 백업

전체 PAL Kit 데이터를 수동으로 백업하려면:

```bash
# DB 파일 백업
cp ~/.pal/pal.db ~/backup/pal-backup-$(date +%Y%m%d).db

# 동기화 디렉토리 백업
tar -czf ~/backup/pal-sync-$(date +%Y%m%d).tar.gz ~/.pal/sync/
```

---

## 문제 해결

### 동기화 실패

```bash
# Git 상태 확인
cd ~/.pal/sync && git status

# 수동으로 충돌 해결 후
cd ~/.pal/sync && git add . && git commit -m "resolve conflicts"
```

### 환경 감지 실패

```bash
# 수동으로 환경 지정
pal env switch <환경이름>

# 환경 재설정
pal env setup --name <새이름> --force
```

### 경로 해석 실패

```bash
# 환경 경로 설정 확인
pal env current -v

# 경로 분석
pal path analyze <문제경로>
```

### DB 손상 시 복구

```bash
# 백업에서 복원
cp ~/backup/pal-backup.db ~/.pal/pal.db

# 또는 동기화에서 복원
pal sync pull --strategy keep_remote
```

---

## 권장 사항

1. **정기 백업**: 중요한 작업 후 `pal sync push` 실행
2. **작업 전 동기화**: 새 환경에서 작업 시작 전 `pal sync pull`
3. **원격 저장소 사용**: 비공개 Git 저장소로 안전하게 백업
4. **환경별 이름 구분**: 명확한 환경 이름 사용 (예: `home-mac`, `office-windows`)

---

## 관련 명령어 요약

| 명령어 | 설명 |
|--------|------|
| `pal env setup` | 환경 등록 |
| `pal env list` | 환경 목록 |
| `pal env current` | 현재 환경 |
| `pal env switch` | 환경 전환 |
| `pal sync init` | 동기화 초기화 |
| `pal sync push` | 데이터 백업 |
| `pal sync pull` | 데이터 복원 |
| `pal sync status` | 동기화 상태 |
| `pal sync export` | YAML 내보내기 |
| `pal sync import` | YAML 가져오기 |
| `pal sync conflicts` | 충돌 목록 |
| `pal sync resolve` | 충돌 해결 |
| `pal sync diff` | 데이터 비교 |
| `pal path to-logical` | 논리 경로 변환 |
| `pal path to-absolute` | 절대 경로 변환 |
| `pal path migrate` | 경로 마이그레이션 |
