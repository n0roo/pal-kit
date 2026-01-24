import { ChevronRight, Home, Folder } from 'lucide-react'
import clsx from 'clsx'

interface DocumentBreadcrumbProps {
  path: string
  onNavigate: (path: string) => void
  className?: string
}

export default function DocumentBreadcrumb({
  path,
  onNavigate,
  className,
}: DocumentBreadcrumbProps) {
  // Split path into segments
  const segments = path.split('/').filter(Boolean)

  // Build breadcrumb items
  const items = [
    { label: 'Home', path: '.', icon: Home },
    ...segments.map((segment, index) => ({
      label: segment,
      path: segments.slice(0, index + 1).join('/'),
      icon: Folder,
    })),
  ]

  return (
    <nav className={clsx('flex items-center gap-1 text-sm', className)}>
      {items.map((item, index) => (
        <div key={item.path} className="flex items-center gap-1">
          {index > 0 && (
            <ChevronRight size={14} className="text-dark-500" />
          )}
          <button
            onClick={() => onNavigate(item.path)}
            className={clsx(
              'flex items-center gap-1 px-1.5 py-0.5 rounded transition-colors',
              index === items.length - 1
                ? 'text-dark-200 bg-dark-700'
                : 'text-dark-400 hover:text-dark-200 hover:bg-dark-700'
            )}
          >
            <item.icon size={12} />
            <span className="max-w-[120px] truncate">{item.label}</span>
          </button>
        </div>
      ))}
    </nav>
  )
}
