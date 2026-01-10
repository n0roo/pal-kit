// PAL Kit Dashboard App

const API_BASE = '';
let refreshInterval;

// Initialize
document.addEventListener('DOMContentLoaded', () => {
    initTabs();
    initRefresh();
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
        loadSessions(),
        loadPorts(),
        loadPipelines(),
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
    setStatValue('pipelines-running', data.pipelines?.running ?? 0);
    setStatValue('docs-total', data.docs?.total ?? 0);
    setStatValue('conventions-enabled', data.conventions?.enabled ?? 0);
    setStatValue('locks-active', data.locks?.active ?? 0);
    setStatValue('escalations-open', data.escalations?.open ?? 0);
    
    document.getElementById('project-root').textContent = data.project_root || '';
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
        tbody.innerHTML = '<tr><td colspan="5" class="empty-state">No sessions</td></tr>';
        return;
    }
    
    tbody.innerHTML = data.map(s => `
        <tr>
            <td>${statusBadge(s.status || 'unknown')}</td>
            <td class="text-sm">${escapeHtml(s.id || '-')}</td>
            <td>${escapeHtml(getNullableString(s.title))}</td>
            <td>${escapeHtml(getNullableString(s.port_id))}</td>
            <td class="text-sm muted">${formatDate(s.started_at)}</td>
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
            <td>${escapeHtml(getNullableString(p.title))}</td>
            <td>${escapeHtml(getNullableString(p.pipeline_id))}</td>
            <td class="text-sm muted">${formatDate(p.created_at)}</td>
        </tr>
    `).join('');
}

// Pipelines
async function loadPipelines() {
    const data = await fetchAPI('pipelines');
    const tbody = document.getElementById('pipelines-table');
    
    if (!data || data.length === 0) {
        tbody.innerHTML = '<tr><td colspan="4" class="empty-state">No pipelines</td></tr>';
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

/**
 * Get value from Go's sql.NullString (or similar nullable types)
 * Handles: {String: "value", Valid: true} or plain string or null
 */
function getNullableString(field) {
    if (field === null || field === undefined) return '-';
    if (typeof field === 'string') return field || '-';
    if (typeof field === 'object') {
        // Go's sql.NullString format
        if (field.Valid === true && field.String !== undefined) {
            return field.String || '-';
        }
        // Go's sql.NullInt64, sql.NullTime, etc.
        if (field.Valid === true && field.Int64 !== undefined) {
            return String(field.Int64);
        }
        if (field.Valid === true && field.Time !== undefined) {
            return field.Time;
        }
    }
    return '-';
}

/**
 * Escape HTML to prevent XSS
 */
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
    
    // Handle Go's sql.NullTime
    if (typeof dateStr === 'object') {
        if (dateStr.Valid === true && dateStr.Time) {
            dateStr = dateStr.Time;
        } else {
            return '-';
        }
    }
    
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
    
    // Handle nullable
    if (typeof bytes === 'object' && bytes.Valid !== undefined) {
        bytes = bytes.Valid ? (bytes.Int64 || 0) : 0;
    }
    
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
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
