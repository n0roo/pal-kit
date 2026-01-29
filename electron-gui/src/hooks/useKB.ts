import { useState, useEffect, useCallback } from 'react'

const API_PATH = '/api/v2/kb'

// Check if running in Electron
const isElectron = () => typeof window !== 'undefined' && window.pal !== undefined

// Resolve API URL for fetch (handles Electron dynamic port)
async function resolveUrl(path: string): Promise<string> {
  if (isElectron() && window.app?.getServerPort) {
    const port = await window.app.getServerPort()
    if (port && port > 0) {
      return `http://localhost:${port}${path}`
    }
  }
  return path
}

// Types
export interface KBStatus {
  initialized: boolean
  vault_path: string
  version?: string
  created_at?: string
  sections?: Record<string, number>  // section name -> doc count
  error?: string
}

export interface KBTocItem {
  section: string
  exists: boolean
  valid?: boolean
  needs_refresh?: boolean
  missing_count?: number
  orphan_count?: number
}

export interface KBTocEntry {
  title: string
  path: string
  summary?: string
  depth: number  // Backend uses 'depth' not 'level'
  is_dir?: boolean
  children?: KBTocEntry[]
}

export interface KBDocument {
  id: string
  title: string
  path: string
  type: string
  status?: string
  tags?: string[]
  summary?: string
  created?: string
  updated?: string
}

export interface KBDocumentDetail extends KBDocument {
  content: string
  aliases?: string[]
  domain?: string
  priority?: string
}

export interface KBSearchParams {
  query?: string
  type?: string
  status?: string
  tags?: string[]
  section?: string
  limit?: number
}

// Hook: KB Status
export function useKBStatus() {
  const [status, setStatus] = useState<KBStatus | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const fetchStatus = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/status`)
      if (!res.ok) throw new Error('Failed to fetch KB status')
      const data = await res.json()
      setStatus(data)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchStatus()
  }, [fetchStatus])

  const initialize = useCallback(async (force = false) => {
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/init`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ force }),
      })
      if (!res.ok) throw new Error('Failed to initialize KB')
      await fetchStatus()
      return true
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      return false
    }
  }, [fetchStatus])

  const rebuildIndex = useCallback(async () => {
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/index`, {
        method: 'POST',
      })
      if (!res.ok) throw new Error('Failed to rebuild index')
      await fetchStatus()
      return true
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error')
      return false
    }
  }, [fetchStatus])

  return { status, loading, error, fetchStatus, initialize, rebuildIndex }
}

// Hook: KB TOC
export function useKBToc() {
  const [toc, setToc] = useState<KBTocItem[]>([])
  const [loading, setLoading] = useState(true)

  const fetchToc = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/toc`)
      if (!res.ok) throw new Error('Failed to fetch TOC')
      const data = await res.json()
      setToc(data)
    } catch (err) {
      console.error('TOC fetch error:', err)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchToc()
  }, [fetchToc])

  const getSectionToc = useCallback(async (section: string): Promise<KBTocEntry[]> => {
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/toc/${encodeURIComponent(section)}`)
      if (!res.ok) return []
      const data = await res.json()
      // Backend returns { section, content, entries: [...] }
      return data.entries || []
    } catch {
      return []
    }
  }, [])

  const generateToc = useCallback(async (section: string, depth = 3, sortBy = 'alphabetical') => {
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/toc/${encodeURIComponent(section)}/generate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ depth, sort_by: sortBy }),
      })
      if (!res.ok) throw new Error('Failed to generate TOC')
      await fetchToc()
      return true
    } catch {
      return false
    }
  }, [fetchToc])

  return { toc, loading, fetchToc, getSectionToc, generateToc }
}

// Search result from backend
interface SearchResult {
  document: {
    id: number
    path: string
    title: string
    type?: string
    status?: string
    domain?: string
    summary?: string
    tags?: string[]
    created_at?: string
    updated_at?: string
    indexed_at: string
  }
  score: number
  highlights?: string[]
}

// Hook: KB Documents
export function useKBDocuments() {
  const [documents, setDocuments] = useState<KBDocument[]>([])
  const [loading, setLoading] = useState(false)
  const [total, setTotal] = useState(0)

  const search = useCallback(async (params: KBSearchParams = {}) => {
    setLoading(true)
    try {
      const searchParams = new URLSearchParams()
      if (params.query) searchParams.set('q', params.query)  // Backend uses 'q'
      if (params.type) searchParams.set('type', params.type)
      if (params.status) searchParams.set('status', params.status)
      if (params.section) searchParams.set('section', params.section)
      if (params.limit) searchParams.set('limit', String(params.limit))
      if (params.tags?.length) searchParams.set('tag', params.tags[0])  // Backend uses 'tag'

      const res = await fetch(`${await resolveUrl(API_PATH)}/documents?${searchParams}`)
      if (!res.ok) throw new Error('Failed to search documents')
      const results: SearchResult[] = await res.json()

      // Extract documents from search results
      const docs = (results || []).map((r) => ({
        id: r.document.path,  // Use path as id
        title: r.document.title,
        path: r.document.path,
        type: r.document.type || '',
        status: r.document.status,
        tags: r.document.tags,
        summary: r.document.summary,
        created: r.document.created_at,
        updated: r.document.updated_at,
      }))

      setDocuments(docs)
      setTotal(docs.length)
    } catch (err) {
      console.error('Document search error:', err)
      setDocuments([])
    } finally {
      setLoading(false)
    }
  }, [])

  const getDocument = useCallback(async (id: string): Promise<KBDocumentDetail | null> => {
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/documents/${encodeURIComponent(id)}`)
      if (!res.ok) return null
      return await res.json()
    } catch {
      return null
    }
  }, [])

  const createDocument = useCallback(async (doc: Partial<KBDocumentDetail>) => {
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/documents`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(doc),
      })
      if (!res.ok) throw new Error('Failed to create document')
      return await res.json()
    } catch (err) {
      console.error('Create document error:', err)
      return null
    }
  }, [])

  const updateDocument = useCallback(async (id: string, doc: Partial<KBDocumentDetail>) => {
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/documents/${encodeURIComponent(id)}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(doc),
      })
      if (!res.ok) throw new Error('Failed to update document')
      return await res.json()
    } catch (err) {
      console.error('Update document error:', err)
      return null
    }
  }, [])

  const deleteDocument = useCallback(async (id: string) => {
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/documents/${encodeURIComponent(id)}`, {
        method: 'DELETE',
      })
      return res.ok
    } catch {
      return false
    }
  }, [])

  const moveDocument = useCallback(async (id: string, targetPath: string) => {
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/documents/${encodeURIComponent(id)}/move`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ target_path: targetPath }),
      })
      return res.ok
    } catch {
      return false
    }
  }, [])

  return {
    documents,
    loading,
    total,
    search,
    getDocument,
    createDocument,
    updateDocument,
    deleteDocument,
    moveDocument,
  }
}

// Section type
export interface KBSection {
  id: string
  name: string
  description: string
  icon?: string
}

// Hook: KB Sections
export function useKBSections() {
  const [sections, setSections] = useState<KBSection[]>([])
  const [loading, setLoading] = useState(true)

  const fetchSections = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/sections`)
      if (!res.ok) throw new Error('Failed to fetch sections')
      const data = await res.json()
      setSections(data || [])
    } catch (err) {
      console.error('Sections fetch error:', err)
      setSections([])
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchSections()
  }, [fetchSections])

  return { sections, loading, fetchSections }
}

// Hook: KB Tags (returns map of tag -> count)
export function useKBTags() {
  const [tags, setTags] = useState<Record<string, number>>({})
  const [loading, setLoading] = useState(true)

  const fetchTags = useCallback(async () => {
    setLoading(true)
    try {
      const res = await fetch(`${await resolveUrl(API_PATH)}/tags`)
      if (!res.ok) throw new Error('Failed to fetch tags')
      const data = await res.json()
      // Backend returns map[string]int directly
      setTags(data || {})
    } catch (err) {
      console.error('Tags fetch error:', err)
      setTags({})
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    fetchTags()
  }, [fetchTags])

  // Helper to get tag names as array
  const tagNames = Object.keys(tags)

  return { tags, tagNames, loading, fetchTags }
}

// Combined hook for full KB functionality
export function useKB() {
  const status = useKBStatus()
  const toc = useKBToc()
  const documents = useKBDocuments()
  const tagsHook = useKBTags()
  const sectionsHook = useKBSections()

  return {
    ...status,
    toc: toc.toc,
    tocLoading: toc.loading,
    getSectionToc: toc.getSectionToc,
    generateToc: toc.generateToc,
    refreshToc: toc.fetchToc,
    documents: documents.documents,
    docsLoading: documents.loading,
    docsTotal: documents.total,
    searchDocs: documents.search,
    getDocument: documents.getDocument,
    createDocument: documents.createDocument,
    updateDocument: documents.updateDocument,
    deleteDocument: documents.deleteDocument,
    moveDocument: documents.moveDocument,
    tags: tagsHook.tags,
    tagNames: tagsHook.tagNames,
    tagsLoading: tagsHook.loading,
    refreshTags: tagsHook.fetchTags,
    sections: sectionsHook.sections,
    sectionsLoading: sectionsHook.loading,
    refreshSections: sectionsHook.fetchSections,
  }
}

export default useKB
