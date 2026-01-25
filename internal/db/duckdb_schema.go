package db

// DuckDB 테이블 스키마
// nolint:unused // schema reserved for DuckDB OLAP integration
const _duckDBSchemaTables = `
CREATE TABLE IF NOT EXISTS metadata (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR PRIMARY KEY,
    port_id VARCHAR,
    title VARCHAR,
    status VARCHAR DEFAULT 'running',
    parent_id VARCHAR,
    root_id VARCHAR,
    depth INTEGER DEFAULT 0,
    path VARCHAR,
    type VARCHAR DEFAULT 'single',
    agent_id VARCHAR,
    agent_version INTEGER,
    substatus VARCHAR,
    attention_score DOUBLE,
    token_budget INTEGER,
    context_snapshot VARCHAR,
    checkpoint_id VARCHAR,
    output_summary VARCHAR,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    project_root VARCHAR,
    project_name VARCHAR,
    claude_session_id VARCHAR,
    transcript_path VARCHAR,
    cwd VARCHAR,
    input_tokens BIGINT DEFAULT 0,
    output_tokens BIGINT DEFAULT 0,
    cache_read_tokens BIGINT DEFAULT 0,
    cache_create_tokens BIGINT DEFAULT 0,
    cost_usd DOUBLE DEFAULT 0,
    compact_count INTEGER DEFAULT 0,
    last_compact_at TIMESTAMP,
    created_env VARCHAR,
    last_env VARCHAR,
    session_type VARCHAR,
    parent_session VARCHAR,
    jsonl_path VARCHAR
);

CREATE TABLE IF NOT EXISTS ports (
    id VARCHAR PRIMARY KEY,
    title VARCHAR,
    status VARCHAR DEFAULT 'pending',
    port_type VARCHAR DEFAULT 'atomic',
    session_id VARCHAR,
    file_path VARCHAR,
    agent_id VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    input_tokens BIGINT DEFAULT 0,
    output_tokens BIGINT DEFAULT 0,
    cost_usd DOUBLE DEFAULT 0,
    duration_secs INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS port_dependencies (
    port_id VARCHAR NOT NULL,
    depends_on VARCHAR NOT NULL,
    dependency_type VARCHAR,
    required_outputs VARCHAR,
    satisfied BOOLEAN DEFAULT false,
    satisfied_at TIMESTAMP,
    PRIMARY KEY (port_id, depends_on)
);

CREATE TABLE IF NOT EXISTS agents (
    id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL,
    type VARCHAR NOT NULL,
    description VARCHAR,
    capabilities VARCHAR,
    current_version INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS agent_versions (
    id VARCHAR PRIMARY KEY,
    agent_id VARCHAR NOT NULL,
    version INTEGER NOT NULL,
    spec_content VARCHAR NOT NULL,
    spec_hash VARCHAR,
    change_summary VARCHAR,
    change_reason VARCHAR,
    avg_attention_score DOUBLE,
    avg_completion_rate DOUBLE,
    avg_token_efficiency DOUBLE,
    usage_count INTEGER DEFAULT 0,
    status VARCHAR DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, version)
);

CREATE TABLE IF NOT EXISTS agent_performance (
    id VARCHAR PRIMARY KEY,
    agent_id VARCHAR NOT NULL,
    agent_version INTEGER NOT NULL,
    session_id VARCHAR NOT NULL,
    attention_avg DOUBLE,
    attention_min DOUBLE,
    token_used INTEGER,
    compact_count INTEGER,
    completion_time_seconds INTEGER,
    outcome VARCHAR,
    quality_score DOUBLE,
    feedback VARCHAR,
    improvement_suggestions VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS session_attention (
    session_id VARCHAR PRIMARY KEY,
    port_id VARCHAR,
    current_context_hash VARCHAR,
    loaded_tokens INTEGER DEFAULT 0,
    available_tokens INTEGER,
    token_budget INTEGER DEFAULT 200000,
    focus_score DOUBLE DEFAULT 1.0,
    drift_score DOUBLE DEFAULT 0.0,
    drift_count INTEGER DEFAULT 0,
    last_compaction_at TIMESTAMP,
    loaded_files VARCHAR,
    loaded_conventions VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS compact_events (
    id VARCHAR PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    trigger_reason VARCHAR,
    before_tokens INTEGER,
    after_tokens INTEGER,
    preserved_context VARCHAR,
    discarded_context VARCHAR,
    checkpoint_before VARCHAR,
    recovery_hint VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS messages (
    id VARCHAR PRIMARY KEY,
    conversation_id VARCHAR NOT NULL,
    from_session VARCHAR NOT NULL,
    to_session VARCHAR,
    type VARCHAR NOT NULL,
    subtype VARCHAR,
    payload VARCHAR NOT NULL,
    attention_score DOUBLE,
    context_snapshot VARCHAR,
    token_count INTEGER,
    cumulative_tokens INTEGER,
    status VARCHAR DEFAULT 'pending',
    priority INTEGER DEFAULT 5,
    port_id VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS port_handoffs (
    id VARCHAR PRIMARY KEY,
    from_port_id VARCHAR NOT NULL,
    to_port_id VARCHAR NOT NULL,
    handoff_type VARCHAR,
    content VARCHAR NOT NULL,
    token_count INTEGER,
    max_token_budget INTEGER DEFAULT 2000,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS orchestration_ports (
    id VARCHAR PRIMARY KEY,
    title VARCHAR NOT NULL,
    description VARCHAR,
    atomic_ports VARCHAR NOT NULL,
    status VARCHAR DEFAULT 'pending',
    current_port_id VARCHAR,
    progress_percent INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS worker_sessions (
    id VARCHAR PRIMARY KEY,
    orchestration_id VARCHAR,
    port_id VARCHAR NOT NULL,
    worker_type VARCHAR NOT NULL,
    impl_session_id VARCHAR,
    test_session_id VARCHAR,
    status VARCHAR DEFAULT 'pending',
    substatus VARCHAR,
    result VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS build_outputs (
    id VARCHAR PRIMARY KEY,
    build_session_id VARCHAR NOT NULL,
    operator_count INTEGER DEFAULT 0,
    worker_count INTEGER DEFAULT 0,
    test_count INTEGER DEFAULT 0,
    total_ports INTEGER DEFAULT 0,
    port_hierarchy VARCHAR,
    dependency_graph VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE IF NOT EXISTS session_events_id_seq;
CREATE TABLE IF NOT EXISTS session_events (
    id INTEGER DEFAULT nextval('session_events_id_seq') PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    event_type VARCHAR NOT NULL,
    event_data VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS locks (
    resource VARCHAR PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    acquired_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE IF NOT EXISTS escalations_id_seq;
CREATE TABLE IF NOT EXISTS escalations (
    id INTEGER DEFAULT nextval('escalations_id_seq') PRIMARY KEY,
    from_session VARCHAR,
    from_port VARCHAR,
    to_session VARCHAR,
    issue VARCHAR NOT NULL,
    type VARCHAR,
    severity VARCHAR DEFAULT 'medium',
    context VARCHAR,
    suggestion VARCHAR,
    resolution VARCHAR,
    status VARCHAR DEFAULT 'open',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP
);

CREATE SEQUENCE IF NOT EXISTS compactions_id_seq;
CREATE TABLE IF NOT EXISTS compactions (
    id INTEGER DEFAULT nextval('compactions_id_seq') PRIMARY KEY,
    session_id VARCHAR NOT NULL,
    triggered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    trigger_type VARCHAR DEFAULT 'auto',
    context_summary VARCHAR,
    tokens_before INTEGER
);

CREATE TABLE IF NOT EXISTS projects (
    root VARCHAR PRIMARY KEY,
    name VARCHAR,
    description VARCHAR,
    logical_root VARCHAR,
    last_active TIMESTAMP,
    session_count INTEGER DEFAULT 0,
    total_tokens BIGINT DEFAULT 0,
    total_cost DOUBLE DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pipelines (
    id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL,
    session_id VARCHAR,
    status VARCHAR DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP
);

CREATE TABLE IF NOT EXISTS pipeline_ports (
    pipeline_id VARCHAR NOT NULL,
    port_id VARCHAR NOT NULL,
    group_order INTEGER DEFAULT 0,
    status VARCHAR DEFAULT 'pending',
    PRIMARY KEY (pipeline_id, port_id)
);

CREATE TABLE IF NOT EXISTS documents (
    id VARCHAR PRIMARY KEY,
    path VARCHAR NOT NULL UNIQUE,
    type VARCHAR,
    domain VARCHAR,
    status VARCHAR DEFAULT 'active',
    priority VARCHAR,
    tokens INTEGER DEFAULT 0,
    summary VARCHAR,
    content_hash VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS document_tags (
    document_id VARCHAR NOT NULL,
    tag VARCHAR NOT NULL,
    PRIMARY KEY (document_id, tag)
);

CREATE TABLE IF NOT EXISTS document_links (
    from_id VARCHAR NOT NULL,
    to_id VARCHAR NOT NULL,
    link_type VARCHAR NOT NULL,
    PRIMARY KEY (from_id, to_id)
);

CREATE SEQUENCE IF NOT EXISTS code_markers_id_seq;
CREATE TABLE IF NOT EXISTS code_markers (
    id INTEGER DEFAULT nextval('code_markers_id_seq') PRIMARY KEY,
    port VARCHAR NOT NULL,
    layer VARCHAR,
    domain VARCHAR,
    adapter VARCHAR,
    generated BOOLEAN DEFAULT false,
    file_path VARCHAR NOT NULL,
    line INTEGER NOT NULL,
    project_root VARCHAR,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(file_path, line)
);

CREATE TABLE IF NOT EXISTS code_marker_deps (
    from_port VARCHAR NOT NULL,
    to_port VARCHAR NOT NULL,
    PRIMARY KEY (from_port, to_port)
);

CREATE TABLE IF NOT EXISTS marker_port_links (
    marker_port VARCHAR NOT NULL,
    document_id VARCHAR NOT NULL,
    PRIMARY KEY (marker_port, document_id)
);

CREATE SEQUENCE IF NOT EXISTS file_manifests_id_seq;
CREATE TABLE IF NOT EXISTS file_manifests (
    id INTEGER DEFAULT nextval('file_manifests_id_seq') PRIMARY KEY,
    project_root VARCHAR NOT NULL,
    file_path VARCHAR NOT NULL,
    file_type VARCHAR NOT NULL,
    hash VARCHAR NOT NULL,
    size INTEGER DEFAULT 0,
    mtime TIMESTAMP,
    managed_by VARCHAR DEFAULT 'pal',
    status VARCHAR DEFAULT 'synced',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_root, file_path)
);

CREATE SEQUENCE IF NOT EXISTS file_changes_id_seq;
CREATE TABLE IF NOT EXISTS file_changes (
    id INTEGER DEFAULT nextval('file_changes_id_seq') PRIMARY KEY,
    project_root VARCHAR NOT NULL,
    file_path VARCHAR NOT NULL,
    change_type VARCHAR NOT NULL,
    old_hash VARCHAR,
    new_hash VARCHAR,
    changed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    session_id VARCHAR
);

CREATE TABLE IF NOT EXISTS environments (
    id VARCHAR PRIMARY KEY,
    name VARCHAR NOT NULL UNIQUE,
    hostname VARCHAR,
    paths VARCHAR,
    is_current BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_active TIMESTAMP
);

CREATE TABLE IF NOT EXISTS sync_meta (
    key VARCHAR PRIMARY KEY,
    value VARCHAR,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE SEQUENCE IF NOT EXISTS sync_history_id_seq;
CREATE TABLE IF NOT EXISTS sync_history (
    id INTEGER DEFAULT nextval('sync_history_id_seq') PRIMARY KEY,
    direction VARCHAR NOT NULL,
    env_id VARCHAR,
    items_count INTEGER DEFAULT 0,
    conflicts_count INTEGER DEFAULT 0,
    synced_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
`
