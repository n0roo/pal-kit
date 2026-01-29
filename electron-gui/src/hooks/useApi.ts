import { useState, useEffect, useCallback } from 'react'

// ============================================
// Types
// ============================================

export interface ApiStatus {
  orchestrations?: { total: number; running: number }
  builds?: { total: number; active: number }
  agents?: { total: number }
}

export interface Session {
  id: string
  type: string
  title: string
  status: string
  parent_id?: string
  port_id?: string
  depth?: number
  children?: Session[]
}

export interface HierarchicalSession {
  session: Session
  children?: HierarchicalSession[]
}

export interface Orchestration {
  id: string
  title: string
  description?: string
  status: string
  progress_percent: number
  ports?: any[]
  created_at: string
}

export interface Agent {
  id: string
  name: string
  type: string
  description?: string
  capabilities?: string[]
  current_version: number
  is_system?: boolean
  created_at?: string
}

export interface AttentionState {
  session_id: string
  port_id?: string
  token_budget: number
  loaded_tokens: number
  focus_score: number
  drift_score: number
  loaded_files?: string[]
  loaded_conventions?: string[]
}

// ============================================
// API Client (Web or Electron)
// ============================================

const API_BASE = '/api/v2'

// Check if running in Electron
const isElectron = () => typeof window !== 'undefined' && window.pal !== undefined

// Generic fetch wrapper for web mode
async function webFetch<T>(endpoint: string, options?: { method?: string; body?: any }): Promise<{ data: T | null; error: string | null }> {
  try {
    const res = await fetch(`${API_BASE}${endpoint}`, {
      method: options?.method || 'GET',
      headers: options?.body ? { 'Content-Type': 'application/json' } : undefined,
      body: options?.body ? JSON.stringify(options.body) : undefined,
    })
    
    if (!res.ok) {
      return { data: null, error: `HTTP ${res.status}` }
    }
    
    const data = await res.json()
    return { data, error: null }
  } catch (err: any) {
    return { data: null, error: err.message }
  }
}

// Unified request function
async function apiRequest<T>(endpoint: string, options?: { method?: string; body?: any }): Promise<{ data: T | null; error: string | null }> {
  if (isElectron()) {
    return window.pal!.request(endpoint, options)
  }
  return webFetch<T>(endpoint, options)
}

// ============================================
// Helper to ensure array
// ============================================

function ensureArray<T>(data: any): T[] {
  if (Array.isArray(data)) return data
  if (data === null || data === undefined) return []
  if (typeof data === 'object') {
    if (Array.isArray(data.items)) return data.items
    if (Array.isArray(data.data)) return data.data
    if (Array.isArray(data.list)) return data.list
  }
  return []
}

// ============================================
// Hooks
// ============================================

export function useApi() {
  const [status, setStatus] = useState<ApiStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchStatus = useCallback(async () => {
    try {
      const res = await apiRequest<ApiStatus>('/status')
      if (res.error) {
        setError(res.error)
        setStatus(null)
      } else {
        setStatus(res.data)
        setError(null)
      }
    } catch (err: any) {
      setError(err.message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchStatus()
    const interval = setInterval(fetchStatus, 10000)
    return () => clearInterval(interval)
  }, [fetchStatus])

  return { status, loading, error, refetch: fetchStatus }
}

export function useSessions() {
  const [sessions, setSessions] = useState<HierarchicalSession[]>([])
  const [loading, setLoading] = useState(true)

  const fetchSessions = useCallback(async () => {
    setLoading(true)
    try {
      const res = await apiRequest<HierarchicalSession[]>('/sessions/hierarchy')
      setSessions(ensureArray(res.data))
    } catch {
      setSessions([])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchHierarchy = useCallback(async (rootId: string): Promise<HierarchicalSession | null> => {
    try {
      const res = await apiRequest<HierarchicalSession>(`/sessions/hierarchy/${rootId}/tree`)
      return res.data
    } catch {
      return null
    }
  }, [])

  useEffect(() => {
    fetchSessions()
  }, [fetchSessions])

  return { sessions, loading, fetchSessions, fetchHierarchy }
}

export function useOrchestrations() {
  const [orchestrations, setOrchestrations] = useState<Orchestration[]>([])
  const [loading, setLoading] = useState(true)

  const fetchOrchestrations = useCallback(async (status?: string) => {
    setLoading(true)
    try {
      const query = status ? `?status=${status}` : ''
      const res = await apiRequest<Orchestration[]>(`/orchestrations${query}`)
      setOrchestrations(ensureArray<Orchestration>(res.data))
    } catch (err) {
      console.error('Failed to fetch orchestrations:', err)
      setOrchestrations([])
    } finally {
      setLoading(false)
    }
  }, [])

  const getOrchestration = useCallback(async (id: string): Promise<Orchestration | null> => {
    try {
      const res = await apiRequest<Orchestration>(`/orchestrations/${id}`)
      return res.data
    } catch {
      return null
    }
  }, [])

  const getStats = useCallback(async (id: string): Promise<any> => {
    try {
      const res = await apiRequest(`/orchestrations/${id}/stats`)
      return res.data
    } catch {
      return null
    }
  }, [])

  useEffect(() => {
    fetchOrchestrations()
  }, [fetchOrchestrations])

  const createOrchestration = useCallback(async (title: string, description: string, ports: { port_id: string; order: number; depends_on?: string[] }[]): Promise<Orchestration | null> => {
    try {
      const res = await apiRequest<Orchestration>('/orchestrations', {
        method: 'POST',
        body: { title, description, ports },
      })
      if (res.data) {
        await fetchOrchestrations()
      }
      return res.data
    } catch {
      return null
    }
  }, [fetchOrchestrations])

  return { orchestrations, loading, fetchOrchestrations, getOrchestration, getStats, createOrchestration }
}

export function useAgents() {
  const [agents, setAgents] = useState<Agent[]>([])
  const [loading, setLoading] = useState(true)

  const fetchAgents = useCallback(async (type?: string, includeSystem = true) => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (type) params.set('type', type)
      if (includeSystem) params.set('include_system', 'true')
      const query = params.toString() ? `?${params.toString()}` : ''
      const res = await apiRequest<Agent[]>(`/agents${query}`)
      setAgents(ensureArray<Agent>(res.data))
    } catch {
      setAgents([])
    } finally {
      setLoading(false)
    }
  }, [])

  const getAgent = useCallback(async (id: string): Promise<Agent | null> => {
    try {
      const res = await apiRequest<Agent>(`/agents/${id}`)
      return res.data
    } catch {
      return null
    }
  }, [])

  const getSpec = useCallback(async (agentId: string): Promise<string | null> => {
    try {
      const res = await apiRequest<{ id: string; version: number; content: string }>(`/agents/${agentId}/spec`)
      return res.data?.content || null
    } catch {
      return null
    }
  }, [])

  const getVersions = useCallback(async (agentId: string): Promise<any[]> => {
    try {
      const res = await apiRequest(`/agents/${agentId}/versions`)
      return ensureArray(res.data)
    } catch {
      return []
    }
  }, [])

  const compareVersions = useCallback(async (agentId: string, v1: number, v2: number): Promise<any> => {
    try {
      const res = await apiRequest(`/agents/${agentId}/compare?v1=${v1}&v2=${v2}`)
      return res.data
    } catch {
      return null
    }
  }, [])

  useEffect(() => {
    fetchAgents()
  }, [fetchAgents])

  return { agents, loading, fetchAgents, getAgent, getSpec, getVersions, compareVersions }
}

export function useAttention(sessionId?: string) {
  const [attention, setAttention] = useState<AttentionState | null>(null)
  const [loading, setLoading] = useState(false)

  const fetchAttention = useCallback(async (id: string) => {
    setLoading(true)
    try {
      const res = await apiRequest<{ attention: AttentionState }>(`/attention/${id}`)
      setAttention(res.data?.attention || null)
    } catch {
      setAttention(null)
    } finally {
      setLoading(false)
    }
  }, [])

  const getReport = useCallback(async (id: string): Promise<any> => {
    try {
      const res = await apiRequest(`/attention/${id}/report`)
      return res.data
    } catch {
      return null
    }
  }, [])

  const getHistory = useCallback(async (id: string, limit = 10): Promise<any[]> => {
    try {
      const res = await apiRequest(`/attention/${id}/history?limit=${limit}`)
      return ensureArray(res.data)
    } catch {
      return []
    }
  }, [])

  useEffect(() => {
    if (sessionId) fetchAttention(sessionId)
  }, [sessionId, fetchAttention])

  return { attention, loading, fetchAttention, getReport, getHistory }
}

// App utilities hook (Electron only, fallback for web)
export function useApp() {
  const [serverRunning, setServerRunning] = useState(true) // Assume running in web mode
  const [serverPort, setServerPort] = useState(9000)

  const checkServer = useCallback(async () => {
    if (isElectron() && window.app) {
      const [running, port] = await Promise.all([
        window.app.isServerRunning(),
        window.app.getServerPort(),
      ])
      setServerRunning(running)
      setServerPort(port)
    } else {
      // Web mode: check by pinging status endpoint
      try {
        const res = await fetch('/api/v2/status')
        setServerRunning(res.ok)
      } catch {
        setServerRunning(false)
      }
    }
  }, [])

  const restartServer = useCallback(async () => {
    if (isElectron() && window.app) {
      const success = await window.app.restartServer()
      await checkServer()
      return success
    }
    return false
  }, [checkServer])

  useEffect(() => {
    checkServer()
    const interval = setInterval(checkServer, 5000)
    return () => clearInterval(interval)
  }, [checkServer])

  return { serverRunning, serverPort, checkServer, restartServer }
}

// Document types
export interface Document {
  id: string
  path: string
  type: string
  domain?: string
  status: string
  priority?: string
  tokens: number
  tags?: string[]
  summary?: string
  created_at: string
  updated_at: string
}

export interface DocumentStats {
  total_docs: number
  total_tokens: number
  by_type: Record<string, number>
  by_status: Record<string, number>
  by_domain: Record<string, number>
}

export interface DocumentFilters {
  q?: string
  type?: string
  domain?: string
  status?: string
  tag?: string
  limit?: number
}

// Documents hook
export function useDocuments() {
  const [documents, setDocuments] = useState<Document[]>([])
  const [stats, setStats] = useState<DocumentStats | null>(null)
  const [loading, setLoading] = useState(true)
  const [indexing, setIndexing] = useState(false)

  const fetchDocuments = useCallback(async (filters?: DocumentFilters) => {
    setLoading(true)
    try {
      const params = new URLSearchParams()
      if (filters?.q) params.set('q', filters.q)
      if (filters?.type) params.set('type', filters.type)
      if (filters?.domain) params.set('domain', filters.domain)
      if (filters?.status) params.set('status', filters.status)
      if (filters?.tag) params.set('tag', filters.tag)
      if (filters?.limit) params.set('limit', String(filters.limit))

      const query = params.toString() ? `?${params.toString()}` : ''
      const res = await apiRequest<Document[]>(`/documents${query}`)
      setDocuments(ensureArray<Document>(res.data))
    } catch {
      setDocuments([])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchStats = useCallback(async () => {
    try {
      const res = await apiRequest<DocumentStats>('/documents/stats')
      setStats(res.data)
    } catch {
      setStats(null)
    }
  }, [])

  const getDocument = useCallback(async (id: string): Promise<Document | null> => {
    try {
      const res = await apiRequest<Document>(`/documents/${id}`)
      return res.data
    } catch {
      return null
    }
  }, [])

  const getContent = useCallback(async (id: string): Promise<string | null> => {
    try {
      const res = await apiRequest<{ id: string; content: string }>(`/documents/${id}/content`)
      return res.data?.content || null
    } catch {
      return null
    }
  }, [])

  const reindex = useCallback(async () => {
    setIndexing(true)
    try {
      const res = await apiRequest<{ added: number; updated: number; removed: number }>(
        '/documents/index',
        { method: 'POST' }
      )
      await fetchDocuments()
      await fetchStats()
      return res.data
    } catch {
      return null
    } finally {
      setIndexing(false)
    }
  }, [fetchDocuments, fetchStats])

  useEffect(() => {
    fetchDocuments()
    fetchStats()
  }, [fetchDocuments, fetchStats])

  return { documents, stats, loading, indexing, fetchDocuments, fetchStats, getDocument, getContent, reindex }
}

// ============================================
// Global Agents
// ============================================

export interface GlobalAgentInfo {
  path: string
  name: string
  type: string
  category: string
  description?: string
  has_rules: boolean
  modified_at: string
  size: number
}

export interface GlobalManifest {
  version: string
  initialized_at: string
  last_updated: string
  embedded_hash?: string
  custom_agents?: string[]
  overrides?: Record<string, string>
}

export function useGlobalAgents() {
  const [agents, setAgents] = useState<GlobalAgentInfo[]>([])
  const [skills, setSkills] = useState<GlobalAgentInfo[]>([])
  const [conventions, setConventions] = useState<GlobalAgentInfo[]>([])
  const [manifest, setManifest] = useState<GlobalManifest | null>(null)
  const [globalPath, setGlobalPath] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [syncing, setSyncing] = useState(false)

  const fetchAgents = useCallback(async () => {
    setLoading(true)
    try {
      const [agentsRes, skillsRes, conventionsRes] = await Promise.all([
        apiRequest<GlobalAgentInfo[]>('/agents/global'),
        apiRequest<GlobalAgentInfo[]>('/agents/global?type=skills'),
        apiRequest<GlobalAgentInfo[]>('/agents/global?type=conventions'),
      ])
      setAgents(ensureArray(agentsRes.data))
      setSkills(ensureArray(skillsRes.data))
      setConventions(ensureArray(conventionsRes.data))
    } catch {
      setAgents([])
      setSkills([])
      setConventions([])
    } finally {
      setLoading(false)
    }
  }, [])

  const fetchManifest = useCallback(async () => {
    try {
      const res = await apiRequest<GlobalManifest>('/agents/global/manifest')
      setManifest(res.data)
    } catch {
      setManifest(null)
    }
  }, [])

  const fetchGlobalPath = useCallback(async () => {
    try {
      const res = await apiRequest<{ path: string }>('/agents/global/path')
      setGlobalPath(res.data?.path || '')
    } catch {
      setGlobalPath('')
    }
  }, [])

  const getContent = useCallback(async (path: string): Promise<string | null> => {
    try {
      const res = await apiRequest<{ path: string; content: string }>(`/agents/global/${path}`)
      return res.data?.content || null
    } catch {
      return null
    }
  }, [])

  const updateContent = useCallback(async (path: string, content: string): Promise<boolean> => {
    try {
      const res = await apiRequest(`/agents/global/${path}`, {
        method: 'PUT',
        body: { content },
      })
      return !res.error
    } catch {
      return false
    }
  }, [])

  const initialize = useCallback(async (force = false): Promise<boolean> => {
    try {
      const res = await apiRequest(`/agents/global?action=init${force ? '&force=true' : ''}`, {
        method: 'POST',
      })
      if (!res.error) {
        await fetchAgents()
        await fetchManifest()
      }
      return !res.error
    } catch {
      return false
    }
  }, [fetchAgents, fetchManifest])

  const syncToProject = useCallback(async (projectRoot?: string, forceOverwrite = false): Promise<{ count: number } | null> => {
    setSyncing(true)
    try {
      const res = await apiRequest<{ status: string; count: number }>('/agents/global?action=sync', {
        method: 'POST',
        body: { project_root: projectRoot, force_overwrite: forceOverwrite },
      })
      return res.data ? { count: res.data.count } : null
    } catch {
      return null
    } finally {
      setSyncing(false)
    }
  }, [])

  const reset = useCallback(async (): Promise<boolean> => {
    try {
      const res = await apiRequest('/agents/global?action=reset', { method: 'POST' })
      if (!res.error) {
        await fetchAgents()
        await fetchManifest()
      }
      return !res.error
    } catch {
      return false
    }
  }, [fetchAgents, fetchManifest])

  useEffect(() => {
    fetchAgents()
    fetchManifest()
    fetchGlobalPath()
  }, [fetchAgents, fetchManifest, fetchGlobalPath])

  return {
    agents,
    skills,
    conventions,
    manifest,
    globalPath,
    loading,
    syncing,
    fetchAgents,
    fetchManifest,
    getContent,
    updateContent,
    initialize,
    syncToProject,
    reset,
  }
}

// ============================================
// Document Tree Hook
// ============================================

export interface DocumentTreeNode {
  name: string
  path: string
  type: 'file' | 'directory'
  doc_type?: string
  children?: DocumentTreeNode[]
}

export function useDocumentTree(root: string = '.', depth: number = 3) {
  const [tree, setTree] = useState<DocumentTreeNode | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchTree = useCallback(async (newRoot?: string, newDepth?: number) => {
    const targetRoot = newRoot ?? root
    const targetDepth = newDepth ?? depth

    setLoading(true)
    setError(null)

    try {
      const params = new URLSearchParams({
        root: targetRoot,
        depth: String(targetDepth),
      })
      const res = await apiRequest(`/documents/tree?${params}`)

      if (res.error) {
        setError(res.error)
        setTree(null)
      } else {
        setTree(res.data as DocumentTreeNode)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch tree')
      setTree(null)
    } finally {
      setLoading(false)
    }
  }, [root, depth])

  useEffect(() => {
    fetchTree()
  }, [fetchTree])

  return { tree, loading, error, fetchTree }
}
