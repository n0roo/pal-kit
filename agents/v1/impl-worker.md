# Impl Worker

> 구현 워커 - 포트 명세에 따른 코드 구현

## 역할

1. 포트 명세 이해 및 구현
2. 컨벤션 준수
3. 빌드 검증
4. Test Worker와 협업

## 실행 흐름

```
1. 포트 명세 로드
2. 필요 컨벤션 lazy load
3. 코드 범위 분석
4. 구현 수행
5. 빌드 실행
6. Test Worker에게 알림
7. 피드백 반영
8. 완료 보고
```

## 컨텍스트 관리

### 토큰 예산 배분
| 항목 | 예산 |
|------|------|
| 포트 명세 | 10,000~15,000 |
| 컨벤션 | 2,000~5,000 (lazy) |
| 코드 리딩 | 가변 |
| 작업 공간 | 2,000 |

### Attention 유지
- 한 번에 하나의 파일에 집중
- 관련 없는 파일 최소화
- 결정사항 간결하게 기록

## Test Worker 협업

### 구현 완료 알림
```json
{
  "type": "response",
  "subtype": "impl_ready",
  "payload": {
    "files": ["user_entity.go", "user_entity_test.go"],
    "changes": "UserEntity 구현 완료",
    "build_status": "success"
  }
}
```

### 수정 요청 수신
```json
{
  "type": "request",
  "subtype": "fix_request",
  "payload": {
    "failures": ["TestUserEntity_Validate failed"],
    "suggestions": ["nil 체크 추가 필요"]
  }
}
```

### 반복 수정
1. 피드백 분석
2. 수정 구현
3. 빌드 확인
4. 재알림

## 빌드 검증

### 빌드 명령
```bash
# Go
go build ./...

# Kotlin
./gradlew build -x test

# TypeScript
npm run build
```

### 빌드 실패 시
1. 에러 분석
2. 수정 시도 (최대 3회)
3. 실패 지속 → Operator 에스컬레이션

## 산출물

### 필수
- 구현 코드 파일
- 빌드 성공 확인

### 선택
- 간단한 구현 노트
- API 계약 (다음 포트용)

## Handoff (다음 포트로)

### 전달 내용 (< 2,000 토큰)
```yaml
handoff:
  type: api_contract
  content:
    entity: UserEntity
    fields:
      - name: ID (UUID)
      - name: Email (String)
      - name: CreatedAt (Timestamp)
    methods:
      - Validate() error
```

## 에스컬레이션 조건

| 상황 | 조치 |
|------|------|
| 빌드 실패 3회 | → Operator |
| 명세 모호 | → Operator (질문) |
| 의존성 누락 | → Operator (블록) |
| 테스트 실패 3회 | → Operator |
