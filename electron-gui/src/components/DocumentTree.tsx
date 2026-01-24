import { useState } from 'react'
import {
  ChevronRight, ChevronDown, Folder, FolderOpen,
  FileText, FileCode, FileCog, FileSearch, File
} from 'lucide-react'
import clsx from 'clsx'
import type { DocumentTreeNode } from '../hooks/useApi'

interface DocumentTreeProps {
  tree: DocumentTreeNode | null
  selectedPath: string | null
  onSelectNode: (node: DocumentTreeNode) => void
  onNavigate?: (path: string) => void
  loading?: boolean
}

interface TreeNodeProps {
  node: DocumentTreeNode
  selectedPath: string | null
  onSelect: (node: DocumentTreeNode) => void
  onNavigate?: (path: string) => void
  depth?: number
}

const DOC_TYPE_ICONS: Record<string, typeof FileText> = {
  port: FileCog,
  convention: FileCode,
  agent: FileSearch,
  docs: FileText,
  adr: FileText,
  session: FileText,
  markdown: FileText,
  yaml: FileCode,
}

const DOC_TYPE_COLORS: Record<string, string> = {
  port: 'text-blue-400',
  convention: 'text-purple-400',
  agent: 'text-green-400',
  docs: 'text-yellow-400',
  adr: 'text-orange-400',
  session: 'text-cyan-400',
}

function TreeNode({ node, selectedPath, onSelect, onNavigate, depth = 0 }: TreeNodeProps) {
  const [expanded, setExpanded] = useState(depth < 2)
  const isSelected = selectedPath === node.path
  const isDirectory = node.type === 'directory'
  const hasChildren = isDirectory && node.children && node.children.length > 0

  const Icon = isDirectory
    ? (expanded ? FolderOpen : Folder)
    : (DOC_TYPE_ICONS[node.doc_type || ''] || File)

  const colorClass = !isDirectory && node.doc_type
    ? DOC_TYPE_COLORS[node.doc_type] || 'text-dark-400'
    : 'text-dark-400'

  const handleClick = () => {
    if (isDirectory) {
      if (hasChildren) {
        setExpanded(!expanded)
      }
      // Also allow navigation into directory
      if (onNavigate) {
        onNavigate(node.path)
      }
    }
    onSelect(node)
  }

  const handleToggle = (e: React.MouseEvent) => {
    e.stopPropagation()
    setExpanded(!expanded)
  }

  return (
    <div>
      <div
        onClick={handleClick}
        className={clsx(
          'flex items-center gap-1 py-1 px-2 rounded cursor-pointer transition-colors',
          isSelected
            ? 'bg-primary-600/20 text-primary-400'
            : 'hover:bg-dark-700 text-dark-300 hover:text-dark-100'
        )}
        style={{ paddingLeft: `${depth * 16 + 8}px` }}
      >
        {/* Expand/Collapse toggle */}
        {hasChildren ? (
          <button
            onClick={handleToggle}
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
        <Icon
          size={14}
          className={clsx('flex-shrink-0', colorClass)}
        />

        {/* Name */}
        <span className="text-sm truncate">{node.name}</span>

        {/* Doc type badge for files */}
        {!isDirectory && node.doc_type && node.doc_type !== 'other' && (
          <span className={clsx(
            'ml-auto text-[10px] px-1.5 py-0.5 rounded',
            DOC_TYPE_COLORS[node.doc_type] || 'text-dark-400',
            'bg-dark-700/50'
          )}>
            {node.doc_type}
          </span>
        )}
      </div>

      {/* Children */}
      {hasChildren && expanded && (
        <div>
          {node.children!.map((child) => (
            <TreeNode
              key={child.path}
              node={child}
              selectedPath={selectedPath}
              onSelect={onSelect}
              onNavigate={onNavigate}
              depth={depth + 1}
            />
          ))}
        </div>
      )}
    </div>
  )
}

export default function DocumentTree({
  tree,
  selectedPath,
  onSelectNode,
  onNavigate,
  loading,
}: DocumentTreeProps) {
  if (loading) {
    return (
      <div className="flex items-center justify-center h-32 text-dark-400">
        <span className="text-sm">로딩 중...</span>
      </div>
    )
  }

  if (!tree) {
    return (
      <div className="flex items-center justify-center h-32 text-dark-400">
        <div className="text-center">
          <Folder size={24} className="mx-auto mb-2 opacity-50" />
          <p className="text-sm">트리를 불러올 수 없습니다</p>
        </div>
      </div>
    )
  }

  return (
    <div className="py-2">
      {tree.children?.map((child) => (
        <TreeNode
          key={child.path}
          node={child}
          selectedPath={selectedPath}
          onSelect={onSelectNode}
          onNavigate={onNavigate}
        />
      ))}
    </div>
  )
}
