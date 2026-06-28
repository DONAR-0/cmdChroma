import { Link, useLocation } from '@tanstack/react-router'
import { cn } from '@/lib/utils'
import { useAppStore } from '@/lib/store/app'
import {
  LayoutDashboard,
  Database,
  Search,
  MessageSquare,
  Upload,
  Settings,
  ChevronLeft,
} from 'lucide-react'

const navItems = [
  { to: '/', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/collections', label: 'Collections', icon: Database },
  { to: '/import', label: 'Import', icon: Upload },
  { to: '/query', label: 'Semantic Search', icon: Search },
  { to: '/chat', label: 'RAG Chat', icon: MessageSquare },
  { to: '/settings', label: 'Settings', icon: Settings },
]

export function Sidebar() {
  const { sidebarOpen, toggleSidebar } = useAppStore()
  const location = useLocation()

  return (
    <aside
      className={cn(
        'flex flex-col border-r border-border bg-background transition-all duration-200',
        sidebarOpen ? 'w-56' : 'w-12',
      )}
    >
      <div className="flex h-14 items-center justify-between border-b border-border px-3">
        {sidebarOpen && (
          <span className="font-semibold text-sm tracking-tight">cmdChroma</span>
        )}
        <button
          onClick={toggleSidebar}
          className="rounded-md p-1.5 hover:bg-accent text-muted-foreground"
        >
          <ChevronLeft className={cn('h-4 w-4 transition-transform', !sidebarOpen && 'rotate-180')} />
        </button>
      </div>

      <nav className="flex-1 space-y-1 p-2">
        {navItems.map((item) => {
          const isActive = location.pathname === item.to ||
            (item.to !== '/' && location.pathname.startsWith(item.to))
          return (
            <Link
              key={item.to}
              to={item.to}
              className={cn(
                'flex items-center gap-3 rounded-md px-2 py-2 text-sm transition-colors',
                isActive
                  ? 'bg-accent text-accent-foreground font-medium'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground',
              )}
            >
              <item.icon className="h-4 w-4 shrink-0" />
              {sidebarOpen && <span>{item.label}</span>}
            </Link>
          )
        })}
      </nav>
    </aside>
  )
}
