# Spring MSA Spec Skill

> Spring Cloud Microservices 명세 스킬

---

## 도메인 특성

### 핵심 개념

Spring Cloud 기반 마이크로서비스 아키텍처입니다.

```
API Gateway
    ↓
┌─────────────────────────────────────┐
│  Service A  │  Service B  │  Service C  │
│  (Domain)   │  (Domain)   │  (Domain)   │
└─────────────────────────────────────┘
    ↓               ↓               ↓
    └───────── Message Broker ───────┘
```

### 서비스 유형

| 유형 | 역할 | 기술 |
|------|------|------|
| API Gateway | 라우팅, 인증 | Spring Cloud Gateway |
| Domain Service | 도메인 로직 | Spring Boot |
| Config Server | 설정 중앙화 | Spring Cloud Config |
| Discovery | 서비스 등록/발견 | Eureka, Consul |

### 통신 패턴

| 패턴 | 용도 | 기술 |
|------|------|------|
| Sync | 즉시 응답 필요 | REST, gRPC |
| Async | 이벤트 기반 | Kafka, RabbitMQ |
| Saga | 분산 트랜잭션 | Choreography/Orchestration |

---

## 템플릿

### Domain Service Port

```yaml
---
type: port
layer: domain-service
domain: {domain}
title: "{Domain} Service"
priority: {priority}
dependencies: []
---

# {Domain} Service Port

## 목표
{Domain} 도메인의 마이크로서비스 구현

## 범위
- 도메인 로직
- REST API
- 이벤트 발행/구독

## API 설계

### REST Endpoints
| Method | Path | 설명 |
|--------|------|------|
| POST | /api/v1/{domain} | 생성 |
| GET | /api/v1/{domain}/{id} | 조회 |
| PUT | /api/v1/{domain}/{id} | 수정 |
| DELETE | /api/v1/{domain}/{id} | 삭제 |

### Request/Response
```json
// POST /api/v1/{domain}
{
  "field1": "value1",
  "field2": "value2"
}

// Response
{
  "id": "uuid",
  "field1": "value1",
  "field2": "value2",
  "createdAt": "2024-01-01T00:00:00Z"
}
```

## 이벤트 설계

### 발행 이벤트
| 이벤트 | 토픽 | 페이로드 |
|--------|------|----------|
| {Domain}Created | {domain}.created | {payload} |
| {Domain}Updated | {domain}.updated | {payload} |

### 구독 이벤트
| 이벤트 | 토픽 | 핸들러 |
|--------|------|--------|
| OtherEvent | other.topic | handleOtherEvent() |

## 의존성
- DB: PostgreSQL
- Message: Kafka
- Cache: Redis (선택)

## 검증 규칙
- [ ] 12-Factor App 준수
- [ ] API 버저닝 적용
- [ ] 이벤트 스키마 정의됨
```

### API Gateway Port

```yaml
---
type: port
layer: gateway
title: "API Gateway"
priority: critical
dependencies: []
---

# API Gateway Port

## 목표
API 라우팅, 인증, Rate Limiting 구현

## 범위
- 라우팅 규칙
- 인증/인가
- Rate Limiting
- Circuit Breaker

## 라우팅 규칙

| Path | Service | 설명 |
|------|---------|------|
| /api/v1/users/** | user-service | 사용자 서비스 |
| /api/v1/orders/** | order-service | 주문 서비스 |

## 인증

### JWT 검증
- Issuer: auth-service
- Audience: api-gateway
- Claims: userId, roles

### 인가 규칙
| Path | Roles |
|------|-------|
| /api/v1/admin/** | ADMIN |
| /api/v1/users/** | USER, ADMIN |

## Rate Limiting
| Path | Limit |
|------|-------|
| /api/v1/** | 100 req/min |
| /api/v1/auth/** | 10 req/min |

## 검증 규칙
- [ ] 모든 경로 라우팅 정의됨
- [ ] JWT 검증 설정됨
- [ ] Rate Limit 설정됨
- [ ] Circuit Breaker 설정됨
```

### Event-Driven Port

```yaml
---
type: port
layer: event
domain: {domain}
title: "{Event} Event Handler"
priority: {priority}
dependencies: [service-xxx]
---

# {Event} Event Handler Port

## 목표
{Event} 이벤트 처리 로직 구현

## 범위
- 이벤트 구독
- 이벤트 처리
- 보상 트랜잭션 (필요시)

## 이벤트 스키마

### {Event}Event
```json
{
  "eventId": "uuid",
  "eventType": "{Event}Created",
  "timestamp": "2024-01-01T00:00:00Z",
  "payload": {
    "id": "uuid",
    "data": {}
  }
}
```

## 처리 로직

1. 이벤트 수신
2. 유효성 검증
3. 비즈니스 로직 실행
4. 결과 이벤트 발행 (필요시)

## 에러 처리

| 에러 유형 | 전략 |
|----------|------|
| 일시적 오류 | Retry (지수 백오프) |
| 영구 오류 | Dead Letter Queue |
| 비즈니스 오류 | 보상 이벤트 발행 |

## 검증 규칙
- [ ] 멱등성 보장
- [ ] 재시도 정책 정의됨
- [ ] DLQ 설정됨
```

---

## 컨벤션

### 프로젝트 구조

```
{service-name}/
├── src/main/java/com/example/{service}/
│   ├── api/                # REST Controller
│   ├── application/        # Application Service
│   ├── domain/             # Domain Entity, Repository
│   ├── infrastructure/     # 외부 연동
│   └── event/              # Event Handler
├── src/main/resources/
│   ├── application.yml
│   └── bootstrap.yml
└── src/test/
```

### 네이밍 규칙

| 구성요소 | 패턴 | 예시 |
|----------|------|------|
| Service | {Domain}Service | UserService |
| Controller | {Domain}Controller | UserController |
| Repository | {Domain}Repository | UserRepository |
| Event | {Domain}{Action}Event | UserCreatedEvent |
| Handler | {Event}EventHandler | UserCreatedEventHandler |

### 설정 관리

```yaml
# application.yml
spring:
  application:
    name: {service-name}
  cloud:
    config:
      uri: http://config-server:8888
  kafka:
    bootstrap-servers: ${KAFKA_SERVERS}
```

---

## 검증 기준

### 12-Factor App
- [ ] 설정의 환경 분리
- [ ] 로그의 스트림 처리
- [ ] 프로세스의 Stateless
- [ ] 포트 바인딩

### API 설계
- [ ] RESTful 원칙 준수
- [ ] 버저닝 적용 (/api/v1/...)
- [ ] 에러 응답 표준화
- [ ] OpenAPI 스펙 정의

### 이벤트 설계
- [ ] 스키마 버저닝
- [ ] 멱등성 보장
- [ ] 재시도 정책

---

## 참조 문서

- Spring Cloud 공식 문서
- MSA 패턴 (microservices.io)
- 12-Factor App (12factor.net)
