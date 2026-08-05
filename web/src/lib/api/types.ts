export interface HealthResponse {
  status: string
  chroma: string
  embedder: string
}

export interface CollectionItem {
  name: string
  description: string
}

export interface ChromaCollection {
  id: string
  name: string
  tenant: string
  database: string
  metadata: Record<string, unknown> | null
  dimension: number | null
  configuration_json: Record<string, unknown> | null
  count: number
}

export interface DocumentRecord {
  id: string
  content: string
  metadata: Record<string, unknown> | null
}

export interface QueryDocument {
  id: string
  content: string
  distance: number
  metadata: Record<string, unknown> | null
}

export interface QueryResponse {
  documents: QueryDocument[]
  count: number
}

export interface ChatRequest {
  collection: string
  message: string
  model?: string
  session_id?: string
  n_results?: number
  distance_threshold?: number
}

export interface ChatTokenEvent {
  content: string
}

export interface ChatDoneEvent {
  total_tokens: number
}

export interface ChatErrorEvent {
  error: string
}

export interface SessionItem {
  id: string
  collection: string
  message_count: number
  created_at: string
}
