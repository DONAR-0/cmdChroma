import { useHealth } from '@/lib/api/hooks'
import { cn } from '@/lib/utils'

export function Header() {
  const { data: health } = useHealth()

  const statusColor = health?.status === 'ok' ? 'bg-emerald-500' : 'bg-red-500'

  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-background px-6">
      <div className="flex items-center gap-3">
        <h1 className="text-sm font-medium text-foreground">cmdChroma Web UI</h1>
      </div>

      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2">
          <span className={cn('h-2 w-2 rounded-full', statusColor)} />
          <span className="text-xs text-muted-foreground">
            {health?.status === 'ok' ? 'Connected' : health ? health.status : 'Checking...'}
          </span>
        </div>
        {health && (
          <>
            <span className="text-xs text-muted-foreground">|</span>
            <span className="text-xs text-muted-foreground">Chroma: {health.chroma}</span>
            <span className="text-xs text-muted-foreground">|</span>
            <span className="text-xs text-muted-foreground">Embedder: {health.embedder}</span>
          </>
        )}
      </div>
    </header>
  )
}
