import { useState, useEffect, useCallback } from 'react'

const API_BASE = 'http://localhost:9000/api/v2/projects'

export interface Project {
  root: string
  name: string
  description?: string
  last_active?: string
  session_count: number
  port_count: number
  active_ports: number
  total_tokens: number
  created_at?: string
  initialized: boolean
}

export function useProjects() {
  const [projects, setProjects] = useState<Project[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchProjects = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(API_BASE)
      if (!res.ok) throw new Error('Failed to fetch projects')
      const data = await res.json()
      setProjects(data || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      setProjects([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchProjects()
  }, [fetchProjects])

  const importProject = useCallback(async (path: string, name?: string): Promise<Project | null> => {
    try {
      const res = await fetch(`${API_BASE}/import`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path, name }),
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || 'Failed to import project')
      }
      await fetchProjects()
      return await res.json()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      return null
    }
  }, [fetchProjects])

  const initProject = useCallback(async (path: string, name?: string): Promise<Project | null> => {
    try {
      const res = await fetch(`${API_BASE}/init`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path, name }),
      })
      if (!res.ok) {
        const data = await res.json()
        throw new Error(data.error || 'Failed to initialize project')
      }
      await fetchProjects()
      return await res.json()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      return null
    }
  }, [fetchProjects])

  const removeProject = useCallback(async (root: string): Promise<boolean> => {
    try {
      const res = await fetch(`${API_BASE}/${encodeURIComponent(root)}`, {
        method: 'DELETE',
      })
      if (!res.ok) throw new Error('Failed to remove project')
      await fetchProjects()
      return true
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      return false
    }
  }, [fetchProjects])

  const getProject = useCallback(async (root: string): Promise<Project | null> => {
    try {
      const res = await fetch(`${API_BASE}/${encodeURIComponent(root)}`)
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  }, [])

  return {
    projects,
    loading,
    error,
    fetchProjects,
    importProject,
    initProject,
    removeProject,
    getProject,
  }
}

export default useProjects
