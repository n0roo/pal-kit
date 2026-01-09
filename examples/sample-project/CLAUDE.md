# Sample Project

> PAL 기능 테스트를 위한 샘플 프로젝트

## 프로젝트 개요

가상의 주문 관리 시스템을 구현하는 예제입니다.

## 아키텍처

```
L1 (Domain)
├─ Order (Entity)
├─ OrderItem (Entity)
└─ OrderStatus (Enum)

L2 (Application)
├─ OrderService
└─ OrderRepository (Interface)

API
└─ OrderController
```

## 컨벤션

### 네이밍
- Entity: PascalCase
- Service: `{Domain}Service`
- Repository: `{Domain}Repository`

### 레이어 규칙
- L2는 L1만 참조 가능
- API는 L2만 참조 가능
- Repository 직접 참조 금지 (L2에서)

## 현재 작업 컨텍스트

<!-- pal:context:start -->
> 이 섹션은 `pal context inject`에 의해 동적으로 업데이트됩니다.

### 활성 작업
- 없음

### 파이프라인 진행
- 없음

### 에스컬레이션
- 없음
<!-- pal:context:end -->

## 참고 문서

- 포트 명세: `ports/` 디렉토리
- 에이전트 정의: `.claude/agents/` 디렉토리
