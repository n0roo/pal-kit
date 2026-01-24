import { useState, useEffect } from 'react'
import { 
  Settings as SettingsIcon, Server, Database, FolderOpen, 
  RefreshCw, Check, X, Globe, Terminal, Cpu, Palette,
  Bell, Shield, Key, Save, RotateCcw
} from 'lucide-react'
import clsx from 'clsx'

interface ServerConfig {
  port: number
  projectRoot: string
  dbPath: string
  autoStart: boolean
}

interface UIConfig {
  theme: 'dark' | 'light' | 'system'
  compactMode: boolean
  showNotifications: boolean
  autoRefresh: boolean
  refreshInterval: number
}

interface MCPConfig {
  enabled: boolean
  serverPath: string
  claudeDesktopPath: string
}

export default function Settings() {
  const [activeTab, setActiveTab] = useState<'server' | 'ui' | 'mcp' | 'about'>('server')
  const [serverConfig, setServerConfig] = useState<ServerConfig>({
    port: 9000,
    projectRoot: '',
    dbPath: '.pal/pal.db',
    autoStart: true,
  })
  const [uiConfig, setUIConfig] = useState<UIConfig>({
    theme: 'dark',
    compactMode: false,
    showNotifications: true,
    autoRefresh: true,
    refreshInterval: 5000,
  })
  const [mcpConfig, setMCPConfig] = useState<MCPConfig>({
    enabled: false,
    serverPath: '',
    claudeDesktopPath: '',
  })
  const [saving, setSaving] = useState(false)
  const [saved, setSaved] = useState(false)

  // Load config on mount
  useEffect(() => {
    // In Electron mode, load from electron-store
    // In web mode, load from localStorage
    const loadedUI = localStorage.getItem('pal-ui-config')
    if (loadedUI) {
      try {
        setUIConfig(JSON.parse(loadedUI))
      } catch {}
    }
  }, [])

  const handleSave = async () => {
    setSaving(true)
    
    // Save UI config
    localStorage.setItem('pal-ui-config', JSON.stringify(uiConfig))
    
    // In Electron mode, would save server config via IPC
    await new Promise(resolve => setTimeout(resolve, 500))
    
    setSaving(false)
    setSaved(true)
    setTimeout(() => setSaved(false), 2000)
  }

  const tabs = [
    { id: 'server', label: '서버', icon: Server },
    { id: 'ui', label: 'UI', icon: Palette },
    { id: 'mcp', label: 'MCP', icon: Terminal },
    { id: 'about', label: '정보', icon: SettingsIcon },
  ] as const

  return (
    <div className="h-full flex">
      {/* Sidebar */}
      <div className="w-48 border-r border-dark-700 p-4">
        <h1 className="text-lg font-semibold mb-4 flex items-center gap-2">
          <SettingsIcon size={20} />
          설정
        </h1>
        
        <nav className="space-y-1">
          {tabs.map(tab => (
            <button
              key={tab.id}
              onClick={() => setActiveTab(tab.id)}
              className={clsx(
                'w-full flex items-center gap-2 px-3 py-2 rounded-lg text-sm transition-colors',
                activeTab === tab.id
                  ? 'bg-primary-600 text-white'
                  : 'text-dark-300 hover:bg-dark-700'
              )}
            >
              <tab.icon size={16} />
              {tab.label}
            </button>
          ))}
        </nav>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto p-6">
        {activeTab === 'server' && (
          <div className="max-w-xl space-y-6">
            <h2 className="text-xl font-semibold mb-4">서버 설정</h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-dark-300 mb-1.5">서버 포트</label>
                <input
                  type="number"
                  value={serverConfig.port}
                  onChange={(e) => setServerConfig({ ...serverConfig, port: Number(e.target.value) })}
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-600 rounded-lg"
                />
                <p className="text-xs text-dark-400 mt-1">PAL 서버가 실행될 포트 번호</p>
              </div>

              <div>
                <label className="block text-sm text-dark-300 mb-1.5">프로젝트 경로</label>
                <div className="flex gap-2">
                  <input
                    type="text"
                    value={serverConfig.projectRoot}
                    onChange={(e) => setServerConfig({ ...serverConfig, projectRoot: e.target.value })}
                    className="flex-1 px-3 py-2 bg-dark-800 border border-dark-600 rounded-lg"
                    placeholder="/path/to/project"
                  />
                  <button className="px-3 py-2 bg-dark-700 rounded-lg hover:bg-dark-600">
                    <FolderOpen size={18} />
                  </button>
                </div>
              </div>

              <div>
                <label className="block text-sm text-dark-300 mb-1.5">데이터베이스 경로</label>
                <input
                  type="text"
                  value={serverConfig.dbPath}
                  onChange={(e) => setServerConfig({ ...serverConfig, dbPath: e.target.value })}
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-600 rounded-lg"
                />
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm">자동 시작</div>
                  <div className="text-xs text-dark-400">앱 실행 시 서버 자동 시작</div>
                </div>
                <button
                  onClick={() => setServerConfig({ ...serverConfig, autoStart: !serverConfig.autoStart })}
                  className={clsx(
                    'w-12 h-6 rounded-full transition-colors relative',
                    serverConfig.autoStart ? 'bg-primary-600' : 'bg-dark-600'
                  )}
                >
                  <div className={clsx(
                    'w-5 h-5 rounded-full bg-white absolute top-0.5 transition-all',
                    serverConfig.autoStart ? 'left-6' : 'left-0.5'
                  )} />
                </button>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'ui' && (
          <div className="max-w-xl space-y-6">
            <h2 className="text-xl font-semibold mb-4">UI 설정</h2>
            
            <div className="space-y-4">
              <div>
                <label className="block text-sm text-dark-300 mb-1.5">테마</label>
                <select
                  value={uiConfig.theme}
                  onChange={(e) => setUIConfig({ ...uiConfig, theme: e.target.value as any })}
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-600 rounded-lg"
                >
                  <option value="dark">다크</option>
                  <option value="light">라이트</option>
                  <option value="system">시스템</option>
                </select>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm">컴팩트 모드</div>
                  <div className="text-xs text-dark-400">UI 요소 크기 축소</div>
                </div>
                <button
                  onClick={() => setUIConfig({ ...uiConfig, compactMode: !uiConfig.compactMode })}
                  className={clsx(
                    'w-12 h-6 rounded-full transition-colors relative',
                    uiConfig.compactMode ? 'bg-primary-600' : 'bg-dark-600'
                  )}
                >
                  <div className={clsx(
                    'w-5 h-5 rounded-full bg-white absolute top-0.5 transition-all',
                    uiConfig.compactMode ? 'left-6' : 'left-0.5'
                  )} />
                </button>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm">알림 표시</div>
                  <div className="text-xs text-dark-400">이벤트 알림 표시</div>
                </div>
                <button
                  onClick={() => setUIConfig({ ...uiConfig, showNotifications: !uiConfig.showNotifications })}
                  className={clsx(
                    'w-12 h-6 rounded-full transition-colors relative',
                    uiConfig.showNotifications ? 'bg-primary-600' : 'bg-dark-600'
                  )}
                >
                  <div className={clsx(
                    'w-5 h-5 rounded-full bg-white absolute top-0.5 transition-all',
                    uiConfig.showNotifications ? 'left-6' : 'left-0.5'
                  )} />
                </button>
              </div>

              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm">자동 새로고침</div>
                  <div className="text-xs text-dark-400">데이터 자동 갱신</div>
                </div>
                <button
                  onClick={() => setUIConfig({ ...uiConfig, autoRefresh: !uiConfig.autoRefresh })}
                  className={clsx(
                    'w-12 h-6 rounded-full transition-colors relative',
                    uiConfig.autoRefresh ? 'bg-primary-600' : 'bg-dark-600'
                  )}
                >
                  <div className={clsx(
                    'w-5 h-5 rounded-full bg-white absolute top-0.5 transition-all',
                    uiConfig.autoRefresh ? 'left-6' : 'left-0.5'
                  )} />
                </button>
              </div>

              {uiConfig.autoRefresh && (
                <div>
                  <label className="block text-sm text-dark-300 mb-1.5">새로고침 간격 (ms)</label>
                  <input
                    type="number"
                    value={uiConfig.refreshInterval}
                    onChange={(e) => setUIConfig({ ...uiConfig, refreshInterval: Number(e.target.value) })}
                    className="w-full px-3 py-2 bg-dark-800 border border-dark-600 rounded-lg"
                    min={1000}
                    step={1000}
                  />
                </div>
              )}
            </div>
          </div>
        )}

        {activeTab === 'mcp' && (
          <div className="max-w-xl space-y-6">
            <h2 className="text-xl font-semibold mb-4">MCP 연동 (Claude Desktop)</h2>
            
            <div className="bg-dark-800 rounded-lg p-4 border border-dark-600">
              <h3 className="font-medium mb-2 flex items-center gap-2">
                <Terminal size={16} className="text-primary-400" />
                MCP 서버란?
              </h3>
              <p className="text-sm text-dark-300">
                Model Context Protocol (MCP)은 Claude Desktop과 외부 도구를 연결하는 프로토콜입니다. 
                PAL-Kit을 MCP 서버로 실행하면 Claude가 직접 문서를 읽고, 세션을 관리하고, 
                오케스트레이션을 실행할 수 있습니다.
              </p>
            </div>

            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <div>
                  <div className="text-sm">MCP 서버 활성화</div>
                  <div className="text-xs text-dark-400">Claude Desktop 연동 활성화</div>
                </div>
                <button
                  onClick={() => setMCPConfig({ ...mcpConfig, enabled: !mcpConfig.enabled })}
                  className={clsx(
                    'w-12 h-6 rounded-full transition-colors relative',
                    mcpConfig.enabled ? 'bg-primary-600' : 'bg-dark-600'
                  )}
                >
                  <div className={clsx(
                    'w-5 h-5 rounded-full bg-white absolute top-0.5 transition-all',
                    mcpConfig.enabled ? 'left-6' : 'left-0.5'
                  )} />
                </button>
              </div>

              <div className="border-t border-dark-700 pt-4">
                <h3 className="font-medium mb-3">Claude Desktop 설정 방법</h3>
                <div className="bg-dark-900 rounded-lg p-3 font-mono text-sm">
                  <div className="text-dark-400 mb-2">// claude_desktop_config.json</div>
                  <pre className="text-green-400 whitespace-pre-wrap">{`{
  "mcpServers": {
    "pal-kit": {
      "command": "pal",
      "args": ["mcp", "--project", "/path/to/project"],
      "env": {}
    }
  }
}`}</pre>
                </div>
                <p className="text-xs text-dark-400 mt-2">
                  macOS: ~/Library/Application Support/Claude/claude_desktop_config.json
                </p>
              </div>

              <div className="border-t border-dark-700 pt-4">
                <h3 className="font-medium mb-3">제공되는 MCP 도구</h3>
                <div className="space-y-2">
                  {[
                    { name: 'read_document', desc: '문서 읽기' },
                    { name: 'list_documents', desc: '문서 목록 조회' },
                    { name: 'list_agents', desc: '에이전트 목록' },
                    { name: 'get_agent_spec', desc: '에이전트 명세 조회' },
                    { name: 'create_session', desc: '세션 생성' },
                    { name: 'list_sessions', desc: '세션 목록' },
                    { name: 'start_orchestration', desc: '오케스트레이션 시작' },
                  ].map(tool => (
                    <div key={tool.name} className="flex items-center justify-between text-sm">
                      <code className="text-primary-400">{tool.name}</code>
                      <span className="text-dark-400">{tool.desc}</span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          </div>
        )}

        {activeTab === 'about' && (
          <div className="max-w-xl space-y-6">
            <h2 className="text-xl font-semibold mb-4">정보</h2>
            
            <div className="bg-dark-800 rounded-lg p-6">
              <div className="flex items-center gap-4 mb-4">
                <div className="w-16 h-16 bg-primary-600 rounded-xl flex items-center justify-center">
                  <Cpu size={32} />
                </div>
                <div>
                  <h3 className="text-xl font-bold">PAL-Kit</h3>
                  <p className="text-dark-400">Claude Desktop Agent Framework</p>
                </div>
              </div>
              
              <div className="space-y-2 text-sm">
                <div className="flex justify-between">
                  <span className="text-dark-400">버전</span>
                  <span>0.1.0</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-dark-400">빌드</span>
                  <span>2024.01.24</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-dark-400">라이선스</span>
                  <span>MIT</span>
                </div>
              </div>
            </div>

            <div className="space-y-2">
              <a href="https://github.com/n0roo/pal-kit" target="_blank" className="block p-3 bg-dark-800 rounded-lg hover:bg-dark-700">
                <div className="flex items-center gap-2">
                  <Globe size={16} />
                  <span>GitHub 저장소</span>
                </div>
              </a>
              <a href="#" className="block p-3 bg-dark-800 rounded-lg hover:bg-dark-700">
                <div className="flex items-center gap-2">
                  <Shield size={16} />
                  <span>개인정보 처리방침</span>
                </div>
              </a>
            </div>
          </div>
        )}

        {/* Save button */}
        <div className="fixed bottom-6 right-6">
          <button
            onClick={handleSave}
            disabled={saving}
            className={clsx(
              'flex items-center gap-2 px-4 py-2 rounded-lg font-medium transition-all',
              saved
                ? 'bg-green-600 text-white'
                : 'bg-primary-600 hover:bg-primary-700 text-white'
            )}
          >
            {saving ? (
              <RefreshCw size={18} className="animate-spin" />
            ) : saved ? (
              <Check size={18} />
            ) : (
              <Save size={18} />
            )}
            {saved ? '저장됨' : '저장'}
          </button>
        </div>
      </div>
    </div>
  )
}
