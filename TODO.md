# PAL Kit TODO

> 개발 로드맵 및 작업 추적

## 완료된 기능 ✅

### Phase 1: 기본 기능 (v0.1.0)
- [x] 프로젝트 초기화 (`pal init`)
- [x] Lock 관리 (`pal lock`)
- [x] 세션 관리 (`pal session`)
- [x] 포트 관리 (`pal port`)
- [x] 사용량 추적 (`pal usage`)
- [x] 컴팩션 기록 (`pal compact`)
- [x] Hook 지원 (`pal hook`)

### Phase 2: 템플릿 & 에스컬레이션 (v0.1.1)
- [x] 템플릿 시스템 (`pal template`)
- [x] 에스컬레이션 관리 (`pal escalation`)

### Phase 3: 파이프라인 & 계층적 세션 (v0.2.0)
- [x] DB 스키마 v2 마이그레이션
- [x] 파이프라인 서비스 (`pal pipeline`)
  - [x] create, add, list, show, status, delete
  - [x] 실행 그룹 (group_order)
  - [x] 의존성 관리 (--after)
  - [x] 트리뷰 출력
- [x] 세션 계층
  - [x] session_type (single, multi, sub, builder)
  - [x] parent_session 관계
  - [x] `pal session tree` 트리 조회
- [x] `.claude/rules/` 연동
  - [x] `pal port activate` - 조건부 규칙 생성
  - [x] `pal port deactivate` - 규칙 삭제
  - [x] `pal port rules` - 활성 규칙 목록
- [x] 컨텍스트 관리 (`pal context`)
  - [x] CLAUDE.md 동적 주입
  - [x] 포트 기반 컨텍스트 생성
- [x] 통합 상태 대시보드 (`pal status`)

### Phase 4: 파이프라인 실행 (v0.2.1)
- [x] `pal pl plan` - 실행 계획 조회
- [x] `pal pl next` - 다음 실행 가능 포트
- [x] `pal pl run` - 실행 스크립트 생성
- [x] `pal pl run --tmux` - tmux 병렬 스크립트
- [x] `pal pl port-status` - 파이프라인 내 포트 상태

### Phase 5: Hook 고도화 (v0.2.2)
- [x] `pal hook port-start` - 포트 시작 (rules + running)
- [x] `pal hook port-end` - 포트 완료 (rules 제거)
- [x] `pal hook sync` - rules ↔ running 동기화
- [x] `pal hook session-start --port` - 세션+포트 동시 시작
- [x] settings.json에 SessionEnd Hook 추가
- [x] init 시 rules 디렉토리 생성

---

## 예정된 기능 📋

### Phase 6: 자동화 강화 (v0.3.0)
- [ ] `pal pl exec <id>` - 실제 파이프라인 실행 (subprocess)
- [ ] `pal pl watch <id>` - 파이프라인 모니터링
- [ ] 포트 타임아웃 설정
- [ ] 실패 시 자동 재시도

### Phase 7: 에이전트 통합 (v0.4.0)
- [ ] 에이전트 프롬프트 로딩
- [ ] 에이전트별 세션 분리
- [ ] 에이전트 간 메시지 전달
- [ ] builder 에이전트 자동 실행

### 개선 사항
- [ ] `pal port show` 포트 명세 내용 표시
- [ ] `pal status --pipeline <id>` 파이프라인 중심 뷰
- [ ] `pal status --watch` 실시간 갱신
- [ ] JSON 스키마 검증
- [ ] 설정 파일 (`pal.yaml`) 지원
- [ ] 테스트 코드 작성
- [ ] bash/zsh 자동완성

---

## 버그 및 기술 부채

- [ ] DB 마이그레이션 롤백 지원
- [ ] 에러 핸들링 일관성
- [ ] 로그 레벨 설정

---

## 문서

- [x] README.md 업데이트
- [x] TODO.md 업데이트
- [ ] 사용 가이드 (USAGE.md)
- [ ] 아키텍처 문서 (ARCHITECTURE.md)
- [ ] 예제 프로젝트 정리

---

## 버전 히스토리

### v0.2.2 (현재)
- Hook 고도화 (port-start, port-end, sync)
- settings.json 개선

### v0.2.1
- 파이프라인 실행 (plan, next, run)
- 파이프라인 내 포트 상태 관리

### v0.2.0
- 파이프라인 관리
- 세션 계층 (builder/sub)
- Rules 연동
- 통합 상태 대시보드

### v0.1.1
- 템플릿 시스템
- 에스컬레이션 관리

### v0.1.0
- 초기 릴리스
- 기본 CLI 기능
