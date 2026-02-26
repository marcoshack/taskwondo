import { useState, useMemo, useEffect, useCallback, useRef } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { useWorkItems, useCreateWorkItem, useBulkUpdateWorkItems, useDeleteWorkItem, type BulkUpdateResult } from '@/hooks/useWorkItems'
import { useProject, useMembers } from '@/hooks/useProjects'
import { useProjectWorkflow, useAvailableStatuses } from '@/hooks/useWorkflows'
import { useMilestones } from '@/hooks/useMilestones'
import { useUserSetting, useSetUserSetting } from '@/hooks/useUserSettings'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useDebounce } from '@/hooks/useDebounce'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Button } from '@/components/ui/Button'
import { Select } from '@/components/ui/Select'
import { Spinner } from '@/components/ui/Spinner'
import { Modal } from '@/components/ui/Modal'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { WorkItemFilters } from '@/components/workitems/WorkItemFilters'
import { WorkItemForm } from '@/components/workitems/WorkItemForm'
import { BoardView } from '@/components/workitems/BoardView'
import { SLAIndicator } from '@/components/SLAIndicator'
import { Tooltip } from '@/components/ui/Tooltip'
import { formatRelativeTime } from '@/utils/duration'
import { User, History, Check, X, LayoutList, LayoutGrid } from 'lucide-react'
import { Input } from '@/components/ui/Input'
import { useAuth } from '@/contexts/AuthContext'
import { useAddToInbox, useInboxItems } from '@/hooks/useInbox'
import { useSavedSearches, useCreateSavedSearch, useUpdateSavedSearch, useDeleteSavedSearch } from '@/hooks/useSavedSearches'
import { SavedSearchSelector } from '@/components/workitems/SavedSearchSelector'
import { SaveSearchModal } from '@/components/workitems/SaveSearchModal'
import { InboxButton } from '@/components/workitems/InboxButton'
import type { AxiosError } from 'axios'
import type { SavedSearch } from '@/api/savedSearches'
import { listWorkItems, type WorkItem, type WorkItemFilter } from '@/api/workitems'

type ViewMode = 'list' | 'board'

const SETTINGS_KEY = 'workitem_filters'
const VIEW_STATE_KEY = 'workitem_view_state'
const closedCategories = new Set(['done', 'cancelled'])

/** Filter keys synced to URL and persisted. */
const FILTER_PARAMS = ['type', 'status', 'priority', 'assignee', 'milestone'] as const
type FilterParam = typeof FILTER_PARAMS[number]

type SavedFilter = Pick<WorkItemFilter, FilterParam>

/** Full view state persisted to user settings — survives navigation. */
interface ViewState {
  filter: SavedFilter
  search: string
  viewMode: ViewMode
  sort: string
  order: 'asc' | 'desc'
  activeSearchId: string | null
  activeSearchSnapshot: { filter: SavedFilter; search: string; viewMode: ViewMode } | null
}

function pickSavedFilter(f: WorkItemFilter): SavedFilter {
  return { type: f.type, status: f.status, priority: f.priority, assignee: f.assignee, milestone: f.milestone }
}

/** Check if URL has any filter/search/view params. */
function urlHasFilterParams(sp: URLSearchParams): boolean {
  for (const key of FILTER_PARAMS) {
    if (sp.has(key)) return true
  }
  return sp.has('q') || sp.has('view') || sp.has('sort')
}

/** Parse filter state from URL search params. */
function parseUrlFilter(sp: URLSearchParams): SavedFilter {
  const filter: SavedFilter = {}
  for (const key of FILTER_PARAMS) {
    const val = sp.get(key)
    if (val) filter[key] = val.split(',')
  }
  return filter
}

/** Build URL search params from filter, search, view, and sort. */
function buildUrlParams(filter: WorkItemFilter, search: string, view: ViewMode, sort: string, order: string): URLSearchParams {
  const sp = new URLSearchParams()
  for (const key of FILTER_PARAMS) {
    const val = filter[key]
    if (val?.length) sp.set(key, val.join(','))
  }
  if (search) sp.set('q', search)
  if (view !== 'list') sp.set('view', view)
  if (sort !== 'created_at' || order !== 'desc') {
    sp.set('sort', sort)
    sp.set('order', order)
  }
  return sp
}

function WorkItemSearchBar({ search, onSearchChange }: { search: string; onSearchChange: (v: string) => void }) {
  const { t } = useTranslation()
  const searchRef = useRef<HTMLInputElement>(null)
  useKeyboardShortcut({ key: '/' }, () => searchRef.current?.focus())
  return (
    <div className="relative">
      <Input
        ref={searchRef}
        placeholder={t('workitems.filters.search')}
        value={search}
        onChange={(e) => onSearchChange(e.target.value)}
        onKeyDown={(e) => { if (e.key === 'Escape') searchRef.current?.blur() }}
        className="!h-[39px] pr-8"
      />
      {search && (
        <button
          onClick={() => { onSearchChange(''); searchRef.current?.focus() }}
          className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          aria-label={t('common.clear')}
        >
          <X className="h-4 w-4" />
        </button>
      )}
    </div>
  )
}

function getDescriptionPreview(description: string): string {
  const line = description.split('\n').find(l => l.trim() !== '')
  if (!line) return ''
  return line.trim().replace(/^#+\s+/, '').replace(/[*_~`[\]]/g, '')
}

export function WorkItemListPage() {
  const { t } = useTranslation()
  const { projectKey } = useParams<{ projectKey: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const { user } = useAuth()
  const { statuses, transitionsMap } = useProjectWorkflow(projectKey ?? '')
  const { data: allStatuses } = useAvailableStatuses(projectKey ?? '')
  const { data: project } = useProject(projectKey ?? '')
  const { data: members } = useMembers(projectKey ?? '')
  const { data: milestones } = useMilestones(projectKey ?? '')

  const currentUserRole = members?.find((m) => m.user_id === user?.id)?.role ?? (user?.global_role === 'admin' ? 'owner' : null)
  const canEdit = user?.global_role === 'admin' || (currentUserRole != null && currentUserRole !== 'viewer')
  const readOnly = !canEdit

  // Load saved view state from user settings (full state: filter + search + view + sort + active search)
  const { data: savedViewState, isLoading: settingsLoading } = useUserSetting<ViewState>(projectKey ?? '', VIEW_STATE_KEY)
  // Legacy: load old filter-only setting as fallback for migration
  const { data: savedFilterLegacy } = useUserSetting<SavedFilter>(projectKey ?? '', SETTINGS_KEY)
  const saveMutation = useSetUserSetting(projectKey ?? '')

  // Show dates preference (global, persisted)
  const { data: showDatesPref } = usePreference<boolean>('showDates')
  const showDates = showDatesPref ?? true
  const setPreferenceMutation = useSetPreference()

  // Compute default open statuses from ALL workflows (exclude done/cancelled)
  const defaultOpenStatuses = useMemo(() => {
    if (!allStatuses?.length) return undefined
    const names = new Set(allStatuses.filter((s) => !closedCategories.has(s.category)).map((s) => s.name))
    return Array.from(names)
  }, [allStatuses])

  // Capture initial URL state once (before any effects run)
  const initialUrlRef = useRef<{ hasParams: boolean; filter: SavedFilter; search: string; view: ViewMode; sort: string; order: 'asc' | 'desc' } | null>(null)
  if (initialUrlRef.current === null) {
    const hasParams = urlHasFilterParams(searchParams)
    initialUrlRef.current = {
      hasParams,
      filter: parseUrlFilter(searchParams),
      search: searchParams.get('q') ?? '',
      view: (searchParams.get('view') === 'board' ? 'board' : 'list') as ViewMode,
      sort: searchParams.get('sort') ?? 'created_at',
      order: (searchParams.get('order') === 'asc' ? 'asc' : 'desc'),
    }
  }

  // Track whether the initial load was from a shared URL (don't save on init).
  // If we have URL params AND a saved view state with the same filters, treat it as a reload
  // (not a shared URL) — this lets us restore activeSearchId while still preventing save-on-init.
  const loadedFromUrlRef = useRef(initialUrlRef.current.hasParams)

  const [filter, setFilter] = useState<WorkItemFilter>(
    initialUrlRef.current.hasParams ? initialUrlRef.current.filter : {}
  )
  const [filterInitialized, setFilterInitialized] = useState(initialUrlRef.current.hasParams)
  const [search, setSearch] = useState(initialUrlRef.current.search)
  const [viewMode, setViewMode] = useState<ViewMode>(initialUrlRef.current.view)
  const [sort, setSort] = useState(initialUrlRef.current.sort)
  const [order, setOrder] = useState<'asc' | 'desc'>(initialUrlRef.current.order)
  const [activeSearchId, setActiveSearchId] = useState<string | null>(null)
  const [activeSearchSnapshot, setActiveSearchSnapshot] = useState<{ filter: SavedFilter; search: string; viewMode: ViewMode } | null>(null)

  // If no URL params, initialize from saved view state or defaults once data loads
  useEffect(() => {
    if (filterInitialized || settingsLoading) return
    if (!allStatuses?.length) return

    const vs = savedViewState
    const legacyFilter = savedFilterLegacy

    if (vs) {
      // Restore full view state
      setFilter(vs.filter)
      if (vs.search) setSearch(vs.search)
      if (vs.viewMode) setViewMode(vs.viewMode)
      if (vs.sort) setSort(vs.sort)
      if (vs.order) setOrder(vs.order)
      if (vs.activeSearchId) setActiveSearchId(vs.activeSearchId)
      if (vs.activeSearchSnapshot) setActiveSearchSnapshot(vs.activeSearchSnapshot)
    } else if (legacyFilter) {
      // Migrate from legacy filter-only setting
      setFilter(legacyFilter)
    } else if (defaultOpenStatuses) {
      setFilter({ status: defaultOpenStatuses })
    }
    setFilterInitialized(true)
  }, [savedViewState, savedFilterLegacy, settingsLoading, defaultOpenStatuses, allStatuses, filterInitialized])

  // When page loads with URL params (reload), restore activeSearchId from saved view state.
  // The main init effect above skips when filterInitialized=true (URL case), but we still
  // need the active search context so the selector shows the correct name.
  const searchIdRestoredRef = useRef(false)
  useEffect(() => {
    if (searchIdRestoredRef.current || !savedViewState || settingsLoading) return
    if (initialUrlRef.current?.hasParams && savedViewState.activeSearchId) {
      setActiveSearchId(savedViewState.activeSearchId)
      if (savedViewState.activeSearchSnapshot) setActiveSearchSnapshot(savedViewState.activeSearchSnapshot)
      // This is a reload, not a shared URL — allow saving view state changes
      loadedFromUrlRef.current = false
    }
    searchIdRestoredRef.current = true
  }, [savedViewState, settingsLoading])

  // Sync URL when filter initializes from settings/defaults (non-URL case)
  const urlSyncedRef = useRef(initialUrlRef.current.hasParams)
  useEffect(() => {
    if (!filterInitialized || urlSyncedRef.current) return
    setSearchParams(buildUrlParams(filter, search, viewMode, sort, order), { replace: true })
    urlSyncedRef.current = true
  }, [filterInitialized, filter, search, viewMode, sort, order, setSearchParams])

  // Save full view state (debounced) — only on manual user changes
  const saveTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const saveMutationRef = useRef(saveMutation)
  saveMutationRef.current = saveMutation
  // Use refs so saveViewState always captures the latest values without re-creating
  const stateRefs = useRef({ search, viewMode, sort, order, activeSearchId, activeSearchSnapshot })
  stateRefs.current = { search, viewMode, sort, order, activeSearchId, activeSearchSnapshot }

  const saveViewState = useCallback((f: WorkItemFilter) => {
    if (!projectKey || !filterInitialized || loadedFromUrlRef.current) return
    clearTimeout(saveTimerRef.current)
    saveTimerRef.current = setTimeout(() => {
      const s = stateRefs.current
      const vs: ViewState = {
        filter: pickSavedFilter(f),
        search: s.search,
        viewMode: s.viewMode,
        sort: s.sort,
        order: s.order as 'asc' | 'desc',
        activeSearchId: s.activeSearchId,
        activeSearchSnapshot: s.activeSearchSnapshot,
      }
      saveMutationRef.current.mutate({ key: VIEW_STATE_KEY, value: vs })
    }, 500)
  }, [projectKey, filterInitialized])

  // Alias for backward compat within this file
  const saveFilter = saveViewState

  function syncUrl(f: WorkItemFilter, q: string, v: ViewMode, s: string = sort, o: 'asc' | 'desc' = order) {
    setSearchParams(buildUrlParams(f, q, v, s, o), { replace: true })
  }

  function handleFilterChange(f: WorkItemFilter) {
    setFilter(f)
    syncUrl(f, search, viewMode)
    saveFilter(f)
  }

  function handleSearchChange(q: string) {
    setSearch(q)
    syncUrl(filter, q, viewMode)
    saveFilter(filter)
  }

  function handleViewChange(v: ViewMode) {
    setViewMode(v)
    syncUrl(filter, search, v)
    saveFilter(filter)
  }

  function handleSort(sortKey: string) {
    let newOrder: 'asc' | 'desc'
    if (sort === sortKey) {
      newOrder = order === 'asc' ? 'desc' : 'asc'
    } else {
      newOrder = ['title', 'type', 'status', 'sla_target_at'].includes(sortKey) ? 'asc' : 'desc'
    }
    setSort(sortKey)
    setOrder(newOrder)
    syncUrl(filter, search, viewMode, sortKey, newOrder)
    saveFilter(filter)
  }

  function handleOrderChange(newOrder: 'asc' | 'desc') {
    setOrder(newOrder)
    syncUrl(filter, search, viewMode, sort, newOrder)
    saveFilter(filter)
  }

  const [showCreate, setShowCreate] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [bulkError, setBulkError] = useState<string | null>(null)
  const [activeRow, setActiveRow] = useState(-1)
  const activeRowStorageKey = `taskwondo_activeRow_${projectKey}`
  const filterStorageKey = `taskwondo_listParams_${projectKey}`
  /** Build current params from state (avoids stale searchParams from useSearchParams). */
  const currentParamsString = useCallback(
    () => buildUrlParams(filter, search, viewMode, sort, order).toString(),
    [filter, search, viewMode, sort, order],
  )
  const restoredRef = useRef(false)

  // --- Saved searches ---
  const { data: savedSearchList } = useSavedSearches(projectKey ?? '')
  const createSearchMutation = useCreateSavedSearch(projectKey ?? '')
  const updateSearchMutation = useUpdateSavedSearch(projectKey ?? '')
  const deleteSearchMutation = useDeleteSavedSearch(projectKey ?? '')
  const [showSaveModal, setShowSaveModal] = useState(false)

  const canManageShared = currentUserRole === 'owner' || currentUserRole === 'admin' || user?.global_role === 'admin'

  const activeSearch = useMemo(
    () => savedSearchList?.find((s) => s.id === activeSearchId) ?? null,
    [savedSearchList, activeSearchId],
  )

  const hasUnsavedChanges = useMemo(() => {
    if (!activeSearchId || !activeSearchSnapshot) return false
    return JSON.stringify(pickSavedFilter(filter)) !== JSON.stringify(activeSearchSnapshot.filter)
      || search !== (activeSearchSnapshot.search ?? '')
      || viewMode !== activeSearchSnapshot.viewMode
  }, [activeSearchId, activeSearchSnapshot, filter, search, viewMode])

  function handleSelectSearch(ss: SavedSearch) {
    const f: WorkItemFilter = {}
    if (ss.filters.type?.length) f.type = ss.filters.type
    if (ss.filters.status?.length) f.status = ss.filters.status
    if (ss.filters.priority?.length) f.priority = ss.filters.priority
    if (ss.filters.assignee?.length) f.assignee = ss.filters.assignee
    if (ss.filters.milestone?.length) f.milestone = ss.filters.milestone

    setFilter(f)
    setSearch(ss.filters.q ?? '')
    setViewMode(ss.view_mode as ViewMode)
    syncUrl(f, ss.filters.q ?? '', ss.view_mode as ViewMode)
    setActiveSearchId(ss.id)
    setActiveSearchSnapshot({ filter: pickSavedFilter(f), search: ss.filters.q ?? '', viewMode: ss.view_mode as ViewMode })
    loadedFromUrlRef.current = false
    saveFilter(f)
  }

  function handleSaveNew(name: string, shared: boolean) {
    createSearchMutation.mutate({
      name,
      filters: { ...pickSavedFilter(filter), q: search || undefined },
      view_mode: viewMode,
      shared,
    }, {
      onSuccess: (saved) => {
        setActiveSearchId(saved.id)
        setActiveSearchSnapshot({ filter: pickSavedFilter(filter), search, viewMode })
        setShowSaveModal(false)
        saveFilter(filter)
      },
    })
  }

  function handleUpdateExisting() {
    if (!activeSearchId) return
    updateSearchMutation.mutate({
      searchId: activeSearchId,
      input: {
        filters: { ...pickSavedFilter(filter), q: search || undefined },
        view_mode: viewMode,
      },
    }, {
      onSuccess: () => {
        setActiveSearchSnapshot({ filter: pickSavedFilter(filter), search, viewMode })
        setShowSaveModal(false)
        saveFilter(filter)
      },
    })
  }

  function handleRenameSearch(ss: SavedSearch, newName: string) {
    updateSearchMutation.mutate({ searchId: ss.id, input: { name: newName } })
  }

  function handleDeleteSearch(ss: SavedSearch) {
    deleteSearchMutation.mutate(ss.id, {
      onSuccess: () => {
        if (activeSearchId === ss.id) {
          setActiveSearchId(null)
          setActiveSearchSnapshot(null)
          saveFilter(filter)
        }
      },
    })
  }

  function handleClearFilters() {
    const f: WorkItemFilter = defaultOpenStatuses ? { status: defaultOpenStatuses } : {}
    setFilter(f)
    setSearch('')
    setViewMode('list')
    setSort('created_at')
    setOrder('desc')
    setActiveSearchId(null)
    setActiveSearchSnapshot(null)
    syncUrl(f, '', 'list', 'created_at', 'desc')
    loadedFromUrlRef.current = false
    // Persist cleared state so it survives navigation
    saveFilter(f)
  }

  useKeyboardShortcut({ key: 'c' }, () => setShowCreate(true), canEdit)

  const debouncedSearch = useDebounce(search, 300)

  const activeFilter = useMemo(() => ({
    ...filter,
    q: debouncedSearch || undefined,
    sort,
    order,
    limit: viewMode === 'board' ? 200 : 50,
  }), [filter, debouncedSearch, viewMode, sort, order])

  const { data: result, isLoading } = useWorkItems(projectKey ?? '', activeFilter)
  const createMutation = useCreateWorkItem(projectKey ?? '')
  const deleteMutation = useDeleteWorkItem(projectKey ?? '')
  const bulkMutation = useBulkUpdateWorkItems(projectKey ?? '')
  const items = result?.data ?? []

  function toggleSelect(itemNumber: number) {
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(itemNumber)) next.delete(itemNumber)
      else next.add(itemNumber)
      return next
    })
  }

  function toggleSelectAll() {
    if (selected.size === items.length) {
      setSelected(new Set())
    } else {
      setSelected(new Set(items.map((i) => i.item_number)))
    }
  }

  function handleBulkResult(result: BulkUpdateResult) {
    if (result.failed.length === 0) {
      setSelected(new Set())
      setBulkError(null)
      return
    }
    // Keep only failed items selected
    setSelected(new Set(result.failed.map((f) => f.itemNumber)))
    // Build display IDs for failed items
    const displayIds = result.failed.map((f) => {
      const item = allItems.find((i) => i.item_number === f.itemNumber)
      return item?.display_id ?? `#${f.itemNumber}`
    })
    setBulkError(t('workitems.bulk.transitionError', { items: displayIds.join(', ') }))
  }

  function handleBulkStatus(status: string) {
    if (!status) return
    setBulkError(null)
    const updates = Array.from(selected).map((itemNumber) => ({ itemNumber, input: { status } }))
    bulkMutation.mutate(updates, { onSuccess: handleBulkResult })
  }

  function handleBulkAssign(value: string) {
    if (!value) return
    setBulkError(null)
    const input = value === 'unassign' ? { assignee_id: null } : { assignee_id: value }
    const updates = Array.from(selected).map((itemNumber) => ({ itemNumber, input }))
    bulkMutation.mutate(updates, { onSuccess: handleBulkResult })
  }

  function handleBulkMilestone(value: string) {
    if (!value) return
    setBulkError(null)
    const input = value === 'none' ? { milestone_id: null } : { milestone_id: value }
    const updates = Array.from(selected).map((itemNumber) => ({ itemNumber, input }))
    bulkMutation.mutate(updates, { onSuccess: handleBulkResult })
  }

  const [loadedPages, setLoadedPages] = useState<WorkItem[][]>([])

  const allItems = useMemo(() => {
    if (loadedPages.length === 0) return items
    return [...items, ...loadedPages.flat()]
  }, [items, loadedPages])

  const filterKey = JSON.stringify(activeFilter)
  const [prevFilterKey, setPrevFilterKey] = useState(filterKey)
  if (filterKey !== prevFilterKey) {
    setPrevFilterKey(filterKey)
    setLoadedPages([])
    setSelected(new Set())
    setActiveRow(-1)
  }

  // Restore active row from sessionStorage after navigating back
  useEffect(() => {
    if (restoredRef.current || allItems.length === 0) return
    const stored = sessionStorage.getItem(activeRowStorageKey)
    if (stored) {
      const itemNumber = parseInt(stored, 10)
      const idx = allItems.findIndex((i) => i.item_number === itemNumber)
      if (idx >= 0) setActiveRow(idx)
      sessionStorage.removeItem(activeRowStorageKey)
    }
    restoredRef.current = true
  }, [allItems, activeRowStorageKey])

  // List navigation: arrows + j/k (vim), o/Enter to open, x to select, # to delete, Escape to deselect
  useKeyboardShortcut([{ key: 'ArrowDown' }, { key: 'j' }], () => setActiveRow((prev) => Math.min(prev + 1, allItems.length - 1)), viewMode === 'list')
  useKeyboardShortcut([{ key: 'ArrowUp' }, { key: 'k' }], () => setActiveRow((prev) => Math.max(prev - 1, 0)), viewMode === 'list')
  useKeyboardShortcut([{ key: 'Enter' }, { key: 'o' }], () => {
    if (activeRow >= 0 && activeRow < allItems.length) {
      sessionStorage.setItem(activeRowStorageKey, String(allItems[activeRow].item_number))
      sessionStorage.setItem(filterStorageKey, currentParamsString())
      navigate(`/projects/${projectKey}/items/${allItems[activeRow].item_number}`)
    }
  }, activeRow >= 0)
  useKeyboardShortcut({ key: 'x' }, () => {
    if (activeRow >= 0 && activeRow < allItems.length) {
      toggleSelect(allItems[activeRow].item_number)
    }
  }, canEdit && viewMode === 'list' && activeRow >= 0)
  useKeyboardShortcut({ key: '#' }, () => setShowDeleteConfirm(true), canEdit && (selected.size > 0 || activeRow >= 0))

  // Inbox: track which work items are already in inbox
  const { data: inboxData } = useInboxItems()
  const inboxByWorkItemId = useMemo(() => {
    const map = new Map<string, string>()
    if (inboxData?.items) {
      for (const item of inboxData.items) map.set(item.work_item_id, item.id)
    }
    return map
  }, [inboxData])

  // 'i' shortcut: send active row to inbox
  const addToInboxMutation = useAddToInbox()
  const [inboxSavedId, setInboxSavedId] = useState<string | null>(null)
  useKeyboardShortcut({ key: 'i' }, () => {
    if (activeRow >= 0 && activeRow < allItems.length) {
      const item = allItems[activeRow]
      addToInboxMutation.mutate(item.id, {
        onSuccess: () => {
          setInboxSavedId(item.id)
          setTimeout(() => setInboxSavedId(null), 2000)
        },
      })
    }
  }, viewMode === 'list' && activeRow >= 0)

  useKeyboardShortcut({ key: 'Escape' }, () => setActiveRow(-1), activeRow >= 0)

  // Items targeted for deletion: selected checkboxes take priority, otherwise highlighted row
  const deleteTargets = useMemo(() => {
    if (selected.size > 0) return Array.from(selected)
    if (activeRow >= 0 && activeRow < allItems.length) return [allItems[activeRow].item_number]
    return []
  }, [selected, activeRow, allItems])

  const columns: Column<WorkItem>[] = [
    ...(!readOnly ? [{
      key: 'select',
      header: '',
      className: 'w-10',
      render: (row: WorkItem) => (
        <input
          type="checkbox"
          checked={selected.has(row.item_number)}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => { e.stopPropagation(); toggleSelect(row.item_number) }}
          onClick={(e: React.MouseEvent) => e.stopPropagation()}
        />
      ),
    }] as Column<WorkItem>[] : []),
    {
      key: 'display_id',
      header: t('workitems.table.id'),
      className: 'w-[102px]',
      sortKey: 'item_number',
      render: (row) => <span className="font-mono text-gray-500 dark:text-gray-400">{row.display_id}</span>,
    },
    {
      key: 'type',
      header: t('workitems.table.type'),
      className: 'w-20',
      sortKey: 'type',
      render: (row) => <TypeBadge type={row.type} />,
    },
    {
      key: 'title',
      header: t('workitems.table.title'),
      sortKey: 'title',
      render: (row) => (
        <div className="flex items-center gap-1 min-w-0">
          <Tooltip content={row.title} className="relative block min-w-0 flex-1">
            <span className={`truncate block ${row.description ? 'text-gray-400 dark:text-gray-500' : 'text-gray-900 dark:text-gray-100'}`}>
              <span className="font-medium text-gray-900 dark:text-gray-100">{row.title}</span>
              {row.description && (
                <span className="font-normal text-xs"> – {getDescriptionPreview(row.description)}</span>
              )}
            </span>
          </Tooltip>
          <span className="shrink-0" onClick={(e) => e.stopPropagation()}>
            {inboxSavedId === row.id ? (
              <Check className="h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />
            ) : (
              <span className="sm:opacity-0 sm:group-hover:opacity-100 transition-opacity">
                <InboxButton workItemId={row.id} inboxItemId={inboxByWorkItemId.get(row.id)} />
              </span>
            )}
          </span>
        </div>
      ),
    },
    {
      key: 'status',
      header: t('workitems.table.status'),
      className: 'w-32',
      sortKey: 'status',
      render: (row) => <StatusBadge status={row.status} statuses={allStatuses ?? statuses} />,
    },
    {
      key: 'priority',
      header: t('workitems.table.priority'),
      className: 'w-28',
      sortKey: 'priority',
      render: (row) => <PriorityBadge priority={row.priority} />,
    },
    {
      key: 'sla',
      header: t('sla.columnHeader'),
      className: 'w-[110px]',
      sortKey: 'sla_target_at',
      render: (row) => <SLAIndicator sla={row.sla} />,
    },
    {
      key: 'updated',
      header: t('workitems.table.updated'),
      className: 'w-[130px]',
      sortKey: 'updated_at',
      render: (row) => <span className="text-gray-500 dark:text-gray-400">{new Date(row.updated_at).toLocaleDateString()}</span>,
    },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 shrink-0">
          <span className="sm:hidden">{t('workitems.titleShort')}</span>
          <span className="hidden sm:inline">{t('workitems.title')}</span>
        </h2>
        <div className="flex items-center gap-2">
          <div className="hidden sm:block flex-1 min-w-0 max-w-md">
            <WorkItemSearchBar search={search} onSearchChange={handleSearchChange} />
          </div>
          <div className="inline-flex rounded-md shadow-sm">
            <button
              className={`flex items-center gap-1.5 px-3 h-[39px] text-sm font-medium rounded-l-md border ${
                viewMode === 'list' ? 'bg-indigo-50 text-indigo-700 border-indigo-300 dark:bg-indigo-900/30 dark:text-indigo-300 dark:border-indigo-700' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700'
              }`}
              onClick={() => handleViewChange('list')}
            >
              <LayoutList className="h-4 w-4" />
              {t('workitems.view.list')}
            </button>
            <button
              className={`flex items-center gap-1.5 px-3 h-[39px] text-sm font-medium rounded-r-md border-t border-r border-b ${
                viewMode === 'board' ? 'bg-indigo-50 text-indigo-700 border-indigo-300 dark:bg-indigo-900/30 dark:text-indigo-300 dark:border-indigo-700' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700'
              }`}
              onClick={() => handleViewChange('board')}
            >
              <LayoutGrid className="h-4 w-4" />
              {t('workitems.view.board')}
            </button>
          </div>
          {!readOnly && (
            <Button onClick={() => setShowCreate(true)} className="h-[39px] border border-transparent">
              <span className="sm:hidden">{t('workitems.newShort')}</span>
              <span className="hidden sm:inline">{t('workitems.new')}</span>
            </Button>
          )}
        </div>
      </div>

      <WorkItemFilters
        filter={filter}
        onFilterChange={handleFilterChange}
        statuses={allStatuses ?? statuses}
        milestones={milestones ?? []}
        members={members ?? []}
        search={search}
        onSearchChange={handleSearchChange}
        sort={sort}
        order={order}
        onSort={handleSort}
        onOrderChange={handleOrderChange}
        showDates={showDates}
        onShowDatesChange={(v) => setPreferenceMutation.mutate({ key: 'showDates', value: v })}
        onSave={() => setShowSaveModal(true)}
        onClearFilters={handleClearFilters}
        hasUnsavedChanges={hasUnsavedChanges}
        hasActiveSearch={!!activeSearchId}
        savedSearchSelector={
          <SavedSearchSelector
            searches={savedSearchList ?? []}
            activeSearchId={activeSearchId}
            hasUnsavedChanges={hasUnsavedChanges}
            onSelect={handleSelectSearch}
            onRename={handleRenameSearch}
            onDelete={handleDeleteSearch}
            canManageShared={canManageShared}
          />
        }
        savedSearchMobileButton={
          <SavedSearchSelector
            searches={savedSearchList ?? []}
            activeSearchId={activeSearchId}
            hasUnsavedChanges={hasUnsavedChanges}
            onSelect={handleSelectSearch}
            onRename={handleRenameSearch}
            onDelete={handleDeleteSearch}
            canManageShared={canManageShared}
            variant="mobile"
          />
        }
      />

      <SaveSearchModal
        open={showSaveModal}
        onClose={() => setShowSaveModal(false)}
        onSaveNew={handleSaveNew}
        onUpdateExisting={handleUpdateExisting}
        activeSearch={activeSearch}
        hasUnsavedChanges={hasUnsavedChanges}
        canManageShared={canManageShared}
      />

      {/* Bulk action toolbar */}
      {!readOnly && selected.size > 0 && (
        <div className="flex items-center gap-3 rounded-md bg-indigo-50 dark:bg-indigo-900/30 px-4 py-2">
          <span className="text-sm font-medium text-indigo-700 dark:text-indigo-300">{t('workitems.selected', { count: selected.size })}</span>
          <div className="w-40">
            <Select onChange={(e) => handleBulkStatus(e.target.value)} value="">
              <option value="">{t('workitems.bulk.changeStatus')}</option>
              {(allStatuses ?? statuses).map((s) => (
                <option key={s.name} value={s.name}>{t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name })}</option>
              ))}
            </Select>
          </div>
          <div className="w-44">
            <Select onChange={(e) => handleBulkAssign(e.target.value)} value="">
              <option value="">{t('workitems.bulk.assign')}</option>
              <option value="unassign">{t('workitems.bulk.unassign')}</option>
              {(members ?? []).map((m) => (
                <option key={m.user_id} value={m.user_id}>{m.display_name}</option>
              ))}
            </Select>
          </div>
          {milestones && milestones.length > 0 && (
            <div className="w-44">
              <Select onChange={(e) => handleBulkMilestone(e.target.value)} value="">
                <option value="">{t('workitems.bulk.milestone')}</option>
                <option value="none">{t('milestones.noMilestone')}</option>
                {milestones.filter((m) => m.status === 'open').map((m) => (
                  <option key={m.id} value={m.id}>{m.name}</option>
                ))}
              </Select>
            </div>
          )}
          <Button variant="ghost" size="sm" onClick={() => { setSelected(new Set()); setBulkError(null) }}>{t('common.clear')}</Button>
          {bulkMutation.isPending && <Spinner size="sm" />}
          {bulkError && (
            <span className="text-sm text-red-600 dark:text-red-400">{bulkError}</span>
          )}
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Spinner size="lg" />
        </div>
      ) : viewMode === 'list' ? (
        <>
          {/* Desktop: table view */}
          <div className="hidden sm:block border dark:border-gray-700 rounded-lg overflow-hidden">
            {!readOnly && (
              <div className="bg-gray-50 dark:bg-gray-800 px-6 py-2 border-b dark:border-gray-700">
                <label className="flex items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
                  <input
                    type="checkbox"
                    checked={items.length > 0 && selected.size === items.length}
                    onChange={toggleSelectAll}
                  />
                  {t('common.selectAll')}
                </label>
              </div>
            )}
            <DataTable
              columns={columns}
              data={allItems}
              onRowClick={(row) => {
                sessionStorage.setItem(activeRowStorageKey, String(row.item_number))
                sessionStorage.setItem(filterStorageKey, currentParamsString())
                navigate(`/projects/${projectKey}/items/${row.item_number}`)
              }}
              emptyMessage={t('workitems.empty')}
              sortBy={sort}
              sortOrder={order}
              onSort={handleSort}
              activeRowIndex={activeRow}
            />
          </div>

          {/* Mobile: card view */}
          <div className="sm:hidden space-y-2">
            {allItems.length === 0 ? (
              <p className="text-center text-sm text-gray-500 dark:text-gray-400 py-12">{t('workitems.empty')}</p>
            ) : (
              allItems.map((item) => {
                const assigneeName = item.assignee_id
                  ? members?.find(m => m.user_id === item.assignee_id)?.display_name ?? t('userPicker.unassigned')
                  : t('userPicker.unassigned')
                return (
                  <div
                    key={item.id}
                    className="relative w-full text-left rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-3 shadow-sm hover:border-indigo-300 dark:hover:border-indigo-600 transition-colors cursor-pointer"
                    onClick={() => {
                      sessionStorage.setItem(activeRowStorageKey, String(item.item_number))
                      sessionStorage.setItem(filterStorageKey, currentParamsString())
                      navigate(`/projects/${projectKey}/items/${item.item_number}`)
                    }}
                    role="button"
                    tabIndex={0}
                    onKeyDown={(e) => {
                      if (e.key === 'Enter' || e.key === ' ') {
                        e.preventDefault()
                        sessionStorage.setItem(activeRowStorageKey, String(item.item_number))
                        sessionStorage.setItem(filterStorageKey, currentParamsString())
                        navigate(`/projects/${projectKey}/items/${item.item_number}`)
                      }
                    }}
                  >
                    <span className="absolute top-2 right-2" onClick={(e) => e.stopPropagation()}>
                      <InboxButton workItemId={item.id} inboxItemId={inboxByWorkItemId.get(item.id)} />
                    </span>
                    <div className="flex items-center gap-2 flex-wrap pr-6">
                      <span className="font-mono text-sm font-semibold text-gray-700 dark:text-gray-300">{item.display_id}</span>
                      <TypeBadge type={item.type} />
                      <StatusBadge status={item.status} statuses={allStatuses ?? statuses} />
                      <PriorityBadge priority={item.priority} />
                      {!showDates && item.sla && (
                        <span className="ml-auto"><SLAIndicator sla={item.sla} compact /></span>
                      )}
                    </div>
                    {showDates && (
                      <div className="flex items-center gap-3 mt-1.5 text-xs text-gray-400 dark:text-gray-500">
                        <span className="inline-flex items-center gap-1">
                          <User className="h-3 w-3" />
                          <span className="truncate max-w-[8rem]">{assigneeName}</span>
                        </span>
                        <span className="inline-flex items-center gap-1">
                          <History className="h-3 w-3" />
                          {formatRelativeTime(item.updated_at)}
                        </span>
                        {item.sla && <SLAIndicator sla={item.sla} />}
                      </div>
                    )}
                    <p className="mt-1.5 text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{item.title}</p>
                    {item.description && (
                      <p className="mt-0.5 text-xs text-gray-500 dark:text-gray-400 truncate">{item.description}</p>
                    )}
                  </div>
                )
              })
            )}
          </div>

          {result?.meta.has_more && (
            <div className="flex justify-center pt-2">
              <Button
                variant="secondary"
                onClick={async () => {
                  const lastItem = allItems[allItems.length - 1]
                  if (!lastItem) return
                  const next = await listWorkItems(projectKey ?? '', { ...activeFilter, cursor: lastItem.id })
                  setLoadedPages((prev) => [...prev, next.data])
                }}
              >
                {t('common.loadMore')}
              </Button>
            </div>
          )}
        </>
      ) : (
        <BoardView
          projectKey={projectKey ?? ''}
          items={allItems}
          statuses={statuses}
          transitionsMap={transitionsMap}
          readOnly={readOnly}
          onItemClick={(item) => {
            sessionStorage.setItem(filterStorageKey, currentParamsString())
            navigate(`/projects/${projectKey}/items/${item.item_number}`)
          }}
        />
      )}

      <Modal open={showCreate} onClose={() => { setShowCreate(false); createMutation.reset() }} title={t('workitems.newTitle')} dismissable={false}>
        <WorkItemForm
          projectKey={projectKey ?? ''}
          mode="create"
          members={members ?? []}
          milestones={milestones}
          allowedComplexityValues={project?.allowed_complexity_values}
          onSubmit={(values) => {
            createMutation.mutate(values as { type: string; title: string }, {
              onSuccess: () => { setShowCreate(false); createMutation.reset() },
            })
          }}
          onCancel={() => { setShowCreate(false); createMutation.reset() }}
          isSubmitting={createMutation.isPending}
          submitError={createMutation.error ? t('workitems.form.submitError', { message: (createMutation.error as AxiosError<{ error?: { message?: string } }>).response?.data?.error?.message ?? t('common.unknown') }) : null}
        />
      </Modal>

      <Modal open={showDeleteConfirm} onClose={() => setShowDeleteConfirm(false)} title={t('workitems.deleteSelectedTitle')}>
        <form onSubmit={(e) => {
          e.preventDefault()
          for (const num of deleteTargets) {
            deleteMutation.mutate(num)
          }
          setShowDeleteConfirm(false)
          setSelected(new Set())
          setActiveRow(-1)
        }}>
          <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
            {t('workitems.deleteSelectedBody', { count: deleteTargets.length })}
          </p>
          <div className="flex justify-end gap-3">
            <Button type="button" variant="secondary" onClick={() => setShowDeleteConfirm(false)}>{t('common.cancel')}</Button>
            <Button
              type="submit"
              variant="danger"
              autoFocus
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? t('common.deleting') : t('common.delete')}
            </Button>
          </div>
        </form>
      </Modal>

    </div>
  )
}
