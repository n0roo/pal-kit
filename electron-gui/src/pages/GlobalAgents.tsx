import { useState, useEffect } from 'react'
import {
  Globe, RefreshCw, Save, RotateCcw, Upload, FolderSync,
  FileText, Code, Folder, ChevronRight, X, Clock, HardDrive,
  BookOpen
} from 'lucide-react'
import { useGlobalAgents } from '../hooks'
import { MarkdownViewer } from '../components'
import clsx from 'clsx'
import type { GlobalAgentInfo } from '../hooks/useApi'

type CategoryTab = 'agents' | 'skills' | 'conventions'
type ContentTab = 'spec' | 'rules'

export default function GlobalAgents() {
  const {
    agents, skills, conventions, manifest, globalPath,
    loading, syncing, fetchAgents, fetchManifest,
    getContent, updateContent, initialize, syncToProject, reset
  } = useGlobalAgents()

  // Category tab (left sidebar)
  const [activeCategory, setActiveCategory] = useState<CategoryTab>('agents')
  const [selectedItem, setSelectedItem] = useState<GlobalAgentInfo | null>(null)

  // Content editing
  const [specContent, setSpecContent] = useState<string>('')
  const [rulesContent, setRulesContent] = useState<string>('')
  const [originalSpec, setOriginalSpec] = useState<string>('')
  const [originalRules, setOriginalRules] = useState<string>('')
  const [activeContent, setActiveContent] = useState<ContentTab>('spec')
  const [loadingContent, setLoadingContent] = useState(false)
  const [saving, setSaving] = useState(false)
  const [editMode, setEditMode] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  const currentList = activeCategory === 'agents' ? agents : activeCategory === 'skills' ? skills : conventions

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text })
    setTimeout(() => setMessage(null), 3000)
  }

  const handleSelectItem = async (item: GlobalAgentInfo) => {
    setSelectedItem(item)
    setEditMode(false)
    setLoadingContent(true)
    setActiveContent('spec')

    // Load spec content
    const specData = await getContent(item.path)
    setSpecContent(specData || '')
    setOriginalSpec(specData || '')

    // Load rules content if exists
    if (item.has_rules) {
      const rulesPath = item.path.replace(/\.yaml$/, '.rules.md')
      const rulesData = await getContent(rulesPath)
      setRulesContent(rulesData || '')
      setOriginalRules(rulesData || '')
    } else {
      setRulesContent('')
      setOriginalRules('')
    }

    setLoadingContent(false)
  }

  // Track changes
  const specHasChanges = specContent !== originalSpec
  const rulesHasChanges = rulesContent !== originalRules
  const hasChanges = specHasChanges || rulesHasChanges

  const handleSave = async () => {
    if (!selectedItem) return

    setSaving(true)
    let success = true

    // Save spec if changed
    if (specHasChanges) {
      const result = await updateContent(selectedItem.path, specContent)
      if (!result) success = false
    }

    // Save rules if changed
    if (rulesHasChanges && selectedItem.has_rules) {
      const rulesPath = selectedItem.path.replace(/\.yaml$/, '.rules.md')
      const result = await updateContent(rulesPath, rulesContent)
      if (!result) success = false
    }

    setSaving(false)

    if (success) {
      setOriginalSpec(specContent)
      setOriginalRules(rulesContent)
      setEditMode(false)
      showMessage('success', '저장되었습니다')
      fetchAgents()
    } else {
      showMessage('error', '저장에 실패했습니다')
    }
  }

  const handleRevert = () => {
    setSpecContent(originalSpec)
    setRulesContent(originalRules)
    setEditMode(false)
  }

  const handleInitialize = async () => {
    const success = await initialize(false)
    if (success) {
      showMessage('success', '초기화되었습니다')
    } else {
      showMessage('error', '초기화에 실패했습니다')
    }
  }

  const handleSync = async () => {
    const result = await syncToProject()
    if (result) {
      showMessage('success', `${result.count}개 파일이 동기화되었습니다`)
    } else {
      showMessage('error', '동기화에 실패했습니다')
    }
  }

  const handleReset = async () => {
    if (!confirm('정말 초기화하시겠습니까? 수정된 내용이 모두 삭제됩니다.')) return

    const success = await reset()
    if (success) {
      setSelectedItem(null)
      setSpecContent('')
      setRulesContent('')
      showMessage('success', '초기 상태로 리셋되었습니다')
    } else {
      showMessage('error', '리셋에 실패했습니다')
    }
  }

  const categoryTabs: { label: string; value: CategoryTab; icon: any; count: number }[] = [
    { label: '에이전트', value: 'agents', icon: Code, count: agents.length },
    { label: '스킬', value: 'skills', icon: FileText, count: skills.length },
    { label: '컨벤션', value: 'conventions', icon: Folder, count: conventions.length },
  ]

  // Current content for the active tab
  const currentContent = activeContent === 'spec' ? specContent : rulesContent
  const setCurrentContent = activeContent === 'spec' ? setSpecContent : setRulesContent

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-dark-700">
        <div className="flex items-center justify-between mb-4">
          <div>
            <h1 className="text-xl font-semibold flex items-center gap-2">
              <Globe size={24} className="text-blue-400" />
              전역 에이전트 관리
            </h1>
            <p className="text-sm text-dark-400 mt-1">
              {globalPath || '~/.pal'} 에서 에이전트 템플릿을 관리합니다
            </p>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={handleInitialize}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-700 hover:bg-dark-600 rounded-lg text-sm"
              disabled={loading}
            >
              <Upload size={14} />
              초기화
            </button>
            <button
              onClick={handleSync}
              disabled={syncing}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 rounded-lg text-sm"
            >
              <FolderSync size={14} className={syncing ? 'animate-spin' : ''} />
              프로젝트 동기화
            </button>
            <button
              onClick={handleReset}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-red-600/20 text-red-400 hover:bg-red-600/30 rounded-lg text-sm"
            >
              <RotateCcw size={14} />
              리셋
            </button>
            <button
              onClick={() => fetchAgents()}
              disabled={loading}
              className="p-2 hover:bg-dark-700 rounded"
            >
              <RefreshCw size={18} className={loading ? 'animate-spin' : ''} />
            </button>
          </div>
        </div>

        {/* Manifest info */}
        {manifest && (
          <div className="flex items-center gap-4 text-xs text-dark-400 bg-dark-800 rounded-lg px-3 py-2">
            <span className="flex items-center gap-1">
              <HardDrive size={12} />
              버전 {manifest.version}
            </span>
            <span className="flex items-center gap-1">
              <Clock size={12} />
              마지막 수정: {new Date(manifest.last_updated).toLocaleString()}
            </span>
            {manifest.custom_agents && manifest.custom_agents.length > 0 && (
              <span className="text-yellow-400">
                커스텀 에이전트 {manifest.custom_agents.length}개
              </span>
            )}
          </div>
        )}
      </div>

      {/* Message */}
      {message && (
        <div className={clsx(
          'mx-4 mt-2 px-4 py-2 rounded-lg text-sm',
          message.type === 'success' ? 'bg-green-600/20 text-green-400' : 'bg-red-600/20 text-red-400'
        )}>
          {message.text}
        </div>
      )}

      {/* Main content */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left: File list */}
        <div className="w-80 border-r border-dark-700 flex flex-col">
          {/* Category tabs */}
          <div className="flex border-b border-dark-700">
            {categoryTabs.map(tab => (
              <button
                key={tab.value}
                onClick={() => {
                  setActiveCategory(tab.value)
                  setSelectedItem(null)
                }}
                className={clsx(
                  'flex-1 py-2 text-sm flex items-center justify-center gap-1.5',
                  activeCategory === tab.value
                    ? 'text-primary-400 border-b-2 border-primary-400'
                    : 'text-dark-400 hover:text-dark-200'
                )}
              >
                <tab.icon size={14} />
                {tab.label}
                <span className="text-xs text-dark-500">({tab.count})</span>
              </button>
            ))}
          </div>

          {/* List */}
          <div className="flex-1 overflow-auto p-2">
            {loading ? (
              <div className="flex items-center justify-center h-32">
                <RefreshCw size={24} className="animate-spin text-dark-400" />
              </div>
            ) : currentList.length === 0 ? (
              <div className="text-center text-dark-400 py-8">
                <Globe size={32} className="mx-auto mb-2 opacity-50" />
                <p className="text-sm">항목이 없습니다</p>
                <button
                  onClick={handleInitialize}
                  className="mt-2 text-xs text-primary-400 hover:underline"
                >
                  초기화하기
                </button>
              </div>
            ) : (
              <div className="space-y-1">
                {currentList.map(item => (
                  <button
                    key={item.path}
                    onClick={() => handleSelectItem(item)}
                    className={clsx(
                      'w-full text-left p-2 rounded-lg transition-colors',
                      selectedItem?.path === item.path
                        ? 'bg-primary-600/20 border border-primary-500/50'
                        : 'hover:bg-dark-700 border border-transparent'
                    )}
                  >
                    <div className="flex items-center gap-2">
                      <FileText size={14} className="text-dark-400 flex-shrink-0" />
                      <div className="flex-1 min-w-0">
                        <div className="font-medium text-sm truncate">{item.name}</div>
                        <div className="text-xs text-dark-500 truncate">{item.path}</div>
                      </div>
                      <ChevronRight size={14} className="text-dark-500 flex-shrink-0" />
                    </div>
                    {item.description && (
                      <p className="text-xs text-dark-400 mt-1 truncate pl-6">
                        {item.description}
                      </p>
                    )}
                    <div className="flex items-center gap-2 mt-1 pl-6">
                      <span className="text-xs px-1.5 py-0.5 bg-dark-700 rounded text-dark-400">
                        {item.type}
                      </span>
                      {item.category && (
                        <span className="text-xs text-dark-500">{item.category}</span>
                      )}
                      {item.has_rules && (
                        <span className="text-xs px-1.5 py-0.5 bg-yellow-500/20 text-yellow-400 rounded flex items-center gap-1">
                          <BookOpen size={10} />
                          rules
                        </span>
                      )}
                    </div>
                  </button>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Right: Content editor with tabs */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {selectedItem ? (
            <>
              {/* Editor header */}
              <div className="p-3 border-b border-dark-700 flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <span className="font-medium">{selectedItem.name}</span>
                  {hasChanges && (
                    <span className="text-xs px-1.5 py-0.5 bg-yellow-500/20 text-yellow-400 rounded">
                      수정됨
                    </span>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  {editMode ? (
                    <>
                      <button
                        onClick={handleRevert}
                        className="flex items-center gap-1 px-2 py-1 text-sm text-dark-400 hover:text-dark-200"
                      >
                        <X size={14} />
                        취소
                      </button>
                      <button
                        onClick={handleSave}
                        disabled={!hasChanges || saving}
                        className="flex items-center gap-1 px-3 py-1 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded text-sm"
                      >
                        <Save size={14} />
                        {saving ? '저장 중...' : '저장'}
                      </button>
                    </>
                  ) : (
                    <button
                      onClick={() => setEditMode(true)}
                      className="flex items-center gap-1 px-3 py-1 bg-dark-700 hover:bg-dark-600 rounded text-sm"
                    >
                      <Code size={14} />
                      편집
                    </button>
                  )}
                </div>
              </div>

              {/* Content tabs (Spec / Rules) */}
              <div className="flex border-b border-dark-700">
                <button
                  onClick={() => setActiveContent('spec')}
                  className={clsx(
                    'px-4 py-2 text-sm flex items-center gap-1.5 border-b-2 transition-colors',
                    activeContent === 'spec'
                      ? 'border-primary-500 text-primary-400'
                      : 'border-transparent text-dark-400 hover:text-dark-200'
                  )}
                >
                  <FileText size={14} />
                  Spec (YAML)
                  {specHasChanges && <span className="text-yellow-400">*</span>}
                </button>
                {selectedItem.has_rules && (
                  <button
                    onClick={() => setActiveContent('rules')}
                    className={clsx(
                      'px-4 py-2 text-sm flex items-center gap-1.5 border-b-2 transition-colors',
                      activeContent === 'rules'
                        ? 'border-primary-500 text-primary-400'
                        : 'border-transparent text-dark-400 hover:text-dark-200'
                    )}
                  >
                    <BookOpen size={14} />
                    Rules (Markdown)
                    {rulesHasChanges && <span className="text-yellow-400">*</span>}
                  </button>
                )}
              </div>

              {/* Content area */}
              <div className="flex-1 overflow-auto p-4">
                {loadingContent ? (
                  <div className="flex items-center justify-center h-32">
                    <RefreshCw size={24} className="animate-spin text-dark-400" />
                  </div>
                ) : editMode ? (
                  <textarea
                    value={currentContent}
                    onChange={(e) => setCurrentContent(e.target.value)}
                    className="w-full h-full bg-dark-800 border border-dark-600 rounded-lg p-4 font-mono text-sm resize-none focus:outline-none focus:border-primary-500"
                    spellCheck={false}
                    placeholder={activeContent === 'spec' ? 'YAML 스펙을 입력하세요...' : 'Markdown 규칙을 입력하세요...'}
                  />
                ) : (
                  <div className="bg-dark-800 rounded-lg p-4 overflow-auto h-full">
                    {activeContent === 'spec' ? (
                      <pre className="text-sm font-mono whitespace-pre-wrap">{specContent}</pre>
                    ) : (
                      <MarkdownViewer content={rulesContent} />
                    )}
                  </div>
                )}
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-dark-400">
              <div className="text-center">
                <Globe size={48} className="mx-auto mb-4 opacity-50" />
                <p>항목을 선택하세요</p>
                <p className="text-sm mt-2">
                  에이전트, 스킬, 컨벤션 파일을 편집할 수 있습니다
                </p>
                <p className="text-xs mt-1 text-dark-500">
                  YAML 스펙과 Rules 마크다운을 함께 관리합니다
                </p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
