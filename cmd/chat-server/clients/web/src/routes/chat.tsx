import { createFileRoute } from '@tanstack/react-router'
import { useState, useRef, useEffect } from 'react'
import { useCollections, useSessions } from '@/lib/api/hooks'
import { api } from '@/lib/api/client'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Send, MessageSquare, Trash2 } from 'lucide-react'

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
}

export const Route = createFileRoute('/chat')({
  component: ChatPage,
})

function ChatPage() {
  const { data: collections } = useCollections()
  const { data: sessions, refetch: refetchSessions } = useSessions()
  const [collection, setCollection] = useState('')
  const [model, setModel] = useState('')
  const [sessionId, setSessionId] = useState('')
  const [input, setInput] = useState('')
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [streaming, setStreaming] = useState(false)
  const [error, setError] = useState('')
  const abortRef = useRef<AbortController | null>(null)
  const messagesEndRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

  async function handleSend(e: React.FormEvent) {
    e.preventDefault()
    if (!input.trim() || !collection || streaming) return

    const userMessage = input.trim()
    setInput('')
    setMessages((prev) => [...prev, { role: 'user', content: userMessage }])
    setStreaming(true)
    setError('')

    const assistantMessage = { role: 'assistant' as const, content: '' }
    setMessages((prev) => [...prev, assistantMessage])

    const controller = api.chatStream(collection, userMessage, {
      model: model || undefined,
      session_id: sessionId || undefined,
      n_results: 3,
      distance_threshold: 0,
      onToken: (token) => {
        setMessages((prev) => {
          const updated = [...prev]
          const last = updated[updated.length - 1]
          if (last.role === 'assistant') {
            updated[updated.length - 1] = { ...last, content: last.content + token }
          }
          return updated
        })
      },
      onDone: () => {
        setStreaming(false)
        refetchSessions()
      },
      onError: (err) => {
        setError(err)
        setStreaming(false)
      },
    })
    abortRef.current = controller
  }

  function handleStop() {
    abortRef.current?.abort()
    setStreaming(false)
  }

  function handleNewSession() {
    setSessionId('')
    setMessages([])
    setError('')
  }

  async function handleClearSession(id: string) {
    await api.clearSession(id)
    refetchSessions()
    if (sessionId === id) {
      setSessionId('')
      setMessages([])
    }
  }

  return (
    <div className="flex h-full gap-6">
      <div className="flex-1 flex flex-col">
        <div className="mb-4">
          <h2 className="text-2xl font-bold tracking-tight">RAG Chat</h2>
          <p className="text-sm text-muted-foreground mt-1">
            Chat with your documents using RAG
          </p>
        </div>

        <div className="flex gap-4 mb-4">
          <div className="flex-1">
            <label className="text-xs font-medium text-muted-foreground">Collection</label>
            <select
              value={collection}
              onChange={(e) => setCollection(e.target.value)}
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm mt-1"
            >
              <option value="">Select...</option>
              {collections?.map((c) => (
                <option key={c.name} value={c.name}>{c.name}</option>
              ))}
            </select>
          </div>
          <div className="flex-1">
            <label className="text-xs font-medium text-muted-foreground">Model (optional)</label>
            <input
              value={model}
              onChange={(e) => setModel(e.target.value)}
              placeholder="e.g. qwen:0.5b or nim://..."
              className="w-full rounded-md border border-border bg-background px-3 py-2 text-sm mt-1"
            />
          </div>
          <div className="flex items-end">
            <Button variant="outline" size="sm" onClick={handleNewSession}>
              <MessageSquare className="h-4 w-4 mr-1" />
              New Chat
            </Button>
          </div>
        </div>

        <Card className="flex-1 flex flex-col">
          <CardContent className="flex-1 flex flex-col p-0">
            <div className="flex-1 overflow-y-auto p-4 space-y-4">
              {messages.length === 0 && (
                <div className="flex flex-col items-center justify-center h-full text-muted-foreground">
                  <MessageSquare className="h-12 w-12 mb-2" />
                  <p className="text-sm">Select a collection and start chatting</p>
                </div>
              )}
              {messages.map((msg, i) => (
                <div key={i} className={`flex ${msg.role === 'user' ? 'justify-end' : 'justify-start'}`}>
                  <div
                    className={`max-w-[80%] rounded-lg px-4 py-2 text-sm ${
                      msg.role === 'user'
                        ? 'bg-primary text-primary-foreground'
                        : 'bg-muted text-muted-foreground'
                    }`}
                  >
                    <p className="whitespace-pre-wrap">{msg.content || (i === messages.length - 1 && streaming ? '...' : '')}</p>
                  </div>
                </div>
              ))}
              <div ref={messagesEndRef} />
            </div>

            <div className="border-t border-border p-4">
              {error && (
                <p className="text-xs text-destructive mb-2">{error}</p>
              )}
              <form onSubmit={handleSend} className="flex gap-2">
                <input
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  placeholder={collection ? 'Ask a question...' : 'Select a collection first'}
                  disabled={!collection || streaming}
                  className="flex-1 rounded-md border border-border bg-background px-3 py-2 text-sm"
                />
                {streaming ? (
                  <Button type="button" variant="destructive" size="sm" onClick={handleStop}>
                    Stop
                  </Button>
                ) : (
                  <Button type="submit" disabled={!collection || !input.trim()}>
                    <Send className="h-4 w-4" />
                  </Button>
                )}
              </form>
            </div>
          </CardContent>
        </Card>
      </div>

      <div className="w-64 shrink-0">
        <h3 className="text-sm font-medium mb-3">Sessions</h3>
        {sessions && sessions.length > 0 ? (
          <div className="space-y-2">
            {sessions.map((s) => (
              <Card
                key={s.id}
                className={`cursor-pointer transition-colors ${
                  sessionId === s.id ? 'ring-1 ring-primary' : ''
                }`}
                onClick={() => setSessionId(s.id)}
              >
                <CardContent className="p-3">
                  <div className="flex items-center justify-between">
                    <div className="flex-1 min-w-0">
                      <p className="text-xs font-medium truncate">{s.collection}</p>
                      <p className="text-xs text-muted-foreground">{s.message_count} messages</p>
                    </div>
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        handleClearSession(s.id)
                      }}
                      className="text-muted-foreground hover:text-destructive ml-2"
                    >
                      <Trash2 className="h-3 w-3" />
                    </button>
                  </div>
                </CardContent>
              </Card>
            ))}
          </div>
        ) : (
          <p className="text-xs text-muted-foreground">No active sessions</p>
        )}
      </div>
    </div>
  )
}
