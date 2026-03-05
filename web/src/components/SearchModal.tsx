import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  Search,
  FileText,
  FolderKanban,
  MessageSquare,
  Milestone,
  Layers,
  Paperclip,
  Loader2,
  SearchX,
} from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { useSearch } from '@/hooks/useSearch'
import { useDebounce } from '@/hooks/useDebounce'
import type { SearchResult } from '@/api/search'

const ENTITY_TYPE_ORDER = [
  'project',
  'work_item',
  'milestone',
  'queue',
  'comment',
  'attachment',
] as const

function entityIcon(type: string) {
  switch (type) {
    case 'project':
      return FolderKanban
    case 'work_item':
      return FileText
    case 'milestone':
      return Milestone
    case 'queue':
      return Layers
    case 'comment':
      return MessageSquare
    case 'attachment':
      return Paperclip
    default:
      return FileText
  }
}

interface ResultGroup {
  entityType: string
  results: SearchResult[]
}

function groupResults(results: SearchResult[]): ResultGroup[] {
  const map = new Map<string, SearchResult[]>()
  for (const r of results) {
    const list = map.get(r.entity_type)
    if (list) list.push(r)
    else map.set(r.entity_type, [r])
  }

  return ENTITY_TYPE_ORDER.filter((t) => map.has(t)).map((t) => ({
    entityType: t,
    results: map.get(t)!,
  }))
}

/** Build a flat list of selectable items from grouped results. */
function flattenGroups(groups: ResultGroup[]): SearchResult[] {
  return groups.flatMap((g) => g.results)
}

interface SearchModalProps {
  open: boolean
  onClose: () => void
}

export function SearchModal({ open, onClose }: SearchModalProps) {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const [limit, setLimit] = useState(20)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)

  const debouncedQuery = useDebounce(query, 400)
  const { results, total, isLoading, mode } = useSearch({
    query: debouncedQuery,
    limit,
  })

  const groups = useMemo(() => groupResults(results), [results])
  const flatItems = useMemo(() => flattenGroups(groups), [groups])

  // Reset state when modal opens
  useEffect(() => {
    if (open) {
      setQuery('')
      setSelectedIndex(0)
      setLimit(20)
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

  // Reset selection when results change
  useEffect(() => {
    setSelectedIndex(0)
  }, [results])

  const scrollSelectedIntoView = useCallback((index: number) => {
    const container = listRef.current
    if (!container) return
    const items = container.querySelectorAll('[data-search-item]')
    const item = items[index] as HTMLElement | undefined
    item?.scrollIntoView({ block: 'nearest' })
  }, [])

  const navigateToResult = useCallback(
    (result: SearchResult) => {
      onClose()
      // Parse snippet for display_id pattern (e.g. "TF-123: Title")
      const displayIdMatch = result.snippet.match(/^([A-Z][A-Z0-9]+-\d+):/)

      switch (result.entity_type) {
        case 'project':
          // snippet is the project name/key — we don't have the key directly,
          // but entity_id is the project UUID. Best effort: search projects.
          // For now, navigate to projects list as fallback
          navigate('/projects')
          break
        case 'work_item':
          if (displayIdMatch) {
            const [projectKey, itemNum] = displayIdMatch[1].split('-')
            navigate(`/projects/${projectKey}/items/${itemNum}`)
          }
          break
        case 'milestone':
          // Milestone snippet may not have routing info — navigate to projects
          navigate('/projects')
          break
        case 'queue':
          navigate('/projects')
          break
        case 'comment':
          // Comments reference work items — try to extract display_id
          if (displayIdMatch) {
            const [projectKey, itemNum] = displayIdMatch[1].split('-')
            navigate(`/projects/${projectKey}/items/${itemNum}`)
          }
          break
        case 'attachment':
          if (displayIdMatch) {
            const [projectKey, itemNum] = displayIdMatch[1].split('-')
            navigate(`/projects/${projectKey}/items/${itemNum}`)
          }
          break
      }
    },
    [navigate, onClose],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        const next = Math.min(selectedIndex + 1, flatItems.length - 1)
        setSelectedIndex(next)
        scrollSelectedIntoView(next)
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        const prev = Math.max(selectedIndex - 1, 0)
        setSelectedIndex(prev)
        scrollSelectedIntoView(prev)
      } else if (e.key === 'Enter' && flatItems.length > 0) {
        e.preventDefault()
        navigateToResult(flatItems[selectedIndex])
      }
    },
    [selectedIndex, flatItems, navigateToResult, scrollSelectedIntoView],
  )

  const hasMore = total > flatItems.length
  const showResults = debouncedQuery.length >= 2

  return (
    <Modal
      open={open}
      onClose={onClose}
      position="top"
      className="!max-w-2xl"
    >
      <div className="flex items-center gap-2 mb-3">
        <Search className="h-5 w-5 text-gray-400 shrink-0" />
        <input
          ref={inputRef}
          type="text"
          placeholder={t('search.placeholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          className="block w-full text-sm bg-transparent border-none outline-none text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500"
        />
        {isLoading && showResults && (
          <Loader2 className="h-4 w-4 text-gray-400 animate-spin shrink-0" />
        )}
        {mode === 'fts' && showResults && !isLoading && (
          <span className="text-xs text-gray-400 shrink-0 whitespace-nowrap">
            {t('search.ftsMode')}
          </span>
        )}
      </div>

      <div className="border-t border-gray-200 dark:border-gray-700" />

      {!showResults ? (
        <p className="text-sm text-gray-500 dark:text-gray-400 py-6 text-center">
          {t('search.hint')}
        </p>
      ) : isLoading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 text-gray-400 animate-spin" />
        </div>
      ) : flatItems.length === 0 ? (
        <div className="flex flex-col items-center py-8 gap-2">
          <SearchX className="h-8 w-8 text-gray-300 dark:text-gray-600" />
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t('search.noResults')}
          </p>
        </div>
      ) : (
        <div ref={listRef} className="max-h-80 overflow-y-auto py-2">
          {groups.map((group) => {
            const Icon = entityIcon(group.entityType)
            return (
              <div key={group.entityType} className="mb-2 last:mb-0">
                <div className="flex items-center gap-2 px-3 py-1.5">
                  <Icon className="h-3.5 w-3.5 text-gray-400" />
                  <span className="text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                    {t(`search.entityType.${group.entityType}`)}
                  </span>
                </div>
                {group.results.map((result) => {
                  const globalIndex = flatItems.indexOf(result)
                  return (
                    <button
                      key={`${result.entity_type}-${result.entity_id}`}
                      data-search-item
                      onClick={() => navigateToResult(result)}
                      onMouseEnter={() => setSelectedIndex(globalIndex)}
                      className={`w-full text-left flex items-center gap-3 px-3 py-2 rounded-md text-sm ${
                        globalIndex === selectedIndex
                          ? 'bg-indigo-50 dark:bg-indigo-900/30'
                          : 'hover:bg-gray-100 dark:hover:bg-gray-700'
                      }`}
                    >
                      <span className="text-gray-900 dark:text-gray-100 truncate flex-1">
                        {result.snippet}
                      </span>
                      {mode === 'semantic' && result.score > 0 && (
                        <span className="shrink-0 flex items-center gap-1.5">
                          <div className="w-12 h-1.5 bg-gray-200 dark:bg-gray-700 rounded-full overflow-hidden">
                            <div
                              className="h-full bg-indigo-500 rounded-full"
                              style={{
                                width: `${Math.round(result.score * 100)}%`,
                              }}
                            />
                          </div>
                          <span className="text-xs text-gray-400 w-8 text-right">
                            {Math.round(result.score * 100)}%
                          </span>
                        </span>
                      )}
                    </button>
                  )
                })}
              </div>
            )
          })}
          {hasMore && (
            <button
              onClick={() => setLimit((prev) => prev + 20)}
              className="w-full text-center py-2 text-sm text-indigo-600 dark:text-indigo-400 hover:bg-gray-50 dark:hover:bg-gray-800 rounded-md"
            >
              {t('search.showMore')}
            </button>
          )}
        </div>
      )}

      <div className="border-t border-gray-200 dark:border-gray-700 pt-2 mt-1 flex items-center justify-between">
        <div className="flex items-center gap-2 text-xs text-gray-400">
          <kbd className="inline-flex items-center justify-center min-w-[1.25rem] px-1 py-0.5 font-mono text-[10px] text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded">
            ↑↓
          </kbd>
          <span>{t('search.navigate')}</span>
          <kbd className="inline-flex items-center justify-center min-w-[1.25rem] px-1 py-0.5 font-mono text-[10px] text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded">
            ↵
          </kbd>
          <span>{t('search.open')}</span>
          <kbd className="inline-flex items-center justify-center min-w-[1.25rem] px-1 py-0.5 font-mono text-[10px] text-gray-500 dark:text-gray-400 bg-gray-100 dark:bg-gray-700 border border-gray-300 dark:border-gray-600 rounded">
            esc
          </kbd>
          <span>{t('search.dismiss')}</span>
        </div>
      </div>
    </Modal>
  )
}
