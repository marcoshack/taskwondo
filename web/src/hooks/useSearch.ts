import { useState, useEffect, useRef, useCallback } from 'react'
import { unifiedSearch } from '@/api/search'
import type { SearchResult } from '@/api/search'

interface UseSearchOptions {
  query: string
  limit?: number
}

interface UseSearchReturn {
  ftsResults: SearchResult[]
  semanticResults: SearchResult[]
  semanticAvailable: boolean
  semanticStatus: 'pending' | 'complete' | 'error'
  semanticError: string | null
  isLoading: boolean
}

export function useSearch({ query, limit = 20 }: UseSearchOptions): UseSearchReturn {
  const [ftsResults, setFtsResults] = useState<SearchResult[]>([])
  const [semanticResults, setSemanticResults] = useState<SearchResult[]>([])
  const [semanticAvailable, setSemanticAvailable] = useState(false)
  const [semanticStatus, setSemanticStatus] = useState<'pending' | 'complete' | 'error'>('complete')
  const [semanticError, setSemanticError] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const abortRef = useRef<AbortController | null>(null)

  const resetState = useCallback(() => {
    setFtsResults([])
    setSemanticResults([])
    setSemanticAvailable(false)
    setSemanticStatus('complete')
    setSemanticError(null)
    setIsLoading(false)
  }, [])

  useEffect(() => {
    abortRef.current?.abort()

    if (query.length < 2) {
      resetState()
      return
    }

    const controller = new AbortController()
    abortRef.current = controller

    setFtsResults([])
    setSemanticResults([])
    setSemanticError(null)
    setIsLoading(true)

    unifiedSearch(query, { limit, signal: controller.signal })
      .then((data) => {
        setFtsResults(data.fts.results ?? [])
        setSemanticAvailable(data.semantic.available)

        if (data.semantic.status === 'error') {
          setSemanticError('Semantic search is temporarily unavailable')
          setSemanticStatus('error')
        } else {
          setSemanticResults(data.semantic.results ?? [])
          setSemanticStatus('complete')
        }
        setIsLoading(false)
      })
      .catch((err) => {
        if (err.name !== 'AbortError' && err.name !== 'CanceledError') {
          setIsLoading(false)
        }
      })

    return () => {
      controller.abort()
    }
  }, [query, limit, resetState])

  return {
    ftsResults,
    semanticResults,
    semanticAvailable,
    semanticStatus,
    semanticError,
    isLoading,
  }
}
