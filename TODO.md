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

---

## 진행 중 🔄

(현재 없음)

---

## 예정된 기능 📋

### Phase 4: 실행 자동화 (v0.3.0)
- [ ] `pal pipeline run <id>` - 파이프라인 실행
- [ ] `pal pipeline wait <id>` - 완료 대기
- [ ] tmux 연동 (병렬 실행)
- [ ] Task 연동 (순차 실행)

### Phase 5: 고급 기능 (v0.4.0)
- [ ] 포트 의존성 자동 분석
- [ ] 포트 명세에서 파일 패턴 자동 추출
- [ ] Hook 자동 설정
- [ ] 세션 복구 (`pal session resume`)

### 개선 사항
- [ ] `pal port show` 포트 명세 내용 표시
- [ ] `pal status --pipeline <id>` 파이프라인 중심 뷰
- [ ] JSON 스키마 검증
- [ ] 설정 파일 (`pal.yaml`) 지원
- [ ] 테스트 코드 작성

---

## 버그 및 기술 부채

- [ ] 외래키 제약 조건 에러 메시지 개선
- [ ] DB 마이그레이션 롤백 지원
- [ ] 에러 핸들링 일관성

---

## 문서

- [x] README.md 업데이트
- [ ] 사용 가이드 (USAGE.md)
- [ ] 아키텍처 문서 (ARCHITECTURE.md)
- [ ] 예제 프로젝트

---

## 버전 히스토리

### v0.2.0 (현재)
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
