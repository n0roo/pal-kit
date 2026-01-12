// PAL Kit Dashboard App

const API_BASE = '';
let refreshInterval;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initTabs();
    initRefresh();
    initModal();
    loadAllData();
    
    // Auto refresh every 10 seconds
    refreshInterval = setInterval(loadAllData, 10000);
});

// Tab Navigation
function initTabs() {
    const navBtns = document.querySelectorAll('.nav-btn');
    navBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            const tab = btn.dataset.tab;
            switchTab(tab);
        });
    });
}

function switchTab(tab) {
    document.querySelectorAll('.nav-btn').forEach(btn => {
        btn.classList.toggle('active', btn.dataset.tab === tab);
    });
    
    document.querySelectorAll('.tab-content').forEach(content => {
        content.classList.toggle('active', content.id === `tab-${tab}`);
    });
}

// Modal
function initModal() {
    const modal = document.getElementById('session-modal');
    const closeBtn = modal.querySelector('.modal-close');

    closeBtn.addEventListener('click', () => modal.classList.add('hidden'));
    modal.addEventListener('click', (e) => {
        if (e.target === modal) modal.classList.add('hidden');
    });
}

// Overview Modal - shows detailed lists for each card type
async function showOverviewModal(type) {
    const modal = document.getElementById('session-modal');
    const body = document.getElementById('session-modal-body');
    const header = modal.querySelector('.modal-header h3');

    let content = '';

    switch (type) {
        case 'active-sessions': {
            header.textContent = 'Active Sessions';
            const data = await fetchAPI('sessions?status=active');
            if (!data || data.length === 0) {
                content = '<div class="empty-state">No active sessions</div>';
            } else {
                content = `
                    <div class="children-list">
                        ${data.map(s => `
                            <div class="child-item" onclick="showSessionDetail('${s.id}')">
                                ${statusBadge(s.status)}
                                <span>${escapeHtml(s.id)}</span>
                                <span class="muted">${escapeHtml(s.title || '')}</span>
                            </div>
                        `).join('')}
                    </div>
                `;
            }
            break;
        }
        case 'completed-sessions': {
            header.textContent = 'Completed Sessions';
            const data = await fetchAPI('sessions?status=complete&limit=20');
            if (!data || data.length === 0) {
                content = '<div class="empty-state">No completed sessions</div>';
            } else {
                content = `
                    <div class="children-list">
                        ${data.map(s => `
                            <div class="child-item" onclick="showSessionDetail('${s.id}')">
                                ${statusBadge(s.status)}
                                <span>${escapeHtml(s.id)}</span>
                                <span class="muted">${escapeHtml(s.title || '')}</span>
                            </div>
                        `).join('')}
                    </div>
                `;
            }
            break;
        }
        case 'ports': {
            header.textContent = 'Ports';
            const data = await fetchAPI('ports');
            if (!data || data.length === 0) {
                content = '<div class="empty-state">No ports</div>';
            } else {
                const running = data.filter(p => p.status === 'running');
                const complete = data.filter(p => p.status === 'complete');
                const pending = data.filter(p => p.status === 'pending' || p.status === 'draft');

                content = '';
                if (running.length > 0) {
                    content += `<h4>Running (${running.length})</h4><div class="children-list">${running.map(p => `
                        <div class="child-item">
                            ${statusBadge(p.status)}
                            <span>${escapeHtml(p.id)}</span>
                            <span class="muted">${escapeHtml(p.title || '')}</span>
                        </div>
                    `).join('')}</div>`;
                }
                if (complete.length > 0) {
                    content += `<h4>Complete (${complete.length})</h4><div class="children-list">${complete.map(p => `
                        <div class="child-item">
                            ${statusBadge(p.status)}
                            <span>${escapeHtml(p.id)}</span>
                            <span class="muted">${escapeHtml(p.title || '')}</span>
                        </div>
                    `).join('')}</div>`;
                }
                if (pending.length > 0) {
                    content += `<h4>Pending (${pending.length})</h4><div class="children-list">${pending.map(p => `
                        <div class="child-item">
                            ${statusBadge(p.status)}
                            <span>${escapeHtml(p.id)}</span>
                            <span class="muted">${escapeHtml(p.title || '')}</span>
                        </div>
                    `).join('')}</div>`;
                }
            }
            break;
        }
        case 'workflows': {
            header.textContent = 'Workflows';
            const data = await fetchAPI('pipelines');
            if (!data || data.length === 0) {
                content = '<div class="empty-state">No workflows</div>';
            } else {
                content = `
                    <div class="children-list">
                        ${data.map(p => `
                            <div class="child-item">
                                ${statusBadge(p.status)}
                                <span>${escapeHtml(p.id)}</span>
                                <span class="muted">${escapeHtml(p.name || '')}</span>
                            </div>
                        `).join('')}
                    </div>
                `;
            }
            break;
        }
        case 'tokens': {
            header.textContent = 'Token Usage';
            const data = await fetchAPI('sessions/stats');
            if (!data) {
                content = '<div class="empty-state">No data</div>';
            } else {
                content = `
                    <div class="detail-grid">
                        <div class="detail-item">
                            <label>Input Tokens</label>
                            <span>${formatNumber(data.total_input_tokens || 0)}</span>
                        </div>
                        <div class="detail-item">
                            <label>Output Tokens</label>
                            <span>${formatNumber(data.total_output_tokens || 0)}</span>
                        </div>
                        <div class="detail-item">
                            <label>Cache Read</label>
                            <span>${formatNumber(data.total_cache_read_tokens || 0)}</span>
                        </div>
                        <div class="detail-item">
                            <label>Cache Create</label>
                            <span>${formatNumber(data.total_cache_create_tokens || 0)}</span>
                        </div>
                        <div class="detail-item highlight">
                            <label>Total</label>
                            <span>${formatNumber((data.total_input_tokens || 0) + (data.total_output_tokens || 0))}</span>
                        </div>
                    </div>
                `;
            }
            break;
        }
        case 'cost': {
            header.textContent = 'Cost Breakdown';
            const data = await fetchAPI('sessions/stats');
            if (!data) {
                content = '<div class="empty-state">No data</div>';
            } else {
                content = `
                    <div class="detail-grid">
                        <div class="detail-item highlight">
                            <label>Total Cost</label>
                            <span>$${(data.total_cost_usd || 0).toFixed(4)}</span>
                        </div>
                        <div class="detail-item">
                            <label>Sessions</label>
                            <span>${data.total_sessions || 0}</span>
                        </div>
                        <div class="detail-item">
                            <label>Avg per Session</label>
                            <span>$${data.total_sessions ? ((data.total_cost_usd || 0) / data.total_sessions).toFixed(4) : '0.0000'}</span>
                        </div>
                    </div>
                `;
            }
            break;
        }
        case 'time': {
            header.textContent = 'Time Summary';
            const data = await fetchAPI('sessions/stats');
            if (!data) {
                content = '<div class="empty-state">No data</div>';
            } else {
                content = `
                    <div class="detail-grid">
                        <div class="detail-item highlight">
                            <label>Total Time</label>
                            <span>${formatDuration(data.total_duration_secs || 0)}</span>
                        </div>
                        <div class="detail-item">
                            <label>Sessions</label>
                            <span>${data.total_sessions || 0}</span>
                        </div>
                        <div class="detail-item">
                            <label>Avg per Session</label>
                            <span>${formatDuration(data.total_sessions ? Math.floor((data.total_duration_secs || 0) / data.total_sessions) : 0)}</span>
                        </div>
                    </div>
                `;
            }
            break;
        }
        case 'escalations': {
            header.textContent = 'Escalations';
            const data = await fetchAPI('escalations');
            if (!data || data.length === 0) {
                content = '<div class="empty-state">No escalations</div>';
            } else {
                content = `
                    <div class="children-list">
                        ${data.map(e => `
                            <div class="child-item">
                                ${statusBadge(e.status)}
                                <span>${escapeHtml(e.title || e.id)}</span>
                                <span class="muted">${escapeHtml(e.message || '')}</span>
                            </div>
                        `).join('')}
                    </div>
                `;
            }
            break;
        }
        default:
            content = '<div class="empty-state">Unknown type</div>';
    }

    body.innerHTML = content;
    modal.classList.remove('hidden');
}

// Current session for event filtering
let currentSessionId = null;

function showSessionDetail(sessionId, eventTypeFilter = '') {
    currentSessionId = sessionId;
    
    // Build events URL with optional filter
    let eventsUrl = `sessions/${sessionId}/events?limit=50`;
    if (eventTypeFilter) {
        eventsUrl += `&type=${eventTypeFilter}`;
    }
    
    // Fetch both session detail and events in parallel
    Promise.all([
        fetchAPI(`sessions/${sessionId}`),
        fetchAPI(eventsUrl)
    ]).then(([data, events]) => {
        if (!data) return;
        
        const modal = document.getElementById('session-modal');
        const body = document.getElementById('session-modal-body');
        const s = data.session;
        const children = data.children || [];
        const eventList = events || [];
        
        body.innerHTML = `
            <div class="detail-grid">
                <div class="detail-item">
                    <label>ID</label>
                    <span>${escapeHtml(s.id)}</span>
                </div>
                <div class="detail-item">
                    <label>Title</label>
                    <span>${escapeHtml(s.title || '-')}</span>
                </div>
                <div class="detail-item">
                    <label>Status</label>
                    <span>${statusBadge(s.status)}</span>
                </div>
                <div class="detail-item">
                    <label>Type</label>
                    <span>${escapeHtml(s.session_type || 'single')}</span>
                </div>
                <div class="detail-item">
                    <label>Parent</label>
                    <span>${s.parent ? `<a href="#" onclick="showSessionDetail('${s.parent}')">${s.parent}</a>` : '-'}</span>
                </div>
                <div class="detail-item">
                    <label>Port</label>
                    <span>${escapeHtml(s.port_id || '-')}</span>
                </div>
                <div class="detail-item">
                    <label>Duration</label>
                    <span>${escapeHtml(s.duration_str || '-')}</span>
                </div>
                <div class="detail-item">
                    <label>Started</label>
                    <span>${formatDate(s.started_at)}</span>
                </div>
                <div class="detail-item">
                    <label>Ended</label>
                    <span>${formatDate(s.ended_at)}</span>
                </div>
            </div>
            
            <h4>Token Usage</h4>
            <div class="detail-grid">
                <div class="detail-item">
                    <label>Input</label>
                    <span>${formatNumber(s.input_tokens)}</span>
                </div>
                <div class="detail-item">
                    <label>Output</label>
                    <span>${formatNumber(s.output_tokens)}</span>
                </div>
                <div class="detail-item">
                    <label>Cache Read</label>
                    <span>${formatNumber(s.cache_read_tokens)}</span>
                </div>
                <div class="detail-item">
                    <label>Cache Create</label>
                    <span>${formatNumber(s.cache_create_tokens)}</span>
                </div>
                <div class="detail-item highlight">
                    <label>Cost (USD)</label>
                    <span>\${(s.cost_usd || 0).toFixed(4)}</span>
                </div>
                <div class="detail-item">
                    <label>Compactions</label>
                    <span>${s.compact_count || 0}</span>
                </div>
            </div>
            
            <div class="timeline-header-row">
                <h4>Event Timeline (${eventList.length})</h4>
                <select id="event-type-filter" onchange="filterEvents(this.value)">
                    <option value=""${!eventTypeFilter ? ' selected' : ''}>All Events</option>
                    <option value="session_start"${eventTypeFilter === 'session_start' ? ' selected' : ''}>🚀 Session Start</option>
                    <option value="session_end"${eventTypeFilter === 'session_end' ? ' selected' : ''}>🏁 Session End</option>
                    <option value="pre_compact"${eventTypeFilter === 'pre_compact' ? ' selected' : ''}>📦 Compact</option>
                    <option value="tool_use"${eventTypeFilter === 'tool_use' ? ' selected' : ''}>🔧 Tool Use</option>
                    <option value="notification"${eventTypeFilter === 'notification' ? ' selected' : ''}>📢 Notification</option>
                </select>
            </div>
            ${eventList.length > 0 ? `
                <div class="timeline">
                    ${eventList.map(e => `
                        <div class="timeline-item">
                            <div class="timeline-marker ${getEventTypeClass(e.event_type)}"></div>
                            <div class="timeline-content">
                                <div class="timeline-header">
                                    <span class="event-type">${eventTypeLabel(e.event_type)}</span>
                                    <span class="event-time">${formatTimeAgo(e.created_at)}</span>
                                </div>
                                ${e.event_data ? `<div class="event-data">${formatEventData(e.event_data)}</div>` : ''}
                            </div>
                        </div>
                    `).join('')}
                </div>
            ` : '<div class="empty-state">No events found</div>'}
            
            ${children.length > 0 ? `
                <h4>Child Sessions (${children.length})</h4>
                <div class="children-list">
                    ${children.map(c => `
                        <div class="child-item" onclick="showSessionDetail('${c.id}')">
                            ${statusBadge(c.status)}
                            <span>${escapeHtml(c.id)}</span>
                            <span class="muted">${escapeHtml(c.title || '')}</span>
                        </div>
                    `).join('')}
                </div>
            ` : ''}
        `;
        
        modal.classList.remove('hidden');
    });
}

// Filter events by type
function filterEvents(eventType) {
    if (currentSessionId) {
        showSessionDetail(currentSessionId, eventType);
    }
}

// Event type helpers
function getEventTypeClass(type) {
    const classes = {
        'session_start': 'start',
        'session_end': 'end',
        'pre_compact': 'compact',
        'tool_use': 'tool',
        'notification': 'info'
    };
    return classes[type] || 'default';
}

function eventTypeLabel(type) {
    const labels = {
        'session_start': '🚀 Session Start',
        'session_end': '🏁 Session End',
        'pre_compact': '📦 Compact',
        'tool_use': '🔧 Tool Use',
        'notification': '📢 Notification'
    };
    return labels[type] || type;
}

function formatEventData(data) {
    try {
        const parsed = JSON.parse(data);
        return Object.entries(parsed)
            .filter(([k, v]) => v !== '' && v !== null)
            .map(([k, v]) => `<span class="event-kv"><strong>${escapeHtml(k)}:</strong> ${escapeHtml(String(v))}</span>`)
            .join(' ');
    } catch {
        return escapeHtml(data);
    }
}

function formatTimeAgo(dateStr) {
    if (!dateStr) return '-';
    try {
        const date = new Date(dateStr);
        const now = new Date();
        const diff = Math.floor((now - date) / 1000);
        
        if (diff < 60) return 'just now';
        if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
        if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
        return date.toLocaleDateString();
    } catch {
        return '-';
    }
}

// Refresh
function initRefresh() {
    document.getElementById('refresh-btn').addEventListener('click', loadAllData);
}

function updateLastRefresh() {
    const el = document.getElementById('last-update');
    el.textContent = `Updated: ${new Date().toLocaleTimeString()}`;
}

// API Calls
async function fetchAPI(endpoint) {
    try {
        const response = await fetch(`${API_BASE}/api/${endpoint}`);
        if (!response.ok) throw new Error(`HTTP ${response.status}`);
        return await response.json();
    } catch (error) {
        console.error(`API Error (${endpoint}):`, error);
        return null;
    }
}

// Load All Data
async function loadAllData() {
    await Promise.all([
        loadStatus(),
        loadSessionStats(),
        loadSessions(),
        loadHistory(),
        loadPorts(),
        loadWorkflows(),
        loadDocs(),
        loadConventions(),
        loadAgents()
    ]);
    updateLastRefresh();
}

// Status
async function loadStatus() {
    const data = await fetchAPI('status');
    if (!data) return;

    setStatValue('sessions-active', data.sessions?.active ?? 0);
    setStatValue('ports-total', data.ports?.total ?? 0);
    setStatValue('workflows-running', data.pipelines?.running ?? 0);
    setStatValue('escalations-open', data.escalations?.open ?? 0);

    // Ports breakdown
    const running = data.ports?.running ?? 0;
    const complete = data.ports?.complete ?? 0;
    const pending = data.ports?.pending ?? 0;
    setStatValue('ports-breakdown', `${running} running, ${complete} complete, ${pending} pending`);

    document.getElementById('project-root').textContent = data.project_root || '';
}

// Session Stats
async function loadSessionStats() {
    const data = await fetchAPI('sessions/stats');
    if (!data) return;
    
    setStatValue('sessions-completed', data.completed_sessions ?? 0);
    
    const totalTokens = (data.total_input_tokens || 0) + (data.total_output_tokens || 0);
    setStatValue('total-tokens', formatNumber(totalTokens));
    setStatValue('total-cost', `$${(data.total_cost_usd || 0).toFixed(2)}`);
    setStatValue('total-duration', formatDuration(data.total_duration_secs || 0));
}

function setStatValue(id, value) {
    const el = document.getElementById(`stat-${id}`);
    if (el) el.textContent = value;
}

// Sessions
async function loadSessions() {
    const data = await fetchAPI('sessions');
    const tbody = document.getElementById('sessions-table');
    
    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="empty-state">No sessions</td></tr>';
        return;
    }
    
    tbody.innerHTML = data.map(s => `
        <tr onclick="showSessionDetail('${s.id}')" style="cursor:pointer">
            <td>${statusBadge(s.status || 'unknown')}</td>
            <td class="text-sm">${escapeHtml(s.id || '-')}</td>
            <td>${escapeHtml(s.title || '-')}</td>
            <td>${escapeHtml(s.session_type || 'single')}</td>
            <td>${escapeHtml(s.duration_str || '-')}</td>
            <td class="text-sm">${formatNumber(s.input_tokens + s.output_tokens)}</td>
            <td class="text-sm">$${(s.cost_usd || 0).toFixed(4)}</td>
            <td>${s.children_count || 0}</td>
        </tr>
    `).join('');
}

// History
async function loadHistory() {
    const data = await fetchAPI('sessions/history?days=30');
    const tbody = document.getElementById('history-table');
    
    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="7" class="empty-state">No history</td></tr>';
        return;
    }
    
    tbody.innerHTML = data.map(h => `
        <tr>
            <td>${escapeHtml(h.date)}</td>
            <td>${h.count}</td>
            <td>${h.completed}</td>
            <td class="text-sm">${formatNumber(h.input_tokens)}</td>
            <td class="text-sm">${formatNumber(h.output_tokens)}</td>
            <td>${escapeHtml(h.duration_str || '-')}</td>
            <td>$${(h.cost_usd || 0).toFixed(4)}</td>
        </tr>
    `).join('');
}

// Ports
async function loadPorts() {
    const data = await fetchAPI('ports');
    const tbody = document.getElementById('ports-table');
    
    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No ports</td></tr>';
        return;
    }
    
    tbody.innerHTML = data.map(p => `
        <tr>
            <td>${statusBadge(p.status || 'unknown')}</td>
            <td class="text-sm">${escapeHtml(p.id || '-')}</td>
            <td>${escapeHtml(p.title || '-')}</td>
            <td>${escapeHtml(p.session_id || '-')}</td>
            <td class="text-sm muted">${formatDate(p.created_at)}</td>
        </tr>
    `).join('');
}

// Workflows (formerly Pipelines)
async function loadWorkflows() {
    const data = await fetchAPI('pipelines');
    const tbody = document.getElementById('workflows-table');

    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="empty-state">No workflows</td></tr>';
        return;
    }

    tbody.innerHTML = data.map(p => `
        <tr>
            <td>${statusBadge(p.status || 'unknown')}</td>
            <td class="text-sm">${escapeHtml(p.id || '-')}</td>
            <td>${escapeHtml(p.name || '-')}</td>
            <td class="text-sm muted">${formatDate(p.created_at)}</td>
        </tr>
    `).join('');
}

// Documents
async function loadDocs() {
    const data = await fetchAPI('docs');
    const tbody = document.getElementById('docs-table');
    
    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No documents</td></tr>';
        return;
    }
    
    tbody.innerHTML = data.map(d => `
        <tr>
            <td>${statusBadge(d.status || 'unknown')}</td>
            <td>${escapeHtml(d.relative_path || d.path || '-')}</td>
            <td>${escapeHtml(d.type || '-')}</td>
            <td class="text-sm muted">${formatBytes(d.size)}</td>
            <td class="text-sm muted">${formatDate(d.modified_at)}</td>
        </tr>
    `).join('');
}

// Conventions
async function loadConventions() {
    const data = await fetchAPI('conventions');
    const tbody = document.getElementById('conventions-table');
    
    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="empty-state">No conventions</td></tr>';
        return;
    }
    
    tbody.innerHTML = data.map(c => `
        <tr>
            <td>${statusBadge(c.enabled ? 'enabled' : 'disabled')}</td>
            <td class="text-sm">${escapeHtml(c.id || '-')}</td>
            <td>${escapeHtml(c.name || '-')}</td>
            <td>${escapeHtml(c.type || '-')}</td>
            <td>${Array.isArray(c.rules) ? c.rules.length : 0}</td>
            <td>${c.priority ?? '-'}</td>
        </tr>
    `).join('');
}

// Agents
async function loadAgents() {
    const data = await fetchAPI('agents');
    const grid = document.getElementById('agents-grid');
    
    if (!data || data.length === 0) {
        grid.innerHTML = '<div class="empty-state"><div class="icon">🤖</div><p>No agents configured</p></div>';
        return;
    }
    
    grid.innerHTML = data.map(a => `
        <div class="agent-card">
            <h3>
                ${typeEmoji(a.type)} ${escapeHtml(a.name || a.id || 'Unknown')}
                <span class="type-badge">${escapeHtml(a.type || 'custom')}</span>
            </h3>
            <p>${escapeHtml(a.description || 'No description')}</p>
            ${Array.isArray(a.tools) && a.tools.length ? `
                <div class="agent-tools">
                    ${a.tools.map(t => `<span class="tool-tag">${escapeHtml(t)}</span>`).join('')}
                </div>
            ` : ''}
        </div>
    `).join('');
}

// Helpers
function escapeHtml(text) {
    if (text === null || text === undefined) return '-';
    const str = String(text);
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
}

function statusBadge(status) {
    const safeStatus = escapeHtml(status || 'unknown');
    return `<span class="status"><span class="status-dot ${safeStatus}"></span>${safeStatus}</span>`;
}

function formatDate(dateStr) {
    if (!dateStr) return '-';
    try {
        const date = new Date(dateStr);
        if (isNaN(date.getTime())) return '-';
        return date.toLocaleString();
    } catch {
        return '-';
    }
}

function formatBytes(bytes) {
    if (bytes === null || bytes === undefined || bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatNumber(num) {
    if (num === null || num === undefined) return '0';
    if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
    if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
    return num.toString();
}

function formatDuration(secs) {
    if (!secs || secs === 0) return '0s';
    if (secs < 60) return `${secs}s`;
    if (secs < 3600) return `${Math.floor(secs/60)}m ${secs%60}s`;
    const hours = Math.floor(secs / 3600);
    const mins = Math.floor((secs % 3600) / 60);
    return `${hours}h ${mins}m`;
}

function typeEmoji(type) {
    const emojis = {
        builder: '🏗️',
        worker: '👷',
        reviewer: '🔍',
        planner: '📋',
        tester: '🧪',
        docs: '📝',
        architect: '🏛️',
        engineer: '⚙️',
        custom: '🤖'
    };
    return emojis[type] || '🤖';
}
