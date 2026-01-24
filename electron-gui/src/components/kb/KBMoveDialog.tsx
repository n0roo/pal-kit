import { useState } from 'react'
import { X, Folder, ChevronRight, Move } from 'lucide-react'
import clsx from 'clsx'
import type { KBTocItem } from '../../hooks/useKB'

interface KBMoveDialogProps {
  documentTitle: string
  currentPath: string
  sections: KBTocItem[]
  onMove: (targetPath: string) => Promise<boolean>
  onClose: () => void
}

const SECTION_LABELS: Record<string, string> = {
  '00-System': 'System',
  '10-Domains': 'Domains',
  '20-Projects': 'Projects',
  '30-References': 'References',
  '40-Archive': 'Archive',
}

export default function KBMoveDialog({
  documentTitle,
  currentPath,
  sections,
  onMove,
  onClose,
}: KBMoveDialogProps) {
  const [selectedSection, setSelectedSection] = useState<string | null>(null)
  const [customPath, setCustomPath] = useState('')
  const [moving, setMoving] = useState(false)

  const handleMove = async () => {
    const targetPath = customPath || selectedSection
    if (!targetPath) return

    setMoving(true)
    const success = await onMove(targetPath)
    setMoving(false)

    if (success) {
      onClose()
    }
  }

  const currentSection = currentPath.split('/')[0]

  return (
    <div className="fixed inset-0 bg-black/50 flex items-center justify-center z-50">
      <div className="bg-dark-800 rounded-lg w-full max-w-md mx-4 shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b border-dark-700">
          <div className="flex items-center gap-2">
            <Move size={18} className="text-primary-400" />
            <span className="font-medium">문서 이동</span>
          </div>
          <button
            onClick={onClose}
            className="p-1 text-dark-400 hover:text-dark-200 hover:bg-dark-700 rounded"
          >
            <X size={18} />
          </button>
        </div>

        {/* Content */}
        <div className="p-4">
          {/* Document info */}
          <div className="mb-4 p-3 bg-dark-700 rounded-lg">
            <div className="text-sm font-medium">{documentTitle}</div>
            <div className="text-xs text-dark-400 mt-1 flex items-center gap-1">
              <Folder size={12} />
              {currentPath}
            </div>
          </div>

          {/* Section selection */}
          <div className="mb-4">
            <label className="block text-xs text-dark-400 mb-2">이동할 섹션</label>
            <div className="space-y-1">
              {sections.map((section) => {
                const isCurrentSection = section.section === currentSection
                const isSelected = selectedSection === section.section

                return (
                  <button
                    key={section.section}
                    onClick={() => {
                      setSelectedSection(section.section)
                      setCustomPath('')
                    }}
                    disabled={isCurrentSection}
                    className={clsx(
                      'w-full flex items-center gap-2 p-2 rounded transition-colors text-left',
                      isSelected
                        ? 'bg-primary-600/20 border border-primary-500/50'
                        : isCurrentSection
                        ? 'bg-dark-700/50 text-dark-500 cursor-not-allowed'
                        : 'hover:bg-dark-700 border border-transparent'
                    )}
                  >
                    <Folder size={16} className={isSelected ? 'text-primary-400' : 'text-dark-400'} />
                    <div className="flex-1">
                      <div className="text-sm">
                        {SECTION_LABELS[section.section] || section.section}
                      </div>
                      <div className="text-xs text-dark-500">{section.section}</div>
                    </div>
                    {isCurrentSection && (
                      <span className="text-xs text-dark-500">현재 위치</span>
                    )}
                    {isSelected && (
                      <ChevronRight size={14} className="text-primary-400" />
                    )}
                  </button>
                )
              })}
            </div>
          </div>

          {/* Custom path input */}
          <div className="mb-4">
            <label className="block text-xs text-dark-400 mb-2">
              또는 직접 경로 입력
            </label>
            <input
              type="text"
              value={customPath}
              onChange={(e) => {
                setCustomPath(e.target.value)
                setSelectedSection(null)
              }}
              placeholder="예: 20-Projects/my-project/ports"
              className="w-full px-3 py-2 bg-dark-700 border border-dark-600 rounded text-sm focus:outline-none focus:border-primary-500"
            />
          </div>

          {/* Preview */}
          {(selectedSection || customPath) && (
            <div className="mb-4 p-3 bg-dark-700/50 rounded-lg border border-dashed border-dark-600">
              <div className="text-xs text-dark-400 mb-1">이동 후 경로</div>
              <div className="text-sm flex items-center gap-1">
                <Folder size={14} className="text-primary-400" />
                {customPath || `${selectedSection}/${currentPath.split('/').slice(1).join('/')}`}
              </div>
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="flex justify-end gap-2 p-4 border-t border-dark-700">
          <button
            onClick={onClose}
            className="px-4 py-2 bg-dark-700 hover:bg-dark-600 rounded text-sm"
          >
            취소
          </button>
          <button
            onClick={handleMove}
            disabled={(!selectedSection && !customPath) || moving}
            className="px-4 py-2 bg-primary-600 hover:bg-primary-700 disabled:bg-dark-600 rounded text-sm flex items-center gap-2"
          >
            <Move size={14} />
            {moving ? '이동 중...' : '이동'}
          </button>
        </div>
      </div>
    </div>
  )
}
