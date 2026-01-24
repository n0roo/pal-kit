import { FileText, Tag, Clock, Search, Plus, Filter } from 'lucide-react'
import clsx from 'clsx'
import type { KBDocument } from '../../hooks/useKB'

interface KBDocumentListProps {
  documents: KBDocument[]
  selectedId: string | null
  onSelectDocument: (doc: KBDocument) => void
  onCreateDocument: () => void
  searchQuery: string
  onSearchChange: (query: string) => void
  loading?: boolean
  total?: number
}

const STATUS_COLORS: Record<string, string> = {
  draft: 'text-yellow-400 bg-yellow-500/20',
  active: 'text-green-400 bg-green-500/20',
  archived: 'text-gray-400 bg-gray-500/20',
  review: 'text-blue-400 bg-blue-500/20',
}

export default function KBDocumentList({
  documents,
  selectedId,
  onSelectDocument,
  onCreateDocument,
  searchQuery,
  onSearchChange,
  loading,
  total,
}: KBDocumentListProps) {
  return (
    <div className="flex flex-col h-full">
      {/* Header with search */}
      <div className="p-3 border-b border-dark-700">
        <div className="flex items-center gap-2 mb-3">
          <div className="relative flex-1">
            <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-dark-400" />
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              placeholder="문서 검색..."
              className="w-full pl-8 pr-3 py-1.5 bg-dark-800 border border-dark-600 rounded text-sm focus:outline-none focus:border-primary-500"
            />
          </div>
          <button
            onClick={onCreateDocument}
            className="p-1.5 bg-primary-600 hover:bg-primary-700 rounded"
            title="새 문서"
          >
            <Plus size={16} />
          </button>
        </div>

        {/* Stats */}
        <div className="flex items-center justify-between text-xs text-dark-400">
          <span>{total ?? documents.length}개 문서</span>
          {loading && <span className="text-primary-400">검색 중...</span>}
        </div>
      </div>

      {/* Document list */}
      <div className="flex-1 overflow-auto">
        {documents.length === 0 ? (
          <div className="flex items-center justify-center h-32 text-dark-400">
            <div className="text-center">
              <FileText size={24} className="mx-auto mb-2 opacity-50" />
              <p className="text-sm">문서가 없습니다</p>
            </div>
          </div>
        ) : (
          <div className="p-2 space-y-1">
            {documents.map((doc) => {
              const isSelected = selectedId === doc.id

              return (
                <button
                  key={doc.id}
                  onClick={() => onSelectDocument(doc)}
                  className={clsx(
                    'w-full text-left p-2.5 rounded-lg transition-colors',
                    isSelected
                      ? 'bg-primary-600/20 border border-primary-500/50'
                      : 'hover:bg-dark-700 border border-transparent'
                  )}
                >
                  {/* Title */}
                  <div className="flex items-start gap-2">
                    <FileText size={14} className="flex-shrink-0 mt-0.5 text-dark-400" />
                    <div className="flex-1 min-w-0">
                      <div className="font-medium text-sm truncate">{doc.title}</div>

                      {/* Summary */}
                      {doc.summary && (
                        <p className="text-xs text-dark-400 mt-0.5 line-clamp-2">
                          {doc.summary}
                        </p>
                      )}

                      {/* Meta */}
                      <div className="flex items-center gap-2 mt-1.5 flex-wrap">
                        {/* Type */}
                        <span className="text-[10px] px-1.5 py-0.5 bg-dark-700 rounded text-dark-300">
                          {doc.type}
                        </span>

                        {/* Status */}
                        {doc.status && (
                          <span className={clsx(
                            'text-[10px] px-1.5 py-0.5 rounded',
                            STATUS_COLORS[doc.status] || 'text-dark-400 bg-dark-700'
                          )}>
                            {doc.status}
                          </span>
                        )}

                        {/* Tags (first 2) */}
                        {doc.tags?.slice(0, 2).map((tag) => (
                          <span
                            key={tag}
                            className="text-[10px] px-1.5 py-0.5 bg-dark-600 rounded text-dark-300 flex items-center gap-0.5"
                          >
                            <Tag size={8} />
                            {tag}
                          </span>
                        ))}
                        {doc.tags && doc.tags.length > 2 && (
                          <span className="text-[10px] text-dark-500">
                            +{doc.tags.length - 2}
                          </span>
                        )}
                      </div>

                      {/* Date */}
                      {doc.updated && (
                        <div className="flex items-center gap-1 mt-1 text-[10px] text-dark-500">
                          <Clock size={10} />
                          {new Date(doc.updated).toLocaleDateString()}
                        </div>
                      )}
                    </div>
                  </div>
                </button>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}
