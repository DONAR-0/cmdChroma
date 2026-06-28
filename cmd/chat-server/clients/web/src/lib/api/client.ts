import ky from 'ky'
import type {
  HealthResponse,
  CollectionItem,
  ChromaCollection,
  DocumentRecord,
  QueryResponse,
  QueryDocument,
  SessionItem,
} from './types'

class ApiClient {
  private client: typeof ky
  private apiKey: string

  constructor(baseUrl: string, apiKey?: string) {
    this.apiKey = apiKey || ''
    const prefix = baseUrl.endsWith('/api') ? baseUrl : `${baseUrl}/api`
    this.client = ky.create({
      prefix,
      headers: this.apiKey ? { 'X-API-Key': this.apiKey } : {},
      timeout: 30000,
    })
  }

  async health(): Promise<HealthResponse> {
    const base = import.meta.env.VITE_API_URL || ''
    const res = await fetch(`${base}/health`)
    if (!res.ok) throw new Error(`health check failed: ${res.status}`)
    return res.json()
  }

  async collections(): Promise<CollectionItem[]> {
    const res = await this.client.get('collections').json<{ collections: CollectionItem[] }>()
    return res.collections
  }

  async listChromaCollections(): Promise<ChromaCollection[]> {
    const res = await this.client.get('collections').json<{ collections: ChromaCollection[] }>()
    return res.collections
  }

  async createCollection(name: string): Promise<{ id: string; name: string }> {
    return this.client.post('collections', { json: { name } }).json()
  }

  async deleteCollection(name: string): Promise<void> {
    await this.client.delete(`collections/${encodeURIComponent(name)}`)
  }

  async listDocuments(name: string): Promise<DocumentRecord[]> {
    const res = await this.client.get(`collections/${encodeURIComponent(name)}/documents`).json<{ documents: DocumentRecord[]; count: number }>()
    return res.documents
  }

  async addDocuments(name: string, documents: string[], ids?: string[], metadatas?: Record<string, unknown>[]): Promise<void> {
    await this.client.post(`collections/${encodeURIComponent(name)}/documents`, {
      json: { documents, ids: ids ?? [], metadatas },
    })
  }

  async deleteDocuments(name: string, ids: string[]): Promise<void> {
    await this.client.delete(`collections/${encodeURIComponent(name)}/documents`, {
      json: { ids },
    })
  }

  async query(collection: string, query: string, nResults = 5, distanceThreshold = 0): Promise<QueryDocument[]> {
    const res = await this.client.post('query', {
      json: { collection, query, n_results: nResults, distance_threshold: distanceThreshold },
    }).json<QueryResponse>()
    return res.documents
  }

  async sessions(): Promise<SessionItem[]> {
    const res = await this.client.get('sessions').json<{ sessions: SessionItem[] }>()
    return res.sessions
  }

  async clearSession(id: string): Promise<void> {
    await this.client.delete(`sessions/${id}`)
  }

  chatStream(
    collection: string,
    message: string,
    options: {
      model?: string
      session_id?: string
      n_results?: number
      distance_threshold?: number
      onToken: (token: string) => void
      onDone: (totalTokens: number) => void
      onError: (error: string) => void
    },
  ): AbortController {
    const controller = new AbortController()

    const body = JSON.stringify({
      collection,
      message,
      model: options.model,
      session_id: options.session_id,
      n_results: options.n_results ?? 3,
      distance_threshold: options.distance_threshold ?? 0,
    })

    fetch(`/api/chat`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-API-Key': this.apiKey,
      },
      body,
      signal: controller.signal,
    }).then(async (response) => {
      if (!response.ok) {
        const err = await response.text()
        options.onError(err || `HTTP ${response.status}`)
        return
      }

      const reader = response.body?.getReader()
      if (!reader) {
        options.onError('No response body')
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

        for (const line of lines) {
          if (line.startsWith('event: ')) {
          } else if (line.startsWith('data: ')) {
            const data = line.slice(6)
            try {
              const parsed = JSON.parse(data)
              if ('content' in parsed) {
                options.onToken(parsed.content)
              } else if ('total_tokens' in parsed) {
                options.onDone(parsed.total_tokens)
              } else if ('error' in parsed) {
                options.onError(parsed.error)
              }
            } catch {
              // ignore parse errors on partial lines
            }
          }
        }
      }
    }).catch((err) => {
      if (err.name !== 'AbortError') {
        options.onError(err.message)
      }
    })

    return controller
  }
}

const apiUrl = import.meta.env.VITE_API_URL || ''
const apiKey = import.meta.env.VITE_API_KEY || ''
export const api = new ApiClient(apiUrl ? `${apiUrl}/api` : '/api', apiKey)
