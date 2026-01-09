# Port: test-pipeline

> Pipeline 패키지 테스트 코드 작성

## 목표

`internal/pipeline` 패키지의 유닛 테스트 작성

## 범위

- `pipeline.go` - 파이프라인 CRUD, 의존성 관리
- `runner.go` - 실행 계획, 스크립트 생성
- 테스트 파일: `internal/pipeline/pipeline_test.go`

## 테스트 케이스

1. **Create/Get/Delete**: 기본 CRUD
2. **AddPort/GetPorts**: 포트 관리
3. **AddDependency/GetDependencies**: 의존성
4. **CanExecutePort**: 실행 가능 여부
5. **BuildExecutionPlan**: 실행 계획
6. **GetNextPorts**: 다음 실행 가능 포트
7. **GenerateRunScript**: 스크립트 생성

## 의존성

- test-session 완료 필요 (세션 연동 테스트)

## 완료 조건

- [ ] 테스트 파일 생성
- [ ] 모든 테스트 통과
- [ ] `go test ./internal/pipeline/...` 성공
