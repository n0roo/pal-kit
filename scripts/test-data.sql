-- PAL Kit 테스트 데이터 SQL
-- 사용법: sqlite3 ~/.pal/pal.db < test-data.sql

-- 에이전트 생성
INSERT OR IGNORE INTO agents (id, name, type, description, capabilities, current_version, created_at, updated_at)
VALUES 
  ('agent-spec-001', 'spec-agent', 'spec', '명세 설계 에이전트 - 요구사항을 분석하고 Atomic Port로 분해', '["requirement_analysis", "port_decomposition", "spec_writing"]', 1, datetime('now'), datetime('now')),
  ('agent-op-001', 'operator-agent', 'operator', 'Operator 에이전트 - Worker 관리 및 진행 조율', '["worker_management", "progress_tracking", "escalation_handling"]', 1, datetime('now'), datetime('now')),
  ('agent-impl-001', 'impl-worker', 'worker', '구현 Worker - 코드 구현 담당', '["code_implementation", "refactoring", "bug_fixing"]', 1, datetime('now'), datetime('now')),
  ('agent-test-001', 'test-worker', 'worker', '테스트 Worker - 테스트 작성 및 실행', '["test_writing", "test_execution", "coverage_analysis"]', 1, datetime('now'), datetime('now'));

-- 에이전트 버전 생성
INSERT OR IGNORE INTO agent_versions (id, agent_id, version, spec_content, change_summary, change_reason, created_at)
VALUES
  ('ver-spec-1', 'agent-spec-001', 1, '# Spec Agent v1\n\n명세 설계 전문 에이전트', '초기 버전', 'initial', datetime('now')),
  ('ver-op-1', 'agent-op-001', 1, '# Operator Agent v1\n\nWorker 조율 에이전트', '초기 버전', 'initial', datetime('now')),
  ('ver-impl-1', 'agent-impl-001', 1, '# Impl Worker v1\n\n구현 전문 워커', '초기 버전', 'initial', datetime('now')),
  ('ver-test-1', 'agent-test-001', 1, '# Test Worker v1\n\n테스트 전문 워커', '초기 버전', 'initial', datetime('now'));

-- Orchestration 생성
INSERT OR IGNORE INTO orchestration_ports (id, title, description, atomic_ports, status, progress_percent, created_at)
VALUES 
  ('orch-001', 'user-service-impl', '사용자 서비스 구현', '[{"port_id":"port-auth","order":1,"status":"complete"},{"port_id":"port-user-crud","order":2,"status":"running","depends_on":["port-auth"]},{"port_id":"port-user-api","order":3,"status":"pending","depends_on":["port-user-crud"]}]', 'running', 45, datetime('now')),
  ('orch-002', 'payment-service', '결제 서비스 구현', '[{"port_id":"port-payment-init","order":1,"status":"pending"},{"port_id":"port-payment-process","order":2,"status":"pending","depends_on":["port-payment-init"]},{"port_id":"port-payment-webhook","order":3,"status":"pending","depends_on":["port-payment-process"]}]', 'pending', 0, datetime('now'));

-- Build 세션 생성
INSERT OR IGNORE INTO sessions (id, title, status, session_type, parent_session, type, depth, path, root_id, agent_id, token_budget, started_at, updated_at)
VALUES 
  ('sess-build-001', 'User Service Build', 'running', 'builder', NULL, 'build', 0, 'sess-build-001', 'sess-build-001', 'agent-spec-001', 15000, datetime('now', '-2 hours'), datetime('now')),
  ('sess-build-002', 'Payment Service Build', 'running', 'builder', NULL, 'build', 0, 'sess-build-002', 'sess-build-002', 'agent-spec-001', 15000, datetime('now', '-1 hours'), datetime('now'));

-- Operator 세션 생성
INSERT OR IGNORE INTO sessions (id, title, status, session_type, parent_session, type, depth, path, root_id, agent_id, token_budget, started_at, updated_at)
VALUES 
  ('sess-op-001', 'User Service Operator', 'running', 'sub', 'sess-build-001', 'operator', 1, 'sess-build-001/sess-op-001', 'sess-build-001', 'agent-op-001', 15000, datetime('now', '-1 hours'), datetime('now'));

-- Worker 세션 생성
INSERT OR IGNORE INTO sessions (id, title, status, session_type, parent_session, type, depth, path, root_id, agent_id, token_budget, started_at, updated_at)
VALUES 
  ('sess-worker-001', 'Auth Impl Worker', 'complete', 'sub', 'sess-op-001', 'worker', 2, 'sess-build-001/sess-op-001/sess-worker-001', 'sess-build-001', 'agent-impl-001', 15000, datetime('now', '-30 minutes'), datetime('now')),
  ('sess-worker-002', 'User CRUD Worker', 'running', 'sub', 'sess-op-001', 'worker', 2, 'sess-build-001/sess-op-001/sess-worker-002', 'sess-build-001', 'agent-impl-001', 15000, datetime('now', '-15 minutes'), datetime('now'));

-- Attention 데이터 생성
INSERT OR IGNORE INTO session_attention (session_id, port_id, loaded_tokens, available_tokens, focus_score, drift_count, loaded_files, loaded_conventions, updated_at)
VALUES 
  ('sess-build-001', NULL, 8500, 15000, 0.92, 1, '["CLAUDE.md", "specs/user-service.md"]', '["go-conventions", "api-design"]', datetime('now')),
  ('sess-op-001', NULL, 6200, 15000, 0.88, 2, '["orchestration.md"]', '["worker-management"]', datetime('now')),
  ('sess-worker-001', 'port-auth', 12000, 15000, 0.75, 3, '["auth/handler.go", "auth/service.go", "auth/middleware.go"]', '["go-conventions"]', datetime('now')),
  ('sess-worker-002', 'port-user-crud', 9800, 15000, 0.85, 1, '["user/handler.go", "user/repository.go"]', '["go-conventions", "db-patterns"]', datetime('now'));

-- Compact 이벤트 생성
INSERT OR IGNORE INTO compact_events (id, session_id, trigger_reason, before_tokens, after_tokens, preserved_context, discarded_context, recovery_hint, created_at)
VALUES 
  ('compact-001', 'sess-worker-001', 'token_limit', 14500, 8000, '["핵심 구현 코드", "API 스펙"]', '["디버깅 로그", "임시 테스트"]', 'auth 모듈 핵심 로직 보존됨', datetime('now', '-20 minutes')),
  ('compact-002', 'sess-build-001', 'focus_drift', 13000, 8500, '["요구사항", "포트 분해"]', '["브레인스토밍 내용"]', '명세 핵심 내용 보존', datetime('now', '-1 hours'));

-- Worker 세션 (orchestration 연결)
INSERT OR IGNORE INTO worker_sessions (id, orchestration_id, port_id, worker_type, impl_session_id, test_session_id, status, substatus, created_at, updated_at)
VALUES 
  ('ws-001', 'orch-001', 'port-auth', 'impl_test_pair', 'sess-worker-001', NULL, 'complete', NULL, datetime('now', '-30 minutes'), datetime('now')),
  ('ws-002', 'orch-001', 'port-user-crud', 'impl_test_pair', 'sess-worker-002', NULL, 'running', 'implementing', datetime('now', '-15 minutes'), datetime('now'));

-- Handoff 데이터
INSERT OR IGNORE INTO port_handoffs (id, from_port_id, to_port_id, type, content, token_count, created_at)
VALUES 
  ('ho-001', 'port-auth', 'port-user-crud', 'api_contract', '{"entity":"User","methods":[{"name":"Authenticate","params":["credentials"],"returns":"token"}]}', 150, datetime('now', '-25 minutes')),
  ('ho-002', 'port-auth', 'port-user-api', 'type_def', '{"types":["AuthToken","UserCredentials","AuthError"]}', 80, datetime('now', '-25 minutes'));

SELECT 'Test data created successfully!' as result;
SELECT '에이전트: ' || COUNT(*) FROM agents;
SELECT 'Orchestration: ' || COUNT(*) FROM orchestration_ports;
SELECT '세션: ' || COUNT(*) FROM sessions;
SELECT 'Attention: ' || COUNT(*) FROM session_attention;
