import { useState, useEffect } from 'react'
import { Activity, RefreshCw, AlertTriangle, FileText, BookOpen, Clock } from 'lucide-react'
import { useAttention, useSessions, type HierarchicalSession } from '../hooks'
import { AttentionGauge } from '../components'
import { formatDistanceToNow } from 'date-fns'
import { ko } from 'date-fns/locale'
import clsx from 'clsx'

interface AttentionProps {
  alerts: any[]
}

// Normalize session data
function normalizeSession(data: any): HierarchicalSession | null {
  if (!data) return null
  if (data.session && data.session.id) return data as HierarchicalSession
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
      children: []
    }
  }
  return null
}

export default function Attention({ alerts }: AttentionProps) {
  const { sessions: rawSessions } = useSessions()
  const [selectedSessionId, setSelectedSessionId] = useState<string | null>(null)
  const { attention, loading, fetchAttention, getReport, getHistory } = useAttention()
  const [report, setReport] = useState<any>(null)
  const [history, setHistory] = useState<any[]>([])

  // Normalize sessions
  const sessions = rawSessions.map(normalizeSession).filter(Boolean) as HierarchicalSession[]
  const activeSessions = sessions.filter(s => s?.session?.status === 'running')

  useEffect(() => {
    if (selectedSessionId) {
      fetchAttention(selectedSessionId)
      getReport(selectedSessionId).then(r => setReport(r))
      getHistory(selectedSessionId).then(h => setHistory(h || []))
    }
  }, [selectedSessionId, fetchAttention, getReport, getHistory])

  // Default attention for display when no data
  const displayAttention = attention || {
    session_id: selectedSessionId || '',
    token_budget: 15000,
    loaded_tokens: 0,
    focus_score: 0,
    drift_score: 0,
    loaded_files: [],
    loaded_conventions: []
  }

  return (
    <div className="h-full flex">
      {/* Left panel - Session selector */}
      <div className="w-72 border-r border-dark-700 flex flex-col">
        <div className="p-4 border-b border-dark-700">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Activity size={20} className="text-blue-400" />
            Attention 모니터
          </h2>
        </div>

        {/* Alerts */}
        {alerts.length > 0 && (
          <div className="p-4 border-b border-dark-700">
            <h3 className="text-sm font-medium text-yellow-400 flex items-center gap-2 mb-2">
              <AlertTriangle size={16} />
              경고 ({alerts.length})
            </h3>
            <div className="space-y-2 max-h-32 overflow-auto">
              {alerts.slice(0, 3).map(alert => (
                <div
                  key={alert.id}
                  className="p-2 bg-yellow-900/30 border border-yellow-700/50 rounded text-xs cursor-pointer hover:bg-yellow-900/50"
                  onClick={() => setSelectedSessionId(alert.session_id)}
                >
                  <div className="text-yellow-300">
                    {alert.session_id?.slice(0, 8)}...
                  </div>
                  <div className="text-yellow-500">
                    {alert.data?.token_percent?.toFixed(1)}% 사용
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Active sessions */}
        <div className="flex-1 overflow-auto p-2">
          <div className="text-xs text-dark-400 px-2 mb-2">
            활성 세션 ({activeSessions.length})
          </div>
          {activeSessions.length === 0 ? (
            <div className="text-center py-4 text-dark-500 text-sm">
              활성 세션이 없습니다
            </div>
          ) : (
            activeSessions.map(hs => (
              <div
                key={hs.session.id}
                onClick={() => setSelectedSessionId(hs.session.id)}
                className={clsx(
                  'p-3 rounded-lg cursor-pointer transition-colors',
                  selectedSessionId === hs.session.id
                    ? 'bg-primary-600/20 border border-primary-600/50'
                    : 'hover:bg-dark-700'
                )}
              >
                <div className="font-medium truncate">
                  {hs.session.title || hs.session.id.slice(0, 12)}
                </div>
                <div className="text-xs text-dark-400 capitalize">
                  {hs.session.type}
                </div>
              </div>
            ))
          )}
        </div>
      </div>

      {/* Right panel - Attention detail */}
      <div className="flex-1 overflow-auto">
        {selectedSessionId ? (
          <div className="p-6 space-y-6">
            {loading ? (
              <div className="flex items-center justify-center h-64">
                <RefreshCw size={32} className="animate-spin text-dark-400" />
              </div>
            ) : (
              <>
                {/* Attention Gauge */}
                <div className="bg-dark-800 border border-dark-700 rounded-lg p-6">
                  <h3 className="text-lg font-semibold mb-4">토큰 사용량</h3>
                  <AttentionGauge attention={displayAttention} size="lg" />
                </div>

                {/* Loaded context */}
                <div className="grid grid-cols-2 gap-4">
                  {/* Files */}
                  <div className="bg-dark-800 border border-dark-700 rounded-lg p-4">
                    <h3 className="text-sm font-medium text-dark-300 mb-3 flex items-center gap-2">
                      <FileText size={16} />
                      로드된 파일 ({displayAttention.loaded_files?.length || 0})
                    </h3>
                    <div className="space-y-1 max-h-40 overflow-auto">
                      {displayAttention.loaded_files && displayAttention.loaded_files.length > 0 ? (
                        displayAttention.loaded_files.map((file, i) => (
                          <div key={i} className="text-xs text-dark-400 truncate font-mono">
                            {file}
                          </div>
                        ))
                      ) : (
                        <div className="text-xs text-dark-500">없음</div>
                      )}
                    </div>
                  </div>

                  {/* Conventions */}
                  <div className="bg-dark-800 border border-dark-700 rounded-lg p-4">
                    <h3 className="text-sm font-medium text-dark-300 mb-3 flex items-center gap-2">
                      <BookOpen size={16} />
                      로드된 컨벤션 ({displayAttention.loaded_conventions?.length || 0})
                    </h3>
                    <div className="space-y-1 max-h-40 overflow-auto">
                      {displayAttention.loaded_conventions && displayAttention.loaded_conventions.length > 0 ? (
                        displayAttention.loaded_conventions.map((conv, i) => (
                          <div key={i} className="text-xs text-dark-400 truncate">
                            {conv}
                          </div>
                        ))
                      ) : (
                        <div className="text-xs text-dark-500">없음</div>
                      )}
                    </div>
                  </div>
                </div>

                {/* Report recommendations */}
                {report?.recommendations && report.recommendations.length > 0 && (
                  <div className="bg-dark-800 border border-dark-700 rounded-lg p-4">
                    <h3 className="text-sm font-medium text-dark-300 mb-3">권장사항</h3>
                    <ul className="space-y-2">
                      {report.recommendations.map((rec: string, i: number) => (
                        <li key={i} className="text-sm text-dark-400 flex items-start gap-2">
                          <span className="text-primary-400">•</span>
                          {rec}
                        </li>
                      ))}
                    </ul>
                  </div>
                )}

                {/* Compact history */}
                <div className="bg-dark-800 border border-dark-700 rounded-lg p-4">
                  <h3 className="text-sm font-medium text-dark-300 mb-3 flex items-center gap-2">
                    <Clock size={16} />
                    Compact 이력 ({history.length})
                  </h3>
                  {history.length === 0 ? (
                    <div className="text-sm text-dark-500">Compact 이력이 없습니다</div>
                  ) : (
                    <div className="space-y-2">
                      {history.map((event, i) => (
                        <div key={i} className="p-3 bg-dark-700 rounded-lg">
                          <div className="flex items-center justify-between">
                            <span className="text-sm font-medium">
                              {event.trigger_reason}
                            </span>
                            <span className="text-xs text-dark-400">
                              {event.created_at && formatDistanceToNow(new Date(event.created_at), {
                                addSuffix: true,
                                locale: ko
                              })}
                            </span>
                          </div>
                          <div className="text-xs text-dark-400 mt-1">
                            {event.before_tokens?.toLocaleString()} → {event.after_tokens?.toLocaleString()} 토큰
                          </div>
                          {event.recovery_hint && (
                            <div className="text-xs text-primary-400 mt-1">
                              💡 {event.recovery_hint}
                            </div>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              </>
            )}
          </div>
        ) : (
          <div className="h-full flex items-center justify-center text-dark-400">
            <div className="text-center">
              <Activity size={48} className="mx-auto mb-4 opacity-50" />
              <p>세션을 선택하세요</p>
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
