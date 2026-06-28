import { createFileRoute } from '@tanstack/react-router'
import { useState, useRef } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useChromaCollections } from '@/lib/api/hooks'
import { Card, CardContent } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Upload, Link, Loader2, CheckCircle2, XCircle, FileText } from 'lucide-react'

export const Route = createFileRoute('/import')({
  component: ImportPage,
})

interface ImportEvent {
  event: 'start' | 'progress' | 'done' | 'error'
  data: Record<string, unknown>
}

function ImportPage() {
  const queryClient = useQueryClient()
  const { data: collections } = useChromaCollections()
  const [collection, setCollection] = useState('')
  const [mode, setMode] = useState<'file' | 'url'>('file')
  const [url, setUrl] = useState('')
  const [importing, setImporting] = useState(false)
  const [events, setEvents] = useState<ImportEvent[]>([])
  const [done, setDone] = useState(false)
  const [error, setError] = useState('')
  const abortRef = useRef<AbortController | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  function addEvent(event: string, data: Record<string, unknown>) {
    setEvents((prev) => [...prev, { event, data } as ImportEvent])
  }

  async function handleFileUpload(file: File) {
    if (!collection || !file) return

    setImporting(true)
    setDone(false)
    setError('')
    setEvents([])

    const formData = new FormData()
    formData.append('file', file)

    const controller = new AbortController()
    abortRef.current = controller

    try {
      const response = await fetch(`/api/collections/${encodeURIComponent(collection)}/import`, {
        method: 'POST',
        headers: {
          'X-API-Key': import.meta.env.VITE_API_KEY || '',
        },
        body: formData,
        signal: controller.signal,
      })

      if (!response.ok) {
        const err = await response.text()
        setError(err || `HTTP ${response.status}`)
        setImporting(false)
        return
      }

      const reader = response.body?.getReader()
      if (!reader) {
        setError('No response body')
        setImporting(false)
        return
      }

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() ?? ''

        for (let i = 0; i < lines.length; i++) {
          const line = lines[i]
          if (line.startsWith('event: ')) {
            const eventType = line.slice(7).trim()
            const dataLine = lines[i + 1]
            if (dataLine?.startsWith('data: ')) {
              try {
                const data = JSON.parse(dataLine.slice(6))
                addEvent(eventType, data)
                if (eventType === 'done') {
                  setDone(true)
                  queryClient.invalidateQueries({ queryKey: ['chroma-collections'] })
                  queryClient.invalidateQueries({ queryKey: ['collections'] })
                }
                if (eventType === 'error') {
                  setError(data.error as string)
                  setImporting(false)
                  return
                }
              } catch { /* ignore parse errors */ }
            }
          }
        }
      }
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        setError((err as Error).message)
      }
    } finally {
      setImporting(false)
    }
  }

  async function handleURLImport() {
    if (!collection || !url.trim()) return

    setImporting(true)
    setDone(false)
    setError('')
    setEvents([])

    const controller = new AbortController()
    abortRef.current = controller

    try {
      const response = await fetch(`/api/collections/${encodeURIComponent(collection)}/import/url`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': import.meta.env.VITE_API_KEY || '',
        },
        body: JSON.stringify({ url: url.trim() }),
        signal: controller.signal,
      })

      if (!response.ok) {
        const err = await response.text()
        setError(err || `HTTP ${response.status}`)
        setImporting(false)
        return
      }

      const reader = response.body?.getReader()
      if (!reader) {
        setError('No response body')
        setImporting(false)
        return
      }

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() ?? ''

        for (let i = 0; i < lines.length; i++) {
          const line = lines[i]
          if (line.startsWith('event: ')) {
            const eventType = line.slice(7).trim()
            const dataLine = lines[i + 1]
            if (dataLine?.startsWith('data: ')) {
              try {
                const data = JSON.parse(dataLine.slice(6))
                addEvent(eventType, data)
                if (eventType === 'done') {
                  setDone(true)
                  queryClient.invalidateQueries({ queryKey: ['chroma-collections'] })
                  queryClient.invalidateQueries({ queryKey: ['collections'] })
                }
                if (eventType === 'error') {
                  setError(data.error as string)
                  setImporting(false)
                  return
                }
              } catch { /* ignore */ }
            }
          }
        }
      }
    } catch (err) {
      if ((err as Error).name !== 'AbortError') {
        setError((err as Error).message)
      }
    } finally {
      setImporting(false)
    }
  }

  function handleCancel() {
    abortRef.current?.abort()
    setImporting(false)
  }

  const lastProgress = events.filter((e) => e.event === 'progress').pop()?.data?.processed as number | undefined

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Import Documents</h2>
        <p className="text-sm text-muted-foreground mt-1">
          Import documents into a collection from a file or URL
        </p>
      </div>

      <Card>
        <CardContent className="pt-6 space-y-4">
          <div className="space-y-2">
            <label className="text-sm font-medium">Collection</label>
            <select
              value={collection}
              onChange={(e) => setCollection(e.target.value)}
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
              disabled={importing}
            >
              <option value="">Select collection...</option>
              {collections?.map((c) => (
                <option key={c.id} value={c.name}>{c.name}</option>
              ))}
            </select>
          </div>

          <div className="flex gap-2">
            <Button
              variant={mode === 'file' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setMode('file')}
              disabled={importing}
            >
              <FileText className="h-4 w-4 mr-1" />
              JSONL File
            </Button>
            <Button
              variant={mode === 'url' ? 'default' : 'outline'}
              size="sm"
              onClick={() => setMode('url')}
              disabled={importing}
            >
              <Link className="h-4 w-4 mr-1" />
              URL
            </Button>
          </div>

          {mode === 'file' ? (
            <div className="space-y-3">
              <div
                className="border-2 border-dashed border-border rounded-lg p-8 text-center cursor-pointer hover:border-primary/50 transition-colors"
                onClick={() => fileInputRef.current?.click()}
              >
                <Upload className="h-8 w-8 mx-auto mb-2 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">
                  Click to select a .jsonl or .parquet file
                </p>
              </div>
              <input
                ref={fileInputRef}
                type="file"
                accept=".jsonl,.parquet"
                className="hidden"
                disabled={importing || !collection}
                onChange={(e) => {
                  const file = e.target.files?.[0]
                  if (file) handleFileUpload(file)
                }}
              />
            </div>
          ) : (
            <div className="space-y-3">
              <div className="flex gap-2">
                <input
                  value={url}
                  onChange={(e) => setUrl(e.target.value)}
                  placeholder="https://huggingface.co/datasets/.../resolve/main/data.jsonl"
                  className="flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
                  disabled={importing}
                />
                <Button
                  onClick={handleURLImport}
                  disabled={importing || !url.trim() || !collection}
                >
                  {importing ? <Loader2 className="h-4 w-4 animate-spin" /> : <Link className="h-4 w-4" />}
                  Import
                </Button>
              </div>
              <p className="text-xs text-muted-foreground">
                Paste a direct download link to a .jsonl or .parquet file
              </p>
            </div>
          )}

          {importing && (
            <div className="flex items-center gap-2">
              <Loader2 className="h-4 w-4 animate-spin" />
              <span className="text-sm text-muted-foreground">
                Importing{lastProgress ? ` (${lastProgress} documents processed)` : '...'}
              </span>
              <Button variant="outline" size="sm" onClick={handleCancel}>
                Cancel
              </Button>
            </div>
          )}

          {error && (
            <div className="flex items-start gap-2 text-destructive">
              <XCircle className="h-4 w-4 mt-0.5 shrink-0" />
              <p className="text-sm">{error}</p>
            </div>
          )}

          {done && !error && (
            <div className="flex items-center gap-2 text-emerald-600">
              <CheckCircle2 className="h-4 w-4" />
              <span className="text-sm font-medium">
                Import completed — {lastProgress ?? 0} documents imported
              </span>
            </div>
          )}

          {events.length > 0 && (
            <div className="space-y-1">
              <p className="text-xs font-medium text-muted-foreground">Log</p>
              <div className="max-h-40 overflow-y-auto space-y-1 bg-muted rounded-md p-3">
                {events.map((ev, i) => (
                  <div key={i} className="flex items-center gap-2 text-xs">
                    <Badge variant="outline" className="text-[10px] px-1 py-0">
                      {ev.event}
                    </Badge>
                    <span className="text-muted-foreground">
                      {JSON.stringify(ev.data)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
