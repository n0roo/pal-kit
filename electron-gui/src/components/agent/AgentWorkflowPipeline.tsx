import { useMemo, useState } from 'react'
import {
  FileEdit, Wrench, CheckSquare, HelpCircle,
  Lightbulb, PenTool, Code, TestTube,
  ScrollText, Headphones, X, Bot
} from 'lucide-react'
import clsx from 'clsx'
import AgentPhaseColumn, { type WorkflowPhase } from './AgentPhaseColumn'
import type { WorkflowAgent } from './AgentWorkflowCard'
import type { Agent } from '../../hooks/useApi'

interface AgentWorkflowPipelineProps {
  agents: Agent[]
  onSelectAgent?: (agent: Agent) => void
  compact?: boolean
}

// Define workflow phases (Support is now under Spec as helper)
const WORKFLOW_PHASES: WorkflowPhase[] = [
  {
    id: 'spec',
    name: 'Spec',
    description: '명세 설계, 계획 및 문서 지원',
    order: 1,
    icon: FileEdit,
    color: 'text-purple-400',
    bgColor: 'bg-purple-500/5',
    borderColor: 'border-purple-500/20',
  },
  {
    id: 'execution',
    name: 'Execution',
    description: '구현 및 실행',
    order: 2,
    icon: Wrench,
    color: 'text-blue-400',
    bgColor: 'bg-blue-500/5',
    borderColor: 'border-blue-500/20',
  },
  {
    id: 'validation',
    name: 'Validation',
    description: '검증 및 테스트',
    order: 3,
    icon: CheckSquare,
    color: 'text-orange-400',
    bgColor: 'bg-orange-500/5',
    borderColor: 'border-orange-500/20',
  },
]

// Agent type to phase mapping (Support is under Spec as helper)
const AGENT_PHASE_MAP: Record<string, string> = {
  // Spec phase (includes support as helper)
  planner: 'spec',
  architect: 'spec',
  'spec-writer': 'spec',
  'spec-reviewer': 'spec',
  spec: 'spec',
  docs: 'spec',        // Support - 문서 참조 지원
  support: 'spec',     // Support - Spec의 하위 에이전트
  documentation: 'spec',

  // Execution phase
  builder: 'execution',
  operator: 'execution',
  worker: 'execution',
  'impl-worker': 'execution',
  developer: 'execution',

  // Validation phase
  tester: 'validation',
  'test-worker': 'validation',
  reviewer: 'validation',
  logger: 'validation',
  test: 'validation',
}

// Agent purpose descriptions (for display)
const AGENT_PURPOSES: Record<string, string> = {
  planner: '프로젝트 전체 계획 수립 및 작업 분해',
  architect: '시스템 아키텍처 설계 및 기술 결정',
  'spec-writer': '상세 명세서 작성',
  'spec-reviewer': '명세서 검토 및 피드백',
  builder: '빌드 프로세스 관리 및 통합',
  operator: '워커 조율 및 작업 흐름 관리',
  'impl-worker': '실제 코드 구현',
  tester: '테스트 케이스 작성 및 실행',
  'test-worker': '단위/통합 테스트 수행',
  reviewer: '코드 리뷰 및 품질 검증',
  logger: '작업 로그 기록 및 추적',
  docs: '문서 작성 및 관리',
  support: '사용자 지원 및 Q&A',
}

function convertToWorkflowAgent(agent: Agent): WorkflowAgent {
  return {
    id: agent.id,
    name: agent.name,
    type: agent.type,
    description: agent.description,
    purpose: AGENT_PURPOSES[agent.id.toLowerCase()] || AGENT_PURPOSES[agent.type] || agent.description,
    capabilities: agent.capabilities,
    version: agent.current_version,
    status: 'idle',
  }
}

function getAgentPhase(agent: Agent): string {
  // Check by agent ID first
  const byId = AGENT_PHASE_MAP[agent.id.toLowerCase()]
  if (byId) return byId

  // Check by agent name (lowercase)
  const byName = AGENT_PHASE_MAP[agent.name.toLowerCase()]
  if (byName) return byName

  // Check by type
  const byType = AGENT_PHASE_MAP[agent.type]
  if (byType) return byType

  // Default based on type
  if (agent.type === 'spec') return 'spec'
  if (agent.type === 'operator') return 'execution'
  if (agent.type === 'worker') return 'execution'
  if (agent.type === 'test') return 'validation'

  return 'execution' // Default
}

export default function AgentWorkflowPipeline({
  agents,
  onSelectAgent,
  compact = false,
}: AgentWorkflowPipelineProps) {
  const [selectedAgentId, setSelectedAgentId] = useState<string | null>(null)
  const [showDetail, setShowDetail] = useState(false)

  // Group agents by phase (3 phases: Spec, Execution, Validation)
  const agentsByPhase = useMemo(() => {
    const grouped: Record<string, WorkflowAgent[]> = {
      spec: [],
      execution: [],
      validation: [],
    }

    agents.forEach((agent) => {
      const phase = getAgentPhase(agent)
      if (grouped[phase]) {
        grouped[phase].push(convertToWorkflowAgent(agent))
      }
    })

    return grouped
  }, [agents])

  const handleSelectAgent = (workflowAgent: WorkflowAgent) => {
    setSelectedAgentId(workflowAgent.id)
    setShowDetail(true)

    const originalAgent = agents.find((a) => a.id === workflowAgent.id)
    if (originalAgent && onSelectAgent) {
      onSelectAgent(originalAgent)
    }
  }

  const selectedAgent = useMemo(() => {
    if (!selectedAgentId) return null
    return agents.find((a) => a.id === selectedAgentId)
  }, [selectedAgentId, agents])

  return (
    <div className="flex flex-col h-full">
      {/* Pipeline header */}
      <div className="p-4 border-b border-dark-700">
        <h2 className="text-lg font-semibold mb-1">에이전트 워크플로우</h2>
        <p className="text-sm text-dark-400">
          명세 → 실행 → 검증 → 지원 단계로 구성된 에이전트 파이프라인
        </p>
      </div>

      {/* Pipeline visualization */}
      <div className="flex-1 overflow-auto p-4">
        <div className="flex items-stretch min-w-max">
          {WORKFLOW_PHASES.map((phase, index) => (
            <AgentPhaseColumn
              key={phase.id}
              phase={phase}
              agents={agentsByPhase[phase.id] || []}
              selectedAgent={selectedAgentId}
              onSelectAgent={handleSelectAgent}
              showConnector={index < WORKFLOW_PHASES.length - 1}
              compact={compact}
            />
          ))}
        </div>
      </div>

      {/* Selected agent detail */}
      {showDetail && selectedAgent && (
        <div className="border-t border-dark-700 p-4 bg-dark-850">
          <div className="flex items-start justify-between mb-3">
            <div className="flex items-center gap-3">
              <div className="p-2 bg-primary-600/20 rounded-lg">
                <Bot size={20} className="text-primary-400" />
              </div>
              <div>
                <h3 className="font-semibold">{selectedAgent.name}</h3>
                <p className="text-xs text-dark-400">{selectedAgent.type}</p>
              </div>
            </div>
            <button
              onClick={() => setShowDetail(false)}
              className="p-1 text-dark-400 hover:text-dark-200"
            >
              <X size={16} />
            </button>
          </div>

          <div className="grid grid-cols-2 gap-4">
            {/* Purpose */}
            <div>
              <h4 className="text-xs text-dark-400 mb-1 flex items-center gap-1">
                <Lightbulb size={12} />
                목적
              </h4>
              <p className="text-sm">
                {AGENT_PURPOSES[selectedAgent.id.toLowerCase()] ||
                  AGENT_PURPOSES[selectedAgent.type] ||
                  selectedAgent.description ||
                  '설명 없음'}
              </p>
            </div>

            {/* Capabilities */}
            <div>
              <h4 className="text-xs text-dark-400 mb-1 flex items-center gap-1">
                <Code size={12} />
                능력
              </h4>
              <div className="flex flex-wrap gap-1">
                {selectedAgent.capabilities?.map((cap) => (
                  <span
                    key={cap}
                    className="text-xs px-1.5 py-0.5 bg-dark-700 rounded"
                  >
                    {cap}
                  </span>
                )) || <span className="text-xs text-dark-500">없음</span>}
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
