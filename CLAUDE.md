# pal-kit

> PAL Kit 관리 프로젝트 | 생성일: 2026-01-12

---

## 🚀 PAL Kit 초기 설정 필요

이 프로젝트는 **PAL Kit 환경이 구성되지 않았습니다.**

### Claude에게 요청하세요:

```
이 프로젝트의 PAL Kit 환경을 설정해줘
```

### 설정 과정:

1. **프로젝트 분석** - 기술 스택, 구조 파악
2. **워크플로우 선택** - simple / single / integrate / multi
3. **에이전트 구성** - 워크플로우에 맞는 에이전트 설정
4. **컨벤션 정의** - 프로젝트 규칙 설정

---

## 워크플로우 타입 안내

| 타입 | 설명 | 적합한 경우 |
|------|------|------------|
| **simple** | 대화형 협업, 종합 에이전트 | 간단한 작업, 학습 |
| **single** | 단일 세션, 역할 전환 | 중간 규모 기능 |
| **integrate** | 빌더 관리, 서브세션 | 복잡한 기능, 여러 기술 |
| **multi** | 복수 integrate | 대규모 프로젝트 |

---

## PAL Kit 기본 명령어

```bash
# 상태 확인
pal status

# 포트 관리
pal port list
pal port create <id> --title "작업명"

# 작업 시작/종료
pal hook port-start <id>
pal hook port-end <id>

# 파이프라인
pal pipeline list
pal pl plan <n>

# 대시보드
pal serve
```

---

## 디렉토리 구조

```
.
├── CLAUDE.md           # 이 파일 (프로젝트 컨텍스트)
├── agents/             # 에이전트 정의
├── ports/              # 포트 명세
├── conventions/        # 컨벤션 문서
├── .claude/
│   ├── settings.json   # Claude Code Hook 설정
│   └── rules/          # 조건부 규칙
└── .pal/
    └── config.yaml     # PAL Kit 설정 (설정 후 생성)
```

---

<!-- pal:config:status=pending -->
<!-- 
  PAL Kit 설정 상태: 미완료
  설정 완료 후 이 섹션이 업데이트됩니다.
  
  설정 완료 시 포함될 내용:
  - 선택된 워크플로우 타입
  - 구성된 에이전트 목록
  - 적용된 컨벤션
  - 작업 시작 가이드
-->
