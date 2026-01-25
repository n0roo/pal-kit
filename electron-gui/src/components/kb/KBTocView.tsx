import { useState } from 'react'
import { ChevronRight, ChevronDown, FileText, Folder } from 'lucide-react'
import clsx from 'clsx'
import type { KBTocEntry } from '../../hooks/useKB'

interface KBTocViewProps {
  entries: KBTocEntry[]
  selectedPath: string | null
  onSelectEntry: (entry: KBTocEntry) => void
  loading?: boolean
}

interface TocNodeProps {
  entry: KBTocEntry
  selectedPath: string | null
  onSelect: (entry: KBTocEntry) => void
  depth?: number
}

function TocNode({ entry, selectedPath, onSelect, depth = 0 }: TocNodeProps) {
  const [expanded, setExpanded] = useState(depth < 2)
  const hasChildren = entry.children && entry.children.length > 0
  const isSelected = selectedPath === entry.path

  return (
    <div>
      <div
        className={clsx(
          'flex items-start gap-1 py-1.5 px-2 rounded cursor-pointer transition-colors',
          isSelected
            ? 'bg-primary-600/20 text-primary-400'
            : 'hover:bg-dark-700 text-dark-300 hover:text-dark-100'
        )}
        style={{ paddingLeft: `${depth * 12 + 8}px` }}
        onClick={() => {
          if (hasChildren) {
            setExpanded(!expanded)
          }
          onSelect(entry)
        }}
      >
        {/* Expand/Collapse icon */}
        {hasChildren ? (
          <button
            onClick={(e) => {
              e.stopPropagation()
              setExpanded(!expanded)
            }}
            className="p-0.5 -ml-1 text-dark-400 hover:text-dark-200"
          >
            {expanded ? (
              <ChevronDown size={12} />
            ) : (
              <ChevronRight size={12} />
            )}
          </button>
        ) : (
          <span className="w-4" />
        )}

        {/* Icon */}
        {hasChildren ? (
          <Folder size={14} className="flex-shrink-0 mt-0.5 text-dark-400" />
        ) : (
          <FileText size={14} className="flex-shrink-0 mt-0.5 text-dark-400" />
        )}

        {/* Content */}
        <div className="flex-1 min-w-0">
          <div className="text-sm font-medium truncate">{entry.title}</div>
          {entry.summary && (
            <div className="text-xs text-dark-500 truncate mt-0.5">
              {entry.summary}
            </div>
          )}
        </div>
      </div>

      {/* Children */}
      {hasChildren && expanded && (
        <div>
          {entry.children!.map((child, index) => (
            <TocNode
              key={child.path || index}
              entry={child}
              selectedPath={selectedPath}
              onSelect={onSelect}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export default function KBTocView({
  entries,
  selectedPath,
  onSelectEntry,
  loading,
}: KBTocViewProps) {
  if (loading) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-dark-400 text-sm">로딩 중...</div>
      </div>
    )
  }

  if (!entries || entries.length === 0) {
    return (
      <div className="flex-1 flex items-center justify-center">
        <div className="text-center text-dark-400">
          <Folder size={32} className="mx-auto mb-2 opacity-50" />
          <p className="text-sm">목차가 없습니다</p>
          <p className="text-xs mt-1">TOC를 생성해주세요</p>
        </div>
      </div>
    )
  }

  return (
    <div className="flex-1 overflow-auto py-2">
      {entries.map((entry, index) => (
        <TocNode
          key={entry.path || index}
          entry={entry}
          selectedPath={selectedPath}
          onSelect={onSelectEntry}
        />
      ))}
    </div>
  )
}
