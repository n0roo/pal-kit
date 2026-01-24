#!/bin/bash
# PAL Kit 계층 구조 테스트 스크립트

set -e

echo "=========================================="
echo "PAL Kit 계층 구조 테스트"
echo "=========================================="

# 1. 테스트 데이터 생성
echo ""
echo "📦 1단계: 계층 테스트 데이터 생성"
echo "------------------------------------------"

sqlite3 ~/.pal/pal.db << 'EOF'
-- Build 세션 (최상위)
INSERT OR REPLACE INTO sessions (id, title, status, type, root_id, depth, path, started_at, token_budget)
VALUES ('build-001', 'Feature: User Authentication', 'running', 'build', 'build-001', 0, 'build-001', datetime('now', '-2 hours'), 200000);

-- Operator 세션 (Build 하위)
INSERT OR REPLACE INTO sessions (id, title, status, type, parent_id, root_id, depth, path, started_at, token_budget)
VALUES ('op-001', 'Auth Module Operator', 'running', 'operator', 'build-001', 'build-001', 1, 'build-001/op-001', datetime('now', '-1 hour'), 150000);

-- Worker 세션들 (Operator 하위)
INSERT OR REPLACE INTO sessions (id, title, status, type, parent_id, root_id, depth, path, started_at, token_budget, attention_score)
VALUES 
  ('worker-001', 'UserEntity Implementation', 'complete', 'worker', 'op-001', 'build-001', 2, 'build-001/op-001/worker-001', datetime('now', '-50 minutes'), 15000, 0.95),
  ('worker-002', 'UserRepository Implementation', 'running', 'worker', 'op-001', 'build-001', 2, 'build-001/op-001/worker-002', datetime('now', '-30 minutes'), 15000, 0.78),
  ('worker-003', 'AuthService Implementation', 'pending', 'worker', 'op-001', 'build-001', 2, 'build-001/op-001/worker-003', datetime('now', '-10 minutes'), 15000, NULL);

-- Test 세션들 (Worker와 페어)
INSERT OR REPLACE INTO sessions (id, title, status, type, parent_id, root_id, depth, path, started_at, token_budget)
VALUES 
  ('test-001', 'UserEntity Tests', 'complete', 'test', 'worker-001', 'build-001', 3, 'build-001/op-001/worker-001/test-001', datetime('now', '-45 minutes'), 10000),
  ('test-002', 'UserRepository Tests', 'running', 'test', 'worker-002', 'build-001', 3, 'build-001/op-001/worker-002/test-002', datetime('now', '-20 minutes'), 10000);

-- Attention 데이터
INSERT OR REPLACE INTO session_attention (session_id, port_id, loaded_tokens, available_tokens, focus_score, drift_count, updated_at)
VALUES 
  ('build-001', 'auth-feature', 45000, 200000, 0.92, 0, datetime('now')),
  ('op-001', 'auth-module', 32000, 150000, 0.88, 1, datetime('now')),
  ('worker-001', 'user-entity', 12000, 15000, 0.95, 0, datetime('now')),
  ('worker-002', 'user-repo', 11500, 15000, 0.78, 2, datetime('now'));
EOF

echo "✅ 테스트 데이터 생성 완료"

# 2. 데이터 확인
echo ""
echo "📊 2단계: 데이터 확인"
echo "------------------------------------------"
echo "세션 계층:"
sqlite3 -header -column ~/.pal/pal.db "SELECT id, type, depth, status, SUBSTR(title, 1, 30) as title FROM sessions WHERE root_id = 'build-001' OR id = 'build-001' ORDER BY depth, id;"

echo ""
echo "Attention 데이터:"
sqlite3 -header -column ~/.pal/pal.db "SELECT session_id, loaded_tokens, available_tokens, focus_score FROM session_attention WHERE session_id IN ('build-001', 'op-001', 'worker-001', 'worker-002');"

# 3. 서버 상태 확인
echo ""
echo "🌐 3단계: API 테스트"
echo "------------------------------------------"

# 서버 실행 확인
if ! lsof -i :9000 | grep -q LISTEN; then
    echo "⚠️  서버가 실행되지 않음. 시작합니다..."
    cd ~/playground/CodeSpace/pal-kit
    ./pal serve &
    sleep 2
fi

echo ""
echo "📍 Sessions Hierarchy:"
curl -s http://localhost:9000/api/v2/sessions/hierarchy | jq '.[:3]' 2>/dev/null || echo "❌ API 호출 실패"

echo ""
echo "📍 Build Session Tree:"
curl -s http://localhost:9000/api/v2/sessions/hierarchy/build-001/tree | jq '.' 2>/dev/null || echo "❌ API 호출 실패"

echo ""
echo "📍 Attention (build-001):"
curl -s http://localhost:9000/api/v2/attention/build-001 | jq '.' 2>/dev/null || echo "❌ API 호출 실패"

echo ""
echo "📍 Attention (worker-002):"
curl -s http://localhost:9000/api/v2/attention/worker-002 | jq '.' 2>/dev/null || echo "❌ API 호출 실패"

echo ""
echo "📍 V2 Status:"
curl -s http://localhost:9000/api/v2/status | jq '.' 2>/dev/null || echo "❌ API 호출 실패"

echo ""
echo "=========================================="
echo "✅ 테스트 완료"
echo "=========================================="
