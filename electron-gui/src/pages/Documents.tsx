import { useState, useEffect } from 'react'
import { 
  FileText, Search, RefreshCw, Database, FolderTree, Tag,
  ChevronRight, X, Eye, Hash, Clock, Filter, List
} from 'lucide-react'
import clsx from 'clsx'
import { formatDistanceToNow } from 'date-fns'
import { ko } from 'date-fns/locale'
import { useDocuments, type Document, type DocumentFilters } from '../hooks'
import { MarkdownViewer, TocViewer } from '../components'

type TypeFilter = '' | 'port' | 'convention' | 'agent' | 'docs' | 'session' | 'adr'
type StatusFilter = '' | 'active' | 'draft' | 'archived' | 'deprecated'
type ViewTab = 'preview' | 'toc'

export default function Documents() {
  const { documents, stats, loading, indexing, fetchDocuments, reindex, getContent } = useDocuments()
  
  const [searchQuery, setSearchQuery] = useState('')
  const [typeFilter, setTypeFilter] = useState<TypeFilter>('')
  const [statusFilter, setStatusFilter] = useState<StatusFilter>('')
  const [selectedDoc, setSelectedDoc] = useState<Document | null>(null)
  const [docContent, setDocContent] = useState<string | null>(null)
  const [loadingContent, setLoadingContent] = useState(false)
  const [viewTab, setViewTab] = useState<ViewTab>('preview')

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

  const handleSelectDoc = async (doc: Document) => {
    setSelectedDoc(doc)
    setLoadingContent(true)
    const content = await getContent(doc.id)
    setDocContent(content)
    setLoadingContent(false)
  }

  const handleReindex = async () => {
    const result = await reindex()
    if (result) {
      console.log('Reindex result:', result)
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
    <div className="h-full flex">
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
                onClick={handleReindex}
                disabled={indexing}
                className="flex items-center gap-2 px-3 py-1.5 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded-lg text-sm"
              >
                <Database size={16} className={indexing ? 'animate-pulse' : ''} />
                {indexing ? '인덱싱 중...' : '다시 인덱싱'}
              </button>
            </div>
          </div>

          {/* Stats */}
          {stats && (
            <div className="grid grid-cols-4 gap-3 mb-4">
              <div className="bg-dark-800 rounded-lg p-3">
                <div className="text-2xl font-bold">{stats.total_docs}</div>
                <div className="text-xs text-dark-400">총 문서</div>
              </div>
              <div className="bg-dark-800 rounded-lg p-3">
                <div className="text-2xl font-bold">{(stats.total_tokens / 1000).toFixed(1)}K</div>
                <div className="text-xs text-dark-400">총 토큰</div>
              </div>
              <div className="bg-dark-800 rounded-lg p-3">
                <div className="text-2xl font-bold">{Object.keys(stats.by_type || {}).length}</div>
                <div className="text-xs text-dark-400">문서 타입</div>
              </div>
              <div className="bg-dark-800 rounded-lg p-3">
                <div className="text-2xl font-bold">{Object.keys(stats.by_domain || {}).length}</div>
                <div className="text-xs text-dark-400">도메인</div>
              </div>
            </div>
          )}

          {/* Search */}
          <div className="relative mb-4">
            <Search size={18} className="absolute left-3 top-1/2 -translate-y-1/2 text-dark-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              placeholder="문서 검색 (경로, ID, 키워드...)"
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
            {/* Type filter */}
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

            {/* Status filter */}
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
                        <FolderTree size={12} />
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
      {selectedDoc && (
        <div className="w-[420px] border-l border-dark-700 flex flex-col overflow-hidden">
          <div className="p-4 border-b border-dark-700">
            <div className="flex items-center justify-between mb-2">
              <h2 className="font-semibold truncate">{selectedDoc.id}</h2>
              <button
                onClick={() => {
                  setSelectedDoc(null)
                  setDocContent(null)
                }}
                className="text-dark-400 hover:text-dark-200"
              >
                <X size={18} />
              </button>
            </div>
            
            <div className="flex items-center gap-2 text-sm text-dark-400 mb-3">
              <FolderTree size={14} />
              <span className="truncate">{selectedDoc.path}</span>
            </div>

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

          {/* View tabs */}
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

          {/* Content */}
          <div className="flex-1 overflow-auto p-4">
            {loadingContent ? (
              <div className="flex items-center justify-center h-32">
                <RefreshCw size={24} className="animate-spin text-dark-400" />
              </div>
            ) : docContent ? (
              viewTab === 'preview' ? (
                <div className="bg-dark-800 rounded-lg p-4 overflow-auto max-h-[calc(100vh-450px)]">
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
  )
}
