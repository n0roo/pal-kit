import { useMemo } from 'react'
import clsx from 'clsx'
import type { AttentionState } from '../hooks/useApi'

interface AttentionGaugeProps {
  attention: AttentionState
  size?: 'sm' | 'md' | 'lg'
}

export default function AttentionGauge({ attention, size = 'md' }: AttentionGaugeProps) {
  const tokenPercent = useMemo(() => {
    if (!attention.token_budget) return 0
    return (attention.loaded_tokens / attention.token_budget) * 100
  }, [attention.loaded_tokens, attention.token_budget])

  const getStatusColor = (percent: number) => {
    if (percent >= 95) return 'text-red-500'
    if (percent >= 80) return 'text-yellow-500'
    return 'text-green-500'
  }

  const getBarColor = (percent: number) => {
    if (percent >= 95) return 'bg-red-500'
    if (percent >= 80) return 'bg-yellow-500'
    return 'bg-green-500'
  }

  const getStatusLabel = (percent: number) => {
    if (percent >= 95) return 'Critical'
    if (percent >= 80) return 'Warning'
    if (percent >= 60) return 'Drifting'
    return 'Focused'
  }

  const statusColor = getStatusColor(tokenPercent)
  const barColor = getBarColor(tokenPercent)
  const statusLabel = getStatusLabel(tokenPercent)

  const sizeClasses = {
    sm: 'h-1.5',
    md: 'h-2',
    lg: 'h-3',
  }

  return (
    <div className="space-y-2">
      {/* Header */}
      <div className="flex items-center justify-between">
        <span className="text-sm text-dark-300">Token Usage</span>
        <span className={clsx('text-sm font-medium', statusColor)}>
          {statusLabel}
        </span>
      </div>

      {/* Progress bar */}
      <div className={clsx(
        'w-full bg-dark-700 rounded-full overflow-hidden',
        sizeClasses[size]
      )}>
        <div
          className={clsx(
            'h-full rounded-full transition-all duration-300',
            barColor,
            tokenPercent >= 95 && 'alert-pulse'
          )}
          style={{ width: `${Math.min(tokenPercent, 100)}%` }}
        />
      </div>

      {/* Stats */}
      <div className="flex items-center justify-between text-xs text-dark-400">
        <span>
          {attention.loaded_tokens.toLocaleString()} / {attention.token_budget.toLocaleString()}
        </span>
        <span className={statusColor}>
          {tokenPercent.toFixed(1)}%
        </span>
      </div>

      {/* Focus & Drift scores */}
      <div className="grid grid-cols-2 gap-2 pt-2">
        <div className="bg-dark-700 rounded p-2">
          <div className="text-xs text-dark-400">Focus Score</div>
          <div className={clsx(
            'text-lg font-semibold',
            attention.focus_score >= 0.7 ? 'text-green-400' : 
            attention.focus_score >= 0.4 ? 'text-yellow-400' : 'text-red-400'
          )}>
            {(attention.focus_score * 100).toFixed(0)}%
          </div>
        </div>
        <div className="bg-dark-700 rounded p-2">
          <div className="text-xs text-dark-400">Drift Score</div>
          <div className={clsx(
            'text-lg font-semibold',
            attention.drift_score <= 0.3 ? 'text-green-400' : 
            attention.drift_score <= 0.6 ? 'text-yellow-400' : 'text-red-400'
          )}>
            {(attention.drift_score * 100).toFixed(0)}%
          </div>
        </div>
      </div>
    </div>
  )
}
