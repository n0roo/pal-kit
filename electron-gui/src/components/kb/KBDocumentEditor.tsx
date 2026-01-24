import { useState, useEffect } from 'react'
import {
  Save, X, Trash2, Move, Copy, Eye, Code,
  Tag, FileText, AlertTriangle
} from 'lucide-react'
import clsx from 'clsx'
import type { KBDocumentDetail } from '../../hooks/useKB'
import { MarkdownViewer } from '../'

interface KBDocumentEditorProps {
  document: KBDocumentDetail | null
  isNew?: boolean
  onSave: (doc: Partial<KBDocumentDetail>) => Promise<boolean>
  onDelete: () => Promise<boolean>
  onMove: () => void
  onDuplicate: () => void
  onClose: () => void
  saving?: boolean
}

const DOC_TYPES = [
  { value: 'concept', label: '개념' },
  { value: 'guide', label: '가이드' },
  { value: 'port', label: '포트' },
  { value: 'adr', label: 'ADR' },
  { value: 'template', label: '템플릿' },
  { value: 'reference', label: '참조' },
  { value: 'note', label: '노트' },
]

const DOC_STATUSES = [
  { value: 'draft', label: '초안' },
  { value: 'active', label: '활성' },
  { value: 'review', label: '검토' },
  { value: 'archived', label: '아카이브' },
]

export default function KBDocumentEditor({
  document,
  isNew = false,
  onSave,
  onDelete,
  onMove,
  onDuplicate,
  onClose,
  saving,
}: KBDocumentEditorProps) {
  const [editMode, setEditMode] = useState(isNew)
  const [title, setTitle] = useState('')
  const [type, setType] = useState('note')
  const [status, setStatus] = useState('draft')
  const [tags, setTags] = useState<string[]>([])
  const [tagInput, setTagInput] = useState('')
  const [summary, setSummary] = useState('')
  const [content, setContent] = useState('')
  const [hasChanges, setHasChanges] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  // Initialize form with document data
  useEffect(() => {
    if (document) {
      setTitle(document.title || '')
      setType(document.type || 'note')
      setStatus(document.status || 'draft')
      setTags(document.tags || [])
      setSummary(document.summary || '')
      setContent(document.content || '')
      setHasChanges(false)
    } else if (isNew) {
      setTitle('')
      setType('note')
      setStatus('draft')
      setTags([])
      setSummary('')
      setContent('')
      setHasChanges(false)
    }
  }, [document, isNew])

  // Track changes
  useEffect(() => {
    if (!document && !isNew) return

    const original = document || { title: '', type: 'note', status: 'draft', tags: [], summary: '', content: '' }
    const changed =
      title !== (original.title || '') ||
      type !== (original.type || 'note') ||
      status !== (original.status || 'draft') ||
      JSON.stringify(tags) !== JSON.stringify(original.tags || []) ||
      summary !== (original.summary || '') ||
      content !== (original.content || '')

    setHasChanges(changed)
  }, [title, type, status, tags, summary, content, document, isNew])

  const handleSave = async () => {
    const success = await onSave({
      title,
      type,
      status,
      tags,
      summary,
      content,
    })
    if (success) {
      setEditMode(false)
      setHasChanges(false)
    }
  }

  const handleAddTag = () => {
    const tag = tagInput.trim().toLowerCase()
    if (tag && !tags.includes(tag)) {
      setTags([...tags, tag])
    }
    setTagInput('')
  }

  const handleRemoveTag = (tag: string) => {
    setTags(tags.filter((t) => t !== tag))
  }

  const handleDelete = async () => {
    const success = await onDelete()
    if (success) {
      setShowDeleteConfirm(false)
    }
  }

  if (!document && !isNew) {
    return (
      <div className="flex-1 flex items-center justify-center text-dark-400">
        <div className="text-center">
          <FileText size={48} className="mx-auto mb-4 opacity-50" />
          <p>문서를 선택하거나 새로 만드세요</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 flex flex-col h-full overflow-hidden">
      {/* Header */}
      <div className="p-3 border-b border-dark-700 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm truncate max-w-[200px]">
            {isNew ? '새 문서' : title || '제목 없음'}
          </span>
          {hasChanges && (
            <span className="text-xs px-1.5 py-0.5 bg-yellow-500/20 text-yellow-400 rounded">
              수정됨
            </span>
          )}
        </div>

        <div className="flex items-center gap-1">
          {editMode ? (
            <>
              <button
                onClick={() => setEditMode(false)}
                className="p-1.5 text-dark-400 hover:text-dark-200 hover:bg-dark-700 rounded"
                title="미리보기"
              >
                <Eye size={16} />
              </button>
              <button
                onClick={handleSave}
                disabled={saving || !title.trim()}
                className="flex items-center gap-1 px-2 py-1 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded text-sm"
              >
                <Save size={14} />
                {saving ? '저장 중...' : '저장'}
              </button>
            </>
          ) : (
            <>
              <button
                onClick={() => setEditMode(true)}
                className="p-1.5 text-dark-400 hover:text-dark-200 hover:bg-dark-700 rounded"
                title="편집"
              >
                <Code size={16} />
              </button>
              {!isNew && (
                <>
                  <button
                    onClick={onMove}
                    className="p-1.5 text-dark-400 hover:text-dark-200 hover:bg-dark-700 rounded"
                    title="이동"
                  >
                    <Move size={16} />
                  </button>
                  <button
                    onClick={onDuplicate}
                    className="p-1.5 text-dark-400 hover:text-dark-200 hover:bg-dark-700 rounded"
                    title="복제"
                  >
                    <Copy size={16} />
                  </button>
                  <button
                    onClick={() => setShowDeleteConfirm(true)}
                    className="p-1.5 text-red-400 hover:text-red-300 hover:bg-red-500/20 rounded"
                    title="삭제"
                  >
                    <Trash2 size={16} />
                  </button>
                </>
              )}
            </>
          )}
          <button
            onClick={onClose}
            className="p-1.5 text-dark-400 hover:text-dark-200 hover:bg-dark-700 rounded ml-2"
          >
            <X size={16} />
          </button>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto">
        {editMode ? (
          <div className="p-4 space-y-4">
            {/* Title */}
            <div>
              <label className="block text-xs text-dark-400 mb-1">제목</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="문서 제목"
                className="w-full px-3 py-2 bg-dark-800 border border-dark-600 rounded focus:outline-none focus:border-primary-500"
              />
            </div>

            {/* Type & Status */}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs text-dark-400 mb-1">타입</label>
                <select
                  value={type}
                  onChange={(e) => setType(e.target.value)}
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-600 rounded focus:outline-none focus:border-primary-500"
                >
                  {DOC_TYPES.map((t) => (
                    <option key={t.value} value={t.value}>
                      {t.label}
                    </option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-xs text-dark-400 mb-1">상태</label>
                <select
                  value={status}
                  onChange={(e) => setStatus(e.target.value)}
                  className="w-full px-3 py-2 bg-dark-800 border border-dark-600 rounded focus:outline-none focus:border-primary-500"
                >
                  {DOC_STATUSES.map((s) => (
                    <option key={s.value} value={s.value}>
                      {s.label}
                    </option>
                  ))}
                </select>
              </div>
            </div>

            {/* Tags */}
            <div>
              <label className="block text-xs text-dark-400 mb-1">태그</label>
              <div className="flex flex-wrap gap-1 mb-2">
                {tags.map((tag) => (
                  <span
                    key={tag}
                    className="flex items-center gap-1 px-2 py-0.5 bg-dark-700 rounded text-xs"
                  >
                    <Tag size={10} />
                    {tag}
                    <button
                      onClick={() => handleRemoveTag(tag)}
                      className="ml-1 text-dark-400 hover:text-red-400"
                    >
                      <X size={10} />
                    </button>
                  </span>
                ))}
              </div>
              <div className="flex gap-2">
                <input
                  type="text"
                  value={tagInput}
                  onChange={(e) => setTagInput(e.target.value)}
                  onKeyDown={(e) => e.key === 'Enter' && handleAddTag()}
                  placeholder="태그 추가"
                  className="flex-1 px-3 py-1.5 bg-dark-800 border border-dark-600 rounded text-sm focus:outline-none focus:border-primary-500"
                />
                <button
                  onClick={handleAddTag}
                  className="px-3 py-1.5 bg-dark-700 hover:bg-dark-600 rounded text-sm"
                >
                  추가
                </button>
              </div>
            </div>

            {/* Summary */}
            <div>
              <label className="block text-xs text-dark-400 mb-1">요약</label>
              <textarea
                value={summary}
                onChange={(e) => setSummary(e.target.value)}
                placeholder="문서 요약 (목차에 표시됨)"
                rows={2}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-600 rounded resize-none focus:outline-none focus:border-primary-500"
              />
            </div>

            {/* Content */}
            <div>
              <label className="block text-xs text-dark-400 mb-1">내용</label>
              <textarea
                value={content}
                onChange={(e) => setContent(e.target.value)}
                placeholder="Markdown 형식으로 작성..."
                rows={15}
                className="w-full px-3 py-2 bg-dark-800 border border-dark-600 rounded font-mono text-sm resize-none focus:outline-none focus:border-primary-500"
              />
            </div>
          </div>
        ) : (
          <div className="p-4">
            {/* Meta info */}
            <div className="mb-4 pb-4 border-b border-dark-700">
              <h1 className="text-xl font-semibold mb-2">{title}</h1>
              <div className="flex items-center gap-2 flex-wrap text-xs">
                <span className="px-2 py-0.5 bg-dark-700 rounded">{type}</span>
                <span className={clsx(
                  'px-2 py-0.5 rounded',
                  status === 'active' ? 'bg-green-500/20 text-green-400' :
                  status === 'draft' ? 'bg-yellow-500/20 text-yellow-400' :
                  status === 'review' ? 'bg-blue-500/20 text-blue-400' :
                  'bg-gray-500/20 text-gray-400'
                )}>
                  {status}
                </span>
                {tags.map((tag) => (
                  <span key={tag} className="px-2 py-0.5 bg-dark-600 rounded flex items-center gap-1">
                    <Tag size={10} />
                    {tag}
                  </span>
                ))}
              </div>
              {summary && (
                <p className="mt-2 text-sm text-dark-400">{summary}</p>
              )}
            </div>

            {/* Content preview */}
            <div className="prose prose-invert prose-sm max-w-none">
              <MarkdownViewer content={content} />
            </div>
          </div>
        )}
      </div>

      {/* Delete confirmation */}
      {showDeleteConfirm && (
        <div className="absolute inset-0 bg-black/50 flex items-center justify-center z-50">
          <div className="bg-dark-800 rounded-lg p-4 max-w-sm mx-4">
            <div className="flex items-center gap-2 text-red-400 mb-3">
              <AlertTriangle size={20} />
              <span className="font-medium">문서 삭제</span>
            </div>
            <p className="text-sm text-dark-300 mb-4">
              "{title}" 문서를 삭제하시겠습니까? 이 작업은 되돌릴 수 없습니다.
            </p>
            <div className="flex justify-end gap-2">
              <button
                onClick={() => setShowDeleteConfirm(false)}
                className="px-3 py-1.5 bg-dark-700 hover:bg-dark-600 rounded text-sm"
              >
                취소
              </button>
              <button
                onClick={handleDelete}
                className="px-3 py-1.5 bg-red-600 hover:bg-red-700 rounded text-sm"
              >
                삭제
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
