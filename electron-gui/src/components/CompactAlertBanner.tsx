import { useState, useEffect } from 'react'
import { AlertTriangle, X, RotateCcw, GitBranch, Play } from 'lucide-react'
import clsx from 'clsx'
import { useSSE, CompactTriggeredData } from '../hooks/useSSE'

interface CompactAlertBannerProps {
  onRecover?: (checkpointId: string) => void
  onSplit?: (sessionId: string) => void
  onContinue?: (sessionId: string) => void
  apiBaseUrl?: string
}

export function CompactAlertBanner({ 
  onRecover, 
  onSplit, 
  onContinue,
  apiBaseUrl = 'http://localhost:8080'
}: CompactAlertBannerProps) {
  const [visible, setVisible] = useState(false)
  const [compactData, setCompactData] = useState<{
    sessionId: string
    checkpointId?: string
    trigger: string
    recoveryHint?: string
    timestamp: Date
  } | null>(null)

  // Listen for compact events
  const { latestEvent } = useSSE({
    filters: ['compact:triggered', 'attention:critical'],
    baseUrl: apiBaseUrl,
  })

  useEffect(() => {
    if (!latestEvent) return

    if (latestEvent.type === 'compact:triggered') {
      const data = latestEvent.data as CompactTriggeredData
      setCompactData({
        sessionId: latestEvent.session_id || '',
        checkpointId: data?.checkpoint_id,
        trigger: data?.trigger || 'auto',
        recoveryHint: data?.recovery_hint,
        timestamp: new Date(latestEvent.timestamp),
      })
      setVisible(true)
    }
  }, [latestEvent])

  const handleRecover = () => {
    if (compactData?.checkpointId && onRecover) {
      onRecover(compactData.checkpointId)
    }
    setVisible(false)
  }

  const handleSplit = () => {
    if (compactData?.sessionId && onSplit) {
      onSplit(compactData.sessionId)
    }
    setVisible(false)
  }

  const handleContinue = () => {
    if (compactData?.sessionId && onContinue) {
      onContinue(compactData.sessionId)
    }
    setVisible(false)
  }

  if (!visible || !compactData) return null

  return (
    <div className="fixed top-0 left-0 right-0 z-50 animate-in slide-in-from-top">
      <div className="bg-orange-900/95 border-b border-orange-700 backdrop-blur-sm">
        <div className="max-w-6xl mx-auto px-4 py-3">
          <div className="flex items-center justify-between">
            {/* Alert content */}
            <div className="flex items-center gap-3">
              <div className="p-2 bg-orange-800 rounded-lg">
                <AlertTriangle className="text-orange-300" size={20} />
              </div>
              <div>
                <div className="font-medium text-orange-100">
                  📦 Compact 발생
                </div>
                <div className="text-sm text-orange-300">
                  세션: {compactData.sessionId.slice(0, 8)}... • 
                  트리거: {compactData.trigger}
                  {compactData.checkpointId && (
                    <> • 체크포인트: {compactData.checkpointId.slice(0, 8)}...</>
                  )}
                </div>
                {compactData.recoveryHint && (
                  <div className="text-xs text-orange-400 mt-1">
                    💡 {compactData.recoveryHint}
                  </div>
                )}
              </div>
            </div>

            {/* Actions */}
            <div className="flex items-center gap-2">
              {compactData.checkpointId && (
                <button
                  onClick={handleRecover}
                  className={clsx(
                    'flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium',
                    'bg-blue-600 hover:bg-blue-500 text-white transition-colors'
                  )}
                >
                  <RotateCcw size={14} />
                  체크포인트 복구
                </button>
              )}
              
              <button
                onClick={handleSplit}
                className={clsx(
                  'flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium',
                  'bg-purple-600 hover:bg-purple-500 text-white transition-colors'
                )}
              >
                <GitBranch size={14} />
                세션 분리
              </button>
              
              <button
                onClick={handleContinue}
                className={clsx(
                  'flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm font-medium',
                  'bg-gray-600 hover:bg-gray-500 text-white transition-colors'
                )}
              >
                <Play size={14} />
                계속 진행
              </button>

              {/* Close button */}
              <button
                onClick={() => setVisible(false)}
                className="p-1.5 hover:bg-orange-800 rounded-lg text-orange-400 hover:text-orange-200"
              >
                <X size={16} />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

// Attention Warning Toast
interface AttentionToastProps {
  apiBaseUrl?: string
  maxToasts?: number
}

interface ToastData {
  id: string
  type: 'warning' | 'critical'
  sessionId: string
  usagePercent: number
  message: string
  timestamp: Date
}

export function AttentionToasts({ 
  apiBaseUrl = 'http://localhost:8080',
  maxToasts = 3
}: AttentionToastProps) {
  const [toasts, setToasts] = useState<ToastData[]>([])

  const { latestEvent } = useSSE({
    filters: ['attention:warning', 'attention:critical'],
    baseUrl: apiBaseUrl,
  })

  useEffect(() => {
    if (!latestEvent) return

    const isWarning = latestEvent.type === 'attention:warning'
    const isCritical = latestEvent.type === 'attention:critical'

    if (!isWarning && !isCritical) return

    const data = latestEvent.data as { 
      usage_percent?: number
      warning?: string 
    }

    const newToast: ToastData = {
      id: `${latestEvent.session_id}-${Date.now()}`,
      type: isCritical ? 'critical' : 'warning',
      sessionId: latestEvent.session_id || '',
      usagePercent: data?.usage_percent || 0,
      message: data?.warning || `Token usage: ${data?.usage_percent?.toFixed(1)}%`,
      timestamp: new Date(latestEvent.timestamp),
    }

    setToasts(prev => [newToast, ...prev].slice(0, maxToasts))

    // Auto dismiss after 10 seconds
    setTimeout(() => {
      setToasts(prev => prev.filter(t => t.id !== newToast.id))
    }, 10000)
  }, [latestEvent, maxToasts])

  const dismissToast = (id: string) => {
    setToasts(prev => prev.filter(t => t.id !== id))
  }

  if (toasts.length === 0) return null

  return (
    <div className="fixed bottom-4 right-4 z-50 space-y-2 max-w-sm">
      {toasts.map(toast => (
        <div
          key={toast.id}
          className={clsx(
            'p-3 rounded-lg shadow-lg border backdrop-blur-sm animate-in slide-in-from-right',
            toast.type === 'critical' 
              ? 'bg-red-900/90 border-red-700' 
              : 'bg-yellow-900/90 border-yellow-700'
          )}
        >
          <div className="flex items-start gap-2">
            <AlertTriangle 
              className={toast.type === 'critical' ? 'text-red-400' : 'text-yellow-400'} 
              size={16} 
            />
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between">
                <span className={clsx(
                  'text-sm font-medium',
                  toast.type === 'critical' ? 'text-red-200' : 'text-yellow-200'
                )}>
                  {toast.type === 'critical' ? '🚨 Critical' : '⚠️ Warning'}
                </span>
                <button
                  onClick={() => dismissToast(toast.id)}
                  className={clsx(
                    'p-0.5',
                    toast.type === 'critical' 
                      ? 'text-red-400 hover:text-red-200' 
                      : 'text-yellow-400 hover:text-yellow-200'
                  )}
                >
                  <X size={14} />
                </button>
              </div>
              
              <p className={clsx(
                'text-xs mt-1',
                toast.type === 'critical' ? 'text-red-300' : 'text-yellow-300'
              )}>
                {toast.message}
              </p>
              
              <p className={clsx(
                'text-xs mt-0.5',
                toast.type === 'critical' ? 'text-red-400' : 'text-yellow-400'
              )}>
                Session: {toast.sessionId.slice(0, 8)}...
              </p>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}

export default CompactAlertBanner
