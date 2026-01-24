import { ChevronRight } from 'lucide-react'
import clsx from 'clsx'
import AgentWorkflowCard, { type WorkflowAgent } from './AgentWorkflowCard'

export interface WorkflowPhase {
  id: string
  name: string
  description: string
  order: number
  icon: React.ElementType
  color: string
  bgColor: string
  borderColor: string
}

interface AgentPhaseColumnProps {
  phase: WorkflowPhase
  agents: WorkflowAgent[]
  selectedAgent: string | null
  onSelectAgent: (agent: WorkflowAgent) => void
  showConnector?: boolean
  compact?: boolean
}

export default function AgentPhaseColumn({
  phase,
  agents,
  selectedAgent,
  onSelectAgent,
  showConnector = true,
  compact = false,
}: AgentPhaseColumnProps) {
  const PhaseIcon = phase.icon

  return (
    <div className="flex items-stretch">
      {/* Phase column */}
      <div
        className={clsx(
          'flex flex-col rounded-xl border overflow-hidden',
          phase.bgColor,
          phase.borderColor,
          compact ? 'w-56' : 'w-72'
        )}
      >
        {/* Phase header */}
        <div className={clsx('p-3 border-b', phase.borderColor)}>
          <div className="flex items-center gap-2 mb-1">
            <div className={clsx('p-1.5 rounded-lg', phase.color.replace('text-', 'bg-') + '/20')}>
              <PhaseIcon size={16} className={phase.color} />
            </div>
            <div>
              <h3 className="font-semibold text-sm">{phase.name}</h3>
              <span className="text-[10px] text-dark-500">
                {agents.length}개 에이전트
              </span>
            </div>
          </div>
          <p className="text-xs text-dark-400">{phase.description}</p>
        </div>

        {/* Agent list */}
        <div className={clsx('flex-1 p-2 space-y-2 overflow-auto', compact ? 'max-h-72' : 'max-h-96')}>
          {agents.length === 0 ? (
            <div className="text-center py-4 text-dark-500 text-xs">
              에이전트 없음
            </div>
          ) : (
            agents.map((agent) => (
              <AgentWorkflowCard
                key={agent.id}
                agent={agent}
                onClick={() => onSelectAgent(agent)}
                selected={selectedAgent === agent.id}
                compact={compact}
              />
            ))
          )}
        </div>
      </div>

      {/* Connector arrow */}
      {showConnector && (
        <div className="flex items-center px-2">
          <div className="flex items-center text-dark-600">
            <div className="w-4 h-px bg-dark-600" />
            <ChevronRight size={16} className="-ml-1" />
          </div>
        </div>
      )}
    </div>
  )
}
