import { User, Cog, TestTube2, FileText, GitCommit, Server, Folder } from 'lucide-react'
import clsx from 'clsx'
import type { Agent } from '../hooks/useApi'

interface AgentCardProps {
  agent: Agent
  onClick?: () => void
  selected?: boolean
}

const typeIcons = {
  spec: FileText,
  operator: User,
  worker: Cog,
  test: TestTube2,
}

const typeColors = {
  spec: 'text-blue-400 bg-blue-500/10 border-blue-500/30',
  operator: 'text-purple-400 bg-purple-500/10 border-purple-500/30',
  worker: 'text-green-400 bg-green-500/10 border-green-500/30',
  test: 'text-orange-400 bg-orange-500/10 border-orange-500/30',
}

export default function AgentCard({ agent, onClick, selected }: AgentCardProps) {
  const Icon = typeIcons[agent.type as keyof typeof typeIcons] || User
  const colorClass = typeColors[agent.type as keyof typeof typeColors] || 'text-gray-400 bg-gray-500/10 border-gray-500/30'

  return (
    <div
      className={clsx(
        'bg-dark-800 border rounded-lg p-4 card-hover cursor-pointer transition-all',
        selected 
          ? 'border-primary-500/50 ring-1 ring-primary-500/30' 
          : 'border-dark-700 hover:border-dark-500'
      )}
      onClick={onClick}
    >
      {/* Header */}
      <div className="flex items-start gap-3">
        <div className={clsx(
          'p-2 rounded-lg border',
          colorClass
        )}>
          <Icon size={24} />
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <h3 className="font-medium truncate">{agent.name}</h3>
            {agent.is_system && (
              <Server size={12} className="text-purple-400 flex-shrink-0" />
            )}
          </div>
          <div className="flex items-center gap-1.5 mt-1">
            <span className={clsx(
              'text-xs px-2 py-0.5 rounded capitalize border',
              colorClass
            )}>
              {agent.type}
            </span>
            {agent.is_system ? (
              <span className="text-xs px-1.5 py-0.5 bg-purple-500/10 text-purple-400 rounded border border-purple-500/30">
                시스템
              </span>
            ) : (
              <span className="text-xs px-1.5 py-0.5 bg-dark-700 text-dark-400 rounded">
                프로젝트
              </span>
            )}
          </div>
        </div>
      </div>

      {/* Description */}
      {agent.description && (
        <p className="text-sm text-dark-400 mt-3 line-clamp-2">
          {agent.description}
        </p>
      )}

      {/* Capabilities */}
      {agent.capabilities && agent.capabilities.length > 0 && (
        <div className="flex flex-wrap gap-1 mt-3">
          {agent.capabilities.slice(0, 4).map(cap => (
            <span 
              key={cap} 
              className="text-xs px-1.5 py-0.5 bg-dark-700 text-dark-400 rounded"
            >
              {cap}
            </span>
          ))}
          {agent.capabilities.length > 4 && (
            <span className="text-xs text-dark-500">
              +{agent.capabilities.length - 4}
            </span>
          )}
        </div>
      )}

      {/* Footer */}
      <div className="flex items-center justify-between mt-4 pt-3 border-t border-dark-700">
        <div className="flex items-center gap-1 text-xs text-dark-400">
          <GitCommit size={12} />
          <span>v{agent.current_version}</span>
        </div>
        <span className="text-xs text-dark-500 truncate max-w-[100px]">
          {agent.id.startsWith('system-') ? agent.id.replace('system-', '') : agent.id.slice(0, 8)}
        </span>
      </div>
    </div>
  )
}
