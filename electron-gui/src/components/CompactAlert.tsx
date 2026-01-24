import { AlertTriangle, X } from 'lucide-react'
import { formatDistanceToNow } from 'date-fns'
import { ko } from 'date-fns/locale'

interface Alert {
  id: string
  session_id?: string
  data?: {
    token_percent?: number
    focus_score?: number
  }
  timestamp: string
}

interface CompactAlertProps {
  alerts: Alert[]
  onDismiss: (id: string) => void
}

export default function CompactAlert({ alerts, onDismiss }: CompactAlertProps) {
  if (alerts.length === 0) return null

  return (
    <div className="fixed bottom-20 right-4 flex flex-col gap-2 max-w-sm z-50">
      {alerts.slice(0, 3).map(alert => (
        <div
          key={alert.id}
          className="bg-yellow-900/90 border border-yellow-700 rounded-lg p-3 shadow-lg backdrop-blur-sm animate-in slide-in-from-right"
        >
          <div className="flex items-start gap-2">
            <AlertTriangle className="text-yellow-500 flex-shrink-0 mt-0.5" size={16} />
            <div className="flex-1 min-w-0">
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-yellow-200">
                  Attention 경고
                </span>
                <button
                  onClick={() => onDismiss(alert.id)}
                  className="text-yellow-400 hover:text-yellow-200 p-0.5"
                >
                  <X size={14} />
                </button>
              </div>
              
              <p className="text-xs text-yellow-300 mt-1">
                세션: {alert.session_id?.slice(0, 8)}...
              </p>
              
              {alert.data && (
                <div className="flex gap-4 mt-2 text-xs">
                  {alert.data.token_percent !== undefined && (
                    <span className="text-yellow-400">
                      토큰: {alert.data.token_percent.toFixed(1)}%
                    </span>
                  )}
                  {alert.data.focus_score !== undefined && (
                    <span className="text-yellow-400">
                      Focus: {(alert.data.focus_score * 100).toFixed(0)}%
                    </span>
                  )}
                </div>
              )}
              
              <p className="text-xs text-yellow-500 mt-1">
                {formatDistanceToNow(new Date(alert.timestamp), { 
                  addSuffix: true, 
                  locale: ko 
                })}
              </p>
            </div>
          </div>
        </div>
      ))}
      
      {alerts.length > 3 && (
        <div className="text-xs text-dark-400 text-center">
          +{alerts.length - 3}개 더...
        </div>
      )}
    </div>
  )
}
