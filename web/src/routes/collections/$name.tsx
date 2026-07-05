import { createFileRoute } from '@tanstack/react-router'
import { useState } from 'react'
import { useDocuments, useAddDocuments, useDeleteDocuments, useChromaCollections } from '@/lib/api/hooks'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ArrowLeft, Plus, Trash2, FileText, Loader2 } from 'lucide-react'

export const Route = createFileRoute('/collections/$name')({
  component: CollectionDetailPage,
})

function CollectionDetailPage() {
  const { name } = Route.useParams()
  const { data: documents, isLoading } = useDocuments(name)
  const addMutation = useAddDocuments(name)
  const deleteMutation = useDeleteDocuments(name)
  const { data: collections } = useChromaCollections()
  const collection = collections?.find((c) => c.name === name)

  const [showAdd, setShowAdd] = useState(false)
  const [newDocs, setNewDocs] = useState('')
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set())
  const [expandedId, setExpandedId] = useState<string | null>(null)

  function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    const docs = newDocs.split('\n').filter(Boolean)
    if (docs.length === 0) return
    addMutation.mutate(
      { documents: docs },
      {
        onSuccess: () => {
          setNewDocs('')
          setShowAdd(false)
        },
      },
    )
  }

  function handleDeleteSelected() {
    if (selectedIds.size === 0) return
    deleteMutation.mutate(Array.from(selectedIds), {
      onSuccess: () => setSelectedIds(new Set()),
    })
  }

  function toggleSelect(id: string) {
    setSelectedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  function selectAll() {
    if (!documents) return
    if (selectedIds.size === documents.length) {
      setSelectedIds(new Set())
    } else {
      setSelectedIds(new Set(documents.map((d) => d.id)))
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-4">
        <a
          href="/collections"
          onClick={(e) => { e.preventDefault(); window.history.back() }}
          className="text-muted-foreground hover:text-foreground"
        >
          <ArrowLeft className="h-5 w-5" />
        </a>
        <div className="flex-1">
          <div className="flex items-center gap-3">
            <h2 className="text-2xl font-bold tracking-tight">{name}</h2>
            {collection?.dimension && (
              <Badge variant="secondary">{collection.dimension}d vectors</Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground mt-1">
            {documents?.length ?? 0} document{documents?.length !== 1 ? 's' : ''}
          </p>
        </div>
        <div className="flex gap-2">
          {selectedIds.size > 0 && (
            <Button variant="destructive" size="sm" onClick={handleDeleteSelected} disabled={deleteMutation.isPending}>
              <Trash2 className="h-4 w-4 mr-1" />
              Delete ({selectedIds.size})
            </Button>
          )}
          <Button size="sm" onClick={() => setShowAdd(true)}>
            <Plus className="h-4 w-4 mr-1" />
            Add Documents
          </Button>
        </div>
      </div>

      {showAdd && (
        <Card>
          <CardHeader>
            <CardTitle className="text-base">Add Documents</CardTitle>
            <CardDescription>One document per line</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleAdd} className="space-y-3">
              <textarea
                value={newDocs}
                onChange={(e) => setNewDocs(e.target.value)}
                placeholder={`Document 1\nDocument 2\nDocument 3`}
                rows={6}
                className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm resize-none font-mono"
              />
              <div className="flex gap-2">
                <Button type="submit" disabled={!newDocs.trim() || addMutation.isPending}>
                  {addMutation.isPending && <Loader2 className="h-4 w-4 animate-spin mr-1" />}
                  Add {newDocs.split('\n').filter(Boolean).length} Documents
                </Button>
                <Button type="button" variant="outline" onClick={() => setShowAdd(false)}>
                  Cancel
                </Button>
              </div>
              {addMutation.isError && (
                <p className="text-xs text-destructive">{addMutation.error.message}</p>
              )}
            </form>
          </CardContent>
        </Card>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : !documents || documents.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <FileText className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-sm text-muted-foreground">No documents in this collection</p>
            <p className="text-xs text-muted-foreground mt-1">
              Add documents using the button above
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-2">
          <div className="flex items-center gap-2 px-1">
            <input
              type="checkbox"
              checked={documents.length > 0 && selectedIds.size === documents.length}
              onChange={selectAll}
              className="rounded border-border"
            />
            <span className="text-xs text-muted-foreground">
              {selectedIds.size > 0
                ? `${selectedIds.size} selected`
                : `${documents.length} documents`}
            </span>
          </div>

          {documents.map((doc) => (
            <Card key={doc.id} className="hover:shadow-sm transition-shadow">
              <CardContent className="p-4">
                <div className="flex items-start gap-3">
                  <input
                    type="checkbox"
                    checked={selectedIds.has(doc.id)}
                    onChange={() => toggleSelect(doc.id)}
                    className="mt-1 rounded border-border"
                  />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <Badge variant="outline" className="text-xs font-mono max-w-[200px] truncate">
                        {doc.id}
                      </Badge>
                      <button
                        onClick={() => setExpandedId(expandedId === doc.id ? null : doc.id)}
                        className="text-xs text-muted-foreground hover:text-foreground"
                      >
                        {expandedId === doc.id ? 'Collapse' : 'Expand'}
                      </button>
                    </div>
                    <p className={`text-sm whitespace-pre-wrap ${expandedId === doc.id ? '' : 'line-clamp-2'}`}>
                      {doc.content}
                    </p>
                    {doc.metadata && Object.keys(doc.metadata).length > 0 && (
                      <div className="mt-2 flex flex-wrap gap-1">
                        {Object.entries(doc.metadata).map(([k, v]) => (
                          <Badge key={k} variant="secondary" className="text-xs">
                            {k}: {String(v)}
                          </Badge>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
