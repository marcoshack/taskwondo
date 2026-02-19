import { useState, useMemo, useEffect, useCallback, useRef } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { useWorkItems, useCreateWorkItem, useBulkUpdateWorkItems, useDeleteWorkItem } from '@/hooks/useWorkItems'
import { useMembers } from '@/hooks/useProjects'
import { useProjectWorkflow } from '@/hooks/useWorkflows'
import { useUserSetting, useSetUserSetting } from '@/hooks/useUserSettings'
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
import { listWorkItems, type WorkItem, type WorkItemFilter } from '@/api/workitems'

type ViewMode = 'list' | 'board'

const SETTINGS_KEY = 'workitem_filters'
const closedCategories = new Set(['done', 'cancelled'])

/** Filter keys synced to URL and persisted. */
const FILTER_PARAMS = ['type', 'status', 'priority', 'assignee'] as const
type FilterParam = typeof FILTER_PARAMS[number]

type SavedFilter = Pick<WorkItemFilter, FilterParam>

function pickSavedFilter(f: WorkItemFilter): SavedFilter {
  return { type: f.type, status: f.status, priority: f.priority, assignee: f.assignee }
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

export function WorkItemListPage() {
  const { t } = useTranslation()
  const { projectKey } = useParams<{ projectKey: string }>()
  const navigate = useNavigate()
  const [searchParams, setSearchParams] = useSearchParams()

  const { statuses, transitionsMap } = useProjectWorkflow(projectKey ?? '')
  const { data: members } = useMembers(projectKey ?? '')

  // Load saved filter from user settings
  const { data: savedFilter, isLoading: settingsLoading } = useUserSetting<SavedFilter>(projectKey ?? '', SETTINGS_KEY)
  const saveMutation = useSetUserSetting(projectKey ?? '')

  // Compute default open statuses (exclude done/cancelled)
  const defaultOpenStatuses = useMemo(() => {
    if (!statuses.length) return undefined
    return statuses.filter((s) => !closedCategories.has(s.category)).map((s) => s.name)
  }, [statuses])

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

  // Track whether the initial load was from a shared URL (don't save on init)
  const loadedFromUrlRef = useRef(initialUrlRef.current.hasParams)

  const [filter, setFilter] = useState<WorkItemFilter>(
    initialUrlRef.current.hasParams ? initialUrlRef.current.filter : {}
  )
  const [filterInitialized, setFilterInitialized] = useState(initialUrlRef.current.hasParams)
  const [search, setSearch] = useState(initialUrlRef.current.search)
  const [viewMode, setViewMode] = useState<ViewMode>(initialUrlRef.current.view)
  const [sort, setSort] = useState(initialUrlRef.current.sort)
  const [order, setOrder] = useState<'asc' | 'desc'>(initialUrlRef.current.order)

  // If no URL params, initialize from saved settings or defaults once data loads
  useEffect(() => {
    if (filterInitialized || settingsLoading) return
    if (!statuses.length) return

    if (savedFilter) {
      setFilter(savedFilter)
    } else if (defaultOpenStatuses) {
      setFilter({ status: defaultOpenStatuses })
    }
    setFilterInitialized(true)
  }, [savedFilter, settingsLoading, defaultOpenStatuses, statuses, filterInitialized])

  // Sync URL when filter initializes from settings/defaults (non-URL case)
  const urlSyncedRef = useRef(initialUrlRef.current.hasParams)
  useEffect(() => {
    if (!filterInitialized || urlSyncedRef.current) return
    setSearchParams(buildUrlParams(filter, search, viewMode, sort, order), { replace: true })
    urlSyncedRef.current = true
  }, [filterInitialized, filter, search, viewMode, sort, order, setSearchParams])

  // Save filter preferences (debounced) — only on manual user changes
  const saveTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined)
  const saveMutationRef = useRef(saveMutation)
  saveMutationRef.current = saveMutation

  const saveFilter = useCallback((f: WorkItemFilter) => {
    if (!projectKey || !filterInitialized || loadedFromUrlRef.current) return
    clearTimeout(saveTimerRef.current)
    saveTimerRef.current = setTimeout(() => {
      saveMutationRef.current.mutate({ key: SETTINGS_KEY, value: pickSavedFilter(f) })
    }, 500)
  }, [projectKey, filterInitialized])

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
  }

  function handleViewChange(v: ViewMode) {
    setViewMode(v)
    syncUrl(filter, search, v)
  }

  function handleSort(sortKey: string) {
    let newOrder: 'asc' | 'desc'
    if (sort === sortKey) {
      newOrder = order === 'asc' ? 'desc' : 'asc'
    } else {
      newOrder = ['title', 'type', 'status'].includes(sortKey) ? 'asc' : 'desc'
    }
    setSort(sortKey)
    setOrder(newOrder)
    syncUrl(filter, search, viewMode, sortKey, newOrder)
  }

  function handleOrderChange(newOrder: 'asc' | 'desc') {
    setOrder(newOrder)
    syncUrl(filter, search, viewMode, sort, newOrder)
  }

  const [showCreate, setShowCreate] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [activeRow, setActiveRow] = useState(-1)

  useKeyboardShortcut({ key: 'c' }, () => setShowCreate(true))

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

  function handleBulkStatus(status: string) {
    if (!status) return
    const updates = Array.from(selected).map((itemNumber) => ({ itemNumber, input: { status } }))
    bulkMutation.mutate(updates, { onSuccess: () => setSelected(new Set()) })
  }

  function handleBulkAssign(value: string) {
    if (!value) return
    const input = value === 'unassign' ? { assignee_id: null } : { assignee_id: value }
    const updates = Array.from(selected).map((itemNumber) => ({ itemNumber, input }))
    bulkMutation.mutate(updates, { onSuccess: () => setSelected(new Set()) })
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

  // List navigation: arrows + j/k (vim), o/Enter to open, # to delete, Escape to deselect
  useKeyboardShortcut([{ key: 'ArrowDown' }, { key: 'j' }], () => setActiveRow((prev) => Math.min(prev + 1, allItems.length - 1)), viewMode === 'list')
  useKeyboardShortcut([{ key: 'ArrowUp' }, { key: 'k' }], () => setActiveRow((prev) => Math.max(prev - 1, 0)), viewMode === 'list')
  useKeyboardShortcut([{ key: 'Enter' }, { key: 'o' }], () => {
    if (activeRow >= 0 && activeRow < allItems.length) {
      navigate(`/projects/${projectKey}/items/${allItems[activeRow].item_number}`)
    }
  }, activeRow >= 0)
  useKeyboardShortcut({ key: '#' }, () => setShowDeleteConfirm(true), selected.size > 0 || activeRow >= 0)
  useKeyboardShortcut({ key: 'Escape' }, () => setActiveRow(-1), activeRow >= 0)

  // Items targeted for deletion: selected checkboxes take priority, otherwise highlighted row
  const deleteTargets = useMemo(() => {
    if (selected.size > 0) return Array.from(selected)
    if (activeRow >= 0 && activeRow < allItems.length) return [allItems[activeRow].item_number]
    return []
  }, [selected, activeRow, allItems])

  const columns: Column<WorkItem>[] = [
    {
      key: 'select',
      header: '',
      className: 'w-10',
      render: (row) => (
        <input
          type="checkbox"
          checked={selected.has(row.item_number)}
          onChange={(e) => { e.stopPropagation(); toggleSelect(row.item_number) }}
          onClick={(e) => e.stopPropagation()}
        />
      ),
    },
    {
      key: 'display_id',
      header: t('workitems.table.id'),
      className: 'w-28',
      sortKey: 'item_number',
      render: (row) => <span className="font-mono text-gray-500 dark:text-gray-400">{row.display_id}</span>,
    },
    {
      key: 'type',
      header: t('workitems.table.type'),
      className: 'w-24',
      sortKey: 'type',
      render: (row) => <TypeBadge type={row.type} />,
    },
    {
      key: 'title',
      header: t('workitems.table.title'),
      sortKey: 'title',
      render: (row) => <span className="font-medium text-gray-900 dark:text-gray-100 truncate block">{row.title}</span>,
    },
    {
      key: 'status',
      header: t('workitems.table.status'),
      className: 'w-32',
      sortKey: 'status',
      render: (row) => <StatusBadge status={row.status} statuses={statuses} />,
    },
    {
      key: 'priority',
      header: t('workitems.table.priority'),
      className: 'w-28',
      sortKey: 'priority',
      render: (row) => <PriorityBadge priority={row.priority} />,
    },
    {
      key: 'updated',
      header: t('workitems.table.updated'),
      className: 'w-36',
      sortKey: 'updated_at',
      render: (row) => <span className="text-gray-500 dark:text-gray-400">{new Date(row.updated_at).toLocaleDateString()}</span>,
    },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('workitems.title')}</h2>
        <div className="flex items-center gap-2">
          <div className="inline-flex rounded-md shadow-sm">
            <button
              className={`px-3 py-1.5 text-sm font-medium rounded-l-md border ${
                viewMode === 'list' ? 'bg-indigo-50 text-indigo-700 border-indigo-300 dark:bg-indigo-900/30 dark:text-indigo-300 dark:border-indigo-700' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700'
              }`}
              onClick={() => handleViewChange('list')}
            >
              {t('workitems.view.list')}
            </button>
            <button
              className={`px-3 py-1.5 text-sm font-medium rounded-r-md border-t border-r border-b ${
                viewMode === 'board' ? 'bg-indigo-50 text-indigo-700 border-indigo-300 dark:bg-indigo-900/30 dark:text-indigo-300 dark:border-indigo-700' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700'
              }`}
              onClick={() => handleViewChange('board')}
            >
              {t('workitems.view.board')}
            </button>
          </div>
          <Button onClick={() => setShowCreate(true)}>{t('workitems.new')}</Button>
        </div>
      </div>

      <WorkItemFilters
        filter={filter}
        onFilterChange={handleFilterChange}
        statuses={statuses}
        search={search}
        onSearchChange={handleSearchChange}
        sort={sort}
        order={order}
        onSort={handleSort}
        onOrderChange={handleOrderChange}
      />

      {/* Bulk action toolbar */}
      {selected.size > 0 && (
        <div className="flex items-center gap-3 rounded-md bg-indigo-50 dark:bg-indigo-900/30 px-4 py-2">
          <span className="text-sm font-medium text-indigo-700 dark:text-indigo-300">{t('workitems.selected', { count: selected.size })}</span>
          <div className="w-40">
            <Select onChange={(e) => handleBulkStatus(e.target.value)} value="">
              <option value="">{t('workitems.bulk.changeStatus')}</option>
              {statuses.map((s) => (
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
          <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>{t('common.clear')}</Button>
          {bulkMutation.isPending && <Spinner size="sm" />}
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
            <DataTable
              columns={columns}
              data={allItems}
              onRowClick={(row) => navigate(`/projects/${projectKey}/items/${row.item_number}`)}
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
              allItems.map((item) => (
                <button
                  key={item.id}
                  onClick={() => navigate(`/projects/${projectKey}/items/${item.item_number}`)}
                  className="w-full text-left rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-3 shadow-sm hover:border-indigo-300 dark:hover:border-indigo-600 transition-colors"
                >
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="font-mono text-sm font-semibold text-gray-700 dark:text-gray-300">{item.display_id}</span>
                    <TypeBadge type={item.type} />
                    <StatusBadge status={item.status} statuses={statuses} />
                    <PriorityBadge priority={item.priority} />
                    <span className="ml-auto text-xs text-gray-400 dark:text-gray-500">{new Date(item.updated_at).toLocaleDateString()}</span>
                  </div>
                  <p className="mt-1.5 text-sm font-medium text-gray-900 dark:text-gray-100 truncate">{item.title}</p>
                  {item.description && (
                    <p className="mt-0.5 text-xs text-gray-500 dark:text-gray-400 truncate">{item.description}</p>
                  )}
                </button>
              ))
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
          onItemClick={(item) => navigate(`/projects/${projectKey}/items/${item.item_number}`)}
        />
      )}

      <Modal open={showCreate} onClose={() => setShowCreate(false)} title={t('workitems.newTitle')}>
        <WorkItemForm
          projectKey={projectKey ?? ''}
          mode="create"
          members={members ?? []}
          onSubmit={(values) => {
            createMutation.mutate(values as { type: string; title: string }, {
              onSuccess: () => setShowCreate(false),
            })
          }}
          onCancel={() => setShowCreate(false)}
          isSubmitting={createMutation.isPending}
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
