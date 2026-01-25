import { Wifi, WifiOff, Activity, GitBranch, Users } from 'lucide-react'
import clsx from 'clsx'
import type { ApiStatus } from '../hooks/useApi'

interface StatusBarProps {
  status: ApiStatus | null
  connected: boolean
}

export default function StatusBar({ status, connected }: StatusBarProps) {
  return (
    <div className="h-6 bg-dark-800 border-t border-dark-700 flex items-center justify-between px-3 text-xs text-dark-400">
      {/* Left side - connection status */}
      <div className="flex items-center gap-4">
        <div className={clsx(
          'flex items-center gap-1',
          connected ? 'text-green-500' : 'text-red-500'
        )}>
          {connected ? <Wifi size={12} /> : <WifiOff size={12} />}
          <span>{connected ? '연결됨' : '연결 끊김'}</span>
        </div>
      </div>

      {/* Right side - stats */}
      <div className="flex items-center gap-4">
        {status?.orchestrations && (
          <div className="flex items-center gap-1">
            <GitBranch size={12} />
            <span>
              {status.orchestrations.running}/{status.orchestrations.total} Orchestrations
            </span>
          </div>
        )}
        
        {status?.builds && (
          <div className="flex items-center gap-1">
            <Activity size={12} />
            <span>
              {status.builds.active}/{status.builds.total} Builds
            </span>
          </div>
        )}
        
        {status?.agents && (
          <div className="flex items-center gap-1">
            <Users size={12} />
            <span>{status.agents.total} Agents</span>
          </div>
        )}
      </div>
    </div>
  )
}
