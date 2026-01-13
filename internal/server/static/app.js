// PAL Kit Dashboard App

const API_BASE = '';
let refreshInterval;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initTabs();
    initRefresh();
    initModal();
    initDocModal();
    initProjectModal();
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

// Document Modal
let currentDocPath = '';
let currentDocContent = '';

function initDocModal() {
    const modal = document.getElementById('doc-modal');
    const closeBtn = modal.querySelector('.modal-close');

    closeBtn.addEventListener('click', () => modal.classList.add('hidden'));
    modal.addEventListener('click', (e) => {
        if (e.target === modal) modal.classList.add('hidden');
    });
}

// Show document in viewer
async function showDocViewer(path) {
    const modal = document.getElementById('doc-modal');
    const title = document.getElementById('doc-modal-title');
    const body = document.getElementById('doc-modal-body');

    title.textContent = path;
    body.innerHTML = '<div class="empty-state">Loading...</div>';
    modal.classList.remove('hidden');

    try {
        const data = await fetchAPI(`docs/content?path=${encodeURIComponent(path)}`);
        if (!data || !data.content) {
            body.innerHTML = '<div class="empty-state">Failed to load document</div>';
            return;
        }

        currentDocPath = path;
        currentDocContent = data.content;

        // Determine file type and render
        const ext = path.split('.').pop().toLowerCase();
        if (ext === 'md') {
            // Markdown rendering
            if (typeof marked !== 'undefined') {
                body.innerHTML = marked.parse(data.content);
                // Add copy buttons to code blocks
                addCodeBlockCopyButtons(body);
            } else {
                body.innerHTML = `<pre class="yaml-viewer">${escapeHtml(data.content)}</pre>`;
            }
        } else if (ext === 'yaml' || ext === 'yml') {
            // YAML - syntax highlighted
            body.innerHTML = `<pre class="yaml-viewer">${highlightYaml(data.content)}</pre>`;
        } else {
            // Plain text
            body.innerHTML = `<pre class="yaml-viewer">${escapeHtml(data.content)}</pre>`;
        }
    } catch (err) {
        body.innerHTML = `<div class="empty-state">Error: ${escapeHtml(err.message)}</div>`;
    }
}

// Add copy buttons to code blocks
function addCodeBlockCopyButtons(container) {
    const preBlocks = container.querySelectorAll('pre');
    preBlocks.forEach(pre => {
        const wrapper = document.createElement('div');
        wrapper.className = 'code-block-wrapper';
        pre.parentNode.insertBefore(wrapper, pre);
        wrapper.appendChild(pre);

        const btn = document.createElement('button');
        btn.className = 'code-copy-btn';
        btn.textContent = 'Copy';
        btn.onclick = () => {
            const code = pre.querySelector('code') || pre;
            navigator.clipboard.writeText(code.textContent).then(() => {
                btn.textContent = 'Copied!';
                setTimeout(() => btn.textContent = 'Copy', 2000);
            });
        };
        wrapper.appendChild(btn);
    });
}

// Simple YAML syntax highlighting
function highlightYaml(content) {
    return escapeHtml(content)
        .replace(/^(\s*)([\w-]+):/gm, '$1<span style="color:#7C3AED">$2</span>:')
        .replace(/:\s*(&quot;[^&]*&quot;|&#39;[^&]*&#39;)/g, ': <span style="color:#10B981">$1</span>')
        .replace(/:\s*(\d+)/g, ': <span style="color:#F59E0B">$1</span>')
        .replace(/:\s*(true|false)/gi, ': <span style="color:#F59E0B">$1</span>')
        .replace(/#.*$/gm, '<span style="color:#6B7280">$&</span>');
}

// Copy entire document
function copyDocument() {
    if (!currentDocContent) {
        showToast('No document loaded', true);
        return;
    }

    navigator.clipboard.writeText(currentDocContent).then(() => {
        showToast('Document copied to clipboard');
    }).catch(() => {
        showToast('Failed to copy', true);
    });
}

// Download document
function downloadDocument() {
    if (!currentDocPath || !currentDocContent) {
        showToast('No document loaded', true);
        return;
    }

    const filename = currentDocPath.split('/').pop();
    const blob = new Blob([currentDocContent], { type: 'text/plain' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);

    showToast(`Downloaded ${filename}`);
}

// Show toast notification
function showToast(message, isError = false) {
    const existing = document.querySelector('.toast');
    if (existing) existing.remove();

    const toast = document.createElement('div');
    toast.className = `toast${isError ? ' error' : ''}`;
    toast.textContent = message;
    document.body.appendChild(toast);

    setTimeout(() => toast.remove(), 3000);
}

// Project Modal
function initProjectModal() {
    const modal = document.getElementById('project-modal');
    const closeBtn = modal.querySelector('.modal-close');

    closeBtn.addEventListener('click', () => modal.classList.add('hidden'));
    modal.addEventListener('click', (e) => {
        if (e.target === modal) modal.classList.add('hidden');
    });
}

// Show project detail
async function showProjectDetail(root) {
    const modal = document.getElementById('project-modal');
    const title = document.getElementById('project-modal-title');
    const body = document.getElementById('project-modal-body');

    title.textContent = 'Loading...';
    body.innerHTML = '<div class="empty-state">Loading project details...</div>';
    modal.classList.remove('hidden');

    try {
        const data = await fetchAPI(`projects/detail?root=${encodeURIComponent(root)}`);
        if (!data) {
            body.innerHTML = '<div class="empty-state">Failed to load project</div>';
            return;
        }

        title.textContent = data.name || 'Unknown Project';

        // Build detail HTML
        let configSection = '';
        if (data.config) {
            const agents = data.config.agents || {};
            const settings = data.config.settings || {};
            configSection = `
                <h4>Configuration</h4>
                <div class="detail-grid">
                    <div class="detail-item">
                        <label>Version</label>
                        <span>${escapeHtml(data.config.version || '-')}</span>
                    </div>
                    <div class="detail-item">
                        <label>Workflow</label>
                        <span class="workflow-badge">${escapeHtml(data.config.workflow || '-')}</span>
                    </div>
                    <div class="detail-item full-width">
                        <label>Core Agents</label>
                        <span>${(agents.core || []).map(a => `<span class="tool-tag">${escapeHtml(a)}</span>`).join('') || '-'}</span>
                    </div>
                    <div class="detail-item full-width">
                        <label>Workers</label>
                        <span>${(agents.workers || []).length > 0 ? (agents.workers || []).map(a => `<span class="tool-tag">${escapeHtml(a)}</span>`).join('') : '-'}</span>
                    </div>
                </div>
            `;
        }

        let sessionsSection = '';
        if (data.sessions && data.sessions.length > 0) {
            sessionsSection = `
                <h4>Recent Sessions (${data.sessions.length})</h4>
                <div class="children-list">
                    ${data.sessions.map(s => `
                        <div class="child-item" onclick="showSessionDetail('${s.id}'); document.getElementById('project-modal').classList.add('hidden');">
                            ${statusBadge(s.status)}
                            <span>${escapeHtml(s.id)}</span>
                            <span class="muted">${escapeHtml(s.title || '')}</span>
                            <span class="muted text-sm">$${(s.cost || 0).toFixed(4)}</span>
                        </div>
                    `).join('')}
                </div>
            `;
        }

        let portsSection = '';
        if (data.ports && data.ports.length > 0) {
            portsSection = `
                <h4>Ports (${data.ports.length})</h4>
                <div class="children-list">
                    ${data.ports.map(p => `
                        <div class="child-item">
                            ${statusBadge(p.status)}
                            <span>${escapeHtml(p.id)}</span>
                            <span class="muted">${escapeHtml(p.title || '')}</span>
                        </div>
                    `).join('')}
                </div>
            `;
        }

        body.innerHTML = `
            <div class="detail-grid">
                <div class="detail-item full-width">
                    <label>Path</label>
                    <span class="text-sm">${escapeHtml(data.root)}</span>
                </div>
                <div class="detail-item">
                    <label>Description</label>
                    <span>${escapeHtml(data.description || '-')}</span>
                </div>
                <div class="detail-item">
                    <label>Last Active</label>
                    <span>${formatDate(data.last_active)}</span>
                </div>
                <div class="detail-item">
                    <label>Sessions</label>
                    <span>${data.session_count || 0}</span>
                </div>
                <div class="detail-item">
                    <label>Total Tokens</label>
                    <span>${formatNumber(data.total_tokens || 0)}</span>
                </div>
                <div class="detail-item highlight">
                    <label>Total Cost</label>
                    <span>$${(data.total_cost || 0).toFixed(4)}</span>
                </div>
                <div class="detail-item">
                    <label>Created</label>
                    <span>${formatDate(data.created_at)}</span>
                </div>
            </div>
            ${configSection}
            ${sessionsSection}
            ${portsSection}
        `;
    } catch (err) {
        body.innerHTML = `<div class="empty-state">Error: ${escapeHtml(err.message)}</div>`;
    }
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
        loadProjects(),
        loadSessions(),
        loadSessionTree(),
        loadHistory(),
        loadPorts(),
        loadPortProgress(),
        loadPortFlow(),
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

// Projects
let allProjects = [];

async function loadProjects() {
    const data = await fetchAPI('projects');
    const grid = document.getElementById('projects-grid');

    if (!data || data.length === 0) {
        grid.innerHTML = '<div class="empty-state"><div class="icon">📁</div><p>No projects registered</p></div>';
        allProjects = [];
        return;
    }

    allProjects = data;
    renderProjects(data);
}

function renderProjects(projects) {
    const grid = document.getElementById('projects-grid');

    if (!projects || projects.length === 0) {
        grid.innerHTML = '<div class="empty-state"><div class="icon">📁</div><p>No projects found</p></div>';
        return;
    }

    grid.innerHTML = projects.map(p => `
        <div class="project-card" onclick="showProjectDetail('${escapeHtml(p.root)}')">
            <div class="project-header">
                <h3>📁 ${escapeHtml(p.name || 'Unknown')}</h3>
                <span class="muted text-sm">${formatDate(p.last_active)}</span>
            </div>
            <p class="project-desc">${escapeHtml(p.description || 'No description')}</p>
            <div class="project-stats">
                <div class="project-stat">
                    <span class="stat-icon">📊</span>
                    <span>${p.session_count || 0} sessions</span>
                </div>
                <div class="project-stat">
                    <span class="stat-icon">🔢</span>
                    <span>${formatNumber(p.total_tokens || 0)} tokens</span>
                </div>
                <div class="project-stat highlight">
                    <span class="stat-icon">💰</span>
                    <span>$${(p.total_cost || 0).toFixed(2)}</span>
                </div>
            </div>
            <div class="project-path">${escapeHtml(p.root)}</div>
        </div>
    `).join('');
}

function filterProjects(query) {
    if (!query) {
        renderProjects(allProjects);
        return;
    }

    const lowerQuery = query.toLowerCase();
    const filtered = allProjects.filter(p => {
        const name = (p.name || '').toLowerCase();
        const desc = (p.description || '').toLowerCase();
        const root = (p.root || '').toLowerCase();
        return name.includes(lowerQuery) || desc.includes(lowerQuery) || root.includes(lowerQuery);
    });

    renderProjects(filtered);
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

// Session Tree View
async function loadSessionTree() {
    const container = document.getElementById('session-tree');
    if (!container) return;

    const data = await fetchAPI('sessions/tree');

    if (!data || !data.sessions || data.sessions.length === 0) {
        container.innerHTML = '<div class="empty-state">No session hierarchies</div>';
        return;
    }

    container.innerHTML = data.sessions.map(session => renderTreeNode(session)).join('');
}

function renderTreeNode(node) {
    const icon = getAgentIcon(node.agent || node.session_type);
    const statusClass = getStatusClass(node.status);
    const hasChildren = node.children && node.children.length > 0;

    let html = `
        <div class="tree-node">
            <div class="tree-item" onclick="showSessionDetail('${node.id}')">
                <span class="tree-icon">${icon}</span>
                <span class="tree-name">${escapeHtml(node.name || node.id)}</span>
                <span class="tree-status ${statusClass}">${escapeHtml(node.status)}</span>
            </div>
    `;

    if (hasChildren) {
        html += '<div class="tree-children">';
        html += node.children.map(child => renderTreeNode(child)).join('');
        html += '</div>';
    }

    html += '</div>';
    return html;
}

function getAgentIcon(agent) {
    const icons = {
        'builder': '📂',
        'planner': '📋',
        'architect': '🏗️',
        'worker': '⚙️',
        'tester': '🧪',
        'support': '📚',
        'single': '📍',
        'main': '📂'
    };
    return icons[agent] || '📍';
}

function getStatusClass(status) {
    if (status === 'active' || status === 'running') return 'active';
    if (status === 'complete' || status === 'done') return 'complete';
    return '';
}

// Port Progress Dashboard
async function loadPortProgress() {
    const data = await fetchAPI('ports/progress');

    const completedEl = document.getElementById('ports-completed');
    const inProgressEl = document.getElementById('ports-in-progress');
    const pendingEl = document.getElementById('ports-pending');

    if (!completedEl || !inProgressEl || !pendingEl) return;

    if (!data) {
        completedEl.innerHTML = '<div class="empty-state">-</div>';
        inProgressEl.innerHTML = '<div class="empty-state">-</div>';
        pendingEl.innerHTML = '<div class="empty-state">-</div>';
        return;
    }

    completedEl.innerHTML = renderProgressItems(data.completed || []);
    inProgressEl.innerHTML = renderProgressItems(data.in_progress || []);
    pendingEl.innerHTML = renderProgressItems(data.pending || []);
}

function renderProgressItems(items) {
    if (!items || items.length === 0) {
        return '<div class="empty-state text-sm">None</div>';
    }

    return items.map(item => `
        <div class="progress-item" onclick="showPortDetail('${item.id}')">
            <span class="port-id">${escapeHtml(item.id)}</span>
            ${item.title ? `<span class="port-title">${escapeHtml(item.title)}</span>` : ''}
        </div>
    `).join('');
}

// Port Flow Diagram
async function loadPortFlow() {
    const container = document.getElementById('port-flow');
    if (!container) return;

    const data = await fetchAPI('ports/flow');

    if (!data || !data.ports || data.ports.length === 0) {
        container.innerHTML = '<div class="empty-state">No ports available</div>';
        return;
    }

    // Render as dependency list
    if (!data.dependencies || data.dependencies.length === 0) {
        container.innerHTML = '<div class="empty-state">No dependencies defined</div>';
        return;
    }

    container.innerHTML = `
        <div class="dep-list">
            ${data.dependencies.map(dep => `
                <div class="dep-item">
                    <span class="dep-from">${escapeHtml(dep.from)}</span>
                    <span class="dep-arrow">→</span>
                    <span class="dep-to">${escapeHtml(dep.to)}</span>
                </div>
            `).join('')}
        </div>
    `;
}

// Show port detail (placeholder)
function showPortDetail(portId) {
    // Could open a modal or navigate to port details
    console.log('Show port detail:', portId);
}

// History (Event Log)
let historyPage = 0;
let historyLimit = 50;
let historyTotal = 0;
let historySearchTimeout = null;

async function initHistoryFilters() {
    // Load event types
    const types = await fetchAPI('history/types');
    const typeSelect = document.getElementById('history-event-type');
    if (types && types.length > 0) {
        types.forEach(t => {
            const opt = document.createElement('option');
            opt.value = t;
            opt.textContent = t;
            typeSelect.appendChild(opt);
        });
    }

    // Load projects
    const projects = await fetchAPI('history/projects');
    const projectSelect = document.getElementById('history-project');
    if (projects && projects.length > 0) {
        projects.forEach(p => {
            const opt = document.createElement('option');
            opt.value = p;
            opt.textContent = p;
            projectSelect.appendChild(opt);
        });
    }
}

async function loadEventHistory() {
    const eventType = document.getElementById('history-event-type')?.value || '';
    const project = document.getElementById('history-project')?.value || '';
    const dateRange = document.getElementById('history-date-range')?.value || '7d';
    const search = document.getElementById('history-search')?.value || '';

    // Build query params
    const params = new URLSearchParams();
    if (eventType) params.set('event_type', eventType);
    if (project) params.set('project', project);
    if (search) params.set('search', search);
    params.set('limit', historyLimit);
    params.set('offset', historyPage * historyLimit);

    // Date range
    const now = new Date();
    if (dateRange === 'today') {
        const today = now.toISOString().split('T')[0];
        params.set('start_date', today);
    } else if (dateRange === '7d') {
        const d = new Date(now);
        d.setDate(d.getDate() - 7);
        params.set('start_date', d.toISOString().split('T')[0]);
    } else if (dateRange === '30d') {
        const d = new Date(now);
        d.setDate(d.getDate() - 30);
        params.set('start_date', d.toISOString().split('T')[0]);
    }

    const data = await fetchAPI(`history/events?${params.toString()}`);
    const tbody = document.getElementById('history-table');

    if (!data || !data.events || data.events.length === 0) {
        tbody.innerHTML = '<tr><td colspan="6" class="empty-state">No events found</td></tr>';
        document.getElementById('history-total').textContent = '0 events';
        updateHistoryPagination(0);
        return;
    }

    historyTotal = data.total || 0;
    document.getElementById('history-total').textContent = `${historyTotal} events`;
    updateHistoryPagination(historyTotal);

    tbody.innerHTML = data.events.map(e => `
        <tr>
            <td>${eventStatusBadge(e.status)}</td>
            <td class="text-sm">${formatTimeAgo(e.created_at)}</td>
            <td>${eventTypeIcon(e.event_type)} ${escapeHtml(e.event_type)}</td>
            <td class="text-sm">${e.session_id ? `<a href="#" onclick="showSessionDetail('${e.session_id}'); return false;">${escapeHtml(e.session_id.substring(0, 8))}...</a>` : '-'}</td>
            <td class="text-sm">${escapeHtml(e.project_name || '-')}</td>
            <td class="text-sm event-detail">${formatEventSummary(e)}</td>
        </tr>
    `).join('');
}

function eventStatusBadge(status) {
    const colors = {
        'success': 'complete',
        'error': 'error',
        'warning': 'warning',
        'info': 'active'
    };
    return `<span class="status"><span class="status-dot ${colors[status] || 'pending'}"></span>${escapeHtml(status)}</span>`;
}

function eventTypeIcon(type) {
    const icons = {
        'session_start': '🚀',
        'session_end': '🏁',
        'compact': '📦',
        'port_start': '▶️',
        'port_end': '✅',
        'error': '❌',
        'warning': '⚠️'
    };
    return icons[type] || '📝';
}

function formatEventSummary(event) {
    if (!event.parsed_data) return '-';
    const data = event.parsed_data;
    const parts = [];

    if (data.title) parts.push(escapeHtml(data.title));
    if (data.reason) parts.push(`reason: ${escapeHtml(data.reason)}`);
    if (data.duration) parts.push(`${escapeHtml(data.duration)}`);
    if (data.tokens) parts.push(`${formatNumber(data.tokens)} tokens`);
    if (data.message) parts.push(escapeHtml(data.message.substring(0, 50)));

    return parts.length > 0 ? parts.join(' | ') : '-';
}

function updateHistoryPagination(total) {
    const totalPages = Math.ceil(total / historyLimit);
    const pageInfo = document.getElementById('history-page-info');
    const prevBtn = document.getElementById('history-prev');
    const nextBtn = document.getElementById('history-next');

    pageInfo.textContent = `Page ${historyPage + 1} of ${totalPages || 1}`;
    prevBtn.disabled = historyPage === 0;
    nextBtn.disabled = historyPage >= totalPages - 1;
}

function historyPrevPage() {
    if (historyPage > 0) {
        historyPage--;
        loadEventHistory();
    }
}

function historyNextPage() {
    const totalPages = Math.ceil(historyTotal / historyLimit);
    if (historyPage < totalPages - 1) {
        historyPage++;
        loadEventHistory();
    }
}

function debounceHistorySearch(value) {
    if (historySearchTimeout) clearTimeout(historySearchTimeout);
    historySearchTimeout = setTimeout(() => {
        historyPage = 0;
        loadEventHistory();
    }, 300);
}

async function exportHistory(format) {
    const eventType = document.getElementById('history-event-type')?.value || '';
    const project = document.getElementById('history-project')?.value || '';
    const dateRange = document.getElementById('history-date-range')?.value || '7d';
    const search = document.getElementById('history-search')?.value || '';

    const params = new URLSearchParams();
    params.set('format', format);
    if (eventType) params.set('event_type', eventType);
    if (project) params.set('project', project);
    if (search) params.set('search', search);

    // Date range
    const now = new Date();
    if (dateRange === 'today') {
        params.set('start_date', now.toISOString().split('T')[0]);
    } else if (dateRange === '7d') {
        const d = new Date(now);
        d.setDate(d.getDate() - 7);
        params.set('start_date', d.toISOString().split('T')[0]);
    } else if (dateRange === '30d') {
        const d = new Date(now);
        d.setDate(d.getDate() - 30);
        params.set('start_date', d.toISOString().split('T')[0]);
    }

    try {
        const response = await fetch(`/api/history/export?${params.toString()}`);
        if (!response.ok) throw new Error('Export failed');

        const blob = await response.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `history-export.${format}`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);

        showToast(`Exported as ${format.toUpperCase()}`);
    } catch (err) {
        showToast('Export failed', true);
    }
}

// Legacy loadHistory for session daily summary (keep for reference)
async function loadHistory() {
    await initHistoryFilters();
    await loadEventHistory();
}

// Ports
async function loadPorts() {
    const data = await fetchAPI('ports');
    const tbody = document.getElementById('ports-table');

    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="8" class="empty-state">No ports</td></tr>';
        return;
    }

    tbody.innerHTML = data.map(p => {
        const totalTokens = (p.input_tokens || 0) + (p.output_tokens || 0);
        const tokensStr = totalTokens > 0 ? formatNumber(totalTokens) : '-';
        const costStr = p.cost_usd > 0 ? `$${p.cost_usd.toFixed(2)}` : '-';
        const durationStr = p.duration_str || (p.duration_secs > 0 ? formatDuration(p.duration_secs) : '-');

        return `
        <tr>
            <td>${statusBadge(p.status || 'unknown')}</td>
            <td class="text-sm">${escapeHtml(p.id || '-')}</td>
            <td>${escapeHtml(p.title || '-')}</td>
            <td>${escapeHtml(p.session_id || '-')}</td>
            <td class="text-sm">${durationStr}</td>
            <td class="text-sm">${tokensStr}</td>
            <td class="text-sm">${costStr}</td>
            <td class="text-sm muted">${formatDate(p.created_at)}</td>
        </tr>
    `}).join('');
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
let allDocs = [];

async function loadDocs() {
    const data = await fetchAPI('docs');
    const tbody = document.getElementById('docs-table');

    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No documents</td></tr>';
        allDocs = [];
        return;
    }

    allDocs = data;
    renderDocs(data);
}

function renderDocs(docs) {
    const tbody = document.getElementById('docs-table');

    if (!docs || docs.length === 0) {
        tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No documents found</td></tr>';
        return;
    }

    tbody.innerHTML = docs.map(d => `
        <tr onclick="showDocViewer('${escapeHtml(d.relative_path || d.path)}')" style="cursor:pointer">
            <td>${statusBadge(d.status || 'unknown')}</td>
            <td>${escapeHtml(d.relative_path || d.path || '-')}</td>
            <td>${escapeHtml(d.type || '-')}</td>
            <td class="text-sm muted">${formatBytes(d.size)}</td>
            <td class="text-sm muted">${formatDate(d.modified_at)}</td>
        </tr>
    `).join('');
}

function filterDocs(query) {
    if (!query) {
        renderDocs(allDocs);
        return;
    }

    const lowerQuery = query.toLowerCase();
    const filtered = allDocs.filter(d => {
        const path = (d.relative_path || d.path || '').toLowerCase();
        const type = (d.type || '').toLowerCase();
        return path.includes(lowerQuery) || type.includes(lowerQuery);
    });

    renderDocs(filtered);
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
