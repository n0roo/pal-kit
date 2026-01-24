import { useState, useEffect } from 'react'
import { BrowserRouter, Routes, Route, NavLink } from 'react-router-dom'
import {
  LayoutDashboard,
  GitBranch,
  Layers,
  Users,
  AlertTriangle,
  Activity,
  Settings,
  FileText,
  Globe,
  BookOpen
} from 'lucide-react'
import clsx from 'clsx'

// Pages
import Dashboard from './pages/Dashboard'
import Sessions from './pages/Sessions'
import Orchestrations from './pages/Orchestrations'
import Agents from './pages/Agents'
import Attention from './pages/Attention'
import Documents from './pages/Documents'
import SettingsPage from './pages/Settings'
import GlobalAgents from './pages/GlobalAgents'
import KnowledgeBase from './pages/KnowledgeBase'

// Components
import StatusBar from './components/StatusBar'
import CompactAlert from './components/CompactAlert'

// Hooks
import { useSSE } from './hooks/useSSE'
import { useApi } from './hooks/useApi'

function App() {
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false)
  const [platform, setPlatform] = useState<string>('')
  const { events, connected } = useSSE()
  const { status } = useApi()
  
  // Get platform for macOS-specific styling
  useEffect(() => {
    if (typeof window !== 'undefined' && (window as any).app?.getPlatform) {
      (window as any).app.getPlatform().then((p: string) => setPlatform(p))
    }
  }, [])
  
  const isMac = platform === 'darwin'
  
  // 최근 Attention 경고
  const [attentionAlerts, setAttentionAlerts] = useState<any[]>([])
  
  useEffect(() => {
    const attentionEvents = events.filter(e => e.type === 'attention.warning')
    if (attentionEvents.length > 0) {
      setAttentionAlerts(prev => [...attentionEvents, ...prev].slice(0, 5))
    }
  }, [events])

  const navItems = [
    { path: '/', icon: LayoutDashboard, label: '대시보드' },
    { path: '/sessions', icon: Layers, label: '세션' },
    { path: '/orchestrations', icon: GitBranch, label: 'Orchestration' },
    { path: '/agents', icon: Users, label: '에이전트' },
    { path: '/global-agents', icon: Globe, label: '전역 에이전트' },
    { path: '/documents', icon: FileText, label: '문서' },
    { path: '/knowledge-base', icon: BookOpen, label: 'KB' },
    { path: '/attention', icon: Activity, label: 'Attention' },
  ]

  return (
    <BrowserRouter>
      <div className="flex h-screen bg-dark-900 text-dark-100">
        {/* Sidebar */}
        <aside className={clsx(
          'flex flex-col bg-dark-800 border-r border-dark-700 transition-all duration-300',
          sidebarCollapsed ? 'w-16' : 'w-56'
        )}>
          {/* Logo - with macOS traffic light spacing */}
          <div className={clsx(
            "h-14 flex items-center border-b border-dark-700 drag-region",
            isMac ? "pl-20 pr-4" : "px-4" // macOS: 좌측 트래픽 라이트 공간 확보
          )}>
            <div className="flex items-center gap-2 no-drag">
              <div className="w-8 h-8 rounded-lg bg-primary-600 flex items-center justify-center">
                <span className="text-white font-bold">P</span>
              </div>
              {!sidebarCollapsed && (
                <span className="font-semibold text-lg">PAL Kit</span>
              )}
            </div>
          </div>

          {/* Navigation */}
          <nav className="flex-1 py-4">
            {navItems.map(item => (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) => clsx(
                  'flex items-center gap-3 px-4 py-2.5 mx-2 rounded-lg transition-colors',
                  isActive 
                    ? 'bg-primary-600/20 text-primary-400' 
                    : 'text-dark-400 hover:bg-dark-700 hover:text-dark-200'
                )}
              >
                <item.icon size={20} />
                {!sidebarCollapsed && <span>{item.label}</span>}
              </NavLink>
            ))}
          </nav>

          {/* Collapse toggle */}
          <button
            onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
            className="p-4 border-t border-dark-700 text-dark-400 hover:text-dark-200"
          >
            {sidebarCollapsed ? '→' : '←'}
          </button>
        </aside>

        {/* Main content */}
        <main className="flex-1 flex flex-col overflow-hidden">
          {/* Title bar (for window dragging on macOS) */}
          <div className={clsx(
            "bg-dark-800 border-b border-dark-700 drag-region flex items-center justify-between px-4",
            isMac ? "h-12 pt-2" : "h-10" // Taller on macOS for traffic lights
          )}>
            <div className="no-drag flex items-center gap-2">
              {/* Connection status */}
              <div className={clsx(
                'w-2 h-2 rounded-full',
                connected ? 'bg-green-500' : 'bg-red-500'
              )} />
              <span className="text-xs text-dark-400">
                {connected ? '연결됨' : '연결 끊김'}
              </span>
            </div>
            
            <div className="no-drag flex items-center gap-2">
              {attentionAlerts.length > 0 && (
                <button className="p-1 text-yellow-500 hover:bg-dark-700 rounded">
                  <AlertTriangle size={16} />
                </button>
              )}
              <NavLink 
                to="/settings" 
                className={({ isActive }) => clsx(
                  'p-1 rounded',
                  isActive ? 'text-primary-400 bg-dark-700' : 'text-dark-400 hover:bg-dark-700'
                )}
              >
                <Settings size={16} />
              </NavLink>
            </div>
          </div>

          {/* Page content */}
          <div className="flex-1 overflow-auto">
            <Routes>
              <Route path="/" element={<Dashboard status={status} events={events} />} />
              <Route path="/sessions" element={<Sessions />} />
              <Route path="/orchestrations" element={<Orchestrations />} />
              <Route path="/agents" element={<Agents />} />
              <Route path="/global-agents" element={<GlobalAgents />} />
              <Route path="/documents" element={<Documents />} />
              <Route path="/knowledge-base" element={<KnowledgeBase />} />
              <Route path="/attention" element={<Attention alerts={attentionAlerts} />} />
              <Route path="/settings" element={<SettingsPage />} />
            </Routes>
          </div>

          {/* Status bar */}
          <StatusBar status={status} connected={connected} />
        </main>

        {/* Compact alerts overlay */}
        <CompactAlert alerts={attentionAlerts} onDismiss={(id) => {
          setAttentionAlerts(prev => prev.filter(a => a.id !== id))
        }} />
      </div>
    </BrowserRouter>
  )
}

export default App
