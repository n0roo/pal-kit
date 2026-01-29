import { useState } from 'react'
import { X, BookOpen, AlertTriangle, CheckCircle } from 'lucide-react'
import clsx from 'clsx'
import { useKBDocuments, type KBRegisterResult } from '../../hooks/useKB'

const KB_SECTIONS = [
  { id: '00-System', label: 'System', desc: '시스템 설정 문서' },
  { id: '10-Domains', label: 'Domains', desc: '도메인 지식 문서' },
  { id: '20-Projects', label: 'Projects', desc: '프로젝트 문서' },
  { id: '30-References', label: 'References', desc: '참조 문서' },
  { id: '40-Archive', label: 'Archive', desc: '아카이브' },
]

interface KBRegisterDialogProps {
  documentPath: string
  documentId: string
  onClose: () => void
  onSuccess?: (result: KBRegisterResult) => void
}

export default function KBRegisterDialog({
  documentPath,
  documentId,
  onClose,
  onSuccess,
}: KBRegisterDialogProps) {
  const { registerFromProject } = useKBDocuments()

  const [targetSection, setTargetSection] = useState('20-Projects')
  const [customPath, setCustomPath] = useState('')
  const [registering, setRegistering] = useState(false)
  const [result, setResult] = useState<KBRegisterResult | null>(null)

  const handleRegister = async () => {
    setRegistering(true)
    const res = await registerFromProject(
      documentPath,
      targetSection,
      customPath || undefined
    )
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
      <div className="bg-dark-800 border border-dark-600 rounded-xl p-6 w-[480px] shadow-2xl">
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-lg font-semibold flex items-center gap-2">
            <BookOpen size={20} className="text-primary-400" />
            KB에 등록
          </h3>
          <button onClick={onClose} className="p-1 hover:bg-dark-700 rounded">
            <X size={18} />
          </button>
        </div>

        {/* Source info */}
        <div className="bg-dark-900 rounded-lg p-3 mb-4 text-sm">
          <div className="text-xs text-dark-400 mb-1">소스 문서</div>
          <div className="text-dark-200 truncate">{documentPath}</div>
        </div>

        {/* Result display */}
        {result && (
          <div
            className={clsx(
              'rounded-lg p-3 mb-4 text-sm flex items-start gap-2',
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
              {result.existing_path && (
                <div className="text-xs mt-1 opacity-75">
                  기존 경로: {result.existing_path}
                </div>
              )}
            </div>
          </div>
        )}

        {/* Section selector */}
        {!result && (
          <div className="space-y-4">
            <div>
              <label className="block text-sm text-dark-400 mb-2">대상 섹션</label>
              <div className="space-y-1">
                {KB_SECTIONS.map(section => (
                  <button
                    key={section.id}
                    onClick={() => setTargetSection(section.id)}
                    className={clsx(
                      'w-full text-left px-3 py-2 rounded text-sm transition-colors',
                      targetSection === section.id
                        ? 'bg-primary-600/20 text-primary-400 border border-primary-500/30'
                        : 'bg-dark-700 text-dark-300 hover:bg-dark-600 border border-transparent'
                    )}
                  >
                    <span className="font-medium">{section.label}</span>
                    <span className="text-xs text-dark-500 ml-2">{section.desc}</span>
                  </button>
                ))}
              </div>
            </div>

            <div>
              <label className="block text-sm text-dark-400 mb-1">
                커스텀 경로 <span className="text-dark-500">(선택)</span>
              </label>
              <input
                type="text"
                value={customPath}
                onChange={(e) => setCustomPath(e.target.value)}
                placeholder={`${targetSection}/${documentId}`}
                className="w-full px-3 py-2 bg-dark-900 border border-dark-600 rounded-lg text-sm focus:border-primary-500 focus:outline-none"
              />
            </div>
          </div>
        )}

        {/* Actions */}
        <div className="flex justify-end gap-2 mt-6">
          <button
            onClick={onClose}
            className="px-4 py-2 text-sm text-dark-400 hover:text-dark-200"
          >
            {result ? '닫기' : '취소'}
          </button>
          {!result && (
            <button
              onClick={handleRegister}
              disabled={registering}
              className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded-lg text-sm"
            >
              {registering ? '등록 중...' : '등록'}
            </button>
          )}
        </div>
      </div>
    </div>
  )
}
