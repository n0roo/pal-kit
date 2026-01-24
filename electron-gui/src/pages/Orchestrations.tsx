import { useState } from 'react'
import { GitBranch, Plus, RefreshCw, Filter } from 'lucide-react'
import { useOrchestrations } from '../hooks'
import { OrchestrationProgress } from '../components'
import clsx from 'clsx'

type StatusFilter = '' | 'running' | 'complete' | 'failed' | 'pending'

export default function Orchestrations() {
  const { orchestrations, loading, fetchOrchestrations, getStats } = useOrchestrations()
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('')
  const [selectedOrch, setSelectedOrch] = useState<string | null>(null)
  const [stats, setStats] = useState<any>(null)

  const handleFilterChange = (status: StatusFilter) => {
    setStatusFilter(status)
    fetchOrchestrations(status)
  }

  const handleSelectOrch = async (id: string) => {
    setSelectedOrch(id)
    const orchStats = await getStats(id)
    setStats(orchStats)
  }

  const filters: { label: string; value: StatusFilter }[] = [
    { label: '전체', value: '' },
    { label: '실행 중', value: 'running' },
    { label: '완료', value: 'complete' },
    { label: '실패', value: 'failed' },
    { label: '대기', value: 'pending' },
  ]

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-dark-700">
        <div className="flex items-center justify-between mb-4">
          <h1 className="text-xl font-semibold flex items-center gap-2">
            <GitBranch size={24} className="text-green-400" />
            Orchestrations
          </h1>
          <div className="flex items-center gap-2">
            <button
              onClick={() => fetchOrchestrations(statusFilter)}
              className="p-2 hover:bg-dark-700 rounded"
              disabled={loading}
            >
              <RefreshCw size={18} className={loading ? 'animate-spin' : ''} />
            </button>
            <button className="flex items-center gap-2 px-3 py-2 bg-primary-600 hover:bg-primary-700 rounded-lg text-sm">
              <Plus size={16} />
              새 Orchestration
            </button>
          </div>
        </div>

        {/* Filter tabs */}
        <div className="flex gap-2">
          {filters.map(filter => (
            <button
              key={filter.value}
              onClick={() => handleFilterChange(filter.value)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-sm transition-colors',
                statusFilter === filter.value
                  ? 'bg-primary-600 text-white'
                  : 'bg-dark-700 text-dark-300 hover:bg-dark-600'
              )}
            >
              {filter.label}
            </button>
          ))}
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-4">
        {loading ? (
          <div className="flex items-center justify-center h-64">
            <RefreshCw size={32} className="animate-spin text-dark-400" />
          </div>
        ) : orchestrations.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-64 text-dark-400">
            <GitBranch size={48} className="mb-4 opacity-50" />
            <p>Orchestration이 없습니다</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {orchestrations.map(orch => (
              <OrchestrationProgress
                key={orch.id}
                orchestration={orch}
                onClick={() => handleSelectOrch(orch.id)}
              />
            ))}
          </div>
        )}
      </div>

      {/* Stats sidebar (when selected) */}
      {selectedOrch && stats && (
        <div className="absolute right-0 top-0 h-full w-80 bg-dark-800 border-l border-dark-700 p-4 shadow-xl">
          <div className="flex items-center justify-between mb-4">
            <h3 className="font-semibold">통계</h3>
            <button
              onClick={() => setSelectedOrch(null)}
              className="text-dark-400 hover:text-dark-200"
            >
              ✕
            </button>
          </div>
          
          <div className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <StatCard label="Total Ports" value={stats.total_ports} />
              <StatCard label="Completed" value={stats.completed_ports} color="green" />
              <StatCard label="Running" value={stats.running_ports} color="blue" />
              <StatCard label="Failed" value={stats.failed_ports} color="red" />
            </div>
            
            <div className="pt-3 border-t border-dark-700">
              <div className="text-sm text-dark-400 mb-2">Workers</div>
              <div className="text-2xl font-bold">{stats.total_workers}</div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function StatCard({ label, value, color }: { label: string; value: number; color?: string }) {
  const colorClass = color === 'green' ? 'text-green-400' :
                     color === 'blue' ? 'text-blue-400' :
                     color === 'red' ? 'text-red-400' : 'text-dark-200'
  
  return (
    <div className="bg-dark-700 rounded-lg p-3">
      <div className="text-xs text-dark-400">{label}</div>
      <div className={clsx('text-xl font-bold', colorClass)}>{value}</div>
    </div>
  )
}
