import { useState } from 'react'
import { X, Globe, Eye, Edit3, CheckCircle, AlertTriangle } from 'lucide-react'
import clsx from 'clsx'
import { useKBDocuments, type KBRegisterResult } from '../../hooks/useKB'
import MarkdownViewer from '../MarkdownViewer'

const KB_SECTIONS = [
  { id: '00-System', label: 'System' },
  { id: '10-Domains', label: 'Domains' },
  { id: '20-Projects', label: 'Projects' },
  { id: '30-References', label: 'References' },
  { id: '40-Archive', label: 'Archive' },
]

const DOC_TYPES = [
  { value: '', label: '선택 안함' },
  { value: 'reference', label: 'Reference' },
  { value: 'guide', label: 'Guide' },
  { value: 'concept', label: 'Concept' },
  { value: 'adr', label: 'ADR' },
  { value: 'convention', label: 'Convention' },
]

interface KBExternalDialogProps {
  onClose: () => void
  onSuccess?: (result: KBRegisterResult) => void
}

export default function KBExternalDialog({ onClose, onSuccess }: KBExternalDialogProps) {
  const { registerExternal } = useKBDocuments()

  const [title, setTitle] = useState('')
  const [content, setContent] = useState('')
  const [targetSection, setTargetSection] = useState('30-References')
  const [docType, setDocType] = useState('')
  const [tagsInput, setTagsInput] = useState('')
  const [previewMode, setPreviewMode] = useState(false)
  const [registering, setRegistering] = useState(false)
  const [result, setResult] = useState<KBRegisterResult | null>(null)

  const handleRegister = async () => {
    if (!title.trim()) return

    setRegistering(true)
    const tags = tagsInput
      .split(',')
      .map(t => t.trim())
      .filter(Boolean)

    const res = await registerExternal(title, content, targetSection, {
      type: docType,
      tags,
    })
    setRegistering(false)

    if (res) {
      setResult(res)
      if (res.status === 'registered' && onSuccess) {
        onSuccess(res)
      }
    }
  }

  return (
    <div className="fixed inset-0 z-50 bg-black/50 flex items-center justify-center">
      <div className="bg-dark-800 border border-dark-600 rounded-xl shadow-2xl w-[720px] max-h-[85vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-dark-700">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <Globe size={20} className="text-green-400" />
            외부 문서 등록
          </h3>
          <button onClick={onClose} className="p-1 hover:bg-dark-700 rounded">
            <X size={18} />
          </button>
        </div>

        {/* Result display */}
        {result && (
          <div
            className={clsx(
              'mx-4 mt-4 rounded-lg p-3 text-sm flex items-start gap-2',
              result.status === 'registered' && 'bg-green-600/20 text-green-400',
              result.status === 'duplicate' && 'bg-yellow-600/20 text-yellow-400',
              result.status === 'error' && 'bg-red-600/20 text-red-400'
            )}
          >
            {result.status === 'registered' ? (
              <CheckCircle size={16} className="mt-0.5 flex-shrink-0" />
            ) : (
              <AlertTriangle size={16} className="mt-0.5 flex-shrink-0" />
            )}
            <div>
              <div className="font-medium">{result.message}</div>
              {result.kb_path && (
                <div className="text-xs mt-1 opacity-75">경로: {result.kb_path}</div>
              )}
            </div>
          </div>
        )}

        {/* Body */}
        {!result && (
          <div className="flex-1 overflow-auto p-4 space-y-4">
            {/* Title */}
            <div>
              <label className="block text-sm text-dark-400 mb-1">제목 *</label>
              <input
                type="text"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                placeholder="문서 제목"
                className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm focus:border-primary-500 focus:outline-none"
                autoFocus
              />
            </div>

            {/* Type + Section */}
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm text-dark-400 mb-1">타입</label>
                <select
                  value={docType}
                  onChange={(e) => setDocType(e.target.value)}
                  className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm focus:border-primary-500 focus:outline-none"
                >
                  {DOC_TYPES.map(t => (
                    <option key={t.value} value={t.value}>{t.label}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm text-dark-400 mb-1">대상 섹션</label>
                <select
                  value={targetSection}
                  onChange={(e) => setTargetSection(e.target.value)}
                  className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm focus:border-primary-500 focus:outline-none"
                >
                  {KB_SECTIONS.map(s => (
                    <option key={s.id} value={s.id}>{s.label}</option>
                  ))}
                </select>
              </div>
            </div>

            {/* Tags */}
            <div>
              <label className="block text-sm text-dark-400 mb-1">태그 (쉼표 구분)</label>
              <input
                type="text"
                value={tagsInput}
                onChange={(e) => setTagsInput(e.target.value)}
                placeholder="tag1, tag2, tag3"
                className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm focus:border-primary-500 focus:outline-none"
              />
            </div>

            {/* Content with preview toggle */}
            <div>
              <div className="flex items-center justify-between mb-1">
                <label className="text-sm text-dark-400">내용 (마크다운)</label>
                <button
                  onClick={() => setPreviewMode(!previewMode)}
                  className="flex items-center gap-1 text-xs text-dark-400 hover:text-dark-200"
                >
                  {previewMode ? <Edit3 size={12} /> : <Eye size={12} />}
                  {previewMode ? '편집' : '미리보기'}
                </button>
              </div>

              {previewMode ? (
                <div className="bg-dark-900 border border-dark-600 rounded-lg p-4 min-h-[200px] max-h-[300px] overflow-auto">
                  <MarkdownViewer content={content || '*내용이 없습니다*'} />
                </div>
              ) : (
                <textarea
                  value={content}
                  onChange={(e) => setContent(e.target.value)}
                  placeholder="마크다운 내용을 입력하세요..."
                  className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm font-mono min-h-[200px] max-h-[300px] resize-y focus:border-primary-500 focus:outline-none"
                />
              )}
            </div>
          </div>
        )}

        {/* Footer */}
        <div className="flex justify-end gap-2 p-4 border-t border-dark-700">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-dark-400 hover:text-dark-200"
          >
            {result ? '닫기' : '취소'}
          </button>
          {!result && (
            <button
              onClick={handleRegister}
              disabled={!title.trim() || registering}
              className="px-4 py-2 bg-green-600 hover:bg-green-700 disabled:bg-dark-600 rounded-lg text-sm"
            >
              {registering ? '등록 중...' : '등록'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
