import { useState, useEffect } from 'react'
import {
  Users, RefreshCw, GitCommit, TrendingUp, TrendingDown,
  Server, Folder, Eye, X, ChevronRight, Cpu, Code, FileText,
  GitBranch, LayoutGrid, FolderOpen
} from 'lucide-react'
import { useAgents, useProjects } from '../hooks'
import { AgentCard, MarkdownViewer, AgentWorkflowPipeline } from '../components'
import clsx from 'clsx'
import type { Agent } from '../hooks/useApi'

type TypeFilter = '' | 'spec' | 'operator' | 'worker' | 'test'
type SourceFilter = 'all' | 'system' | 'project'
type ViewMode = 'grid' | 'workflow'

export default function Agents() {
  const { agents, loading, fetchAgents, getSpec, getVersions, compareVersions } = useAgents()
  const { projects } = useProjects()
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('')
  const [sourceFilter, setSourceFilter] = useState<SourceFilter>('all')
  const [selectedProject, setSelectedProject] = useState<string>('')
  const [selectedAgent, setSelectedAgent] = useState<Agent | null>(null)
  const [spec, setSpec] = useState<string | null>(null)
  const [loadingSpec, setLoadingSpec] = useState(false)
  const [versions, setVersions] = useState<any[]>([])
  const [comparison, setComparison] = useState<any>(null)
  const [compareV1, setCompareV1] = useState<number>(0)
  const [compareV2, setCompareV2] = useState<number>(0)
  const [showSpec, setShowSpec] = useState(true)
  const [viewMode, setViewMode] = useState<ViewMode>('workflow')

  const handleFilterChange = (type: TypeFilter) => {
    setTypeFilter(type)
    fetchAgents(type, sourceFilter !== 'project')
  }

  const handleSourceFilterChange = (source: SourceFilter) => {
    setSourceFilter(source)
    fetchAgents(typeFilter, source !== 'project')
  }

  const handleSelectAgent = async (agent: Agent) => {
    setSelectedAgent(agent)
    setLoadingSpec(true)
    setSpec(null)
    
    // Load spec
    const specContent = await getSpec(agent.id)
    setSpec(specContent)
    setLoadingSpec(false)
    
    // Load versions (only for project agents)
    if (!agent.is_system) {
      const v = await getVersions(agent.id)
      setVersions(v)
    } else {
      setVersions([])
    }
    setComparison(null)
  }

  const handleCompare = async () => {
    if (selectedAgent && compareV1 && compareV2) {
      const result = await compareVersions(selectedAgent.id, compareV1, compareV2)
      setComparison(result)
    }
  }

  // Filter agents by source and project
  const filteredAgents = agents.filter(agent => {
    // Source filter
    if (sourceFilter === 'system' && !agent.is_system) return false
    if (sourceFilter === 'project' && agent.is_system) return false

    // Project filter (only for project agents)
    if (selectedProject && !agent.is_system) {
      // Filter by project root path
      if (agent.source && !agent.source.includes(selectedProject)) return false
    }

    return true
  })

  // Group agents by category
  const systemAgents = filteredAgents.filter(a => a.is_system)
  const projectAgents = filteredAgents.filter(a => !a.is_system)

  const typeFilters: { label: string; value: TypeFilter }[] = [
    { label: '전체', value: '' },
    { label: 'Spec', value: 'spec' },
    { label: 'Operator', value: 'operator' },
    { label: 'Worker', value: 'worker' },
    { label: 'Test', value: 'test' },
  ]

  const sourceFilters: { label: string; value: SourceFilter; icon: any }[] = [
    { label: '전체', value: 'all', icon: Users },
    { label: '시스템', value: 'system', icon: Server },
    { label: '프로젝트', value: 'project', icon: Folder },
  ]

  const getTypeIcon = (type: string) => {
    switch(type) {
      case 'spec': return <FileText size={14} />
      case 'operator': return <Cpu size={14} />
      case 'worker': return <Code size={14} />
      default: return <Users size={14} />
    }
  }

  return (
    <div className="h-full flex">
      {/* Main content */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <div className="p-4 border-b border-dark-700">
          <div className="flex items-center justify-between mb-4">
            <h1 className="text-xl font-semibold flex items-center gap-2">
              <Users size={24} className="text-purple-400" />
              에이전트
            </h1>
            <div className="flex items-center gap-2">
              {/* View mode toggle */}
              <div className="flex bg-dark-700 rounded-lg p-0.5">
                <button
                  onClick={() => setViewMode('workflow')}
                  className={clsx(
                    'px-2 py-1 rounded text-xs flex items-center gap-1 transition-colors',
                    viewMode === 'workflow' ? 'bg-primary-600 text-white' : 'text-dark-400 hover:text-dark-200'
                  )}
                  title="워크플로우 뷰"
                >
                  <GitBranch size={14} />
                  워크플로우
                </button>
                <button
                  onClick={() => setViewMode('grid')}
                  className={clsx(
                    'px-2 py-1 rounded text-xs flex items-center gap-1 transition-colors',
                    viewMode === 'grid' ? 'bg-primary-600 text-white' : 'text-dark-400 hover:text-dark-200'
                  )}
                  title="그리드 뷰"
                >
                  <LayoutGrid size={14} />
                  그리드
                </button>
              </div>

              <button
                onClick={() => fetchAgents(typeFilter, sourceFilter !== 'project')}
                className="p-2 hover:bg-dark-700 rounded"
                disabled={loading}
              >
                <RefreshCw size={18} className={loading ? 'animate-spin' : ''} />
              </button>
            </div>
          </div>

          {/* Source filter + Project selector */}
          <div className="flex items-center gap-4 mb-3">
            <div className="flex gap-2">
              {sourceFilters.map(filter => (
                <button
                  key={filter.value}
                  onClick={() => handleSourceFilterChange(filter.value)}
                  className={clsx(
                    'flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm transition-colors',
                    sourceFilter === filter.value
                      ? 'bg-purple-600 text-white'
                      : 'bg-dark-700 text-dark-300 hover:bg-dark-600'
                  )}
                >
                  <filter.icon size={14} />
                  {filter.label}
                </button>
              ))}
            </div>

            {/* Project selector */}
            <div className="flex items-center gap-2">
              <FolderOpen size={16} className="text-dark-400" />
              <select
                value={selectedProject}
                onChange={(e) => setSelectedProject(e.target.value)}
                className="px-3 py-1.5 bg-dark-800 border border-dark-600 rounded-lg text-sm min-w-[180px]"
              >
                <option value="">모든 프로젝트</option>
                {projects.map(p => (
                  <option key={p.root} value={p.root}>{p.name}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Type filter */}
          <div className="flex gap-2">
            {typeFilters.map(filter => (
              <button
                key={filter.value}
                onClick={() => handleFilterChange(filter.value)}
                className={clsx(
                  'px-3 py-1.5 rounded-lg text-sm transition-colors',
                  typeFilter === filter.value
                    ? 'bg-primary-600 text-white'
                    : 'bg-dark-700 text-dark-300 hover:bg-dark-600'
                )}
              >
                {filter.label}
              </button>
            ))}
          </div>
        </div>

        {/* Main content area */}
        <div className="flex-1 overflow-hidden">
          {viewMode === 'workflow' ? (
            /* Workflow Pipeline View */
            <AgentWorkflowPipeline
              agents={filteredAgents}
              onSelectAgent={handleSelectAgent}
            />
          ) : (
            /* Grid View */
            <div className="h-full overflow-auto p-4">
              {loading ? (
                <div className="flex items-center justify-center h-64">
                  <RefreshCw size={32} className="animate-spin text-dark-400" />
                </div>
              ) : filteredAgents.length === 0 ? (
                <div className="flex flex-col items-center justify-center h-64 text-dark-400">
                  <Users size={48} className="mb-4 opacity-50" />
                  <p>에이전트가 없습니다</p>
                </div>
              ) : (
                <div className="space-y-6">
                  {/* System agents section */}
                  {systemAgents.length > 0 && sourceFilter !== 'project' && (
                    <div>
                      <h2 className="text-sm font-medium text-dark-400 mb-3 flex items-center gap-2">
                        <Server size={14} />
                        시스템 에이전트 ({systemAgents.length})
                      </h2>
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
                        {systemAgents.map(agent => (
                          <div
                            key={agent.id}
                            onClick={() => handleSelectAgent(agent)}
                            className={clsx(
                              'p-3 rounded-lg border cursor-pointer transition-colors',
                              selectedAgent?.id === agent.id
                                ? 'bg-purple-600/20 border-purple-500/50'
                                : 'bg-dark-800 border-dark-700 hover:border-dark-500'
                            )}
                          >
                            <div className="flex items-start gap-3">
                              <div className="p-2 rounded-lg bg-purple-500/20 text-purple-400">
                                {getTypeIcon(agent.type)}
                              </div>
                              <div className="flex-1 min-w-0">
                                <div className="flex items-center gap-2">
                                  <span className="font-medium truncate">{agent.name}</span>
                                  <span className="text-xs px-1.5 py-0.5 bg-purple-500/20 text-purple-400 rounded">
                                    시스템
                                  </span>
                                </div>
                                <p className="text-xs text-dark-400 mt-0.5 truncate">
                                  {agent.description}
                                </p>
                                <div className="flex items-center gap-2 mt-1.5">
                                  <span className="text-xs px-1.5 py-0.5 bg-dark-700 text-dark-300 rounded">
                                    {agent.type}
                                  </span>
                                  {agent.capabilities?.slice(0, 2).map(cap => (
                                    <span key={cap} className="text-xs text-dark-500">
                                      {cap}
                                    </span>
                                  ))}
                                </div>
                              </div>
                              <ChevronRight size={16} className="text-dark-500" />
                            </div>
                          </div>
                        ))}
                      </div>
                    </div>
                  )}

                  {/* Project agents section */}
                  {projectAgents.length > 0 && sourceFilter !== 'system' && (
                    <div>
                      <h2 className="text-sm font-medium text-dark-400 mb-3 flex items-center gap-2">
                        <Folder size={14} />
                        프로젝트 에이전트 ({projectAgents.length})
                      </h2>
                      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                        {projectAgents.map(agent => (
                          <AgentCard
                            key={agent.id}
                            agent={agent}
                            onClick={() => handleSelectAgent(agent)}
                          />
                        ))}
                      </div>
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Detail sidebar */}
      {selectedAgent && (
        <div className="w-[500px] border-l border-dark-700 flex flex-col overflow-hidden">
          {/* Header */}
          <div className="p-4 border-b border-dark-700">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <h2 className="font-semibold">{selectedAgent.name}</h2>
                {selectedAgent.is_system && (
                  <span className="text-xs px-1.5 py-0.5 bg-purple-500/20 text-purple-400 rounded">
                    시스템
                  </span>
                )}
              </div>
              <button
                onClick={() => setSelectedAgent(null)}
                className="text-dark-400 hover:text-dark-200"
              >
                <X size={18} />
              </button>
            </div>
            <p className="text-sm text-dark-400 mt-1">{selectedAgent.description}</p>
            
            {/* Metadata */}
            <div className="flex flex-wrap gap-2 mt-3">
              <span className="text-xs px-2 py-1 bg-dark-700 rounded flex items-center gap-1">
                {getTypeIcon(selectedAgent.type)}
                {selectedAgent.type}
              </span>
              <span className="text-xs px-2 py-1 bg-dark-700 rounded">
                v{selectedAgent.current_version}
              </span>
              {selectedAgent.capabilities?.map(cap => (
                <span key={cap} className="text-xs px-2 py-1 bg-dark-700 rounded text-dark-300">
                  {cap}
                </span>
              ))}
            </div>
          </div>

          {/* Tab bar */}
          <div className="flex border-b border-dark-700">
            <button
              onClick={() => setShowSpec(true)}
              className={clsx(
                'flex-1 py-2 text-sm flex items-center justify-center gap-1.5',
                showSpec ? 'text-primary-400 border-b-2 border-primary-400' : 'text-dark-400'
              )}
            >
              <Eye size={14} />
              명세
            </button>
            {!selectedAgent.is_system && (
              <button
                onClick={() => setShowSpec(false)}
                className={clsx(
                  'flex-1 py-2 text-sm flex items-center justify-center gap-1.5',
                  !showSpec ? 'text-primary-400 border-b-2 border-primary-400' : 'text-dark-400'
                )}
              >
                <GitCommit size={14} />
                버전
              </button>
            )}
          </div>

          {/* Content */}
          <div className="flex-1 overflow-auto p-4">
            {showSpec ? (
              // Spec viewer
              loadingSpec ? (
                <div className="flex items-center justify-center h-32">
                  <RefreshCw size={24} className="animate-spin text-dark-400" />
                </div>
              ) : spec ? (
                <div className="bg-dark-800 rounded-lg p-4 overflow-auto">
                  <MarkdownViewer content={spec} />
                </div>
              ) : (
                <div className="text-center text-dark-400 py-8">
                  명세를 불러올 수 없습니다
                </div>
              )
            ) : (
              // Version history (only for project agents)
              <>
                <h3 className="text-sm font-medium text-dark-300 mb-3 flex items-center gap-2">
                  <GitCommit size={16} />
                  버전 히스토리
                </h3>
                
                <div className="space-y-2 mb-6">
                  {versions.length === 0 ? (
                    <p className="text-sm text-dark-400">버전 기록이 없습니다</p>
                  ) : (
                    versions.map(version => (
                      <div
                        key={version.version}
                        className={clsx(
                          'p-3 rounded-lg border',
                          version.version === selectedAgent.current_version
                            ? 'bg-primary-600/10 border-primary-600/50'
                            : 'bg-dark-700 border-dark-600'
                        )}
                      >
                        <div className="flex items-center justify-between">
                          <span className="font-medium">v{version.version}</span>
                          {version.version === selectedAgent.current_version && (
                            <span className="text-xs px-2 py-0.5 bg-primary-600 rounded">
                              Current
                            </span>
                          )}
                        </div>
                        {version.change_summary && (
                          <p className="text-xs text-dark-400 mt-1">
                            {version.change_summary}
                          </p>
                        )}
                      </div>
                    ))
                  )}
                </div>

                {/* Version comparison */}
                {versions.length >= 2 && (
                  <div className="border-t border-dark-700 pt-4">
                    <h3 className="text-sm font-medium text-dark-300 mb-3">버전 비교</h3>
                    <div className="flex items-center gap-2 mb-3">
                      <select
                        value={compareV1}
                        onChange={(e) => setCompareV1(Number(e.target.value))}
                        className="flex-1 px-2 py-1.5 bg-dark-700 border border-dark-600 rounded text-sm"
                      >
                        <option value={0}>v 선택</option>
                        {versions.map(v => (
                          <option key={v.version} value={v.version}>v{v.version}</option>
                        ))}
                      </select>
                      <span className="text-dark-400">vs</span>
                      <select
                        value={compareV2}
                        onChange={(e) => setCompareV2(Number(e.target.value))}
                        className="flex-1 px-2 py-1.5 bg-dark-700 border border-dark-600 rounded text-sm"
                      >
                        <option value={0}>v 선택</option>
                        {versions.map(v => (
                          <option key={v.version} value={v.version}>v{v.version}</option>
                        ))}
                      </select>
                    </div>
                    <button
                      onClick={handleCompare}
                      disabled={!compareV1 || !compareV2}
                      className="w-full py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded-lg text-sm"
                    >
                      비교하기
                    </button>

                    {comparison && (
                      <div className="mt-4 space-y-2">
                        <ComparisonItem
                          label="Attention Score"
                          v1={comparison.v1?.attention_score}
                          v2={comparison.v2?.attention_score}
                        />
                        <ComparisonItem
                          label="Quality Score"
                          v1={comparison.v1?.quality_score}
                          v2={comparison.v2?.quality_score}
                        />
                        <ComparisonItem
                          label="Success Rate"
                          v1={comparison.v1?.success_rate}
                          v2={comparison.v2?.success_rate}
                          isPercent
                        />
                      </div>
                    )}
                  </div>
                )}
              </>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

function ComparisonItem({ 
  label, 
  v1, 
  v2, 
  isPercent 
}: { 
  label: string
  v1?: number
  v2?: number
  isPercent?: boolean 
}) {
  const format = (val?: number) => {
    if (val === undefined) return '-'
    return isPercent ? `${(val * 100).toFixed(1)}%` : val.toFixed(2)
  }

  const diff = v1 !== undefined && v2 !== undefined ? v2 - v1 : 0
  const improved = diff > 0

  return (
    <div className="flex items-center justify-between text-sm">
      <span className="text-dark-400">{label}</span>
      <div className="flex items-center gap-2">
        <span>{format(v1)}</span>
        <span className="text-dark-500">→</span>
        <span>{format(v2)}</span>
        {diff !== 0 && (
          <span className={clsx(
            'flex items-center gap-0.5 text-xs',
            improved ? 'text-green-400' : 'text-red-400'
          )}>
            {improved ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
            {Math.abs(diff).toFixed(2)}
          </span>
        )}
      </div>
    </div>
  )
}
