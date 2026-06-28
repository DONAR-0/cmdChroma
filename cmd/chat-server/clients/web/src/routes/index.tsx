import { createFileRoute } from '@tanstack/react-router'
import { useHealth, useCollections, useSessions } from '@/lib/api/hooks'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Database, MessageSquare, Activity } from 'lucide-react'

export const Route = createFileRoute('/')({
  component: Dashboard,
})

function Dashboard() {
  const { data: health, isLoading: healthLoading } = useHealth()
  const { data: collections } = useCollections()
  const { data: sessions } = useSessions()

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Dashboard</h2>
        <p className="text-sm text-muted-foreground mt-1">
          Overview of your cmdChroma server
        </p>
      </div>

      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Server Status</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            {healthLoading ? (
              <p className="text-sm text-muted-foreground">Checking...</p>
            ) : health ? (
              <div className="space-y-2">
                <div className="flex items-center gap-2">
                  <Badge variant={health.status === 'ok' ? 'success' : 'destructive'}>
                    {health.status}
                  </Badge>
                </div>
                <div className="text-xs text-muted-foreground space-y-1">
                  <p>ChromaDB: {health.chroma}</p>
                  <p>Embedder: {health.embedder}</p>
                </div>
              </div>
            ) : (
              <p className="text-sm text-destructive">Unable to connect</p>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Collections</CardTitle>
            <Database className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold">{collections?.length ?? '-'}</p>
            <p className="text-xs text-muted-foreground mt-1">
              Available collections
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Active Sessions</CardTitle>
            <MessageSquare className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold">{sessions?.length ?? '-'}</p>
            <p className="text-xs text-muted-foreground mt-1">
              RAG chat sessions
            </p>
          </CardContent>
        </Card>
      </div>

      {collections && collections.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle>Collections</CardTitle>
            <CardDescription>Quick overview of your collections</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-2">
              {collections.map((c) => (
                <div key={c.name} className="flex items-center justify-between rounded-md border border-border p-3">
                  <div>
                    <p className="text-sm font-medium">{c.name}</p>
                    {c.description && (
                      <p className="text-xs text-muted-foreground">{c.description}</p>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
