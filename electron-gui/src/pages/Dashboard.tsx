import { useState } from 'react'
import {
  Activity, GitBranch, Users, Layers, TrendingUp, Clock,
  CheckCircle, XCircle, AlertCircle, RefreshCw, Server,
  FolderPlus, FolderOpen, Settings, Trash2, Plus, X
} from 'lucide-react'
import clsx from 'clsx'
import { formatDistanceToNow } from 'date-fns'
import { ko } from 'date-fns/locale'
import type { ApiStatus, SSEEvent } from '../hooks'
import { OrchestrationProgress } from '../components'
import { useOrchestrations, useApp, useProjects, type Project } from '../hooks'

interface DashboardProps {
  status: ApiStatus | null
  events: SSEEvent[]
}

export default function Dashboard({ status, events }: DashboardProps) {
  const { orchestrations, loading: orchLoading } = useOrchestrations()
  const { serverRunning, serverPort, restartServer } = useApp()
  const { projects, loading: projectsLoading, importProject, initProject, removeProject, fetchProjects } = useProjects()

  const [restarting, setRestarting] = useState(false)
  const [showAddDialog, setShowAddDialog] = useState(false)
  const [addMode, setAddMode] = useState<'import' | 'init'>('import')
  const [newProjectPath, setNewProjectPath] = useState('')
  const [newProjectName, setNewProjectName] = useState('')
  const [adding, setAdding] = useState(false)
  const [addError, setAddError] = useState<string | null>(null)
  const [selectedProject, setSelectedProject] = useState<Project | null>(null)

  const runningOrchestrations = orchestrations.filter(o => o.status === 'running')

  const handleRestart = async () => {
    setRestarting(true)
    try {
      await restartServer()
    } finally {
      setRestarting(false)
    }
  }

  const handleAddProject = async () => {
    if (!newProjectPath.trim()) {
      setAddError('경로를 입력하세요')
      return
    }

    setAdding(true)
    setAddError(null)

    try {
      const result = addMode === 'import'
        ? await importProject(newProjectPath.trim(), newProjectName.trim() || undefined)
        : await initProject(newProjectPath.trim(), newProjectName.trim() || undefined)

      if (result) {
        setShowAddDialog(false)
        setNewProjectPath('')
        setNewProjectName('')
      } else {
        setAddError('프로젝트 추가에 실패했습니다')
      }
    } catch (err) {
      setAddError(err instanceof Error ? err.message : '알 수 없는 오류')
    } finally {
      setAdding(false)
    }
  }

  const handleRemoveProject = async (root: string) => {
    if (!confirm('프로젝트를 목록에서 제거하시겠습니까? (파일은 삭제되지 않습니다)')) return
    await removeProject(root)
  }

  const isConnected = status !== null

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold">대시보드</h1>
          <p className="text-dark-400">PAL Kit 프로젝트 관리</p>
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

      {/* Projects Section */}
      <div>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <FolderOpen size={20} className="text-blue-400" />
            프로젝트
          </h2>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setShowAddDialog(true)}
              className="flex items-center gap-2 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 rounded-lg text-sm"
            >
              <Plus size={16} />
              프로젝트 추가
            </button>
            <button
              onClick={() => fetchProjects()}
              disabled={projectsLoading}
              className="p-2 hover:bg-dark-700 rounded"
            >
              <RefreshCw size={16} className={projectsLoading ? 'animate-spin' : ''} />
            </button>
          </div>
        </div>

        {projectsLoading ? (
          <div className="flex items-center justify-center h-32">
            <RefreshCw size={24} className="animate-spin text-dark-400" />
          </div>
        ) : projects.length === 0 ? (
          <div className="bg-dark-800 border border-dark-700 border-dashed rounded-lg p-8 text-center">
            <FolderPlus size={48} className="mx-auto mb-4 text-dark-500" />
            <h3 className="text-lg font-medium mb-2">프로젝트가 없습니다</h3>
            <p className="text-dark-400 mb-4">
              기존 프로젝트를 가져오거나 새 프로젝트를 초기화하세요.
            </p>
            <button
              onClick={() => setShowAddDialog(true)}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 rounded-lg"
            >
              프로젝트 추가
            </button>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
            {projects.map(project => (
              <div
                key={project.root}
                onClick={() => setSelectedProject(selectedProject?.root === project.root ? null : project)}
                className={clsx(
                  'bg-dark-800 border rounded-lg p-4 cursor-pointer transition-all',
                  selectedProject?.root === project.root
                    ? 'border-primary-500 ring-1 ring-primary-500/50'
                    : 'border-dark-700 hover:border-dark-600'
                )}
              >
                <div className="flex items-start justify-between mb-3">
                  <div className="flex items-center gap-2">
                    <FolderOpen size={20} className="text-blue-400" />
                    <div>
                      <h3 className="font-medium">{project.name}</h3>
                      <p className="text-xs text-dark-400 truncate max-w-[200px]">{project.root}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-1">
                    {project.initialized ? (
                      <span className="text-xs px-1.5 py-0.5 bg-green-500/20 text-green-400 rounded">
                        초기화됨
                      </span>
                    ) : (
                      <span className="text-xs px-1.5 py-0.5 bg-yellow-500/20 text-yellow-400 rounded">
                        미초기화
                      </span>
                    )}
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-2 mb-3">
                  <div className="text-center">
                    <div className="text-lg font-bold">{project.session_count}</div>
                    <div className="text-xs text-dark-400">세션</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-bold">
                      {project.active_ports}/{project.port_count}
                    </div>
                    <div className="text-xs text-dark-400">포트</div>
                  </div>
                  <div className="text-center">
                    <div className="text-lg font-bold">
                      {project.total_tokens > 1000
                        ? `${(project.total_tokens / 1000).toFixed(1)}K`
                        : project.total_tokens}
                    </div>
                    <div className="text-xs text-dark-400">토큰</div>
                  </div>
                </div>

                <div className="flex items-center justify-between text-xs text-dark-400">
                  {project.last_active ? (
                    <span className="flex items-center gap-1">
                      <Clock size={12} />
                      {formatDistanceToNow(new Date(project.last_active), { addSuffix: true, locale: ko })}
                    </span>
                  ) : (
                    <span>-</span>
                  )}
                  <button
                    onClick={(e) => {
                      e.stopPropagation()
                      handleRemoveProject(project.root)
                    }}
                    className="p-1 text-dark-500 hover:text-red-400 hover:bg-red-500/10 rounded"
                    title="목록에서 제거"
                  >
                    <Trash2 size={14} />
                  </button>
                </div>
              </div>
            ))}

            {/* Add project card */}
            <div
              onClick={() => setShowAddDialog(true)}
              className="bg-dark-800/50 border border-dark-700 border-dashed rounded-lg p-4 cursor-pointer hover:border-primary-500/50 hover:bg-dark-800 transition-all flex flex-col items-center justify-center min-h-[180px]"
            >
              <Plus size={32} className="text-dark-500 mb-2" />
              <span className="text-dark-400">프로젝트 추가</span>
            </div>
          </div>
        )}
      </div>

      {/* Running Orchestrations */}
      <div>
        <h2 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <Activity size={20} className="text-green-400" />
          실행 중인 Orchestration
        </h2>

        {orchLoading ? (
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

      {/* Add Project Dialog */}
      {showAddDialog && (
        <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-dark-800 border border-dark-700 rounded-lg p-6 w-[480px] max-w-[90vw]">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-lg font-semibold">프로젝트 추가</h3>
              <button
                onClick={() => {
                  setShowAddDialog(false)
                  setAddError(null)
                }}
                className="text-dark-400 hover:text-dark-200"
              >
                <X size={20} />
              </button>
            </div>

            {/* Mode toggle */}
            <div className="flex gap-2 mb-4">
              <button
                onClick={() => setAddMode('import')}
                className={clsx(
                  'flex-1 py-2 rounded-lg text-sm transition-colors',
                  addMode === 'import'
                    ? 'bg-primary-600 text-white'
                    : 'bg-dark-700 text-dark-300 hover:bg-dark-600'
                )}
              >
                <FolderOpen size={16} className="inline mr-1" />
                기존 프로젝트 가져오기
              </button>
              <button
                onClick={() => setAddMode('init')}
                className={clsx(
                  'flex-1 py-2 rounded-lg text-sm transition-colors',
                  addMode === 'init'
                    ? 'bg-primary-600 text-white'
                    : 'bg-dark-700 text-dark-300 hover:bg-dark-600'
                )}
              >
                <FolderPlus size={16} className="inline mr-1" />
                새 프로젝트 초기화
              </button>
            </div>

            <p className="text-sm text-dark-400 mb-4">
              {addMode === 'import'
                ? '기존 프로젝트 디렉토리를 PAL Kit에 등록합니다.'
                : '디렉토리에 .pal 폴더를 생성하고 PAL Kit을 초기화합니다.'}
            </p>

            {/* Path input */}
            <div className="mb-4">
              <label className="block text-sm text-dark-300 mb-1">프로젝트 경로 *</label>
              <input
                type="text"
                value={newProjectPath}
                onChange={(e) => setNewProjectPath(e.target.value)}
                placeholder="/path/to/project"
                className="w-full px-3 py-2 bg-dark-700 border border-dark-600 rounded-lg focus:border-primary-500 focus:outline-none"
              />
            </div>

            {/* Name input */}
            <div className="mb-4">
              <label className="block text-sm text-dark-300 mb-1">프로젝트 이름 (선택)</label>
              <input
                type="text"
                value={newProjectName}
                onChange={(e) => setNewProjectName(e.target.value)}
                placeholder="디렉토리명이 기본값으로 사용됩니다"
                className="w-full px-3 py-2 bg-dark-700 border border-dark-600 rounded-lg focus:border-primary-500 focus:outline-none"
              />
            </div>

            {/* Error */}
            {addError && (
              <div className="mb-4 p-3 bg-red-500/10 border border-red-500/30 rounded-lg text-red-400 text-sm">
                {addError}
              </div>
            )}

            {/* Actions */}
            <div className="flex justify-end gap-2">
              <button
                onClick={() => {
                  setShowAddDialog(false)
                  setAddError(null)
                }}
                className="px-4 py-2 text-dark-300 hover:text-dark-100"
              >
                취소
              </button>
              <button
                onClick={handleAddProject}
                disabled={adding || !newProjectPath.trim()}
                className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded-lg"
              >
                {adding ? '추가 중...' : addMode === 'import' ? '가져오기' : '초기화'}
              </button>
            </div>
          </div>
        </div>
      )}
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
