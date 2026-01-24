import { useState } from 'react'
import { 
  Activity, GitBranch, Users, Layers, TrendingUp, Clock,
  CheckCircle, XCircle, AlertCircle, RefreshCw, Server
} from 'lucide-react'
import clsx from 'clsx'
import { formatDistanceToNow } from 'date-fns'
import { ko } from 'date-fns/locale'
import type { ApiStatus, SSEEvent } from '../hooks'
import { OrchestrationProgress } from '../components'
import { useOrchestrations, useApp } from '../hooks'

interface DashboardProps {
  status: ApiStatus | null
  events: SSEEvent[]
}

export default function Dashboard({ status, events }: DashboardProps) {
  const { orchestrations, loading } = useOrchestrations()
  const { serverRunning, serverPort, restartServer } = useApp()
  const [restarting, setRestarting] = useState(false)

  const runningOrchestrations = orchestrations.filter(o => o.status === 'running')

  const handleRestart = async () => {
    setRestarting(true)
    try {
      await restartServer()
    } finally {
      setRestarting(false)
    }
  }

  const stats = [
    { label: 'Active Builds', value: status?.builds?.active ?? '-', total: status?.builds?.total, icon: Layers, color: 'text-blue-400', bgColor: 'bg-blue-500/10' },
    { label: 'Running Orchestrations', value: status?.orchestrations?.running ?? '-', total: status?.orchestrations?.total, icon: GitBranch, color: 'text-green-400', bgColor: 'bg-green-500/10' },
    { label: 'Total Agents', value: status?.agents?.total ?? '-', icon: Users, color: 'text-purple-400', bgColor: 'bg-purple-500/10' },
  ]

  const isConnected = status !== null

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">대시보드</h1>
          <p className="text-dark-400">PAL Kit 상태 개요</p>
        </div>
        
        <div className="flex items-center gap-3">
          <div className={clsx(
            'flex items-center gap-2 px-3 py-1.5 rounded-lg',
            isConnected ? 'bg-green-900/30 text-green-400' : 'bg-red-900/30 text-red-400'
          )}>
            <Server size={16} />
            <span className="text-sm">
              {isConnected ? `연결됨 :${serverPort}` : '연결 안됨'}
            </span>
          </div>
          
          <button onClick={handleRestart} disabled={restarting}
            className="flex items-center gap-2 px-3 py-1.5 bg-dark-700 hover:bg-dark-600 rounded-lg text-sm disabled:opacity-50">
            <RefreshCw size={16} className={restarting ? 'animate-spin' : ''} />
            {restarting ? '재시작...' : '서버 재시작'}
          </button>
        </div>
      </div>

      {/* Warning */}
      {!isConnected && (
        <div className="bg-yellow-900/30 border border-yellow-700 rounded-lg p-4 flex items-start gap-3">
          <AlertCircle className="text-yellow-500 flex-shrink-0 mt-0.5" size={20} />
          <div>
            <h3 className="font-medium text-yellow-200">서버 연결 확인</h3>
            <p className="text-sm text-yellow-300 mt-1">
              PAL Kit 서버에 연결할 수 없습니다. 서버 재시작을 시도해주세요.
            </p>
          </div>
        </div>
      )}

      {/* Stats */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
        {stats.map(stat => (
          <div key={stat.label} className="bg-dark-800 border border-dark-700 rounded-lg p-4">
            <div className="flex items-center justify-between">
              <div className={clsx('p-2 rounded-lg', stat.bgColor)}>
                <stat.icon size={24} className={stat.color} />
              </div>
              {stat.total !== undefined && <span className="text-xs text-dark-500">/ {stat.total}</span>}
            </div>
            <div className="mt-3">
              <div className="text-3xl font-bold">{stat.value}</div>
              <div className="text-sm text-dark-400">{stat.label}</div>
            </div>
          </div>
        ))}
      </div>

      {/* Running Orchestrations */}
      <div>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Activity size={20} className="text-green-400" />
          실행 중인 Orchestration
        </h2>
        
        {loading ? (
          <div className="bg-dark-800 border border-dark-700 rounded-lg p-8 text-center text-dark-400">
            <Activity size={32} className="mx-auto mb-2 animate-spin" />
          </div>
        ) : runningOrchestrations.length === 0 ? (
          <div className="bg-dark-800 border border-dark-700 rounded-lg p-8 text-center text-dark-400">
            <GitBranch size={32} className="mx-auto mb-2 opacity-50" />
            <p>실행 중인 Orchestration이 없습니다</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {runningOrchestrations.map(orch => (
              <OrchestrationProgress key={orch.id} orchestration={orch} />
            ))}
          </div>
        )}
      </div>

      {/* Recent Events */}
      <div>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <TrendingUp size={20} className="text-blue-400" />
          최근 이벤트
        </h2>
        
        {events.length === 0 ? (
          <div className="bg-dark-800 border border-dark-700 rounded-lg p-8 text-center text-dark-400">
            <Clock size={32} className="mx-auto mb-2 opacity-50" />
            <p>이벤트가 없습니다</p>
          </div>
        ) : (
          <div className="bg-dark-800 border border-dark-700 rounded-lg divide-y divide-dark-700">
            {events.slice(0, 10).map((event, index) => (
              <div key={`${event.timestamp}-${event.type}-${index}`} className="p-3 flex items-center gap-3">
                {event.type.includes('complete') || event.type === 'connected' ? 
                  <CheckCircle size={16} className="text-green-500" /> :
                 event.type.includes('fail') || event.type === 'disconnected' ? 
                  <XCircle size={16} className="text-red-500" /> :
                  <Activity size={16} className="text-blue-500" />}
                <div className="flex-1">
                  <span className="text-sm font-medium">{formatEventType(event.type)}</span>
                </div>
                <span className="text-xs text-dark-500">
                  {formatDistanceToNow(new Date(event.timestamp), { addSuffix: true, locale: ko })}
                </span>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function formatEventType(type: string): string {
  const labels: Record<string, string> = {
    'connected': '서버 연결됨',
    'disconnected': '서버 연결 끊김',
    'session.start': '세션 시작',
    'session.end': '세션 종료',
    'orchestration.start': 'Orchestration 시작',
    'orchestration.complete': 'Orchestration 완료',
  }
  return labels[type] || type
}
