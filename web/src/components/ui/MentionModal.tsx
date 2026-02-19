import { useState, useRef, useEffect, useMemo, useCallback } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import { Spinner } from '@/components/ui/Spinner'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { useProjects } from '@/hooks/useProjects'
import { useWorkItems } from '@/hooks/useWorkItems'
import { useDebounce } from '@/hooks/useDebounce'
import type { DropdownPosition } from '@/hooks/useMentionAutocomplete'

interface MentionItem {
  type: 'project' | 'workitem'
  label: string
  link: string
  projectKey?: string
  displayId?: string
  workItemType?: string
}

interface MentionModalProps {
  open: boolean
  position: DropdownPosition
  onClose: () => void
  onSelect: (markdownLink: string) => void
  projectKey: string
}

export function MentionModal({ open, position, onClose, onSelect, projectKey }: MentionModalProps) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLDivElement>(null)
  const containerRef = useRef<HTMLDivElement>(null)

  const debouncedSearch = useDebounce(search, 300)

  const { data: projects } = useProjects()
  const shouldSearchItems = open && debouncedSearch.length >= 2
  const { data: workItemsData, isFetching } = useWorkItems(
    projectKey,
    shouldSearchItems ? { q: debouncedSearch, limit: 10 } : { limit: 0 },
  )

  const items = useMemo(() => {
    const result: MentionItem[] = []
    const q = debouncedSearch.toLowerCase()

    if (projects) {
      const matchingProjects = projects.filter((p) => {
        if (!q) return true
        return p.name.toLowerCase().includes(q) || p.key.toLowerCase().includes(q)
      }).slice(0, 5)

      for (const p of matchingProjects) {
        result.push({
          type: 'project',
          label: p.name,
          link: `[${p.name}](/projects/${p.key})`,
          projectKey: p.key,
        })
      }
    }

    if (shouldSearchItems && workItemsData?.data) {
      for (const wi of workItemsData.data) {
        result.push({
          type: 'workitem',
          label: wi.title,
          link: `[${wi.display_id}](/projects/${wi.project_key}/items/${wi.item_number})`,
          displayId: wi.display_id,
          workItemType: wi.type,
        })
      }
    }

    return result
  }, [projects, workItemsData, debouncedSearch, shouldSearchItems])

  // Reset selected index when items change
  useEffect(() => {
    setSelectedIndex(0)
  }, [items.length])

  // Auto-focus input and reset state when dropdown opens
  useEffect(() => {
    if (open) {
      setSearch('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

  // Scroll selected item into view
  useEffect(() => {
    if (!listRef.current) return
    const selectedEl = listRef.current.querySelector('[data-selected="true"]')
    if (selectedEl) {
      selectedEl.scrollIntoView({ block: 'nearest' })
    }
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

  // Close on Escape (global)
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

  const projectItems = items.filter((i) => i.type === 'project')
  const workItems = items.filter((i) => i.type === 'workitem')
  const workItemOffset = projectItems.length

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSelectedIndex((prev) => (prev + 1) % (items.length || 1))
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSelectedIndex((prev) => (prev - 1 + (items.length || 1)) % (items.length || 1))
    } else if (e.key === 'Enter') {
      e.preventDefault()
      if (items[selectedIndex]) {
        onSelect(items[selectedIndex].link)
      }
    }
  }, [items, selectedIndex, onSelect])

  if (!open) return null

  // Clamp position so dropdown stays within viewport
  const dropdownWidth = 320
  const viewportW = window.innerWidth
  const viewportH = window.innerHeight
  let top = position.top
  let left = position.left

  // Keep within horizontal bounds
  if (left + dropdownWidth > viewportW - 8) {
    left = viewportW - dropdownWidth - 8
  }
  if (left < 8) left = 8

  // If dropdown would go below viewport, show above the caret instead
  const estimatedHeight = 300
  if (top + estimatedHeight > viewportH + window.scrollY) {
    top = position.top - estimatedHeight - 20
  }

  return createPortal(
    <div
      ref={containerRef}
      className="fixed z-50 w-80 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-600 rounded-lg shadow-xl"
      style={{ top, left }}
      onKeyDown={handleKeyDown}
    >
      <div className="p-2">
        <input
          ref={inputRef}
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-700 dark:text-gray-100 px-2.5 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          placeholder={t('mention.searchPlaceholder')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div ref={listRef} className="max-h-48 overflow-auto pb-1">
        {/* Projects section */}
        {projectItems.length > 0 && (
          <div>
            <div className="px-3 py-1 text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider">
              {t('mention.projects')}
            </div>
            {projectItems.map((item, i) => (
              <button
                key={`project-${item.projectKey}`}
                type="button"
                data-selected={selectedIndex === i}
                className={`w-full text-left px-3 py-1.5 text-sm flex items-center gap-2 ${
                  selectedIndex === i
                    ? 'bg-indigo-50 dark:bg-indigo-900/30'
                    : 'hover:bg-gray-50 dark:hover:bg-gray-700'
                }`}
                onClick={() => onSelect(item.link)}
                onMouseEnter={() => setSelectedIndex(i)}
              >
                <span className="inline-flex items-center rounded px-1.5 py-0.5 text-xs font-mono font-medium bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300">
                  {item.projectKey}
                </span>
                <span className="text-gray-900 dark:text-gray-100 truncate">{item.label}</span>
              </button>
            ))}
          </div>
        )}

        {/* Work Items section */}
        {workItems.length > 0 && (
          <div>
            <div className="px-3 py-1 text-xs font-semibold text-gray-400 dark:text-gray-500 uppercase tracking-wider">
              {t('mention.workItems')}
            </div>
            {workItems.map((item, i) => (
              <button
                key={`workitem-${item.displayId}`}
                type="button"
                data-selected={selectedIndex === workItemOffset + i}
                className={`w-full text-left px-3 py-1.5 text-sm flex items-center gap-2 ${
                  selectedIndex === workItemOffset + i
                    ? 'bg-indigo-50 dark:bg-indigo-900/30'
                    : 'hover:bg-gray-50 dark:hover:bg-gray-700'
                }`}
                onClick={() => onSelect(item.link)}
                onMouseEnter={() => setSelectedIndex(workItemOffset + i)}
              >
                <span className="font-mono font-medium text-indigo-600 dark:text-indigo-400 shrink-0">
                  {item.displayId}
                </span>
                <span className="text-gray-900 dark:text-gray-100 truncate">{item.label}</span>
                {item.workItemType && <TypeBadge type={item.workItemType} />}
              </button>
            ))}
          </div>
        )}

        {/* Loading state */}
        {isFetching && items.length === 0 && (
          <div className="flex items-center justify-center py-3">
            <Spinner size="sm" />
          </div>
        )}

        {/* No results */}
        {!isFetching && items.length === 0 && search.length >= 2 && (
          <div className="px-3 py-3 text-sm text-gray-400 dark:text-gray-500 text-center">
            {t('mention.noResults')}
          </div>
        )}
      </div>
    </div>,
    document.body,
  )
}
