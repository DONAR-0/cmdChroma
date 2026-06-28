import { createFileRoute, Link } from '@tanstack/react-router'
import { useState } from 'react'
import { useChromaCollections, useCreateCollection, useDeleteCollection } from '@/lib/api/hooks'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Database, Plus, Trash2, Loader2 } from 'lucide-react'

export const Route = createFileRoute('/collections')({
  component: CollectionsPage,
})

function CollectionsPage() {
  const { data: collections, isLoading } = useChromaCollections()
  const createMutation = useCreateCollection()
  const deleteMutation = useDeleteCollection()
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)

  function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    if (!newName.trim()) return
    createMutation.mutate(newName.trim(), {
      onSuccess: () => {
        setNewName('')
        setShowCreate(false)
      },
    })
  }

  function handleDelete(name: string) {
    deleteMutation.mutate(name)
    setDeleteConfirm(null)
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold tracking-tight">Collections</h2>
          <p className="text-sm text-muted-foreground mt-1">
            Browse and manage your ChromaDB collections
          </p>
        </div>
        <Button onClick={() => setShowCreate(true)}>
          <Plus className="h-4 w-4 mr-1" />
          Create
        </Button>
      </div>

      {showCreate && (
        <Card>
          <CardContent className="pt-6">
            <form onSubmit={handleCreate} className="flex gap-3">
              <input
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                placeholder="Collection name"
                className="flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
                autoFocus
              />
              <Button type="submit" disabled={!newName.trim() || createMutation.isPending}>
                {createMutation.isPending && <Loader2 className="h-4 w-4 animate-spin mr-1" />}
                Create
              </Button>
              <Button type="button" variant="outline" onClick={() => setShowCreate(false)}>
                Cancel
              </Button>
            </form>
            {createMutation.isError && (
              <p className="text-xs text-destructive mt-2">{createMutation.error.message}</p>
            )}
          </CardContent>
        </Card>
      )}

      {isLoading ? (
        <div className="flex items-center justify-center py-12">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </div>
      ) : !collections || collections.length === 0 ? (
        <Card>
          <CardContent className="flex flex-col items-center justify-center py-12">
            <Database className="h-12 w-12 text-muted-foreground mb-4" />
            <p className="text-sm text-muted-foreground">No collections found</p>
            <p className="text-xs text-muted-foreground mt-1">
              Create one to get started
            </p>
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {collections.map((c) => (
            <Card key={c.id} className="hover:shadow-md transition-shadow relative group">
              <Link to={`/collections/$name`} params={{ name: c.name }} className="block">
                <CardHeader>
                  <div className="flex items-start justify-between">
                    <CardTitle className="text-base">{c.name}</CardTitle>
                    <div className="flex items-center gap-2">
                      <Badge variant="secondary">{c.count} docs</Badge>
                      {c.dimension && (
                        <Badge variant="outline">{c.dimension}d</Badge>
                      )}
                    </div>
                  </div>
                  {c.metadata && (
                    <CardDescription>
                      {Object.entries(c.metadata).map(([k, v]) => (
                        <span key={k} className="mr-2 text-xs">{k}: {String(v)}</span>
                      ))}
                    </CardDescription>
                  )}
                </CardHeader>
              </Link>
              <div className="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                {deleteConfirm === c.name ? (
                  <div className="flex gap-1">
                    <Button
                      size="sm"
                      variant="destructive"
                      onClick={() => handleDelete(c.name)}
                      disabled={deleteMutation.isPending}
                    >
                      Confirm
                    </Button>
                    <Button size="sm" variant="outline" onClick={() => setDeleteConfirm(null)}>
                      Cancel
                    </Button>
                  </div>
                ) : (
                  <Button
                    size="sm"
                    variant="ghost"
                    onClick={(e) => { e.preventDefault(); setDeleteConfirm(c.name) }}
                  >
                    <Trash2 className="h-4 w-4 text-muted-foreground hover:text-destructive" />
                  </Button>
                )}
              </div>
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
