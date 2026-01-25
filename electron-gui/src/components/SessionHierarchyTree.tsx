import { useState, useEffect, useMemo } from 'react'
import { ChevronRight, ChevronDown, Layers, Target, Cog, TestTube2, 
         AlertTriangle, RefreshCw, CheckCircle, XCircle, Clock } from 'lucide-react'
import clsx from 'clsx'
import { useSSE, SSEEvent } from '../hooks/useSSE'

// Session node with attention info
export interface SessionNodeData {
  id: string
  type: 'build' | 'operator' | 'worker' | 'test' | 'main' | 'sub'
  title: string
  status: 'pending' | 'running' | 'complete' | 'failed' | 'blocked'
  portId?: string
  attention?: {
    score: number
    tokensUsed: number
    tokenBudget: number
    focusScore?: number
    driftScore?: number
    compactCount?: number
    lastCompact?: string
  }
  children?: SessionNodeData[]
  testSession?: SessionNodeData
}

interface SessionHierarchyTreeProps {
  initialData?: SessionNodeData[]
  onSelect?: (session: SessionNodeData) => void
  selectedId?: string
  showAttention?: boolean
  apiBaseUrl?: string
}

const typeIcons = {
  build: Layers,
  operator: Target,
  worker: Cog,
  test: TestTube2,
  main: Layers,
  sub: Cog,
}

const typeColors = {
  build: 'text-blue-400',
  operator: 'text-purple-400',
  worker: 'text-green-400',
  test: 'text-orange-400',
  main: 'text-blue-400',
  sub: 'text-green-400',
}

const statusIcons = {
  running: RefreshCw,
  complete: CheckCircle,
  failed: XCircle,
  pending: Clock,
  blocked: AlertTriangle,
}

const statusColors = {
  running: 'bg-green-500 animate-pulse',
  complete: 'bg-gray-500',
  failed: 'bg-red-500',
  pending: 'bg-yellow-500',
  blocked: 'bg-orange-500',
}

interface AttentionBadgeProps {
  attention: SessionNodeData['attention']
  compact?: boolean
}

function AttentionBadge({ attention, compact = true }: AttentionBadgeProps) {
  if (!attention) return null

  const percent = attention.tokenBudget > 0 
    ? (attention.tokensUsed / attention.tokenBudget) * 100 
    : 0
  
  const getColor = (p: number) => {
    if (p >= 90) return 'bg-red-500 text-red-100'
    if (p >= 80) return 'bg-yellow-500 text-yellow-100'
    return 'bg-green-500/30 text-green-400'
  }

  if (compact) {
    return (
      <span 
        className={clsx(
          'text-xs px-1.5 py-0.5 rounded font-mono',
          getColor(percent)
        )}
        title={`${attention.tokensUsed.toLocaleString()} / ${attention.tokenBudget.toLocaleString()} tokens`}
      >
        {percent.toFixed(0)}%
      </span>
    )
  }

  return (
    <div className="space-y-1 text-xs">
      <div className="flex justify-between">
        <span className="text-gray-400">Tokens</span>
        <span className={getColor(percent).replace('bg-', 'text-').split(' ')[0]}>
          {percent.toFixed(1)}%
        </span>
      </div>
      <div className="h-1.5 bg-gray-700 rounded-full overflow-hidden">
        <div 
          className={clsx(
            'h-full rounded-full transition-all',
            percent >= 90 ? 'bg-red-500' : percent >= 80 ? 'bg-yellow-500' : 'bg-green-500'
          )}
          style={{ width: `${Math.min(percent, 100)}%` }}
        />
      </div>
      {attention.compactCount && attention.compactCount > 0 && (
        <div className="text-orange-400">
          📦 Compact: {attention.compactCount}회
        </div>
      )}
    </div>
  )
}

interface SessionNodeComponentProps {
  node: SessionNodeData
  depth: number
  onSelect?: (session: SessionNodeData) => void
  selectedId?: string
  showAttention?: boolean
  updatedIds?: Set<string>
}

function SessionNodeComponent({ 
  node, 
  depth, 
  onSelect, 
  selectedId, 
  showAttention = true,
  updatedIds 
}: SessionNodeComponentProps) {
  const [expanded, setExpanded] = useState(depth < 2)
  const hasChildren = (node.children && node.children.length > 0) || node.testSession
  
  const Icon = typeIcons[node.type] || Layers
  const typeColor = typeColors[node.type] || 'text-gray-400'
  const StatusIcon = statusIcons[node.status] || Clock
  const statusColor = statusColors[node.status] || 'bg-gray-500'
  
  const isSelected = selectedId === node.id
  const isUpdated = updatedIds?.has(node.id)

  return (
    <div>
      <div
        className={clsx(
          'flex items-center gap-2 py-2 px-2 rounded cursor-pointer transition-all',
          isSelected && 'bg-blue-600/20 border border-blue-600/50',
          !isSelected && 'hover:bg-gray-800',
          isUpdated && 'ring-2 ring-yellow-500/50'
        )}
        style={{ paddingLeft: `${depth * 20 + 8}px` }}
        onClick={() => onSelect?.(node)}
      >
        {/* Expand/collapse button */}
        {hasChildren ? (
          <button
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
            className="p-0.5 hover:bg-gray-700 rounded"
          >
            {expanded ? (
              <ChevronDown size={14} className="text-gray-400" />
            ) : (
              <ChevronRight size={14} className="text-gray-400" />
            )}
          </button>
        ) : (
          <span className="w-5" />
        )}

        {/* Status indicator */}
        <div className={clsx('w-2 h-2 rounded-full', statusColor)} />

        {/* Type icon */}
        <Icon size={16} className={typeColor} />

        {/* Title */}
        <span className="flex-1 truncate text-sm">
          {node.title || node.id.slice(0, 8)}
        </span>

        {/* Port ID */}
        {node.portId && (
          <span className="text-xs text-gray-500 bg-gray-800 px-1.5 py-0.5 rounded">
            {node.portId}
          </span>
        )}

        {/* Attention badge */}
        {showAttention && node.attention && (
          <AttentionBadge attention={node.attention} />
        )}

        {/* Status icon */}
        <StatusIcon 
          size={14} 
          className={clsx(
            node.status === 'running' && 'text-green-400 animate-spin',
            node.status === 'complete' && 'text-gray-400',
            node.status === 'failed' && 'text-red-400',
            node.status === 'pending' && 'text-yellow-400',
            node.status === 'blocked' && 'text-orange-400',
          )}
        />
      </div>

      {/* Children */}
      {expanded && hasChildren && (
        <div className="ml-4 border-l border-gray-800">
          {node.children?.map(child => (
            <SessionNodeComponent
              key={child.id}
              node={child}
              depth={depth + 1}
              onSelect={onSelect}
              selectedId={selectedId}
              showAttention={showAttention}
              updatedIds={updatedIds}
            />
          ))}
          {node.testSession && (
            <SessionNodeComponent
              key={node.testSession.id}
              node={node.testSession}
              depth={depth + 1}
              onSelect={onSelect}
              selectedId={selectedId}
              showAttention={showAttention}
              updatedIds={updatedIds}
            />
          )}
        </div>
      )}
    </div>
  )
}

export function SessionHierarchyTree({ 
  initialData = [], 
  onSelect, 
  selectedId,
  showAttention = true,
  apiBaseUrl = 'http://localhost:8080'
}: SessionHierarchyTreeProps) {
  const [sessions, setSessions] = useState<SessionNodeData[]>(initialData)
  const [updatedIds, setUpdatedIds] = useState<Set<string>>(new Set())
  
  // SSE connection for real-time updates
  const { events, connected, latestEvent } = useSSE({
    filters: ['session:start', 'session:end', 'session:update', 
              'attention:warning', 'attention:critical', 
              'port:start', 'port:end'],
    baseUrl: apiBaseUrl,
  })

  // Fetch hierarchy data
  const fetchHierarchy = async () => {
    try {
      const res = await fetch(`${apiBaseUrl}/api/v2/sessions/hierarchy`)
      const data = await res.json()
      if (data.sessions || data.hierarchy) {
        setSessions(data.sessions || data.hierarchy || [])
      }
    } catch (err) {
      console.error('Failed to fetch hierarchy:', err)
    }
  }

  // Initial fetch
  useEffect(() => {
    fetchHierarchy()
  }, [apiBaseUrl])

  // Handle SSE events
  useEffect(() => {
    if (!latestEvent) return

    const sessionId = latestEvent.session_id
    
    // Mark session as updated (visual feedback)
    if (sessionId) {
      setUpdatedIds(prev => new Set(prev).add(sessionId))
      
      // Clear the highlight after 2 seconds
      setTimeout(() => {
        setUpdatedIds(prev => {
          const next = new Set(prev)
          next.delete(sessionId)
          return next
        })
      }, 2000)
    }

    // Refetch on certain events
    if (['session:start', 'session:end', 'port:start', 'port:end'].includes(latestEvent.type)) {
      fetchHierarchy()
    }
  }, [latestEvent])

  // Update initial data when prop changes
  useEffect(() => {
    if (initialData.length > 0) {
      setSessions(initialData)
    }
  }, [initialData])

  if (sessions.length === 0) {
    return (
      <div className="text-center py-8 text-gray-400">
        <Layers size={32} className="mx-auto mb-2 opacity-50" />
        <p>세션이 없습니다</p>
        {!connected && (
          <p className="text-xs text-red-400 mt-2">SSE 연결 끊김</p>
        )}
      </div>
    )
  }

  return (
    <div className="space-y-1">
      {/* Connection status */}
      <div className="flex items-center justify-between px-2 py-1 text-xs text-gray-500">
        <span>
          {sessions.length} session(s)
        </span>
        <div className="flex items-center gap-1">
          <div className={clsx(
            'w-2 h-2 rounded-full',
            connected ? 'bg-green-500' : 'bg-red-500'
          )} />
          <span>{connected ? 'Live' : 'Offline'}</span>
        </div>
      </div>

      {/* Tree */}
      {sessions.map(session => (
        <SessionNodeComponent
          key={session.id}
          node={session}
          depth={0}
          onSelect={onSelect}
          selectedId={selectedId}
          showAttention={showAttention}
          updatedIds={updatedIds}
        />
      ))}
    </div>
  )
}

export default SessionHierarchyTree
