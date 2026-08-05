import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useCollections } from '@/lib/api/hooks'
import { api } from '@/lib/api/client'
import type { QueryDocument } from '@/lib/api/types'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Search, Loader2 } from 'lucide-react'

export const Route = createFileRoute('/query')({
  component: QueryPage,
})

function QueryPage() {
  const { data: collections } = useCollections()
  const [collection, setCollection] = useState('')
  const [query, setQuery] = useState('')
  const [nResults, setNResults] = useState(5)
  const [results, setResults] = useState<QueryDocument[] | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    if (!collection || !query) return

    setLoading(true)
    setError('')
    setResults(null)

    try {
      const docs = await api.query(collection, query, nResults)
      setResults(docs)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Query failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Semantic Search</h2>
        <p className="text-sm text-muted-foreground mt-1">
          Search documents using vector similarity
        </p>
      </div>

      <Card>
        <CardContent className="pt-6">
          <form onSubmit={handleSearch} className="space-y-4">
            <div className="flex gap-4">
              <div className="flex-1 space-y-2">
                <label className="text-sm font-medium">Collection</label>
                <select
                  value={collection}
                  onChange={(e) => setCollection(e.target.value)}
                  className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
                >
                  <option value="">Select collection...</option>
                  {collections?.map((c) => (
                    <option key={c.name} value={c.name}>{c.name}</option>
                  ))}
                </select>
              </div>
              <div className="w-32 space-y-2">
                <label className="text-sm font-medium">Results</label>
                <input
                  type="number"
                  min={1}
                  max={50}
                  value={nResults}
                  onChange={(e) => setNResults(Number(e.target.value))}
                  className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm"
                />
              </div>
            </div>

            <div className="space-y-2">
              <label className="text-sm font-medium">Query</label>
              <textarea
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                placeholder="What are you looking for?"
                rows={3}
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm resize-none"
              />
            </div>

            <Button type="submit" disabled={loading || !collection || !query}>
              {loading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Search className="h-4 w-4" />
              )}
              Search
            </Button>
          </form>
        </CardContent>
      </Card>

      {error && (
        <Card>
          <CardContent className="pt-6">
            <p className="text-sm text-destructive">{error}</p>
          </CardContent>
        </Card>
      )}

      {results && (
        <div className="space-y-3">
          <p className="text-sm text-muted-foreground">
            Found {results.length} result{results.length !== 1 ? 's' : ''}
          </p>
          {results.map((doc, i) => (
            <Card key={doc.id + i}>
              <CardContent className="pt-6">
                <div className="flex items-start justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <Badge variant="secondary">#{i + 1}</Badge>
                    <span className="text-xs text-muted-foreground">ID: {doc.id}</span>
                  </div>
                  <div className="flex items-center gap-1">
                    <div
                      className="h-2 rounded-full bg-primary"
                      style={{
                        width: `${Math.max(2, (1 - doc.distance) * 60)}px`,
                        opacity: 1 - doc.distance,
                      }}
                    />
                    <span className="text-xs text-muted-foreground">
                      {(doc.distance * 100).toFixed(1)}%
                    </span>
                  </div>
                </div>
                <p className="text-sm whitespace-pre-wrap line-clamp-4">{doc.content}</p>
                {doc.metadata && Object.keys(doc.metadata).length > 0 && (
                  <div className="mt-2 flex flex-wrap gap-1">
                    {Object.entries(doc.metadata).map(([k, v]) => (
                      <Badge key={k} variant="outline" className="text-xs">
                        {k}: {String(v)}
                      </Badge>
                    ))}
                  </div>
                )}
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
