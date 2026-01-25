import { useState, useEffect } from 'react'
import {
  BookOpen, RefreshCw, Database, Search, X,
  AlertCircle, CheckCircle, ChevronRight, FileText,
  FolderOpen, Plus, Trash2, Move, Copy
} from 'lucide-react'
import clsx from 'clsx'
import {
  useKBStatus,
  useKBToc,
  useKBDocuments,
  type KBTocEntry,
  type KBDocument,
  type KBDocumentDetail,
} from '../hooks/useKB'
import { MarkdownViewer } from '../components'

// KB Sections
const KB_SECTIONS = [
  { id: '00-System', label: 'System', icon: '⚙️' },
  { id: '10-Domains', label: 'Domains', icon: '🏗️' },
  { id: '20-Projects', label: 'Projects', icon: '📁' },
  { id: '30-References', label: 'References', icon: '📚' },
  { id: '40-Archive', label: 'Archive', icon: '🗄️' },
]

export default function KnowledgeBase() {
  // Hooks
  const { status, loading: statusLoading, fetchStatus, initialize, rebuildIndex } = useKBStatus()
  const { toc, loading: tocLoading, getSectionToc, generateToc, fetchToc } = useKBToc()
  const {
    documents, loading: docsLoading, total, search,
    getDocument, createDocument, updateDocument, deleteDocument, moveDocument,
  } = useKBDocuments()

  // State
  const [selectedSection, setSelectedSection] = useState<string>(KB_SECTIONS[0].id)
  const [sectionToc, setSectionToc] = useState<KBTocEntry[]>([])
  const [selectedDocument, setSelectedDocument] = useState<KBDocumentDetail | null>(null)
  const [searchQuery, setSearchQuery] = useState('')
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)
  const [isEditing, setIsEditing] = useState(false)
  const [editContent, setEditContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(new Set())

  // Load section TOC when section changes
  useEffect(() => {
    loadSectionToc(selectedSection)
    search({ section: selectedSection, limit: 100 })
  }, [selectedSection])

  // Search when query changes (debounced)
  useEffect(() => {
    const timer = setTimeout(() => {
      search({ section: selectedSection, query: searchQuery, limit: 100 })
    }, 300)
    return () => clearTimeout(timer)
  }, [searchQuery, selectedSection])

  const loadSectionToc = async (section: string) => {
    const entries = await getSectionToc(section)
    setSectionToc(entries)
  }

  const showMessage = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text })
    setTimeout(() => setMessage(null), 3000)
  }

  // Toggle folder expansion
  const toggleExpand = (path: string) => {
    const newExpanded = new Set(expandedPaths)
    if (newExpanded.has(path)) {
      newExpanded.delete(path)
    } else {
      newExpanded.add(path)
    }
    setExpandedPaths(newExpanded)
  }

  // Select document from TOC or search
  const handleSelectDocument = async (entry: KBTocEntry | KBDocument) => {
    const id = 'id' in entry ? entry.id : entry.path
    if (id) {
      const doc = await getDocument(id)
      setSelectedDocument(doc)
      setIsEditing(false)
      setEditContent('')
    }
  }

  // Handle save
  const handleSave = async () => {
    if (!selectedDocument) return

    setSaving(true)
    const updated = await updateDocument(selectedDocument.id, { content: editContent })
    setSaving(false)

    if (updated) {
      setSelectedDocument(updated)
      setIsEditing(false)
      showMessage('success', '저장되었습니다')
    } else {
      showMessage('error', '저장에 실패했습니다')
    }
  }

  // Handle delete
  const handleDelete = async () => {
    if (!selectedDocument) return
    if (!confirm('정말 삭제하시겠습니까?')) return

    const success = await deleteDocument(selectedDocument.id)
    if (success) {
      setSelectedDocument(null)
      showMessage('success', '삭제되었습니다')
      loadSectionToc(selectedSection)
    } else {
      showMessage('error', '삭제에 실패했습니다')
    }
  }

  // Handle initialize
  const handleInitialize = async () => {
    const success = await initialize(false)
    if (success) {
      showMessage('success', 'Knowledge Base가 초기화되었습니다')
      fetchToc()
    } else {
      showMessage('error', '초기화에 실패했습니다')
    }
  }

  // Handle rebuild index
  const handleRebuildIndex = async () => {
    const success = await rebuildIndex()
    if (success) {
      showMessage('success', '인덱스가 재구축되었습니다')
    } else {
      showMessage('error', '인덱스 재구축에 실패했습니다')
    }
  }

  // Render TOC tree recursively
  const renderTocTree = (entries: KBTocEntry[], depth = 0) => {
    return entries.map(entry => {
      const hasChildren = entry.children && entry.children.length > 0
      const isExpanded = expandedPaths.has(entry.path || '')
      const isSelected = selectedDocument?.path === entry.path

      return (
        <div key={entry.path || entry.title}>
          <div
            onClick={() => {
              if (hasChildren) {
                toggleExpand(entry.path || '')
              } else if (entry.path) {
                handleSelectDocument(entry)
              }
            }}
            className={clsx(
              'flex items-center gap-2 px-2 py-1.5 cursor-pointer rounded text-sm',
              'hover:bg-dark-700 transition-colors',
              isSelected && 'bg-primary-600/20 text-primary-400',
              depth > 0 && 'ml-4'
            )}
          >
            {hasChildren ? (
              <ChevronRight
                size={14}
                className={clsx('transition-transform', isExpanded && 'rotate-90')}
              />
            ) : (
              <FileText size={14} className="text-dark-400" />
            )}
            <span className="truncate flex-1">{entry.title}</span>
            {entry.summary && (
              <span className="text-xs text-dark-500 truncate max-w-[120px]">
                {entry.summary}
              </span>
            )}
          </div>

          {hasChildren && isExpanded && (
            <div>{renderTocTree(entry.children!, depth + 1)}</div>
          )}
        </div>
      )
    })
  }

  // Not initialized state
  if (!statusLoading && status && !status.initialized) {
    return (
      <div className="h-full flex items-center justify-center">
        <div className="text-center max-w-md">
          <BookOpen size={48} className="mx-auto mb-4 text-primary-400 opacity-50" />
          <h2 className="text-xl font-semibold mb-2">Knowledge Base 초기화 필요</h2>
          <p className="text-dark-400 mb-4">
            Knowledge Base를 사용하려면 먼저 초기화가 필요합니다.
          </p>
          <button
            onClick={handleInitialize}
            className="px-4 py-2 bg-primary-600 hover:bg-primary-700 rounded-lg"
          >
            Knowledge Base 초기화
          </button>
        </div>
      </div>
    )
  }

  return (
    <div className="h-full flex flex-col">
      {/* Header */}
      <div className="p-4 border-b border-dark-700">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold flex items-center gap-2">
              <BookOpen size={24} className="text-primary-400" />
              Knowledge Base
            </h1>
            {status && (
              <p className="text-sm text-dark-400 mt-1">
                {status.vault_path}
                <span className="ml-2">
                  | {Object.values(status.sections || {}).reduce((a: number, b: number) => a + b, 0)}개 문서
                </span>
              </p>
            )}
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={handleRebuildIndex}
              className="flex items-center gap-1.5 px-3 py-1.5 bg-dark-700 hover:bg-dark-600 rounded-lg text-sm"
              title="인덱스 재구축"
            >
              <Database size={14} />
              인덱스
            </button>
            <button
              onClick={() => {
                fetchStatus()
                fetchToc()
              }}
              disabled={statusLoading || tocLoading}
              className="p-2 hover:bg-dark-700 rounded"
              title="새로고침"
            >
              <RefreshCw
                size={18}
                className={statusLoading || tocLoading ? 'animate-spin' : ''}
              />
            </button>
          </div>
        </div>
      </div>

      {/* Message */}
      {message && (
        <div
          className={clsx(
            'mx-4 mt-2 px-4 py-2 rounded-lg text-sm flex items-center gap-2',
            message.type === 'success'
              ? 'bg-green-600/20 text-green-400'
              : 'bg-red-600/20 text-red-400'
          )}
        >
          {message.type === 'success' ? (
            <CheckCircle size={16} />
          ) : (
            <AlertCircle size={16} />
          )}
          {message.text}
        </div>
      )}

      {/* Main content - 2 columns */}
      <div className="flex-1 flex overflow-hidden">
        {/* Left column: Sections + TOC + Search */}
        <div className="w-80 border-r border-dark-700 flex flex-col">
          {/* Section tabs */}
          <div className="flex border-b border-dark-700 overflow-x-auto">
            {KB_SECTIONS.map(section => (
              <button
                key={section.id}
                onClick={() => setSelectedSection(section.id)}
                className={clsx(
                  'px-3 py-2 text-xs whitespace-nowrap border-b-2 transition-colors',
                  selectedSection === section.id
                    ? 'border-primary-500 text-primary-400 bg-dark-800'
                    : 'border-transparent text-dark-400 hover:text-dark-200'
                )}
              >
                <span className="mr-1">{section.icon}</span>
                {section.label}
              </button>
            ))}
          </div>

          {/* Search */}
          <div className="p-2 border-b border-dark-700">
            <div className="relative">
              <Search size={14} className="absolute left-2 top-1/2 -translate-y-1/2 text-dark-400" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="문서 검색..."
                className="w-full pl-8 pr-8 py-1.5 bg-dark-800 border border-dark-600 rounded text-sm focus:border-primary-500 focus:outline-none"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery('')}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-dark-400 hover:text-dark-200"
                >
                  <X size={14} />
                </button>
              )}
            </div>
          </div>

          {/* TOC or Search results */}
          <div className="flex-1 overflow-auto p-2">
            {tocLoading || docsLoading ? (
              <div className="flex items-center justify-center h-32">
                <RefreshCw size={20} className="animate-spin text-dark-400" />
              </div>
            ) : searchQuery ? (
              // Search results
              <div className="space-y-1">
                <div className="text-xs text-dark-400 mb-2">
                  {documents.length}개 검색결과
                </div>
                {documents.map(doc => (
                  <div
                    key={doc.id}
                    onClick={() => handleSelectDocument(doc)}
                    className={clsx(
                      'flex items-start gap-2 px-2 py-2 cursor-pointer rounded text-sm',
                      'hover:bg-dark-700 transition-colors',
                      selectedDocument?.id === doc.id && 'bg-primary-600/20 text-primary-400'
                    )}
                  >
                    <FileText size={14} className="text-dark-400 mt-0.5 flex-shrink-0" />
                    <div className="min-w-0">
                      <div className="truncate font-medium">{doc.title}</div>
                      <div className="text-xs text-dark-500 truncate">{doc.path}</div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              // TOC tree
              <div className="space-y-0.5">
                {renderTocTree(sectionToc)}
              </div>
            )}
          </div>
        </div>

        {/* Right column: Document viewer/editor */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {selectedDocument ? (
            <>
              {/* Document header */}
              <div className="p-4 border-b border-dark-700">
                <div className="flex items-start justify-between">
                  <div className="flex-1 min-w-0">
                    <h2 className="text-lg font-semibold truncate">{selectedDocument.title}</h2>
                    <div className="text-xs text-dark-400 mt-1 truncate">{selectedDocument.path}</div>
                    {selectedDocument.tags && selectedDocument.tags.length > 0 && (
                      <div className="flex flex-wrap gap-1 mt-2">
                        {selectedDocument.tags.map(tag => (
                          <span
                            key={tag}
                            className="px-1.5 py-0.5 bg-primary-600/20 text-primary-400 rounded text-xs"
                          >
                            #{tag}
                          </span>
                        ))}
                      </div>
                    )}
                  </div>

                  <div className="flex items-center gap-1 ml-4">
                    {isEditing ? (
                      <>
                        <button
                          onClick={() => setIsEditing(false)}
                          className="px-3 py-1.5 text-sm text-dark-400 hover:text-dark-200"
                        >
                          취소
                        </button>
                        <button
                          onClick={handleSave}
                          disabled={saving}
                          className="px-3 py-1.5 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded text-sm"
                        >
                          {saving ? '저장 중...' : '저장'}
                        </button>
                      </>
                    ) : (
                      <>
                        <button
                          onClick={() => {
                            setIsEditing(true)
                            setEditContent(selectedDocument.content || '')
                          }}
                          className="px-3 py-1.5 bg-dark-700 hover:bg-dark-600 rounded text-sm"
                        >
                          편집
                        </button>
                        <button
                          onClick={handleDelete}
                          className="p-1.5 text-dark-400 hover:text-red-400"
                          title="삭제"
                        >
                          <Trash2 size={16} />
                        </button>
                        <button
                          onClick={() => setSelectedDocument(null)}
                          className="p-1.5 text-dark-400 hover:text-dark-200"
                        >
                          <X size={16} />
                        </button>
                      </>
                    )}
                  </div>
                </div>
              </div>

              {/* Document content */}
              <div className="flex-1 overflow-auto p-4">
                {isEditing ? (
                  <textarea
                    value={editContent}
                    onChange={(e) => setEditContent(e.target.value)}
                    className="w-full h-full p-4 bg-dark-800 border border-dark-600 rounded-lg resize-none focus:border-primary-500 focus:outline-none font-mono text-sm"
                    placeholder="마크다운 내용을 입력하세요..."
                  />
                ) : (
                  <div className="bg-dark-800 rounded-lg p-4 h-full overflow-auto">
                    <MarkdownViewer content={selectedDocument.content || ''} />
                  </div>
                )}
              </div>
            </>
          ) : (
            <div className="flex-1 flex items-center justify-center text-dark-400">
              <div className="text-center">
                <BookOpen size={48} className="mx-auto mb-4 opacity-30" />
                <p>문서를 선택하세요</p>
                <p className="text-sm mt-1">왼쪽 목차에서 문서를 선택하거나 검색하세요</p>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
