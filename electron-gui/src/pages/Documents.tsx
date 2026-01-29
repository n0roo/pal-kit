import { useState, useEffect } from 'react'
import {
  FileText, Search, RefreshCw, Database, Tag,
  ChevronRight, X, Eye, Hash, Clock, List, PanelLeftClose, PanelLeft,
  Plus, Pencil, Trash2, Save
} from 'lucide-react'
import clsx from 'clsx'
import { formatDistanceToNow } from 'date-fns'
import { ko } from 'date-fns/locale'
import { useDocuments, useDocumentTree, type Document, type DocumentFilters, type DocumentTreeNode } from '../hooks'
import { MarkdownViewer, TocViewer, DocumentTree } from '../components'

type TypeFilter = '' | 'port' | 'convention' | 'agent' | 'docs' | 'session' | 'adr'
type StatusFilter = '' | 'active' | 'draft' | 'archived' | 'deprecated'
type ViewTab = 'preview' | 'toc' | 'edit'

const DOC_TEMPLATES: Record<string, { label: string; dir: string; ext: string; template: string }> = {
  port: {
    label: 'Port',
    dir: 'ports/',
    ext: '.md',
    template: `---
type: port
status: draft
priority: medium
domain: ""
tags: []
---

# Port: {name}

## 배경

## 목표

## 작업 항목

## 완료 기준
`,
  },
  convention: {
    label: 'Convention',
    dir: 'conventions/',
    ext: '.md',
    template: `---
type: convention
status: active
tags: []
---

# Convention: {name}

## 규칙

## 예시
`,
  },
  agent: {
    label: 'Agent',
    dir: 'agents/',
    ext: '.yaml',
    template: `name: "{name}"
type: worker
description: ""
capabilities: []
`,
  },
  docs: {
    label: 'Docs',
    dir: 'docs/',
    ext: '.md',
    template: `---
type: docs
status: draft
tags: []
---

# {name}
`,
  },
  adr: {
    label: 'ADR',
    dir: '.pal/decisions/',
    ext: '.md',
    template: `---
type: adr
status: proposed
decision_date: "{date}"
tags: []
---

# ADR: {name}

## 상태
Proposed

## 컨텍스트

## 결정

## 결과
`,
  },
}

export default function Documents() {
  const {
    documents, stats, loading, indexing,
    fetchDocuments, reindex, getContent,
    createDocument, updateDocument, deleteDocument,
  } = useDocuments()

  const [searchQuery, setSearchQuery] = useState('')
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('')
  const [selectedDoc, setSelectedDoc] = useState<Document | null>(null)
  const [docContent, setDocContent] = useState<string | null>(null)
  const [loadingContent, setLoadingContent] = useState(false)
  const [viewTab, setViewTab] = useState<ViewTab>('preview')
  const [editContent, setEditContent] = useState('')
  const [saving, setSaving] = useState(false)
  const [message, setMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null)

  // Tree view state
  const [showTreePanel, setShowTreePanel] = useState(true)
  const [selectedTreeNode, setSelectedTreeNode] = useState<DocumentTreeNode | null>(null)
  const { tree, loading: treeLoading, fetchTree } = useDocumentTree(4)

  // Create dialog state
  const [showCreateDialog, setShowCreateDialog] = useState(false)
  const [createType, setCreateType] = useState<string>('port')
  const [createName, setCreateName] = useState('')
  const [creating, setCreating] = useState(false)

  // Debounced search
  useEffect(() => {
    const timer = setTimeout(() => {
      const filters: DocumentFilters = {}
      if (searchQuery) filters.q = searchQuery
      if (typeFilter) filters.type = typeFilter
      if (statusFilter) filters.status = statusFilter
      fetchDocuments(filters)
    }, 300)
    return () => clearTimeout(timer)
  }, [searchQuery, typeFilter, statusFilter, fetchDocuments])

  const showMsg = (type: 'success' | 'error', text: string) => {
    setMessage({ type, text })
    setTimeout(() => setMessage(null), 3000)
  }

  const handleSelectDoc = async (doc: Document) => {
    setSelectedDoc(doc)
    setViewTab('preview')
    setLoadingContent(true)
    const content = await getContent(doc.id)
    setDocContent(content)
    setLoadingContent(false)
  }

  const handleReindex = async () => {
    const result = await reindex()
    if (result) {
      fetchTree()
    }
  }

  const handleTreeNodeSelect = (node: DocumentTreeNode) => {
    setSelectedTreeNode(node)
    if (node.type === 'file') {
      const matchingDoc = documents.find(d => d.path.endsWith(node.path) || d.path === node.path)
      if (matchingDoc) {
        handleSelectDoc(matchingDoc)
      }
    }
  }

  // Edit
  const handleStartEdit = () => {
    setEditContent(docContent || '')
    setViewTab('edit')
  }

  const handleSave = async () => {
    if (!selectedDoc) return
    setSaving(true)
    const success = await updateDocument(selectedDoc.id, editContent)
    setSaving(false)
    if (success) {
      setDocContent(editContent)
      setViewTab('preview')
      showMsg('success', '저장되었습니다')
      fetchTree()
    } else {
      showMsg('error', '저장에 실패했습니다')
    }
  }

  // Delete
  const handleDelete = async () => {
    if (!selectedDoc) return
    if (!confirm(`"${selectedDoc.id}" 문서를 삭제하시겠습니까?`)) return

    const success = await deleteDocument(selectedDoc.id)
    if (success) {
      setSelectedDoc(null)
      setDocContent(null)
      showMsg('success', '삭제되었습니다')
      fetchTree()
    } else {
      showMsg('error', '삭제에 실패했습니다')
    }
  }

  // Create
  const handleCreate = async () => {
    if (!createName.trim()) return
    const tmpl = DOC_TEMPLATES[createType]
    if (!tmpl) return

    setCreating(true)
    const fileName = createName.trim().replace(/\s+/g, '-').toLowerCase()
    const path = `${tmpl.dir}${fileName}${tmpl.ext}`
    const content = tmpl.template
      .replace(/{name}/g, createName.trim())
      .replace(/{date}/g, new Date().toISOString().slice(0, 10))

    const success = await createDocument(path, content)
    setCreating(false)
    if (success) {
      setShowCreateDialog(false)
      setCreateName('')
      showMsg('success', `문서가 생성되었습니다: ${path}`)
      fetchTree()
    } else {
      showMsg('error', '문서 생성에 실패했습니다')
    }
  }

  const scrollToHeading = (id: string) => {
    const element = document.getElementById(id)
    if (element) {
      element.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
    setViewTab('preview')
  }

  const typeFilters: { label: string; value: TypeFilter; color: string }[] = [
    { label: '전체', value: '', color: 'bg-dark-600' },
    { label: 'Port', value: 'port', color: 'bg-blue-600' },
    { label: 'Convention', value: 'convention', color: 'bg-purple-600' },
    { label: 'Agent', value: 'agent', color: 'bg-green-600' },
    { label: 'Docs', value: 'docs', color: 'bg-yellow-600' },
    { label: 'Session', value: 'session', color: 'bg-orange-600' },
    { label: 'ADR', value: 'adr', color: 'bg-pink-600' },
  ]

  const statusFilters: { label: string; value: StatusFilter }[] = [
    { label: '전체 상태', value: '' },
    { label: 'Active', value: 'active' },
    { label: 'Draft', value: 'draft' },
    { label: 'Archived', value: 'archived' },
    { label: 'Deprecated', value: 'deprecated' },
  ]

  const getTypeColor = (type: string) => {
    const colors: Record<string, string> = {
      port: 'bg-blue-500/20 text-blue-400 border-blue-500/30',
      convention: 'bg-purple-500/20 text-purple-400 border-purple-500/30',
      agent: 'bg-green-500/20 text-green-400 border-green-500/30',
      docs: 'bg-yellow-500/20 text-yellow-400 border-yellow-500/30',
      session: 'bg-orange-500/20 text-orange-400 border-orange-500/30',
      adr: 'bg-pink-500/20 text-pink-400 border-pink-500/30',
    }
    return colors[type] || 'bg-dark-600 text-dark-300 border-dark-500'
  }

  const getStatusColor = (status: string) => {
    const colors: Record<string, string> = {
      active: 'text-green-400',
      draft: 'text-yellow-400',
      archived: 'text-dark-400',
      deprecated: 'text-red-400',
    }
    return colors[status] || 'text-dark-400'
  }

  return (
    <div className="h-full flex flex-col">
      {/* Message toast */}
      {message && (
        <div
          className={clsx(
            'absolute top-4 right-4 z-50 px-4 py-2 rounded-lg text-sm shadow-lg',
            message.type === 'success' ? 'bg-green-600/90 text-white' : 'bg-red-600/90 text-white'
          )}
        >
          {message.text}
        </div>
      )}

      {/* Create Dialog */}
      {showCreateDialog && (
        <div className="absolute inset-0 z-40 bg-black/50 flex items-center justify-center">
          <div className="bg-dark-800 border border-dark-600 rounded-xl p-6 w-[420px] shadow-2xl">
            <h3 className="text-lg font-semibold mb-4">새 문서 생성</h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm text-dark-400 mb-1">타입</label>
                <div className="flex flex-wrap gap-2">
                  {Object.entries(DOC_TEMPLATES).map(([key, tmpl]) => (
                    <button
                      key={key}
                      onClick={() => setCreateType(key)}
                      className={clsx(
                        'px-3 py-1.5 rounded text-sm border transition-colors',
                        createType === key
                          ? 'bg-primary-600 border-primary-500 text-white'
                          : 'bg-dark-700 border-dark-600 text-dark-300 hover:bg-dark-600'
                      )}
                    >
                      {tmpl.label}
                    </button>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm text-dark-400 mb-1">이름</label>
                <input
                  type="text"
                  value={createName}
                  onChange={(e) => setCreateName(e.target.value)}
                  placeholder="문서 이름"
                  className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm focus:border-primary-500 focus:outline-none"
                  autoFocus
                  onKeyDown={(e) => e.key === 'Enter' && handleCreate()}
                />
                {createName && (
                  <div className="text-xs text-dark-500 mt-1">
                    경로: {DOC_TEMPLATES[createType]?.dir}{createName.trim().replace(/\s+/g, '-').toLowerCase()}{DOC_TEMPLATES[createType]?.ext}
                  </div>
                )}
              </div>
            </div>

            <div className="flex justify-end gap-2 mt-6">
              <button
                onClick={() => { setShowCreateDialog(false); setCreateName('') }}
                className="px-4 py-2 text-sm text-dark-400 hover:text-dark-200"
              >
                취소
              </button>
              <button
                onClick={handleCreate}
                disabled={!createName.trim() || creating}
                className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded-lg text-sm"
              >
                {creating ? '생성 중...' : '생성'}
              </button>
            </div>
          </div>
        </div>
      )}

      {/* Top bar */}
      <div className="px-4 py-2 border-b border-dark-700 bg-dark-800/50 flex items-center gap-3">
        <button
          onClick={() => setShowTreePanel(!showTreePanel)}
          className="p-1.5 hover:bg-dark-700 rounded text-dark-400 hover:text-dark-200"
          title={showTreePanel ? '트리 숨기기' : '트리 보기'}
        >
          {showTreePanel ? <PanelLeftClose size={16} /> : <PanelLeft size={16} />}
        </button>
        <span className="text-sm text-dark-400">관리 문서</span>
        <span className="text-xs text-dark-500">(ports, conventions, agents, docs, sessions, decisions)</span>
      </div>

      <div className="flex-1 flex overflow-hidden">
        {/* Tree panel */}
        {showTreePanel && (
          <div className="w-64 border-r border-dark-700 flex flex-col overflow-hidden bg-dark-800/30">
            <div className="p-2 border-b border-dark-700 flex items-center justify-between">
              <h3 className="text-xs font-medium text-dark-400 uppercase tracking-wide">파일 탐색기</h3>
              <button
                onClick={() => fetchTree()}
                className="p-1 hover:bg-dark-700 rounded text-dark-400 hover:text-dark-200"
                title="새로고침"
              >
                <RefreshCw size={12} />
              </button>
            </div>
            <div className="flex-1 overflow-auto">
              <DocumentTree
                tree={tree}
                selectedPath={selectedTreeNode?.path || null}
                onSelectNode={handleTreeNodeSelect}
                loading={treeLoading}
              />
            </div>
          </div>
        )}

        {/* Main content */}
        <div className="flex-1 flex flex-col overflow-hidden">
          {/* Header */}
          <div className="p-4 border-b border-dark-700">
            <div className="flex items-center justify-between mb-4">
              <h1 className="text-xl font-semibold flex items-center gap-2">
                <FileText size={24} className="text-blue-400" />
                문서 관리
              </h1>
              <div className="flex items-center gap-2">
                <button
                  onClick={() => setShowCreateDialog(true)}
                  className="flex items-center gap-2 px-3 py-1.5 bg-green-600 hover:bg-green-700 rounded-lg text-sm"
                >
                  <Plus size={16} />
                  새 문서
                </button>
                <button
                  onClick={handleReindex}
                  disabled={indexing}
                  className="flex items-center gap-2 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded-lg text-sm"
                >
                  <Database size={16} className={indexing ? 'animate-pulse' : ''} />
                  {indexing ? '인덱싱 중...' : '인덱싱'}
                </button>
              </div>
            </div>

            {/* Stats */}
            {stats && (
              <div className="flex items-center gap-4 text-sm text-dark-400 mb-4">
                <span>{stats.total_docs} 문서</span>
                <span>{(stats.total_tokens / 1000).toFixed(1)}K 토큰</span>
                <span>{Object.keys(stats.by_type || {}).length} 타입</span>
              </div>
            )}

            {/* Search */}
            <div className="relative mb-4">
              <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-dark-400" />
              <input
                type="text"
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                placeholder="문서 검색 (ID, 경로, 키워드, 태그...)"
                className="w-full pl-10 pr-4 py-2 bg-dark-800 border border-dark-600 rounded-lg focus:border-primary-500 focus:outline-none"
              />
              {searchQuery && (
                <button
                  onClick={() => setSearchQuery('')}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-dark-400 hover:text-dark-200"
                >
                  <X size={16} />
                </button>
              )}
            </div>

            {/* Filters */}
            <div className="flex flex-wrap gap-2">
              <div className="flex gap-1">
                {typeFilters.map(filter => (
                  <button
                    key={filter.value}
                    onClick={() => setTypeFilter(filter.value)}
                    className={clsx(
                      'px-2.5 py-1 rounded text-xs transition-colors',
                      typeFilter === filter.value
                        ? 'bg-primary-600 text-white'
                        : 'bg-dark-700 text-dark-300 hover:bg-dark-600'
                    )}
                  >
                    {filter.label}
                  </button>
                ))}
              </div>

              <div className="w-px h-6 bg-dark-600 self-center" />

              <select
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value as StatusFilter)}
                className="px-2.5 py-1 bg-dark-700 border border-dark-600 rounded text-xs"
              >
                {statusFilters.map(filter => (
                  <option key={filter.value} value={filter.value}>{filter.label}</option>
                ))}
              </select>
            </div>
          </div>

          {/* Document list */}
          <div className="flex-1 overflow-auto">
            {loading ? (
              <div className="flex items-center justify-center h-64">
                <RefreshCw size={32} className="animate-spin text-dark-400" />
              </div>
            ) : documents.length === 0 ? (
              <div className="flex flex-col items-center justify-center h-64 text-dark-400">
                <FileText size={48} className="mb-4 opacity-50" />
                <p>문서가 없습니다</p>
                <p className="text-sm mt-1">인덱싱을 실행하여 문서를 스캔하세요</p>
              </div>
            ) : (
              <div className="divide-y divide-dark-700">
                {documents.map(doc => (
                  <div
                    key={doc.id}
                    onClick={() => handleSelectDoc(doc)}
                    className={clsx(
                      'p-4 cursor-pointer hover:bg-dark-800/50 transition-colors',
                      selectedDoc?.id === doc.id && 'bg-dark-800'
                    )}
                  >
                    <div className="flex items-start gap-3">
                      <div className={clsx('p-2 rounded-lg border', getTypeColor(doc.type))}>
                        <FileText size={16} />
                      </div>

                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-2 mb-1">
                          <span className="font-medium truncate">{doc.id}</span>
                          <span className={clsx('text-xs', getStatusColor(doc.status))}>
                            {doc.status}
                          </span>
                        </div>

                        <div className="flex items-center gap-2 text-xs text-dark-400 mb-2">
                          <span className="truncate">{doc.path}</span>
                        </div>

                        <div className="flex items-center gap-3 text-xs">
                          <span className="flex items-center gap-1 text-dark-400">
                            <Hash size={12} />
                            {doc.tokens.toLocaleString()} tokens
                          </span>

                          {doc.domain && (
                            <span className="px-1.5 py-0.5 bg-dark-700 rounded">
                              {doc.domain}
                            </span>
                          )}

                          {doc.tags && doc.tags.length > 0 && (
                            <div className="flex items-center gap-1">
                              <Tag size={12} className="text-dark-400" />
                              {doc.tags.slice(0, 3).map(tag => (
                                <span key={tag} className="px-1.5 py-0.5 bg-primary-600/20 text-primary-400 rounded text-xs">
                                  {tag}
                                </span>
                              ))}
                              {doc.tags.length > 3 && (
                                <span className="text-dark-500">+{doc.tags.length - 3}</span>
                              )}
                            </div>
                          )}
                        </div>
                      </div>

                      <div className="flex flex-col items-end gap-1">
                        <span className="text-xs text-dark-500">
                          {formatDistanceToNow(new Date(doc.updated_at), { addSuffix: true, locale: ko })}
                        </span>
                        <ChevronRight size={16} className="text-dark-500" />
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>

        {/* Detail sidebar */}
        {selectedDoc && docContent !== null && (
          <div className="w-[480px] border-l border-dark-700 flex flex-col overflow-hidden">
            <div className="p-4 border-b border-dark-700">
              <div className="flex items-center justify-between mb-2">
                <h2 className="font-semibold truncate">{selectedDoc.id}</h2>
                <div className="flex items-center gap-1">
                  {viewTab === 'edit' ? (
                    <>
                      <button
                        onClick={() => setViewTab('preview')}
                        className="px-2 py-1 text-xs text-dark-400 hover:text-dark-200"
                      >
                        취소
                      </button>
                      <button
                        onClick={handleSave}
                        disabled={saving}
                        className="flex items-center gap-1 px-2 py-1 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded text-xs"
                      >
                        <Save size={12} />
                        {saving ? '저장 중...' : '저장'}
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        onClick={handleStartEdit}
                        className="p-1.5 text-dark-400 hover:text-blue-400 hover:bg-dark-700 rounded"
                        title="편집"
                      >
                        <Pencil size={14} />
                      </button>
                      <button
                        onClick={handleDelete}
                        className="p-1.5 text-dark-400 hover:text-red-400 hover:bg-dark-700 rounded"
                        title="삭제"
                      >
                        <Trash2 size={14} />
                      </button>
                      <button
                        onClick={() => {
                          setSelectedDoc(null)
                          setDocContent(null)
                        }}
                        className="p-1.5 text-dark-400 hover:text-dark-200 hover:bg-dark-700 rounded"
                      >
                        <X size={14} />
                      </button>
                    </>
                  )}
                </div>
              </div>

              <div className="text-xs text-dark-400 mb-3 truncate">{selectedDoc.path}</div>

              {/* Metadata */}
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div className="bg-dark-800 rounded p-2">
                  <div className="text-xs text-dark-400 mb-0.5">타입</div>
                  <span className={clsx('px-2 py-0.5 rounded text-xs border', getTypeColor(selectedDoc.type))}>
                    {selectedDoc.type}
                  </span>
                </div>
                <div className="bg-dark-800 rounded p-2">
                  <div className="text-xs text-dark-400 mb-0.5">상태</div>
                  <span className={getStatusColor(selectedDoc.status)}>{selectedDoc.status}</span>
                </div>
                <div className="bg-dark-800 rounded p-2">
                  <div className="text-xs text-dark-400 mb-0.5">토큰</div>
                  <span>{selectedDoc.tokens.toLocaleString()}</span>
                </div>
                <div className="bg-dark-800 rounded p-2">
                  <div className="text-xs text-dark-400 mb-0.5">도메인</div>
                  <span>{selectedDoc.domain || '-'}</span>
                </div>
              </div>

              {/* Tags */}
              {selectedDoc.tags && selectedDoc.tags.length > 0 && (
                <div className="mt-3">
                  <div className="text-xs text-dark-400 mb-1.5 flex items-center gap-1">
                    <Tag size={12} /> 태그
                  </div>
                  <div className="flex flex-wrap gap-1">
                    {selectedDoc.tags.map(tag => (
                      <span
                        key={tag}
                        onClick={() => setSearchQuery(tag)}
                        className="px-2 py-0.5 bg-primary-600/20 text-primary-400 rounded text-xs cursor-pointer hover:bg-primary-600/30"
                      >
                        #{tag}
                      </span>
                    ))}
                  </div>
                </div>
              )}

              {/* Timestamps */}
              <div className="mt-3 flex items-center gap-4 text-xs text-dark-400">
                <span className="flex items-center gap-1">
                  <Clock size={12} />
                  수정: {formatDistanceToNow(new Date(selectedDoc.updated_at), { addSuffix: true, locale: ko })}
                </span>
              </div>
            </div>

            {/* View tabs (only shown when not editing) */}
            {viewTab !== 'edit' && (
              <div className="flex border-b border-dark-700">
                <button
                  onClick={() => setViewTab('preview')}
                  className={clsx(
                    'flex-1 py-2 text-sm flex items-center justify-center gap-1.5',
                    viewTab === 'preview' ? 'text-primary-400 border-b-2 border-primary-400' : 'text-dark-400'
                  )}
                >
                  <Eye size={14} />
                  미리보기
                </button>
                <button
                  onClick={() => setViewTab('toc')}
                  className={clsx(
                    'flex-1 py-2 text-sm flex items-center justify-center gap-1.5',
                    viewTab === 'toc' ? 'text-primary-400 border-b-2 border-primary-400' : 'text-dark-400'
                  )}
                >
                  <List size={14} />
                  목차
                </button>
              </div>
            )}

            {/* Content */}
            <div className="flex-1 overflow-auto p-4">
              {loadingContent ? (
                <div className="flex items-center justify-center h-32">
                  <RefreshCw size={24} className="animate-spin text-dark-400" />
                </div>
              ) : viewTab === 'edit' ? (
                <textarea
                  value={editContent}
                  onChange={(e) => setEditContent(e.target.value)}
                  className="w-full h-full p-4 bg-dark-800 border border-dark-600 rounded-lg resize-none focus:border-primary-500 focus:outline-none font-mono text-sm"
                  placeholder="마크다운 내용을 입력하세요..."
                />
              ) : docContent ? (
                viewTab === 'preview' ? (
                  <div className="bg-dark-800 rounded-lg p-4 overflow-auto h-full">
                    <MarkdownViewer content={docContent} />
                  </div>
                ) : (
                  <div className="bg-dark-800 rounded-lg p-4">
                    <h3 className="text-sm font-medium text-dark-300 mb-3 flex items-center gap-2">
                      <List size={14} />
                      문서 목차
                    </h3>
                    <TocViewer
                      content={docContent}
                      onSelect={scrollToHeading}
                    />
                  </div>
                )
              ) : (
                <div className="text-center text-dark-400 py-8">
                  내용을 불러올 수 없습니다
                </div>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  )
}
