import { useState, useEffect } from 'react'
import { Layers, RefreshCw, Search } from 'lucide-react'
import { useSessions, type HierarchicalSession } from '../hooks'
import { SessionTree, AttentionGauge } from '../components'
import type { Session } from '../hooks/useApi'

// Normalize session data - API may return different structures
function normalizeSession(data: any): HierarchicalSession | null {
  if (!data) return null
  
  // Already in { session: {...} } format
  if (data.session && data.session.id) {
    return data as HierarchicalSession
  }
  
  // Direct session object (has id at top level)
  if (data.id) {
    return {
      session: {
        id: data.id,
        title: data.title || data.Title || '',
        status: data.status || data.Status || 'unknown',
        type: data.type || data.Type || 'single',
        parent_id: data.parent_id,
        port_id: data.port_id,
        depth: data.depth || 0,
      },
      children: data.children?.map(normalizeSession).filter(Boolean) || []
    }
  }
  
  return null
}

export default function Sessions() {
  const { sessions: rawSessions, loading, fetchSessions, fetchHierarchy } = useSessions()
  const [selectedSession, setSelectedSession] = useState<HierarchicalSession | null>(null)
  const [hierarchy, setHierarchy] = useState<HierarchicalSession | null>(null)
  const [searchQuery, setSearchQuery] = useState('')

  // Normalize all sessions
  const sessions = rawSessions.map(normalizeSession).filter(Boolean) as HierarchicalSession[]

  useEffect(() => {
    if (selectedSession?.session?.id) {
      fetchHierarchy(selectedSession.session.id).then(h => {
        if (h) setHierarchy(normalizeSession(h))
      })
    }
  }, [selectedSession, fetchHierarchy])

  const filteredSessions = sessions.filter(s => {
    if (!s?.session) return false
    const title = s.session.title || ''
    const id = s.session.id || ''
    return title.toLowerCase().includes(searchQuery.toLowerCase()) || id.includes(searchQuery)
  })

  // Convert HierarchicalSession to Session format for SessionTree
  const convertToSession = (hs: HierarchicalSession): Session => ({
    ...hs.session,
    children: hs.children?.map(convertToSession)
  })

  return (
    <div className="h-full flex">
      {/* Left panel - Session list */}
      <div className="w-80 border-r border-dark-700 flex flex-col">
        <div className="p-4 border-b border-dark-700">
          <div className="flex items-center justify-between mb-3">
            <h2 className="text-lg font-semibold flex items-center gap-2">
              <Layers size={20} className="text-blue-400" />
              세션
            </h2>
            <button
              onClick={() => fetchSessions()}
              className="p-1.5 hover:bg-dark-700 rounded"
              disabled={loading}
            >
              <RefreshCw size={16} className={loading ? 'animate-spin' : ''} />
            </button>
          </div>
          
          {/* Search */}
          <div className="relative">
            <Search size={16} className="absolute left-3 top-1/2 -translate-y-1/2 text-dark-400" />
            <input
              type="text"
              placeholder="세션 검색..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-9 pr-3 py-2 bg-dark-700 border border-dark-600 rounded-lg text-sm focus:outline-none focus:border-primary-500"
            />
          </div>
        </div>

        <div className="flex-1 overflow-auto p-2">
          {loading ? (
            <div className="flex items-center justify-center h-32">
              <RefreshCw size={24} className="animate-spin text-dark-400" />
            </div>
          ) : filteredSessions.length === 0 ? (
            <div className="text-center py-8 text-dark-400">
              <Layers size={32} className="mx-auto mb-2 opacity-50" />
              <p>세션이 없습니다</p>
              <p className="text-xs mt-2">서버가 실행 중인지 확인하세요</p>
              <p className="text-xs text-dark-500">pal serve</p>
            </div>
          ) : (
            <div className="space-y-1">
              {filteredSessions.map(hs => (
                <div
                  key={hs.session.id}
                  onClick={() => setSelectedSession(hs)}
                  className={`p-3 rounded-lg cursor-pointer transition-colors ${
                    selectedSession?.session.id === hs.session.id
                      ? 'bg-primary-600/20 border border-primary-600/50'
                      : 'hover:bg-dark-700'
                  }`}
                >
                  <div className="flex items-center gap-2">
                    <div className={`status-dot ${
                      hs.session.status === 'running' ? 'status-running' :
                      hs.session.status === 'complete' ? 'status-complete' :
                      hs.session.status === 'failed' ? 'status-failed' :
                      'status-pending'
                    }`} />
                    <span className="flex-1 truncate font-medium">
                      {hs.session.title || hs.session.id.slice(0, 12)}
                    </span>
                  </div>
                  <div className="mt-1 flex items-center gap-2 text-xs text-dark-400">
                    <span className="capitalize">{hs.session.type}</span>
                    <span>•</span>
                    <span>{hs.session.id.slice(0, 8)}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      </div>

      {/* Right panel - Session detail & hierarchy */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {selectedSession ? (
          <>
            {/* Session header */}
            <div className="p-4 border-b border-dark-700">
              <h2 className="text-xl font-semibold">
                {selectedSession.session.title || selectedSession.session.id}
              </h2>
              <div className="flex items-center gap-3 mt-2 text-sm text-dark-400">
                <span className="capitalize">{selectedSession.session.type}</span>
                <span>•</span>
                <span className="capitalize">{selectedSession.session.status}</span>
                <span>•</span>
                <span className="font-mono">{selectedSession.session.id}</span>
              </div>
            </div>

            {/* Hierarchy tree */}
            <div className="flex-1 overflow-auto p-4">
              <h3 className="text-sm font-medium text-dark-300 mb-3">세션 계층</h3>
              {hierarchy ? (
                <SessionTree
                  sessions={[convertToSession(hierarchy)]}
                  onSelect={(s) => {
                    // Find the HierarchicalSession for this session
                    const findHS = (hs: HierarchicalSession, id: string): HierarchicalSession | null => {
                      if (hs.session.id === id) return hs
                      for (const child of hs.children || []) {
                        const found = findHS(child, id)
                        if (found) return found
                      }
                      return null
                    }
                    const found = findHS(hierarchy, s.id)
                    if (found) setSelectedSession(found)
                  }}
                  selectedId={selectedSession.session.id}
                />
              ) : (
                <div className="text-center py-8 text-dark-400">
                  계층 정보를 불러오는 중...
                </div>
              )}
            </div>
          </>
        ) : (
          <div className="flex-1 flex items-center justify-center text-dark-400">
            <div className="text-center">
              <Layers size={48} className="mx-auto mb-4 opacity-50" />
              <p>세션을 선택하세요</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
