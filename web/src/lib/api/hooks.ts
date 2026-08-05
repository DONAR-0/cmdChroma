import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { api } from './client'

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: () => api.health(),
    refetchInterval: 30000,
  })
}

export function useCollections() {
  return useQuery({
    queryKey: ['collections'],
    queryFn: () => api.collections(),
  })
}

export function useChromaCollections() {
  return useQuery({
    queryKey: ['chroma-collections'],
    queryFn: () => api.listChromaCollections(),
  })
}

export function useCreateCollection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api.createCollection(name),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chroma-collections'] })
      qc.invalidateQueries({ queryKey: ['collections'] })
    },
  })
}

export function useDeleteCollection() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (name: string) => api.deleteCollection(name),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['chroma-collections'] })
      qc.invalidateQueries({ queryKey: ['collections'] })
    },
  })
}

export function useDocuments(collectionName: string | undefined) {
  return useQuery({
    queryKey: ['documents', collectionName],
    queryFn: () => api.listDocuments(collectionName!),
    enabled: !!collectionName,
  })
}

export function useAddDocuments(collectionName: string | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      documents,
      ids,
      metadatas,
    }: {
      documents: string[]
      ids?: string[]
      metadatas?: Record<string, unknown>[]
    }) => api.addDocuments(collectionName!, documents, ids, metadatas),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', collectionName] })
    },
  })
}

export function useDeleteDocuments(collectionName: string | undefined) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (ids: string[]) => api.deleteDocuments(collectionName!, ids),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', collectionName] })
    },
  })
}

export function useSessions() {
  return useQuery({
    queryKey: ['sessions'],
    queryFn: () => api.sessions(),
    refetchInterval: 10000,
  })
}
