# Cloud Infra Spec Skill

> 클라우드 인프라 (IaC, K8s) 명세 스킬

---

## 도메인 특성

### 핵심 개념

Infrastructure as Code 기반 클라우드 인프라 관리입니다.

```
┌─────────────────────────────────────┐
│            IaC Layer                │
│  Terraform / Pulumi / CloudFormation│
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│         Cloud Provider              │
│    AWS / GCP / Azure                │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│         Kubernetes                  │
│    EKS / GKE / AKS                  │
└─────────────────────────────────────┘
```

### IaC 원칙

| 원칙 | 설명 |
|------|------|
| 선언적 | 원하는 상태를 선언 |
| 멱등성 | 여러 번 실행해도 같은 결과 |
| 버전 관리 | 코드로 관리, Git 추적 |
| 모듈화 | 재사용 가능한 모듈 |

### 환경 분리

| 환경 | 용도 | 특성 |
|------|------|------|
| dev | 개발 | 최소 리소스 |
| staging | 테스트 | 프로덕션 유사 |
| prod | 운영 | HA, 보안 강화 |

---

## 템플릿

### Terraform Module Port

```yaml
---
type: port
layer: infra
domain: {domain}
title: "{Resource} Terraform Module"
priority: {priority}
dependencies: []
---

# {Resource} Terraform Module Port

## 목표
{Resource} 인프라 리소스 프로비저닝

## 범위
- 리소스 정의
- 변수/출력 정의
- 환경별 설정

## 리소스 목록

| 리소스 | 타입 | 설명 |
|--------|------|------|
| {resource}_main | aws_{type} | 주요 리소스 |
| {resource}_sg | aws_security_group | 보안 그룹 |

## 변수 (Variables)

| 변수 | 타입 | 기본값 | 설명 |
|------|------|--------|------|
| environment | string | - | 환경 (dev/staging/prod) |
| region | string | ap-northeast-2 | AWS 리전 |
| instance_type | string | t3.micro | 인스턴스 타입 |

## 출력 (Outputs)

| 출력 | 설명 |
|------|------|
| {resource}_id | 리소스 ID |
| {resource}_arn | 리소스 ARN |
| {resource}_endpoint | 엔드포인트 |

## 구현

```hcl
# modules/{resource}/main.tf
resource "aws_{type}" "main" {
  name = "${var.environment}-{resource}"

  tags = {
    Environment = var.environment
    ManagedBy   = "terraform"
  }
}

# modules/{resource}/variables.tf
variable "environment" {
  type        = string
  description = "Environment name"
}

# modules/{resource}/outputs.tf
output "{resource}_id" {
  value = aws_{type}.main.id
}
```

## 검증 규칙
- [ ] terraform fmt 통과
- [ ] terraform validate 통과
- [ ] 태그 정책 준수
- [ ] 보안 그룹 최소 권한
```

### Kubernetes Manifest Port

```yaml
---
type: port
layer: k8s
domain: {domain}
title: "{Service} K8s Manifests"
priority: {priority}
dependencies: [infra-xxx]
---

# {Service} K8s Manifests Port

## 목표
{Service} 서비스의 Kubernetes 배포 정의

## 범위
- Deployment / StatefulSet
- Service
- ConfigMap / Secret
- Ingress

## 리소스 목록

| 리소스 | 종류 | 설명 |
|--------|------|------|
| {service}-deployment | Deployment | 애플리케이션 |
| {service}-service | Service | 내부 통신 |
| {service}-ingress | Ingress | 외부 노출 |
| {service}-configmap | ConfigMap | 설정 |

## Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {service}
  namespace: {namespace}
spec:
  replicas: 2
  selector:
    matchLabels:
      app: {service}
  template:
    metadata:
      labels:
        app: {service}
    spec:
      containers:
        - name: {service}
          image: {image}:{tag}
          ports:
            - containerPort: 8080
          resources:
            requests:
              cpu: 100m
              memory: 128Mi
            limits:
              cpu: 500m
              memory: 512Mi
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
          readinessProbe:
            httpGet:
              path: /ready
              port: 8080
```

## Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {service}
spec:
  selector:
    app: {service}
  ports:
    - port: 80
      targetPort: 8080
```

## 환경별 설정

| 환경 | Replicas | Resources |
|------|----------|-----------|
| dev | 1 | 100m/128Mi |
| staging | 2 | 200m/256Mi |
| prod | 3+ | 500m/512Mi |

## 검증 규칙
- [ ] 리소스 제한 설정
- [ ] Health Check 정의
- [ ] 환경별 분리
- [ ] 네임스페이스 분리
```

### CI/CD Pipeline Port

```yaml
---
type: port
layer: cicd
domain: {domain}
title: "{Service} CI/CD Pipeline"
priority: {priority}
dependencies: [k8s-xxx]
---

# {Service} CI/CD Pipeline Port

## 목표
{Service} 빌드/배포 자동화

## 범위
- 빌드 파이프라인
- 테스트 자동화
- 배포 자동화
- 롤백 전략

## 파이프라인 단계

```
Build → Test → Scan → Push → Deploy
```

| 단계 | 설명 | 도구 |
|------|------|------|
| Build | 이미지 빌드 | Docker |
| Test | 유닛/통합 테스트 | - |
| Scan | 보안 스캔 | Trivy |
| Push | 레지스트리 푸시 | ECR/GCR |
| Deploy | K8s 배포 | ArgoCD/Flux |

## GitHub Actions

```yaml
name: CI/CD

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: docker build -t {image} .
      - name: Test
        run: make test
      - name: Push
        run: docker push {image}

  deploy:
    needs: build
    runs-on: ubuntu-latest
    steps:
      - name: Deploy to K8s
        run: kubectl apply -f k8s/
```

## 배포 전략

| 전략 | 설명 | 용도 |
|------|------|------|
| Rolling | 점진적 교체 | 기본 |
| Blue/Green | 환경 전환 | 무중단 |
| Canary | 일부 트래픽 | 검증 |

## 검증 규칙
- [ ] 테스트 통과 필수
- [ ] 보안 스캔 통과
- [ ] 승인 절차 (prod)
- [ ] 롤백 가능
```

---

## 컨벤션

### 디렉토리 구조

```
infra/
├── terraform/
│   ├── modules/        # 재사용 모듈
│   │   └── {resource}/
│   ├── environments/   # 환경별 설정
│   │   ├── dev/
│   │   ├── staging/
│   │   └── prod/
│   └── backend.tf      # 상태 저장소
│
├── kubernetes/
│   ├── base/           # 기본 매니페스트
│   └── overlays/       # 환경별 오버레이
│       ├── dev/
│       ├── staging/
│       └── prod/
│
└── .github/
    └── workflows/      # CI/CD
```

### 네이밍 규칙

| 구성요소 | 패턴 | 예시 |
|----------|------|------|
| Terraform Module | {resource} | vpc, rds, eks |
| K8s Namespace | {env}-{service} | prod-api |
| K8s Resource | {service}-{type} | api-deployment |
| Docker Image | {org}/{service} | myorg/api |

### 태그 정책

```hcl
tags = {
  Environment = var.environment
  Service     = "{service}"
  Team        = "{team}"
  ManagedBy   = "terraform"
  CostCenter  = "{cost-center}"
}
```

---

## 검증 기준

### Terraform
- [ ] terraform fmt
- [ ] terraform validate
- [ ] tfsec 보안 스캔
- [ ] 상태 파일 원격 저장

### Kubernetes
- [ ] 리소스 제한
- [ ] Health Check
- [ ] PodDisruptionBudget
- [ ] NetworkPolicy

### CI/CD
- [ ] 테스트 필수
- [ ] 보안 스캔
- [ ] 승인 워크플로우
