import { GitBranch, Play, Pause, CheckCircle, XCircle, Clock } from 'lucide-react'
import clsx from 'clsx'
import type { Orchestration } from '../hooks/useApi'

interface OrchestrationProgressProps {
  orchestration: Orchestration
  onClick?: () => void
}

const statusIcons = {
  pending: Clock,
  running: Play,
  paused: Pause,
  complete: CheckCircle,
  failed: XCircle,
}

const statusColors = {
  pending: 'text-gray-400',
  running: 'text-green-400',
  paused: 'text-yellow-400',
  complete: 'text-blue-400',
  failed: 'text-red-400',
}

export default function OrchestrationProgress({ orchestration, onClick }: OrchestrationProgressProps) {
  const StatusIcon = statusIcons[orchestration.status as keyof typeof statusIcons] || Clock
  const statusColor = statusColors[orchestration.status as keyof typeof statusColors] || 'text-gray-400'

  return (
    <div
      className={clsx(
        'bg-dark-800 border border-dark-700 rounded-lg p-4 card-hover cursor-pointer',
        orchestration.status === 'running' && 'border-green-600/50'
      )}
      onClick={onClick}
    >
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <GitBranch size={18} className="text-primary-400" />
          <span className="font-medium truncate max-w-[200px]">
            {orchestration.title}
          </span>
        </div>
        <div className={clsx('flex items-center gap-1', statusColor)}>
          <StatusIcon size={16} />
          <span className="text-sm capitalize">{orchestration.status}</span>
        </div>
      </div>

      {/* Progress bar */}
      <div className="mb-2">
        <div className="flex items-center justify-between text-xs text-dark-400 mb-1">
          <span>진행률</span>
          <span>{orchestration.progress_percent}%</span>
        </div>
        <div className="h-2 bg-dark-700 rounded-full overflow-hidden">
          <div
            className={clsx(
              'h-full rounded-full transition-all duration-500',
              orchestration.status === 'running' && 'bg-green-500',
              orchestration.status === 'complete' && 'bg-blue-500',
              orchestration.status === 'failed' && 'bg-red-500',
              orchestration.status === 'paused' && 'bg-yellow-500',
              orchestration.status === 'pending' && 'bg-gray-500',
              orchestration.status === 'running' && orchestration.progress_percent < 100 && 'progress-indeterminate'
            )}
            style={{ width: `${orchestration.progress_percent}%` }}
          />
        </div>
      </div>

      {/* Description */}
      {orchestration.description && (
        <p className="text-xs text-dark-400 truncate">
          {orchestration.description}
        </p>
      )}

      {/* Ports count */}
      {orchestration.ports && (
        <div className="mt-2 text-xs text-dark-500">
          {orchestration.ports.length}개 포트
        </div>
      )}
    </div>
  )
}
