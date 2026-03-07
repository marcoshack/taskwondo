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
  AlertCircle,
  FlaskConical,
} from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { TypeBadge } from '@/components/workitems/TypeBadge'
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

/** Extract a clean display line from the indexed content. */
function parseSnippet(result: SearchResult): { type?: string; text: string } {
  const s = result.snippet
  // Work items: "[task] Title\n\nDescription..."
  if (result.entity_type === 'work_item') {
    const m = s.match(/^\[(\w+)]\s*(.*)/)
    if (m) return { type: m[1], text: m[2].split('\n')[0] }
  }
  // Comments: "Comment:\n\nBody..."
  if (result.entity_type === 'comment') {
    const body = s.replace(/^Comment:\s*\n*/, '')
    return { text: body.split('\n')[0] }
  }
  // Attachments: "Attachment filename\nComment: ..."
  if (result.entity_type === 'attachment') {
    const line = s.replace(/^Attachment\s+/, '')
    return { text: line.split('\n')[0] }
  }
  // Everything else: first line
  return { text: s.split('\n')[0] }
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
  const {
    ftsResults,
    semanticResults,
    semanticAvailable,
    semanticStatus,
    semanticError,
    isLoading,
  } = useSearch({ query: debouncedQuery, limit })

  const ftsGroups = useMemo(() => groupResults(ftsResults), [ftsResults])
  const semanticGroups = useMemo(() => groupResults(semanticResults), [semanticResults])
  const flatFts = useMemo(() => flattenGroups(ftsGroups), [ftsGroups])
  const flatSemantic = useMemo(() => flattenGroups(semanticGroups), [semanticGroups])
  const allFlat = useMemo(() => [...flatFts, ...flatSemantic], [flatFts, flatSemantic])

  const hasFtsResults = flatFts.length > 0
  const hasSemanticResults = flatSemantic.length > 0
  const hasAnyResults = hasFtsResults || hasSemanticResults
  // Show section headers when semantic is available and there's something to show
  const showSectionHeaders = semanticAvailable && (hasFtsResults || hasSemanticResults || semanticStatus === 'pending')

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
  }, [ftsResults, semanticResults])

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
      const key = result.project_key
      const num = result.item_number

      switch (result.entity_type) {
        case 'work_item':
          if (key && num != null) navigate(`/projects/${key}/items/${num}`)
          break
        case 'comment':
          if (key && num != null) navigate(`/projects/${key}/items/${num}?tab=comments&highlight=${result.entity_id}`)
          break
        case 'attachment':
          if (key && num != null) navigate(`/projects/${key}/items/${num}?tab=attachments&highlight=${result.entity_id}`)
          break
        case 'project':
          if (key) navigate(`/projects/${key}`)
          break
        case 'milestone':
          if (key) navigate(`/projects/${key}/milestones`)
          break
        case 'queue':
          if (key) navigate(`/projects/${key}/queues`)
          break
      }
    },
    [navigate, onClose],
  )

  const handleKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        const next = Math.min(selectedIndex + 1, allFlat.length - 1)
        setSelectedIndex(next)
        scrollSelectedIntoView(next)
      } else if (e.key === 'ArrowUp') {
        e.preventDefault()
        const prev = Math.max(selectedIndex - 1, 0)
        setSelectedIndex(prev)
        scrollSelectedIntoView(prev)
      } else if (e.key === 'Enter' && allFlat.length > 0) {
        e.preventDefault()
        navigateToResult(allFlat[selectedIndex])
      }
    },
    [selectedIndex, allFlat, navigateToResult, scrollSelectedIntoView],
  )

  const showResults = debouncedQuery.length >= 2
  // Show loading only while waiting for initial FTS results
  const showInitialLoading = isLoading && !hasFtsResults && !hasSemanticResults

  const renderGroups = (
    groups: ResultGroup[],
    showScores: boolean,
  ) =>
    groups.map((group) => {
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
            const globalIndex = allFlat.indexOf(result)
            const parsed = parseSnippet(result)
            const ItemIcon = entityIcon(result.entity_type)
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
                {parsed.type && <TypeBadge type={parsed.type} />}
                {(result.entity_type === 'comment' || result.entity_type === 'attachment') && (
                  <ItemIcon className="h-3.5 w-3.5 text-gray-400 shrink-0" />
                )}
                <span className="text-gray-900 dark:text-gray-100 truncate flex-1">
                  {parsed.text}
                </span>
                {showScores && result.score > 0 && (
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
    })

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
      </div>

      <div className="border-t border-gray-200 dark:border-gray-700" />

      {!showResults ? (
        <p className="text-sm text-gray-500 dark:text-gray-400 py-6 text-center">
          {t('search.hint')}
        </p>
      ) : showInitialLoading ? (
        <div className="flex justify-center py-8">
          <Loader2 className="h-6 w-6 text-gray-400 animate-spin" />
        </div>
      ) : !hasAnyResults && !isLoading ? (
        <div className="flex flex-col items-center py-8 gap-2">
          <SearchX className="h-8 w-8 text-gray-300 dark:text-gray-600" />
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t('search.noResults')}
          </p>
        </div>
      ) : (
        <div ref={listRef} className="max-h-[60vh] overflow-y-auto py-2">
          {/* FTS Results Section */}
          {hasFtsResults && (
            <>
              {showSectionHeaders && (
                <div className="px-3 py-1.5 mb-1">
                  <span className="text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                    {t('search.perfectMatches')}
                  </span>
                </div>
              )}
              {renderGroups(ftsGroups, false)}
            </>
          )}

          {/* Semantic Results Section */}
          {semanticAvailable && (
            <>
              {showSectionHeaders && hasSemanticResults && (
                <div className="flex items-center gap-2 px-3 py-1.5 mt-3 mb-1 border-t border-gray-100 dark:border-gray-700/50 pt-3">
                  <span className="text-xs font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                    {t('search.relatedResults')}
                  </span>
                  <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium bg-gray-100 text-gray-500 dark:bg-gray-700/50 dark:text-gray-400">
                    <FlaskConical className="h-3 w-3" />
                    {t('common.experimental')}
                  </span>
                </div>
              )}
              {hasSemanticResults && renderGroups(semanticGroups, true)}

              {/* Semantic loading spinner */}
              {semanticStatus === 'pending' && (
                <div className="flex items-center gap-2 px-3 py-3 mt-2 border-t border-gray-100 dark:border-gray-700/50">
                  <Loader2 className="h-3.5 w-3.5 text-gray-400 animate-spin" />
                  <span className="text-xs text-gray-400 dark:text-gray-500">
                    {t('search.semanticLoading')}
                  </span>
                </div>
              )}

              {/* Semantic error */}
              {semanticError && (
                <div className="flex items-center gap-2 px-3 py-3 mt-2 border-t border-gray-100 dark:border-gray-700/50">
                  <AlertCircle className="h-3.5 w-3.5 text-gray-400 shrink-0" />
                  <span className="text-xs text-gray-400 dark:text-gray-500">
                    {t('search.semanticError')}
                  </span>
                </div>
              )}
            </>
          )}

          {/* Load more */}
          {flatFts.length >= limit && (
            <button
              onClick={() => setLimit((prev) => prev + 20)}
              className="w-full text-center py-2 text-sm text-indigo-600 dark:text-indigo-400 hover:bg-gray-50 dark:hover:bg-gray-800 rounded-md"
            >
              {t('search.showMore')}
            </button>
          )}
        </div>
      )}

      <div className="border-t border-gray-200 dark:border-gray-700 pt-2 mt-1 hidden [@media(hover:hover)_and_(pointer:fine)]:flex items-center justify-between">
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
