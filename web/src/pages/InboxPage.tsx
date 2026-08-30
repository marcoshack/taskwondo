import { useState, useMemo, useCallback, useRef, useEffect } from 'react'
import { Routes, Route, useNavigate } from 'react-router-dom'
import { toUrlSegment } from '@/hooks/useNamespacePath'
import { Trans, useTranslation } from 'react-i18next'
import { ChevronUp, ChevronDown, X, Search, BrushCleaning, Inbox, Check, Rss, Settings, User, History, Pencil, FolderKanban } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { CreateWorkItemModal } from '@/components/workitems/CreateWorkItemModal'
import { Modal } from '@/components/ui/Modal'
import { Input } from '@/components/ui/Input'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { Tooltip } from '@/components/ui/Tooltip'
import { MultiSelect, type MultiSelectOption } from '@/components/ui/MultiSelect'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { NamespaceIcon } from '@/components/NamespaceIcon'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { RefreshButton, type RefreshInterval } from '@/components/ui/RefreshButton'
import { SLAIndicator } from '@/components/SLAIndicator'
import { AppSidebar } from '@/components/AppSidebar'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLayout } from '@/contexts/LayoutContext'
import { useInboxItems, useRemoveFromInbox, useReorderInboxItem, useClearCompletedInbox, useAddToInbox } from '@/hooks/useInbox'
import { useAllProjects } from '@/hooks/useProjects'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useColumnWidths } from '@/hooks/useColumnWidths'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { useDebounce } from '@/hooks/useDebounce'
import { listInboxItems, type InboxItem } from '@/api/inbox'
import { formatRelativeTime } from '@/utils/duration'
import WatchlistPage from '@/pages/WatchlistPage'

const categoryColors = {
  todo: 'gray',
  in_progress: 'blue',
  done: 'green',
  cancelled: 'red',
} as const

function InboxStatusBadge({ status, category }: { status: string; category: string }) {
  const { t } = useTranslation()
  const color = categoryColors[category as keyof typeof categoryColors] ?? 'gray'
  const key = `workitems.statuses.${status}`
  const translated = t(key)
  return <Badge color={color}>{translated === key ? status : translated}</Badge>
}

// --- Table Row ---

interface InboxRowProps {
  item: InboxItem
  isCompleted: boolean
  isFirst: boolean
  isLast: boolean
  isActive: boolean
  onRemove: (id: string) => void
  onMoveUp: (item: InboxItem) => void
  onMoveDown: (item: InboxItem) => void
  onClick: (item: InboxItem) => void
  removedId: string | null
  reorderedId: string | null
  autoRemove: boolean
  columnWidths: Record<string, number>
}

function getDescriptionPreview(description: string): string {
  const line = description.split('\n').find(l => l.trim() !== '')
  if (!line) return ''
  return line.trim().replace(/^#+\s+/, '').replace(/[*_~`[\]]/g, '')
}

function InboxRow({ item, isCompleted, isFirst, isLast, isActive, onRemove, onMoveUp, onMoveDown, onClick, removedId, reorderedId, autoRemove, columnWidths }: InboxRowProps) {
  const { t } = useTranslation()
  const isRemoving = removedId === item.id
  const rowRef = useRef<HTMLTableRowElement>(null)

  useEffect(() => {
    if (isActive) rowRef.current?.scrollIntoView({ block: 'nearest' })
  }, [isActive])

  return (
    <tr
      ref={rowRef}
      className={`group border-b border-[var(--border)]
        ${reorderedId === item.id ? 'animate-[inbox-highlight_1s_ease-in-out]' : ''}
        ${isRemoving && autoRemove ? 'transition-all duration-200 opacity-0 -translate-y-2' : ''}
        ${isActive ? 'bg-[var(--primary-muted)]' : 'hover:bg-[var(--surface-hover)]'} cursor-pointer`}
      onClick={() => onClick(item)}
    >
      {/* Reorder arrows */}
      <td className={`w-10 px-1 py-3 ${isCompleted && !isRemoving ? 'opacity-40' : ''}`} onClick={(e) => e.stopPropagation()}>
        <div className="flex flex-col items-center -my-1">
          <button
            onClick={() => onMoveUp(item)}
            disabled={isFirst}
            className={`p-0.5 rounded text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] hover:bg-[var(--surface-tertiary)] hover:bg-[var(--surface-hover)] ${isFirst ? 'invisible' : ''}`}
            aria-label={t('inbox.moveUp')}
          >
            <ChevronUp className="h-4 w-4" />
          </button>
          <button
            onClick={() => onMoveDown(item)}
            disabled={isLast}
            className={`p-0.5 rounded text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] hover:bg-[var(--surface-tertiary)] hover:bg-[var(--surface-hover)] ${isLast ? 'invisible' : ''}`}
            aria-label={t('inbox.moveDown')}
          >
            <ChevronDown className="h-4 w-4" />
          </button>
        </div>
      </td>
      {/* Display ID */}
      <td style={columnWidths.display_id ? { width: columnWidths.display_id } : undefined} className={`px-3 py-3 text-sm font-mono whitespace-nowrap ${isCompleted && !isRemoving ? 'text-[var(--foreground-muted)]' : 'text-[var(--foreground-secondary)]'} ${columnWidths.display_id ? '' : 'w-24'}`}>
        {item.display_id}
      </td>
      {/* Type */}
      <td style={columnWidths.type ? { width: columnWidths.type } : undefined} className={`px-3 py-3 whitespace-nowrap ${isCompleted && !isRemoving ? 'opacity-40' : ''} ${columnWidths.type ? '' : 'w-20'}`}>
        <TypeBadge type={item.type} />
      </td>
      {/* Title */}
      <td className={`px-3 py-3 text-sm truncate ${item.description && !isCompleted ? 'text-[var(--foreground-muted)]' : ''}`}>
        <span className={isCompleted && !isRemoving ? 'line-through text-[var(--foreground-muted)]' : 'text-[var(--foreground)]'}>
          {item.title}
        </span>
        {item.description && !isCompleted && (
          <span className="text-xs"> – {getDescriptionPreview(item.description)}</span>
        )}
      </td>
      {/* Status — always shows color for visibility */}
      <td style={columnWidths.status ? { width: columnWidths.status } : undefined} className={`px-3 py-3 whitespace-nowrap ${columnWidths.status ? '' : 'w-28'}`}>
        <InboxStatusBadge status={item.status} category={item.status_category} />
      </td>
      {/* Priority */}
      <td style={columnWidths.priority ? { width: columnWidths.priority } : undefined} className={`px-3 py-3 whitespace-nowrap ${isCompleted && !isRemoving ? 'opacity-40' : ''} ${columnWidths.priority ? '' : 'w-24'}`}>
        <PriorityBadge priority={item.priority} />
      </td>
      {/* SLA */}
      <td style={columnWidths.sla ? { width: columnWidths.sla } : undefined} className={`px-3 py-3 whitespace-nowrap overflow-hidden ${isCompleted && !isRemoving ? 'opacity-40' : ''} ${columnWidths.sla ? '' : 'w-[110px]'}`}>
        <SLAIndicator sla={item.sla} />
      </td>
      {/* Updated */}
      <td style={columnWidths.updated ? { width: columnWidths.updated } : undefined} className={`px-3 py-3 whitespace-nowrap text-sm text-right ${isCompleted && !isRemoving ? 'text-[var(--foreground-secondary)]' : 'text-[var(--foreground-secondary)]'} ${columnWidths.updated ? '' : 'w-[130px]'}`}>
        {new Date(item.updated_at).toLocaleDateString()}
      </td>
      {/* Remove button */}
      <td className="w-10 px-2 py-3" onClick={(e) => e.stopPropagation()}>
        {isRemoving ? (
          <Check className="h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
        ) : (
          <button
            onClick={() => onRemove(item.id)}
            className="lg:opacity-0 lg:group-hover:opacity-100 text-[var(--foreground-muted)] hover:text-[var(--danger)] transition-opacity"
            aria-label={t('inbox.removeFromInbox')}
          >
            <X className="h-4 w-4" />
          </button>
        )}
      </td>
    </tr>
  )
}

// --- Mobile Card ---

interface InboxCardProps {
  item: InboxItem
  isCompleted: boolean
  isFirst: boolean
  isLast: boolean
  isActive: boolean
  editing: boolean
  onRemove: (id: string) => void
  onMoveUp: (item: InboxItem) => void
  onMoveDown: (item: InboxItem) => void
  onClick: (item: InboxItem) => void
  removedId: string | null
  reorderedId: string | null
  autoRemove: boolean
}

function InboxCard({ item, isCompleted, isFirst, isLast, isActive, editing, onRemove, onMoveUp, onMoveDown, onClick, removedId, reorderedId, autoRemove }: InboxCardProps) {
  const { t } = useTranslation()
  const isRemoving = removedId === item.id
  const dimmed = isCompleted && !isRemoving
  const assigneeName = item.assignee_display_name || t('userPicker.unassigned')
  const cardRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isActive) cardRef.current?.scrollIntoView({ block: 'nearest' })
  }, [isActive])

  return (
    <div
      ref={cardRef}
      className={`flex items-stretch gap-0 rounded-lg border bg-[var(--surface)] shadow-sm transition-colors
        ${isActive ? 'border-[var(--primary-border)] dark:border-[var(--primary)] ring-1 ring-indigo-300 dark:ring-indigo-600' : 'border-[var(--border)] hover:border-[var(--primary-border)] dark:hover:border-[var(--primary-border)]'}
        ${reorderedId === item.id ? 'animate-[inbox-highlight_1s_ease-in-out]' : ''}
        ${isRemoving && autoRemove ? 'transition-all duration-200 opacity-0 -translate-y-2' : ''}`}
    >
      {/* Card content */}
      <button
        className="flex-1 text-left min-w-0 p-3"
        onClick={() => onClick(item)}
      >
        {/* Line 1: Display ID + badges */}
        <div className="flex items-center gap-2 overflow-x-auto scrollbar-none">
          <span className={`shrink-0 font-mono text-sm font-semibold ${dimmed ? 'text-[var(--foreground-muted)]' : 'text-[var(--foreground)]'}`}>{item.display_id}</span>
          <span className={`shrink-0 inline-flex ${dimmed ? 'opacity-40' : ''}`}><TypeBadge type={item.type} /></span>
          <span className="shrink-0 inline-flex"><InboxStatusBadge status={item.status} category={item.status_category} /></span>
          <span className={`shrink-0 inline-flex ${dimmed ? 'opacity-40' : ''}`}><PriorityBadge priority={item.priority} /></span>
        </div>
        {/* Line 2: Assignee, Updated, SLA */}
        <div className={`flex items-center gap-3 mt-1.5 text-xs overflow-x-auto scrollbar-none ${dimmed ? 'text-[var(--foreground-secondary)]' : 'text-[var(--foreground-muted)]'}`}>
          <span className="shrink-0 inline-flex items-center gap-1">
            <User className="h-3 w-3" />
            <span className="truncate max-w-[8rem]">{assigneeName}</span>
          </span>
          <span className="shrink-0 inline-flex items-center gap-1">
            <History className="h-3 w-3" />
            {formatRelativeTime(item.updated_at)}
          </span>
          {!dimmed && item.sla && <span className="shrink-0 inline-flex"><SLAIndicator sla={item.sla} /></span>}
        </div>
        {/* Line 3: Title */}
        <p className={`mt-1.5 text-sm font-medium truncate ${dimmed ? 'line-through text-[var(--foreground-muted)]' : 'text-[var(--foreground)]'}`}>
          {item.title}
        </p>
        {/* Line 4: Description (first line) */}
        {item.description && (
          <p className={`mt-0.5 text-xs truncate ${dimmed ? 'text-[var(--foreground-secondary)]' : 'text-[var(--foreground-secondary)]'}`}>
            {item.description}
          </p>
        )}
      </button>
      {/* Edit controls — right column, fits within card's natural height */}
      {editing && (
        <div className="flex flex-col items-center justify-between flex-shrink-0 rounded-r-lg bg-[var(--primary-muted)] border-l border-[var(--primary-border)] dark:border-[var(--primary-border)]/50 px-2 py-1">
          {/* Remove at top */}
          <button
            onClick={() => onRemove(item.id)}
            className="p-1 rounded text-red-400 hover:text-[var(--danger)] hover:bg-red-100 text-[var(--danger)] dark:hover:text-red-300 dark:hover:bg-red-900/30"
            aria-label={t('inbox.removeFromInbox')}
          >
            <X className="h-3.5 w-3.5" />
          </button>
          {/* Reorder arrows — shifted up 10%, spaced 20% apart */}
          <div className="relative -top-[10%] flex flex-col items-center gap-[20%]">
            <button
              onClick={() => onMoveUp(item)}
              disabled={isFirst}
              className={`p-0.5 rounded text-[var(--primary)] hover:text-[var(--primary)] hover:bg-[var(--primary-muted)] dark:text-[var(--primary)] dark:hover:text-[var(--primary)] hover:bg-[var(--primary-muted)] ${isFirst ? 'invisible' : ''}`}
              aria-label={t('inbox.moveUp')}
            >
              <ChevronUp className="h-4 w-4" />
            </button>
            <button
              onClick={() => onMoveDown(item)}
              disabled={isLast}
              className={`p-0.5 rounded text-[var(--primary)] hover:text-[var(--primary)] hover:bg-[var(--primary-muted)] dark:text-[var(--primary)] dark:hover:text-[var(--primary)] hover:bg-[var(--primary-muted)] ${isLast ? 'invisible' : ''}`}
              aria-label={t('inbox.moveDown')}
            >
              <ChevronDown className="h-4 w-4" />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}

// --- Inbox List Page (main inbox content) ---

function InboxListPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()

  const [searchInput, setSearchInput] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)
  const [searchFocused, setSearchFocused] = useState(false)
  const debouncedSearch = useDebounce(searchInput, 500)
  const { data: projectFilterPref } = usePreference<string[]>('inbox_project_filter')
  const { data: strikethroughPref } = usePreference<boolean>('strikethrough_completed')
  const strikethroughEnabled = strikethroughPref ?? true
  const [selectedProjects, setSelectedProjectsRaw] = useState<string[]>([])
  const [projectFilterInit, setProjectFilterInit] = useState(false)
  useEffect(() => {
    if (projectFilterPref !== undefined && !projectFilterInit) {
      setSelectedProjectsRaw(projectFilterPref ?? [])
      setProjectFilterInit(true)
    }
  }, [projectFilterPref, projectFilterInit])
  const [projectFilterOpen, setProjectFilterOpen] = useState(false)
  const { data: projects } = useAllProjects()
  const { showSwitcher: showNamespaces } = useNamespaceContext()

  const { columnWidths, onColumnResize, resetColumnWidth } = useColumnWidths('inbox')

  // Column resize drag logic (mirrors DataTable)
  const resizingRef = useRef<{ key: string; startX: number; startWidth: number } | null>(null)
  const onResizeRef = useRef(onColumnResize)
  useEffect(() => { onResizeRef.current = onColumnResize }, [onColumnResize])

  const handleResizeStart = useCallback((e: React.MouseEvent, colKey: string) => {
    e.preventDefault()
    e.stopPropagation()
    const th = (e.target as HTMLElement).closest('th')
    if (!th) return
    const startWidth = th.getBoundingClientRect().width
    resizingRef.current = { key: colKey, startX: e.clientX, startWidth }
    const handleMove = (e: MouseEvent) => {
      if (!resizingRef.current) return
      const { key, startX, startWidth } = resizingRef.current
      const diff = e.clientX - startX
      const newWidth = Math.max(40, startWidth + diff)
      onResizeRef.current?.(key, newWidth)
    }
    const handleEnd = () => {
      resizingRef.current = null
      document.removeEventListener('mousemove', handleMove)
      document.removeEventListener('mouseup', handleEnd)
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
      const suppress = (e: MouseEvent) => {
        e.stopPropagation()
        e.preventDefault()
        document.removeEventListener('click', suppress, true)
      }
      document.addEventListener('click', suppress, true)
    }
    document.addEventListener('mousemove', handleMove)
    document.addEventListener('mouseup', handleEnd)
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }, [onColumnResize])

  const { data: autoRemovePref } = usePreference<boolean>('inbox_auto_remove')
  const autoRemove = autoRemovePref ?? true
  const { data: skipRemoveConfirmPref } = usePreference<boolean>('inbox_skip_remove_confirm')
  const skipRemoveConfirm = skipRemoveConfirmPref ?? false
  const { data: refreshIntervalPref } = usePreference<number>('inbox_refresh_interval')
  const [refreshInterval, setRefreshInterval] = useState<RefreshInterval>(0)
  useEffect(() => {
    if (refreshIntervalPref != null) setRefreshInterval(refreshIntervalPref as RefreshInterval)
  }, [refreshIntervalPref])
  const setPreferenceMutation = useSetPreference()

  const setSelectedProjects = useCallback((val: string[] | ((prev: string[]) => string[])) => {
    setSelectedProjectsRaw((prev) => {
      const next = typeof val === 'function' ? val(prev) : val
      setPreferenceMutation.mutate({ key: 'inbox_project_filter', value: next })
      return next
    })
  }, [setPreferenceMutation])

  const { data, isLoading, refetch, isFetching } = useInboxItems({
    search: debouncedSearch || undefined,
    include_completed: true,
    project: selectedProjects.length > 0 ? selectedProjects : undefined,
  }, searchFocused ? 0 : refreshInterval)

  const projectOptions: MultiSelectOption[] = useMemo(() =>
    (projects ?? []).map((p) => ({ value: p.key, label: `${p.key} – ${p.name}` })),
    [projects],
  )

  const projectMap = useMemo(() => {
    const map = new Map<string, typeof projects extends (infer T)[] | undefined ? T : never>()
    for (const p of projects ?? []) map.set(p.key, p)
    return map
  }, [projects])

  const renderProjectOption = useCallback((opt: MultiSelectOption) => {
    const p = projectMap.get(opt.value)
    if (!p) return opt.label
    return (
      <span className="flex items-center gap-2 min-w-0 flex-1">
        <ProjectKeyBadge size="icon">{p.key}</ProjectKeyBadge>
        <span className="truncate">{p.name}</span>
        {showNamespaces && p.namespace_slug && (
          <span className="ml-auto flex items-center gap-1 text-[0.7rem] text-[var(--foreground-muted)] shrink-0">
            <span>{p.namespace_slug}</span>
            <NamespaceIcon icon={p.namespace_icon ?? 'building2'} color={p.namespace_color ?? 'slate'} className="h-3 w-3" />
          </span>
        )}
      </span>
    )
  }, [projectMap, showNamespaces])

  const [settingsOpen, setSettingsOpen] = useState(false)
  const [editing, setEditing] = useState(false)
  const [showCreate, setShowCreate] = useState(false)
  const addToInboxMutation = useAddToInbox()

  const [loadedPages, setLoadedPages] = useState<InboxItem[][]>([])
  const [prevSearch, setPrevSearch] = useState(debouncedSearch)
  const [prevProjects, setPrevProjects] = useState(selectedProjects)
  if (debouncedSearch !== prevSearch || selectedProjects !== prevProjects) {
    setPrevSearch(debouncedSearch)
    setPrevProjects(selectedProjects)
    setLoadedPages([])
  }

  const allLoadedItems = useMemo(() => {
    if (!data) return []
    if (loadedPages.length === 0) return data.items
    return [...data.items, ...loadedPages.flat()]
  }, [data, loadedPages])

  const completedItems = useMemo(() =>
    allLoadedItems.filter((i) => i.status_category === 'done' || i.status_category === 'cancelled'),
    [allLoadedItems],
  )

  const allItems = useMemo(() =>
    autoRemove ? allLoadedItems.filter((i) => i.status_category !== 'done' && i.status_category !== 'cancelled') : allLoadedItems,
    [allLoadedItems, autoRemove],
  )

  const hasMore = loadedPages.length > 0
    ? loadedPages[loadedPages.length - 1].length > 0 // approximate
    : (data?.has_more ?? false)

  const removeMutation = useRemoveFromInbox()
  const reorderMutation = useReorderInboxItem()
  const clearCompletedMutation = useClearCompletedInbox()

  const [removedId, setRemovedId] = useState<string | null>(null)
  const [reorderedId, setReorderedId] = useState<string | null>(null)
  const [activeRow, setActiveRow] = useState(-1)
  const activeRowStorageKey = 'taskwondo_activeRow_inbox'
  const restoredRef = useRef(false)
  const [selectedId, setSelectedId] = useState<string | null>(null)
  const [removeConfirmItem, setRemoveConfirmItem] = useState<InboxItem | null>(null)
  const [dontShowAgainChecked, setDontShowAgainChecked] = useState(false)

  // Debounced reorder for rapid j/k with selected item
  const reorderTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const pendingReorderRef = useRef<{ inboxItemId: string; position: number } | null>(null)

  const flushReorder = useCallback(() => {
    clearTimeout(reorderTimerRef.current)
    if (pendingReorderRef.current) {
      const { inboxItemId, position } = pendingReorderRef.current
      pendingReorderRef.current = null
      reorderMutation.mutate({ inboxItemId, position }, {
        onSuccess: () => {
          setReorderedId(inboxItemId)
          setTimeout(() => setReorderedId(null), 1000)
        },
      })
    }
  }, [reorderMutation])

  const scheduleReorder = useCallback((inboxItemId: string, position: number) => {
    pendingReorderRef.current = { inboxItemId, position }
    clearTimeout(reorderTimerRef.current)
    reorderTimerRef.current = setTimeout(flushReorder, 500)
  }, [flushReorder])

  const handleRemove = useCallback((inboxItemId: string) => {
    setRemovedId(inboxItemId)
    // If the removed item was selected, deselect
    if (selectedId === inboxItemId) setSelectedId(null)
    setTimeout(() => {
      removeMutation.mutate(inboxItemId)
      setRemovedId(null)
    }, autoRemove ? 300 : 0)
  }, [removeMutation, autoRemove, selectedId])

  // Remove with confirmation (Shift+#)
  const handleRemoveWithConfirm = useCallback((item: InboxItem) => {
    if (skipRemoveConfirm) {
      handleRemove(item.id)
    } else {
      setRemoveConfirmItem(item)
      setDontShowAgainChecked(false)
    }
  }, [skipRemoveConfirm, handleRemove])

  const handleClick = useCallback((item: InboxItem) => {
    sessionStorage.setItem(activeRowStorageKey, item.id)
    navigate(`/${toUrlSegment(item.namespace_slug)}/projects/${item.project_key}/items/${item.display_id.split('-')[1]}`, { state: { from: 'inbox' } })
  }, [navigate])

  // Reorder via arrow buttons — place item directly before/after neighbor
  const handleMoveUp = useCallback((item: InboxItem) => {
    const index = allItems.findIndex((i) => i.id === item.id)
    if (index <= 0) return
    const prev = allItems[index - 1]
    reorderMutation.mutate({ inboxItemId: item.id, position: prev.position - 1 }, {
      onSuccess: () => {
        setReorderedId(item.id)
        setTimeout(() => setReorderedId(null), 1000)
      },
    })
  }, [allItems, reorderMutation])

  const handleMoveDown = useCallback((item: InboxItem) => {
    const index = allItems.findIndex((i) => i.id === item.id)
    if (index < 0 || index >= allItems.length - 1) return
    const next = allItems[index + 1]
    reorderMutation.mutate({ inboxItemId: item.id, position: next.position + 1 }, {
      onSuccess: () => {
        setReorderedId(item.id)
        setTimeout(() => setReorderedId(null), 1000)
      },
    })
  }, [allItems, reorderMutation])

  // Restore active row from sessionStorage after navigating back
  useEffect(() => {
    if (restoredRef.current || allItems.length === 0) return
    const stored = sessionStorage.getItem(activeRowStorageKey)
    if (stored) {
      const idx = allItems.findIndex((i) => i.id === stored)
      if (idx >= 0) setActiveRow(idx)
      sessionStorage.removeItem(activeRowStorageKey)
    }
    restoredRef.current = true
  }, [allItems])

  // Keyboard navigation
  useKeyboardShortcut({ key: 'c' }, () => setShowCreate(true))
  useKeyboardShortcut({ key: '/' }, () => searchRef.current?.focus())
  useKeyboardShortcut([{ key: 'ArrowDown' }, { key: 'j' }], () => {
    if (selectedId && activeRow >= 0 && activeRow < allItems.length) {
      // Selected item: move it down
      const item = allItems[activeRow]
      if (item.id === selectedId && activeRow < allItems.length - 1) {
        const next = allItems[activeRow + 1]
        scheduleReorder(item.id, next.position + 1)
        setActiveRow(activeRow + 1)
      }
    } else {
      setActiveRow((prev) => Math.min(prev + 1, allItems.length - 1))
    }
  }, allItems.length > 0)
  useKeyboardShortcut([{ key: 'ArrowUp' }, { key: 'k' }], () => {
    if (selectedId && activeRow >= 0 && activeRow < allItems.length) {
      // Selected item: move it up
      const item = allItems[activeRow]
      if (item.id === selectedId && activeRow > 0) {
        const prev = allItems[activeRow - 1]
        scheduleReorder(item.id, prev.position - 1)
        setActiveRow(activeRow - 1)
      }
    } else {
      setActiveRow((prev) => Math.max(prev - 1, 0))
    }
  }, allItems.length > 0)
  useKeyboardShortcut({ key: 'x' }, () => {
    if (activeRow >= 0 && activeRow < allItems.length) {
      const item = allItems[activeRow]
      // Toggle selection — only one item at a time
      if (selectedId === item.id) {
        flushReorder()
        setSelectedId(null)
      } else {
        flushReorder()
        setSelectedId(item.id)
      }
    }
  }, activeRow >= 0)
  useKeyboardShortcut([{ key: 'Enter' }, { key: 'o' }], () => {
    if (activeRow >= 0 && activeRow < allItems.length) {
      handleClick(allItems[activeRow])
    }
  }, activeRow >= 0 && !selectedId)
  useKeyboardShortcut({ key: '#' }, () => {
    if (activeRow >= 0 && activeRow < allItems.length) {
      handleRemoveWithConfirm(allItems[activeRow])
    }
  }, activeRow >= 0)
  useKeyboardShortcut({ key: 'Escape' }, () => {
    if (selectedId) {
      flushReorder()
      setSelectedId(null)
    } else {
      setActiveRow(-1)
    }
  }, activeRow >= 0 || !!selectedId)

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spinner size="lg" />
      </div>
    )
  }

  return (
    <div>
      {/* Header: title + search + actions */}
      <div className="flex items-center justify-between gap-4 mb-4">
        <div className="flex items-center gap-3 min-w-0 shrink lg:shrink-0">
          <h1 className="text-lg font-semibold text-[var(--foreground)]">{t('inbox.title')}</h1>
          {data && (
            <span className="text-sm text-[var(--foreground-secondary)]">
              ({data.total})
            </span>
          )}
        </div>
        <div className="flex items-center gap-2 flex-1 justify-end">
          {/* Desktop: search bar (same position as Watchlist/WorkItems) */}
          <div className="hidden lg:block flex-1 min-w-0 max-w-lg">
            <div className="relative">
              <Input
                ref={searchRef}
                value={searchInput}
                onChange={(e) => setSearchInput(e.target.value)}
                placeholder={t('inbox.searchPlaceholder')}
                className="pr-8"
                onFocus={() => setSearchFocused(true)}
                onBlur={() => setSearchFocused(false)}
                onKeyDown={(e) => { if (e.key === 'Escape') searchRef.current?.blur() }}
              />
              {searchInput && (
                <button
                  onClick={() => { setSearchInput(''); searchRef.current?.focus() }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)]"
                  aria-label={t('common.clear')}
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>
          {/* New Item button */}
          <Button onClick={() => setShowCreate(true)} className="border border-transparent">
            <span className="lg:hidden">{t('workitems.newShort')}</span>
            <span className="hidden lg:inline">{t('workitems.new')}</span>
          </Button>
        </div>
      </div>

      {/* Mobile settings modal */}
      <Modal open={settingsOpen} onClose={() => setSettingsOpen(false)} title={t('inbox.settings')}>
        <div className="py-2">
          <label className="flex items-center justify-between cursor-pointer select-none">
            <span className="text-sm text-[var(--foreground)]">{t('inbox.autoRemove')}</span>
            <button
              type="button"
              role="switch"
              aria-checked={autoRemove}
              onClick={() => setPreferenceMutation.mutate({ key: 'inbox_auto_remove', value: !autoRemove })}
              className={`relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-colors ${
                autoRemove ? 'bg-[var(--primary)]' : 'bg-[var(--surface-tertiary)] dark:bg-[var(--foreground-secondary)]'
              }`}
            >
              <span className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition-transform ${
                autoRemove ? 'translate-x-4' : 'translate-x-0'
              }`} />
            </button>
          </label>
          <p className="text-xs text-[var(--foreground-secondary)] mt-1">{t('inbox.autoRemoveDescription')}</p>
        </div>
      </Modal>

      {/* Desktop: Project filter + actions row */}
      <div className="hidden lg:flex items-center gap-2 mb-4">
        {/* Project filter (max-width capped) */}
        <div className="shrink-0 w-[200px] max-w-[280px]">
          <MultiSelect
            options={projectOptions}
            selected={selectedProjects}
            onChange={setSelectedProjects}
            placeholder={t('inbox.allProjects')}
            searchable
            dropdownWidthClass="left-0 min-w-[340px]"
            renderOption={renderProjectOption}
          />
        </div>
        <div className="flex-1" />
        {/* Auto-hide toggle */}
        <Tooltip content={t('inbox.autoRemoveDescription')}>
          <label className="flex items-center gap-2 cursor-pointer select-none">
            <span className="text-sm text-[var(--foreground-secondary)]">{t('inbox.autoRemove')}</span>
            <button
              type="button"
              role="switch"
              aria-checked={autoRemove}
              onClick={() => setPreferenceMutation.mutate({ key: 'inbox_auto_remove', value: !autoRemove })}
              className={`relative inline-flex h-5 w-9 shrink-0 rounded-full border-2 border-transparent transition-colors ${
                autoRemove ? 'bg-[var(--primary)]' : 'bg-[var(--surface-tertiary)] dark:bg-[var(--foreground-secondary)]'
              }`}
            >
              <span className={`pointer-events-none inline-block h-4 w-4 rounded-full bg-white shadow ring-0 transition-transform ${
                autoRemove ? 'translate-x-4' : 'translate-x-0'
              }`} />
            </button>
          </label>
        </Tooltip>
        {/* Clear completed */}
        <Tooltip content={completedItems.length > 0 ? `${t('inbox.clearCompleted')} (${completedItems.length})` : t('inbox.noCompletedItems')}>
          <button
            onClick={() => clearCompletedMutation.mutate()}
            disabled={clearCompletedMutation.isPending || completedItems.length === 0}
            className={`relative p-2 rounded-lg border border-[var(--border)] transition-colors ${
              completedItems.length === 0
                ? 'opacity-40 cursor-not-allowed text-[var(--foreground-muted)]'
                : 'text-[var(--foreground-secondary)] hover:bg-[var(--surface-tertiary)] hover:text-[var(--foreground)] text-[var(--foreground-muted)] dark:hover:bg-[var(--surface)] dark:hover:text-[var(--foreground-muted)]'
            }`}
            aria-label={t('inbox.clearCompleted')}
          >
            <BrushCleaning className="h-5 w-5" />
            {completedItems.length > 0 && (
              <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
                {completedItems.length}
              </span>
            )}
          </button>
        </Tooltip>
        {/* Refresh / auto-refresh */}
        <RefreshButton
          interval={refreshInterval}
          onIntervalChange={(val) => {
            setRefreshInterval(val)
            setPreferenceMutation.mutate({ key: 'inbox_refresh_interval', value: val })
          }}
          onRefresh={() => refetch()}
          isRefreshing={isFetching}
        />
      </div>

      {/* Mobile: Search + project filter + icons */}
      <div className="flex lg:hidden items-center gap-2 mb-4">
        <div className="relative flex-1 min-w-0">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--foreground-muted)]" />
          <Input
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            placeholder={t('inbox.searchPlaceholder')}
            className="pl-10 pr-8"
            onFocus={() => setSearchFocused(true)}
            onBlur={() => setSearchFocused(false)}
            onKeyDown={(e) => { if (e.key === 'Escape') (e.target as HTMLInputElement).blur() }}
          />
          {searchInput && (
            <button
              onClick={() => setSearchInput('')}
              className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)]"
              aria-label={t('common.clear')}
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
        {/* Mobile: Project filter button */}
        <button
          onClick={() => setProjectFilterOpen(true)}
          className={`relative shrink-0 p-2 rounded-lg border transition-colors ${
            selectedProjects.length > 0
              ? 'border-[var(--primary-border)] bg-[var(--primary-muted)] text-[var(--primary)] dark:border-[var(--primary)]  dark:text-[var(--primary)]'
              : 'border-[var(--border)] text-[var(--foreground-secondary)] hover:bg-[var(--surface-tertiary)] hover:text-[var(--foreground)] text-[var(--foreground-muted)] dark:hover:bg-[var(--surface)] dark:hover:text-[var(--foreground-muted)]'
          }`}
          aria-label={t('inbox.filterByProject')}
        >
          <FolderKanban className="h-5 w-5" />
          {selectedProjects.length > 0 && (
            <span className="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-[var(--primary)] text-[10px] font-bold text-white">
              {selectedProjects.length}
            </span>
          )}
        </button>
        {/* Mobile: Clear completed */}
        <button
          onClick={() => clearCompletedMutation.mutate()}
          disabled={clearCompletedMutation.isPending || completedItems.length === 0}
          className={`relative shrink-0 p-2 rounded-lg border border-[var(--border)] transition-colors ${
            completedItems.length === 0
              ? 'opacity-40 cursor-not-allowed text-[var(--foreground-muted)]'
              : 'text-[var(--foreground-secondary)] hover:bg-[var(--surface-tertiary)] hover:text-[var(--foreground)] text-[var(--foreground-muted)] dark:hover:bg-[var(--surface)] dark:hover:text-[var(--foreground-muted)]'
          }`}
          aria-label={t('inbox.clearCompleted')}
        >
          <BrushCleaning className="h-5 w-5" />
          {completedItems.length > 0 && (
            <span className="absolute -top-1 -right-1 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
              {completedItems.length}
            </span>
          )}
        </button>
        {/* Mobile: edit toggle */}
        <button
          onClick={() => setEditing((v) => !v)}
          className={`p-2 rounded-lg border transition-colors ${
            editing
              ? 'border-[var(--primary-border)] bg-[var(--primary-muted)] text-[var(--primary)] dark:border-[var(--primary)]  dark:text-[var(--primary)]'
              : 'border-[var(--border)] text-[var(--foreground-secondary)] hover:bg-[var(--surface-tertiary)] hover:text-[var(--foreground)] text-[var(--foreground-muted)] dark:hover:bg-[var(--surface)] dark:hover:text-[var(--foreground-muted)]'
          }`}
          aria-label={t('common.edit')}
        >
          <Pencil className="h-5 w-5" />
        </button>
        {/* Mobile: settings button */}
        <button
          onClick={() => setSettingsOpen(true)}
          className="p-2 rounded-lg border border-[var(--border)] text-[var(--foreground-secondary)] hover:bg-[var(--surface-tertiary)] hover:text-[var(--foreground)] text-[var(--foreground-muted)] dark:hover:bg-[var(--surface)] dark:hover:text-[var(--foreground-muted)] transition-colors"
          aria-label={t('inbox.settings')}
        >
          <Settings className="h-5 w-5" />
        </button>
        {/* Mobile: Refresh (last) */}
        <RefreshButton
          interval={refreshInterval}
          onIntervalChange={(val) => {
            setRefreshInterval(val)
            setPreferenceMutation.mutate({ key: 'inbox_refresh_interval', value: val })
          }}
          onRefresh={() => refetch()}
          isRefreshing={isFetching}
        />
      </div>

      {/* Mobile: Project filter modal */}
      <Modal open={projectFilterOpen} onClose={() => setProjectFilterOpen(false)} title={t('inbox.filterByProject')} position="top" containerClassName="!pt-[10.3rem]">
        <MobileProjectFilterContent
          projectOptions={projectOptions}
          selectedProjects={selectedProjects}
          setSelectedProjects={setSelectedProjects}
          projectMap={projectMap}
          showNamespaces={showNamespaces}
        />
      </Modal>

      {allItems.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-16 text-[var(--foreground-secondary)]">
          <Inbox className="h-16 w-16 mb-4 opacity-30" />
          <p className="text-lg font-medium">{t('inbox.empty')}</p>
          <p className="text-sm mt-1">{t('inbox.emptyHint')}</p>
        </div>
      ) : (
        <>
          {/* Desktop table */}
          <div className="hidden lg:block overflow-hidden rounded-lg border border-[var(--border)]">
            <table className="w-full table-fixed">
              <thead className="bg-[var(--surface-secondary)] group/thead">
                <tr>
                  <th className="w-10 px-1 py-3"></th>
                  <th style={columnWidths.display_id ? { width: columnWidths.display_id } : undefined} className={`relative px-3 py-3 text-left text-xs font-medium text-[var(--foreground-secondary)] uppercase ${columnWidths.display_id ? '' : 'w-24'}`}>
                    {t('workitems.table.id')}
                    <div className="absolute right-0.5 top-0 bottom-0 w-1.5 cursor-col-resize opacity-0 group-hover/thead:opacity-100 bg-[var(--primary)]/40 hover:!bg-[var(--primary-hover)]/60 active:!bg-[var(--primary-muted)]/60 dark:bg-[var(--primary-muted)]/30 hover:!bg-[var(--primary-hover)]/50 dark:active:!bg-[var(--primary-muted)]/50 transition-opacity z-10" onMouseDown={(e) => handleResizeStart(e, 'display_id')} onDoubleClick={(e) => { e.stopPropagation(); resetColumnWidth('display_id') }} />
                  </th>
                  <th style={columnWidths.type ? { width: columnWidths.type } : undefined} className={`relative px-3 py-3 text-left text-xs font-medium text-[var(--foreground-secondary)] uppercase ${columnWidths.type ? '' : 'w-20'}`}>
                    {t('workitems.table.type')}
                    <div className="absolute right-0.5 top-0 bottom-0 w-1.5 cursor-col-resize opacity-0 group-hover/thead:opacity-100 bg-[var(--primary)]/40 hover:!bg-[var(--primary-hover)]/60 active:!bg-[var(--primary-muted)]/60 dark:bg-[var(--primary-muted)]/30 hover:!bg-[var(--primary-hover)]/50 dark:active:!bg-[var(--primary-muted)]/50 transition-opacity z-10" onMouseDown={(e) => handleResizeStart(e, 'type')} onDoubleClick={(e) => { e.stopPropagation(); resetColumnWidth('type') }} />
                  </th>
                  <th className="px-3 py-3 text-left text-xs font-medium text-[var(--foreground-secondary)] uppercase">{t('workitems.table.title')}</th>
                  <th style={columnWidths.status ? { width: columnWidths.status } : undefined} className={`relative px-3 py-3 text-left text-xs font-medium text-[var(--foreground-secondary)] uppercase ${columnWidths.status ? '' : 'w-28'}`}>
                    {t('workitems.table.status')}
                    <div className="absolute right-0.5 top-0 bottom-0 w-1.5 cursor-col-resize opacity-0 group-hover/thead:opacity-100 bg-[var(--primary)]/40 hover:!bg-[var(--primary-hover)]/60 active:!bg-[var(--primary-muted)]/60 dark:bg-[var(--primary-muted)]/30 hover:!bg-[var(--primary-hover)]/50 dark:active:!bg-[var(--primary-muted)]/50 transition-opacity z-10" onMouseDown={(e) => handleResizeStart(e, 'status')} onDoubleClick={(e) => { e.stopPropagation(); resetColumnWidth('status') }} />
                  </th>
                  <th style={columnWidths.priority ? { width: columnWidths.priority } : undefined} className={`relative px-3 py-3 text-left text-xs font-medium text-[var(--foreground-secondary)] uppercase ${columnWidths.priority ? '' : 'w-24'}`}>
                    {t('workitems.table.priority')}
                    <div className="absolute right-0.5 top-0 bottom-0 w-1.5 cursor-col-resize opacity-0 group-hover/thead:opacity-100 bg-[var(--primary)]/40 hover:!bg-[var(--primary-hover)]/60 active:!bg-[var(--primary-muted)]/60 dark:bg-[var(--primary-muted)]/30 hover:!bg-[var(--primary-hover)]/50 dark:active:!bg-[var(--primary-muted)]/50 transition-opacity z-10" onMouseDown={(e) => handleResizeStart(e, 'priority')} onDoubleClick={(e) => { e.stopPropagation(); resetColumnWidth('priority') }} />
                  </th>
                  <th style={columnWidths.sla ? { width: columnWidths.sla } : undefined} className={`relative px-3 py-3 text-left text-xs font-medium text-[var(--foreground-secondary)] uppercase ${columnWidths.sla ? '' : 'w-[110px]'}`}>
                    {t('sla.columnHeader')}
                    <div className="absolute right-0.5 top-0 bottom-0 w-1.5 cursor-col-resize opacity-0 group-hover/thead:opacity-100 bg-[var(--primary)]/40 hover:!bg-[var(--primary-hover)]/60 active:!bg-[var(--primary-muted)]/60 dark:bg-[var(--primary-muted)]/30 hover:!bg-[var(--primary-hover)]/50 dark:active:!bg-[var(--primary-muted)]/50 transition-opacity z-10" onMouseDown={(e) => handleResizeStart(e, 'sla')} onDoubleClick={(e) => { e.stopPropagation(); resetColumnWidth('sla') }} />
                  </th>
                  <th style={columnWidths.updated ? { width: columnWidths.updated } : undefined} className={`relative px-3 py-3 text-right text-xs font-medium text-[var(--foreground-secondary)] uppercase ${columnWidths.updated ? '' : 'w-[130px]'}`}>
                    {t('workitems.table.updated')}
                    <div className="absolute right-0.5 top-0 bottom-0 w-1.5 cursor-col-resize opacity-0 group-hover/thead:opacity-100 bg-[var(--primary)]/40 hover:!bg-[var(--primary-hover)]/60 active:!bg-[var(--primary-muted)]/60 dark:bg-[var(--primary-muted)]/30 hover:!bg-[var(--primary-hover)]/50 dark:active:!bg-[var(--primary-muted)]/50 transition-opacity z-10" onMouseDown={(e) => handleResizeStart(e, 'updated')} onDoubleClick={(e) => { e.stopPropagation(); resetColumnWidth('updated') }} />
                  </th>
                  <th className="w-10 px-2 py-3"></th>
                </tr>
              </thead>
              <tbody>
                {allItems.map((item, index) => (
                  <InboxRow
                    key={item.id}
                    item={item}
                    isCompleted={strikethroughEnabled && (item.status_category === 'done' || item.status_category === 'cancelled')}
                    isFirst={index === 0}
                    isLast={index === allItems.length - 1}
                    isActive={index === activeRow}
                    onRemove={handleRemove}
                    onMoveUp={handleMoveUp}
                    onMoveDown={handleMoveDown}
                    onClick={handleClick}
                    removedId={removedId}
                    reorderedId={reorderedId}
                    autoRemove={autoRemove}
                    columnWidths={columnWidths}
                  />
                ))}
              </tbody>
            </table>
          </div>

          {/* Mobile cards */}
          <div className="lg:hidden space-y-2">
            {allItems.map((item, index) => (
              <InboxCard
                key={item.id}
                item={item}
                isCompleted={strikethroughEnabled && (item.status_category === 'done' || item.status_category === 'cancelled')}
                isFirst={index === 0}
                isLast={index === allItems.length - 1}
                isActive={index === activeRow}
                editing={editing}
                onRemove={handleRemove}
                onMoveUp={handleMoveUp}
                onMoveDown={handleMoveDown}
                onClick={handleClick}
                removedId={removedId}
                reorderedId={reorderedId}
                autoRemove={autoRemove}
              />
            ))}
          </div>
        </>
      )}

      {/* Load more */}
      {hasMore && (
        <div className="flex justify-center pt-4">
          <Button
            variant="secondary"
            onClick={async () => {
              const lastItem = allItems[allItems.length - 1]
              if (!lastItem) return
              const next = await listInboxItems({
                search: debouncedSearch || undefined,
                include_completed: true,
                project: selectedProjects.length > 0 ? selectedProjects : undefined,
                cursor: lastItem.id,
              })
              setLoadedPages((prev) => [...prev, next.items])
            }}
          >
            {t('common.loadMore')}
          </Button>
        </div>
      )}

      {/* Remove confirmation modal */}
      <Modal open={!!removeConfirmItem} onClose={() => setRemoveConfirmItem(null)} title={t('inbox.removeConfirmTitle')}>
        <form onSubmit={(e) => {
          e.preventDefault()
          if (removeConfirmItem) {
            if (dontShowAgainChecked) {
              setPreferenceMutation.mutate({ key: 'inbox_skip_remove_confirm', value: true })
            }
            handleRemove(removeConfirmItem.id)
            setRemoveConfirmItem(null)
          }
        }}>
          <p className="text-sm text-[var(--foreground-secondary)] mb-4">
            <Trans i18nKey="inbox.removeConfirmBody" values={{ displayId: removeConfirmItem?.display_id ?? '' }} components={{ bold: <strong /> }} />
          </p>
          <label className="flex items-center gap-2 mb-4 cursor-pointer select-none">
            <input
              type="checkbox"
              checked={dontShowAgainChecked}
              onChange={(e) => setDontShowAgainChecked(e.target.checked)}
            />
            <span className="text-sm text-[var(--foreground-secondary)]">{t('inbox.dontShowAgain')}</span>
          </label>
          <div className="flex justify-end gap-3">
            <Button type="button" variant="secondary" onClick={() => setRemoveConfirmItem(null)}>{t('common.cancel')}</Button>
            <Button type="submit" variant="danger" autoFocus>{t('common.remove')}</Button>
          </div>
        </form>
      </Modal>

      <CreateWorkItemModal
        open={showCreate}
        onClose={() => setShowCreate(false)}
        onCreated={(workItemId) => addToInboxMutation.mutate(workItemId)}
      />
    </div>
  )
}

// --- Feed Page (placeholder) ---

function FeedPage() {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center justify-center h-64 text-[var(--foreground-secondary)]">
      <Rss className="h-12 w-12 mb-4 opacity-30" />
      <p className="text-lg font-medium">{t('user.feedComingSoon')}</p>
    </div>
  )
}

// --- Mobile Project Filter (with search) ---

function MobileProjectFilterContent({ projectOptions, selectedProjects, setSelectedProjects, projectMap, showNamespaces }: {
  projectOptions: MultiSelectOption[]
  selectedProjects: string[]
  setSelectedProjects: (val: string[] | ((prev: string[]) => string[])) => void
  projectMap: Map<string, import('@/api/projects').Project>
  showNamespaces: boolean
}) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    setTimeout(() => searchRef.current?.focus(), 0)
  }, [])

  const filtered = useMemo(() => {
    if (!search) return projectOptions
    const lower = search.toLowerCase()
    return projectOptions.filter((o) => o.label.toLowerCase().includes(lower))
  }, [projectOptions, search])

  return (
    <div>
      <div className="pb-2">
        <Input
          ref={searchRef}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder={t('common.search')}
          className="text-sm"
        />
      </div>
      <div className="flex items-center gap-2 px-1 pb-2 border-b border-[var(--border)]">
        <button type="button" className="text-xs text-[var(--primary)] hover:text-[var(--primary-hover)]" onClick={() => setSelectedProjects(filtered.map((o) => o.value))}>
          {t('common.all')}
        </button>
        <button type="button" className="ml-auto text-xs text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)]" onClick={() => setSelectedProjects([])}>
          {t('common.none')}
        </button>
      </div>
      <div className="max-h-60 overflow-y-auto space-y-1 pt-1">
        {filtered.length === 0 ? (
          <p className="text-sm text-[var(--foreground-muted)] py-2">{t('common.noResults')}</p>
        ) : (
          filtered.map((opt) => {
            const p = projectMap.get(opt.value)
            return (
              <label key={opt.value} className="flex items-center gap-2 px-1 py-2 hover:bg-[var(--surface-hover)] cursor-pointer text-sm text-[var(--foreground)] rounded">
                <input
                  type="checkbox"
                  checked={selectedProjects.includes(opt.value)}
                  onChange={() => setSelectedProjects((prev: string[]) =>
                    prev.includes(opt.value) ? prev.filter((v) => v !== opt.value) : [...prev, opt.value]
                  )}
                  className="rounded border-[var(--border)] text-[var(--primary)] focus:ring-[var(--focus-ring)] shrink-0"
                />
                <ProjectKeyBadge size="icon">{opt.value}</ProjectKeyBadge>
                <span className="truncate">{p?.name ?? opt.label}</span>
                {showNamespaces && p?.namespace_slug && (
                  <span className="ml-auto flex items-center gap-1 text-[0.7rem] text-[var(--foreground-muted)] shrink-0">
                    <span>{p.namespace_slug}</span>
                    <NamespaceIcon icon={p.namespace_icon ?? 'building2'} color={p.namespace_color ?? 'slate'} className="h-3 w-3" />
                  </span>
                )}
              </label>
            )
          })
        )}
      </div>
    </div>
  )
}

// --- Layout Page ---

export default function UserPage() {
  const { collapsed } = useSidebar('app')
  const { containerClass } = useLayout()

  return (
    <div className={`${containerClass(true)} py-6 h-full flex flex-col`}>
      <div className={`flex flex-1 min-h-0 transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <AppSidebar />
        <div className="flex-1 min-w-0">
          <Routes>
            <Route path="inbox" element={<InboxListPage />} />
            <Route path="feed" element={<FeedPage />} />
            <Route path="watchlist" element={<WatchlistPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}
