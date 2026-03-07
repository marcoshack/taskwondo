import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { createPortal } from 'react-dom'
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
  FlaskConical,
} from 'lucide-react'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { useSearch } from '@/hooks/useSearch'
import { useDebounce } from '@/hooks/useDebounce'
import type { SearchResult } from '@/api/search'
import type { DropdownPosition } from '@/hooks/useMentionAutocomplete'

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

function flattenGroups(groups: ResultGroup[]): SearchResult[] {
  return groups.flatMap((g) => g.results)
}

function parseSnippet(result: SearchResult): { type?: string; text: string } {
  const s = result.snippet
  if (result.entity_type === 'work_item') {
    const m = s.match(/^\[(\w+)]\s*(.*)/)
    if (m) return { type: m[1], text: m[2].split('\n')[0] }
  }
  if (result.entity_type === 'comment') {
    const body = s.replace(/^Comment:\s*\n*/, '')
    return { text: body.split('\n')[0] }
  }
  if (result.entity_type === 'attachment') {
    const line = s.replace(/^Attachment\s+/, '')
    return { text: line.split('\n')[0] }
  }
  return { text: s.split('\n')[0] }
}

function buildMarkdownLink(result: SearchResult): string {
  const key = result.project_key
  const num = result.item_number
  const parsed = parseSnippet(result)

  switch (result.entity_type) {
    case 'work_item':
      if (key && num != null) {
        const displayId = `${key}-${num}`
        return `[${displayId}](/projects/${key}/items/${num})`
      }
      return `[${parsed.text}](#)`
    case 'project':
      return `[${parsed.text}](/projects/${key})`
    case 'attachment':
      if (key && num != null)
        return `[${parsed.text}](/api/v1/projects/${key}/items/${num}/attachments/${result.entity_id}/file)`
      return `[${parsed.text}](#)`
    case 'comment':
      if (key && num != null)
        return `[${key}-${num}#comment](/projects/${key}/items/${num}?tab=comments&highlight=${result.entity_id})`
      return `[${parsed.text}](#)`
    case 'milestone':
      if (key)
        return `[${parsed.text}](/projects/${key}/milestones)`
      return `[${parsed.text}](#)`
    case 'queue':
      if (key)
        return `[${parsed.text}](/projects/${key}/queues)`
      return `[${parsed.text}](#)`
    default:
      return `[${parsed.text}](#)`
  }
}

interface MentionSearchModalProps {
  open: boolean
  position: DropdownPosition
  onClose: () => void
  onSelect: (markdownLink: string) => void
}

export function MentionSearchModal({ open, position, onClose, onSelect }: MentionSearchModalProps) {
  const { t } = useTranslation()
  const [query, setQuery] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const debouncedQuery = useDebounce(query, 300)
  const {
    ftsResults,
    semanticResults,
    semanticAvailable,
    semanticStatus,
    isLoading,
  } = useSearch({ query: debouncedQuery, limit: 15 })

  const ftsGroups = useMemo(() => groupResults(ftsResults), [ftsResults])
  const semanticGroups = useMemo(() => groupResults(semanticResults), [semanticResults])
  const flatFts = useMemo(() => flattenGroups(ftsGroups), [ftsGroups])
  const flatSemantic = useMemo(() => flattenGroups(semanticGroups), [semanticGroups])
  const allFlat = useMemo(() => [...flatFts, ...flatSemantic], [flatFts, flatSemantic])

  const hasFtsResults = flatFts.length > 0
  const hasSemanticResults = flatSemantic.length > 0
  const hasAnyResults = hasFtsResults || hasSemanticResults
  const showSectionHeaders = semanticAvailable && (hasFtsResults || hasSemanticResults || semanticStatus === 'pending')

  useEffect(() => {
    setSelectedIndex(0)
  }, [ftsResults, semanticResults])

  useEffect(() => {
    if (open) {
      setQuery('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

  useEffect(() => {
    if (!listRef.current) return
    const selectedEl = listRef.current.querySelector('[data-selected="true"]')
    if (selectedEl) selectedEl.scrollIntoView({ block: 'nearest' })
  }, [selectedIndex])

  // Close on click outside
  const onCloseRef = useRef(onClose)
  onCloseRef.current = onClose
  useEffect(() => {
    if (!open) return
    function handler(e: MouseEvent) {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        onCloseRef.current()
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  useEffect(() => {
    if (!open) return
    function handler(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        e.stopPropagation()
        onCloseRef.current()
      }
    }
    document.addEventListener('keydown', handler, true)
    return () => document.removeEventListener('keydown', handler, true)
  }, [open])

  const renderGroups = (groups: ResultGroup[]) =>
    groups.map((group) => {
      const Icon = entityIcon(group.entityType)
      return (
        <div key={group.entityType} className="mb-1 last:mb-0">
          <div className="flex items-center gap-1.5 px-3 py-1">
            <Icon className="h-3 w-3 text-gray-400" />
            <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
              {t(`search.entityType.${group.entityType}`)}
            </span>
          </div>
          {group.results.map((result) => {
            const globalIndex = allFlat.indexOf(result)
            const parsed = parseSnippet(result)
            const isSelected = globalIndex === selectedIndex
            return (
              <button
                key={`${result.entity_type}-${result.entity_id}`}
                type="button"
                data-selected={isSelected}
                onClick={() => onSelect(buildMarkdownLink(result))}
                onMouseEnter={() => setSelectedIndex(globalIndex)}
                className={`w-full text-left flex items-center gap-2 px-3 py-1.5 text-sm ${
                  isSelected
                    ? 'bg-indigo-50 dark:bg-indigo-900/30'
                    : 'hover:bg-gray-50 dark:hover:bg-gray-700'
                }`}
              >
                {parsed.type && <TypeBadge type={parsed.type} />}
                {result.entity_type === 'work_item' && result.project_key && result.item_number != null && (
                  <span className="font-mono text-xs text-indigo-600 dark:text-indigo-400 shrink-0">
                    {result.project_key}-{result.item_number}
                  </span>
                )}
                <span className="text-gray-900 dark:text-gray-100 truncate flex-1">
                  {parsed.text}
                </span>
                {result.project_key && result.entity_type !== 'project' && result.entity_type !== 'work_item' && (
                  <span className="text-[10px] text-gray-400 shrink-0">
                    {result.project_key}
                  </span>
                )}
              </button>
            )
          })}
        </div>
      )
    })

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex((prev) => Math.min(prev + 1, allFlat.length - 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex((prev) => Math.max(prev - 1, 0))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (allFlat[selectedIndex]) {
        onSelect(buildMarkdownLink(allFlat[selectedIndex]))
      }
    }
  }, [allFlat, selectedIndex, onSelect])

  if (!open) return null

  // Viewport-aware positioning
  const dropdownWidth = 384
  const dropdownMaxHeight = 320
  const viewportW = document.documentElement.clientWidth
  const viewportH = document.documentElement.clientHeight
  let top = position.top
  let left = position.left

  if (left + dropdownWidth > viewportW - 8) {
    left = viewportW - dropdownWidth - 8
  }
  if (left < 8) left = 8

  if (top + dropdownMaxHeight > viewportH) {
    top = position.top - dropdownMaxHeight - 20
  }

  const showResults = debouncedQuery.length >= 2

  return createPortal(
    <div
      ref={containerRef}
      className="fixed z-50 w-96 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg shadow-xl"
      style={{ top, left }}
      onKeyDown={handleKeyDown}
    >
      <div className="flex items-center gap-2 px-3 py-2 border-b border-gray-200 dark:border-gray-700">
        <Search className="h-4 w-4 text-gray-400 shrink-0" />
        <input
          ref={inputRef}
          type="text"
          placeholder={t('mention.searchPlaceholder')}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          className="block w-full text-sm bg-transparent border-none outline-none text-gray-900 dark:text-gray-100 placeholder-gray-400 dark:placeholder-gray-500"
        />
        {isLoading && showResults && (
          <Loader2 className="h-4 w-4 text-gray-400 animate-spin shrink-0" />
        )}
      </div>

      <div ref={listRef} className="max-h-60 overflow-auto py-1">
        {!showResults ? (
          <p className="text-xs text-gray-400 dark:text-gray-500 py-4 text-center">
            {t('search.hint')}
          </p>
        ) : isLoading && !hasAnyResults ? (
          <div className="flex justify-center py-6">
            <Loader2 className="h-5 w-5 text-gray-400 animate-spin" />
          </div>
        ) : !hasAnyResults && !isLoading ? (
          <p className="text-xs text-gray-400 dark:text-gray-500 py-4 text-center">
            {t('mention.noResults')}
          </p>
        ) : (
          <>
            {hasFtsResults && (
              <>
                {showSectionHeaders && (
                  <div className="px-3 py-1">
                    <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                      {t('search.perfectMatches')}
                    </span>
                  </div>
                )}
                {renderGroups(ftsGroups)}
              </>
            )}

            {semanticAvailable && (
              <>
                {showSectionHeaders && hasSemanticResults && (
                  <div className="flex items-center gap-1.5 px-3 py-1 mt-1 border-t border-gray-100 dark:border-gray-700/50 pt-2">
                    <span className="text-[10px] font-semibold uppercase tracking-wider text-gray-400 dark:text-gray-500">
                      {t('search.relatedResults')}
                    </span>
                    <span className="inline-flex items-center gap-0.5 rounded-full px-1.5 py-0.5 text-[9px] font-medium bg-gray-100 text-gray-500 dark:bg-gray-700/50 dark:text-gray-400">
                      <FlaskConical className="h-2.5 w-2.5" />
                      {t('common.experimental')}
                    </span>
                  </div>
                )}
                {hasSemanticResults && renderGroups(semanticGroups)}

                {semanticStatus === 'pending' && (
                  <div className="flex items-center gap-1.5 px-3 py-2 mt-1 border-t border-gray-100 dark:border-gray-700/50">
                    <Loader2 className="h-3 w-3 text-gray-400 animate-spin" />
                    <span className="text-[10px] text-gray-400 dark:text-gray-500">
                      {t('search.semanticLoading')}
                    </span>
                  </div>
                )}
              </>
            )}
          </>
        )}
      </div>
    </div>,
    document.body,
  )
}
