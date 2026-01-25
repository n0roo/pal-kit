import { useMemo, useState, useCallback } from 'react'
import { List, ChevronRight, ChevronDown } from 'lucide-react'
import clsx from 'clsx'

interface TocItem {
  id: string
  text: string
  level: number
}

interface MarkdownViewerProps {
  content: string
  className?: string
  showToc?: boolean
}

// Extract ToC from markdown content
function extractToc(content: string): TocItem[] {
  const toc: TocItem[] = []
  const headerRegex = /^(#{1,3})\s+(.+)$/gm
  let match
  
  while ((match = headerRegex.exec(content)) !== null) {
    const level = match[1].length
    const text = match[2].trim()
    const id = text.toLowerCase().replace(/[^a-z0-9가-힣]+/g, '-').replace(/^-|-$/g, '')
    toc.push({ id, text, level })
  }
  
  return toc
}

// Simple markdown to HTML conversion
function markdownToHtml(content: string): string {
  let html = content
    // Escape HTML
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    
  // YAML frontmatter (process first)
  html = html.replace(/^---\n([\s\S]*?)\n---/m, (_, yaml) => 
    `<div class="bg-dark-900 rounded-lg p-3 my-3 border border-dark-600"><div class="text-xs text-dark-400 mb-2">Frontmatter</div><pre class="text-sm text-yellow-400">${yaml}</pre></div>`
  )
  
  // Headers with IDs
  html = html
    .replace(/^### (.+)$/gm, (_, text) => {
      const id = text.toLowerCase().replace(/[^a-z0-9가-힣]+/g, '-').replace(/^-|-$/g, '')
      return `<h3 id="${id}" class="text-lg font-semibold mt-4 mb-2 text-dark-100 scroll-mt-4">${text}</h3>`
    })
    .replace(/^## (.+)$/gm, (_, text) => {
      const id = text.toLowerCase().replace(/[^a-z0-9가-힣]+/g, '-').replace(/^-|-$/g, '')
      return `<h2 id="${id}" class="text-xl font-semibold mt-5 mb-3 text-dark-50 scroll-mt-4">${text}</h2>`
    })
    .replace(/^# (.+)$/gm, (_, text) => {
      const id = text.toLowerCase().replace(/[^a-z0-9가-힣]+/g, '-').replace(/^-|-$/g, '')
      return `<h1 id="${id}" class="text-2xl font-bold mt-6 mb-4 text-white scroll-mt-4">${text}</h1>`
    })
    // Code blocks
    .replace(/```(\w*)\n([\s\S]*?)```/g, (_, lang, code) => 
      `<pre class="bg-dark-900 rounded-lg p-3 my-3 overflow-x-auto"><code class="text-sm text-green-400">${code.trim()}</code></pre>`
    )
    // Inline code
    .replace(/`([^`]+)`/g, '<code class="bg-dark-700 px-1.5 py-0.5 rounded text-sm text-primary-400">$1</code>')
    // Bold
    .replace(/\*\*([^*]+)\*\*/g, '<strong class="font-semibold text-white">$1</strong>')
    // Italic
    .replace(/\*([^*]+)\*/g, '<em class="italic">$1</em>')
    // Links
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g, '<a href="$2" class="text-primary-400 hover:underline" target="_blank">$1</a>')
    // Blockquotes
    .replace(/^> (.+)$/gm, '<blockquote class="border-l-4 border-primary-500 pl-4 my-2 text-dark-300 italic">$1</blockquote>')
    // Unordered lists
    .replace(/^- (.+)$/gm, '<li class="ml-4 list-disc">$1</li>')
    .replace(/^• (.+)$/gm, '<li class="ml-4 list-disc">$1</li>')
    // Ordered lists
    .replace(/^\d+\. (.+)$/gm, '<li class="ml-4 list-decimal">$1</li>')
    // Horizontal rules
    .replace(/^---$/gm, '<hr class="border-dark-600 my-4" />')
    // Tables (basic)
    .replace(/\|(.+)\|/g, (match) => {
      const cells = match.split('|').filter(Boolean).map(cell => cell.trim())
      if (cells.every(c => /^-+$/.test(c))) {
        return '' // Header separator
      }
      return `<tr>${cells.map(c => `<td class="border border-dark-600 px-3 py-1.5">${c}</td>`).join('')}</tr>`
    })
    // Paragraphs
    .replace(/\n\n/g, '</p><p class="my-2">')
    // Line breaks
    .replace(/\n/g, '<br />')

  // Wrap in paragraph if not starting with HTML tag
  if (!html.startsWith('<')) {
    html = `<p class="my-2">${html}</p>`
  }

  return html
}

export default function MarkdownViewer({ content, className = '', showToc = false }: MarkdownViewerProps) {
  const [tocExpanded, setTocExpanded] = useState(true)
  
  const toc = useMemo(() => extractToc(content), [content])
  const rendered = useMemo(() => markdownToHtml(content), [content])

  const scrollToHeading = useCallback((id: string) => {
    const element = document.getElementById(id)
    if (element) {
      element.scrollIntoView({ behavior: 'smooth', block: 'start' })
    }
  }, [])

  return (
    <div className={`flex gap-4 ${className}`}>
      {/* ToC sidebar */}
      {showToc && toc.length > 0 && (
        <div className="w-48 flex-shrink-0">
          <div className="sticky top-0 bg-dark-800 rounded-lg p-3 border border-dark-700">
            <button
              onClick={() => setTocExpanded(!tocExpanded)}
              className="flex items-center gap-2 w-full text-sm font-medium text-dark-300 hover:text-white"
            >
              {tocExpanded ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
              <List size={14} />
              목차
            </button>
            
            {tocExpanded && (
              <nav className="mt-2 space-y-1">
                {toc.map((item, idx) => (
                  <button
                    key={idx}
                    onClick={() => scrollToHeading(item.id)}
                    className={clsx(
                      'block text-left text-xs text-dark-400 hover:text-primary-400 truncate w-full',
                      item.level === 1 && 'font-medium',
                      item.level === 2 && 'pl-2',
                      item.level === 3 && 'pl-4 text-dark-500'
                    )}
                    title={item.text}
                  >
                    {item.text}
                  </button>
                ))}
              </nav>
            )}
          </div>
        </div>
      )}
      
      {/* Content */}
      <div 
        className="flex-1 prose prose-invert prose-sm max-w-none text-dark-200"
        dangerouslySetInnerHTML={{ __html: rendered }}
      />
    </div>
  )
}

// Standalone ToC component for separate use
interface TocViewerProps {
  content: string
  onSelect?: (id: string) => void
  className?: string
}

export function TocViewer({ content, onSelect, className = '' }: TocViewerProps) {
  const toc = useMemo(() => extractToc(content), [content])

  if (toc.length === 0) {
    return (
      <div className={`text-sm text-dark-400 ${className}`}>
        목차가 없습니다
      </div>
    )
  }

  return (
    <nav className={`space-y-1 ${className}`}>
      {toc.map((item, idx) => (
        <button
          key={idx}
          onClick={() => onSelect?.(item.id)}
          className={clsx(
            'block text-left text-sm hover:text-primary-400 truncate w-full transition-colors',
            item.level === 1 && 'font-medium text-dark-200',
            item.level === 2 && 'pl-3 text-dark-300',
            item.level === 3 && 'pl-6 text-dark-400'
          )}
          title={item.text}
        >
          {item.text}
        </button>
      ))}
    </nav>
  )
}

// Export the extractToc function for use elsewhere
export { extractToc }
export type { TocItem }
