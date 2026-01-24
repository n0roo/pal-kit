#!/bin/bash
# PAL Kit 테스트 데이터 생성 스크립트

set -e

PAL_CMD="./pal"
DB_PATH="$HOME/.pal/pal.db"

echo "=== PAL Kit 테스트 데이터 생성 ==="

# DB 초기화
echo "1. DB 초기화..."
$PAL_CMD init --force 2>/dev/null || true

# 에이전트 생성
echo "2. 에이전트 생성..."
$PAL_CMD agent create "spec-agent" -t spec -d "명세 설계 에이전트" 2>/dev/null || echo "  (이미 존재)"
$PAL_CMD agent create "operator-agent" -t operator -d "Operator 에이전트" 2>/dev/null || echo "  (이미 존재)"
$PAL_CMD agent create "impl-worker" -t worker -d "구현 Worker" 2>/dev/null || echo "  (이미 존재)"
$PAL_CMD agent create "test-worker" -t worker -d "테스트 Worker" 2>/dev/null || echo "  (이미 존재)"

# Orchestration 생성
echo "3. Orchestration 생성..."
ORCH_ID=$($PAL_CMD orchestration create "user-service-impl" -p "port-auth,port-user-crud,port-user-api" 2>/dev/null | grep -o '[a-f0-9-]\{36\}' | head -1)
echo "  Orchestration ID: ${ORCH_ID:-생성됨}"

# 추가 Orchestration
$PAL_CMD orchestration create "payment-service" -p "port-payment-init,port-payment-process,port-payment-webhook" 2>/dev/null || true

echo ""
echo "=== 완료 ==="
echo ""
echo "Electron GUI에서 확인하세요:"
echo "  - Dashboard: 통계 카드"
echo "  - Agents: 4개 에이전트"
echo "  - Orchestrations: 2개 Orchestration"
echo ""
echo "세션과 Attention 테스트는 MCP를 통해 진행하세요:"
echo "  Claude Desktop에서 PAL Kit MCP 연결 후"
echo "  'session_start로 build 세션 시작해줘' 요청"
