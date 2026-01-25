import { Folder, FolderOpen, AlertCircle, CheckCircle, RefreshCw } from 'lucide-react'
import clsx from 'clsx'
import type { KBTocItem } from '../../hooks/useKB'

interface KBSidebarProps {
  sections: KBTocItem[]
  selectedSection: string | null
  onSelectSection: (section: string) => void
  onGenerateToc: (section: string) => void
  loading?: boolean
}

const SECTION_LABELS: Record<string, { label: string; description: string }> = {
  '00-System': { label: 'System', description: '시스템 설정, 템플릿' },
  '10-Domains': { label: 'Domains', description: '도메인 지식' },
  '20-Projects': { label: 'Projects', description: '프로젝트 문서' },
  '30-References': { label: 'References', description: '참조 문서' },
  '40-Archive': { label: 'Archive', description: '아카이브' },
}

export default function KBSidebar({
  sections,
  selectedSection,
  onSelectSection,
  onGenerateToc,
  loading,
}: KBSidebarProps) {
  return (
    <div className="w-14 bg-dark-850 border-r border-dark-700 flex flex-col">
      <div className="p-2 border-b border-dark-700">
        <span className="text-xs text-dark-400 font-medium">KB</span>
      </div>

      <nav className="flex-1 py-2 space-y-1">
        {sections.map((section) => {
          const info = SECTION_LABELS[section.section] || {
            label: section.section.replace(/^\d+-/, ''),
            description: ''
          }
          const isSelected = selectedSection === section.section

          return (
            <div key={section.section} className="relative group">
              <button
                onClick={() => onSelectSection(section.section)}
                className={clsx(
                  'w-full flex flex-col items-center py-2 px-1 transition-colors',
                  isSelected
                    ? 'bg-primary-600/20 text-primary-400'
                    : 'text-dark-400 hover:bg-dark-700 hover:text-dark-200'
                )}
                title={`${info.label}: ${info.description}`}
              >
                {isSelected ? (
                  <FolderOpen size={18} />
                ) : (
                  <Folder size={18} />
                )}
                <span className="text-[10px] mt-1 truncate w-full text-center">
                  {info.label}
                </span>

                {/* Status indicator */}
                {section.exists && (
                  <div className="absolute top-1 right-1">
                    {section.needs_refresh ? (
                      <AlertCircle size={8} className="text-yellow-500" />
                    ) : section.valid ? (
                      <CheckCircle size={8} className="text-green-500" />
                    ) : null}
                  </div>
                )}
              </button>

              {/* Hover tooltip with generate option */}
              <div className="absolute left-full top-0 ml-2 hidden group-hover:block z-50">
                <div className="bg-dark-800 border border-dark-600 rounded-lg p-2 shadow-lg min-w-[160px]">
                  <div className="text-xs font-medium text-dark-200 mb-1">
                    {info.label}
                  </div>
                  <div className="text-[10px] text-dark-400 mb-2">
                    {info.description}
                  </div>

                  {!section.exists ? (
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        onGenerateToc(section.section)
                      }}
                      disabled={loading}
                      className="w-full text-xs px-2 py-1 bg-primary-600 hover:bg-primary-700 rounded text-white flex items-center justify-center gap-1"
                    >
                      <RefreshCw size={10} className={loading ? 'animate-spin' : ''} />
                      TOC 생성
                    </button>
                  ) : section.needs_refresh ? (
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        onGenerateToc(section.section)
                      }}
                      disabled={loading}
                      className="w-full text-xs px-2 py-1 bg-yellow-600 hover:bg-yellow-700 rounded text-white flex items-center justify-center gap-1"
                    >
                      <RefreshCw size={10} className={loading ? 'animate-spin' : ''} />
                      TOC 갱신
                    </button>
                  ) : (
                    <div className="text-[10px] text-green-400 flex items-center gap-1">
                      <CheckCircle size={10} />
                      TOC 최신
                    </div>
                  )}

                  {section.exists && (
                    <div className="mt-2 pt-2 border-t border-dark-600 text-[10px] text-dark-400">
                      {section.missing_count ? (
                        <div className="text-yellow-500">누락: {section.missing_count}</div>
                      ) : null}
                      {section.orphan_count ? (
                        <div className="text-orange-500">고아 링크: {section.orphan_count}</div>
                      ) : null}
                    </div>
                  )}
                </div>
              </div>
            </div>
          )
        })}
      </nav>
    </div>
  )
}
