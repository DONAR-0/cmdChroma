import { createFileRoute } from '@tanstack/react-router'
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from '@/components/ui/card'

export const Route = createFileRoute('/settings')({
  component: SettingsPage,
})

function SettingsPage() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-2xl font-bold tracking-tight">Settings</h2>
        <p className="text-sm text-muted-foreground mt-1">
          Configure your cmdChroma connection
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Connection</CardTitle>
          <CardDescription>
            The UI connects to the cmdChroma chat-server API running on port 8080.
            Configure the server URL and API key using environment variables.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="space-y-3">
            <div>
              <p className="text-sm font-medium">Server URL</p>
              <p className="text-xs text-muted-foreground">
                Default: <code className="bg-muted px-1 rounded">http://localhost:8080</code>
              </p>
            </div>
            <div>
              <p className="text-sm font-medium">API Key</p>
              <p className="text-xs text-muted-foreground">
                Set via <code className="bg-muted px-1 rounded">VITE_API_KEY</code> environment variable
              </p>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
