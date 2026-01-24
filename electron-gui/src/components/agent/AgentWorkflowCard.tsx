import { Bot, Info, Activity, ChevronRight } from 'lucide-react'
import clsx from 'clsx'

export interface WorkflowAgent {
  id: string
  name: string
  type: string
  description?: string
  purpose?: string
  capabilities?: string[]
  status?: 'active' | 'idle' | 'busy'
  version?: number
}

interface AgentWorkflowCardProps {
  agent: WorkflowAgent
  onClick?: () => void
  selected?: boolean
  compact?: boolean
}

const TYPE_COLORS: Record<string, { bg: string; border: string; text: string }> = {
  spec: {
    bg: 'bg-purple-500/10',
    border: 'border-purple-500/30',
    text: 'text-purple-400',
  },
  operator: {
    bg: 'bg-blue-500/10',
    border: 'border-blue-500/30',
    text: 'text-blue-400',
  },
  worker: {
    bg: 'bg-green-500/10',
    border: 'border-green-500/30',
    text: 'text-green-400',
  },
  test: {
    bg: 'bg-orange-500/10',
    border: 'border-orange-500/30',
    text: 'text-orange-400',
  },
  support: {
    bg: 'bg-cyan-500/10',
    border: 'border-cyan-500/30',
    text: 'text-cyan-400',
  },
}

const STATUS_INDICATORS: Record<string, string> = {
  active: 'bg-green-500',
  idle: 'bg-dark-500',
  busy: 'bg-yellow-500 animate-pulse',
}

export default function AgentWorkflowCard({
  agent,
  onClick,
  selected,
  compact = false,
}: AgentWorkflowCardProps) {
  const colors = TYPE_COLORS[agent.type] || TYPE_COLORS.worker

  if (compact) {
    return (
      <button
        onClick={onClick}
        className={clsx(
          'w-full flex items-center gap-2 p-2 rounded-lg border transition-all',
          colors.bg,
          selected ? 'border-primary-500 ring-1 ring-primary-500/50' : colors.border,
          'hover:brightness-110'
        )}
      >
        <div className="relative">
          <Bot size={16} className={colors.text} />
          {agent.status && (
            <div
              className={clsx(
                'absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full',
                STATUS_INDICATORS[agent.status]
              )}
            />
          )}
        </div>
        <span className="text-xs font-medium truncate">{agent.name}</span>
      </button>
    )
  }

  return (
    <button
      onClick={onClick}
      className={clsx(
        'w-full text-left p-3 rounded-lg border transition-all',
        colors.bg,
        selected ? 'border-primary-500 ring-1 ring-primary-500/50' : colors.border,
        'hover:brightness-110'
      )}
    >
      {/* Header */}
      <div className="flex items-start justify-between mb-2">
        <div className="flex items-center gap-2">
          <div className="relative">
            <Bot size={18} className={colors.text} />
            {agent.status && (
              <div
                className={clsx(
                  'absolute -bottom-0.5 -right-0.5 w-2 h-2 rounded-full',
                  STATUS_INDICATORS[agent.status]
                )}
              />
            )}
          </div>
          <span className="font-medium text-sm">{agent.name}</span>
        </div>
        <ChevronRight size={14} className="text-dark-500" />
      </div>

      {/* Purpose / Description */}
      {(agent.purpose || agent.description) && (
        <p className="text-xs text-dark-400 mb-2 line-clamp-2">
          {agent.purpose || agent.description}
        </p>
      )}

      {/* Capabilities preview */}
      {agent.capabilities && agent.capabilities.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {agent.capabilities.slice(0, 3).map((cap) => (
            <span
              key={cap}
              className="text-[10px] px-1.5 py-0.5 bg-dark-700/50 rounded text-dark-300"
            >
              {cap}
            </span>
          ))}
          {agent.capabilities.length > 3 && (
            <span className="text-[10px] text-dark-500">
              +{agent.capabilities.length - 3}
            </span>
          )}
        </div>
      )}

      {/* Version badge */}
      {agent.version && (
        <div className="mt-2 flex items-center gap-1 text-[10px] text-dark-500">
          <Activity size={10} />
          v{agent.version}
        </div>
      )}
    </button>
  )
}
