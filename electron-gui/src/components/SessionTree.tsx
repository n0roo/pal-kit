import { useState } from 'react'
import { ChevronRight, ChevronDown, Layers, Target, Cog, TestTube2 } from 'lucide-react'
import clsx from 'clsx'
import type { Session } from '../hooks/useApi'

interface SessionTreeProps {
  sessions: Session[]
  onSelect?: (session: Session) => void
  selectedId?: string
}

const typeIcons = {
  build: Layers,
  operator: Target,
  worker: Cog,
  test: TestTube2,
}

const typeColors = {
  build: 'text-blue-400',
  operator: 'text-purple-400',
  worker: 'text-green-400',
  test: 'text-orange-400',
}

const statusColors = {
  running: 'bg-green-500',
  complete: 'bg-gray-500',
  failed: 'bg-red-500',
  paused: 'bg-yellow-500',
}

interface SessionNodeProps {
  session: Session
  depth: number
  onSelect?: (session: Session) => void
  selectedId?: string
}

function SessionNode({ session, depth, onSelect, selectedId }: SessionNodeProps) {
  const [expanded, setExpanded] = useState(depth < 2)
  const hasChildren = session.children && session.children.length > 0
  
  const Icon = typeIcons[session.type as keyof typeof typeIcons] || Layers
  const typeColor = typeColors[session.type as keyof typeof typeColors] || 'text-gray-400'
  const statusColor = statusColors[session.status as keyof typeof statusColors] || 'bg-gray-500'
  
  const isSelected = selectedId === session.id

  return (
    <div>
      <div
        className={clsx(
          'session-node flex items-center gap-2 py-1.5 px-2 rounded cursor-pointer',
          isSelected && 'bg-primary-600/20 border border-primary-600/50',
          !isSelected && 'hover:bg-dark-700'
        )}
        style={{ paddingLeft: `${depth * 20 + 8}px` }}
        onClick={() => onSelect?.(session)}
      >
        {/* Expand/collapse button */}
        {hasChildren ? (
          <button
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
            className="p-0.5 hover:bg-dark-600 rounded"
          >
            {expanded ? (
              <ChevronDown size={14} className="text-dark-400" />
            ) : (
              <ChevronRight size={14} className="text-dark-400" />
            )}
          </button>
        ) : (
          <span className="w-5" /> // Spacer
        )}

        {/* Status dot */}
        <div className={clsx('status-dot', statusColor)} />

        {/* Type icon */}
        <Icon size={16} className={typeColor} />

        {/* Title */}
        <span className="flex-1 truncate text-sm">
          {session.title || session.id.slice(0, 8)}
        </span>

        {/* Type badge */}
        <span className={clsx(
          'text-xs px-1.5 py-0.5 rounded',
          typeColor,
          'bg-dark-700'
        )}>
          {session.type}
        </span>
      </div>

      {/* Children */}
      {expanded && hasChildren && (
        <div className="tree-line ml-4">
          {session.children!.map(child => (
            <SessionNode
              key={child.id}
              session={child}
              depth={depth + 1}
              onSelect={onSelect}
              selectedId={selectedId}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export default function SessionTree({ sessions, onSelect, selectedId }: SessionTreeProps) {
  if (sessions.length === 0) {
    return (
      <div className="text-center py-8 text-dark-400">
        <Layers size={32} className="mx-auto mb-2 opacity-50" />
        <p>세션이 없습니다</p>
      </div>
    )
  }

  return (
    <div className="space-y-1">
      {sessions.map(session => (
        <SessionNode
          key={session.id}
          session={session}
          depth={0}
          onSelect={onSelect}
          selectedId={selectedId}
        />
      ))}
    </div>
  )
}
