import { useState, useMemo, useEffect, useCallback, useRef } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { useWorkItems, useCreateWorkItem, useBulkUpdateWorkItems } from '@/hooks/useWorkItems'
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
  return sp.has('q') || sp.has('view')
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

/** Build URL search params from filter, search, and view. */
function buildUrlParams(filter: WorkItemFilter, search: string, view: ViewMode): URLSearchParams {
  const sp = new URLSearchParams()
  for (const key of FILTER_PARAMS) {
    const val = filter[key]
    if (val?.length) sp.set(key, val.join(','))
  }
  if (search) sp.set('q', search)
  if (view !== 'list') sp.set('view', view)
  return sp
}

export function WorkItemListPage() {
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
  const initialUrlRef = useRef<{ hasParams: boolean; filter: SavedFilter; search: string; view: ViewMode } | null>(null)
  if (initialUrlRef.current === null) {
    const hasParams = urlHasFilterParams(searchParams)
    initialUrlRef.current = {
      hasParams,
      filter: parseUrlFilter(searchParams),
      search: searchParams.get('q') ?? '',
      view: (searchParams.get('view') === 'board' ? 'board' : 'list') as ViewMode,
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
    setSearchParams(buildUrlParams(filter, search, viewMode), { replace: true })
    urlSyncedRef.current = true
  }, [filterInitialized, filter, search, viewMode, setSearchParams])

  // Save filter preferences (debounced) — only on manual user changes
  const saveTimerRef = useRef<ReturnType<typeof setTimeout>>()
  const saveMutationRef = useRef(saveMutation)
  saveMutationRef.current = saveMutation

  const saveFilter = useCallback((f: WorkItemFilter) => {
    if (!projectKey || !filterInitialized) return
    clearTimeout(saveTimerRef.current)
    saveTimerRef.current = setTimeout(() => {
      saveMutationRef.current.mutate({ key: SETTINGS_KEY, value: pickSavedFilter(f) })
    }, 500)
  }, [projectKey, filterInitialized])

  function syncUrl(f: WorkItemFilter, q: string, v: ViewMode) {
    setSearchParams(buildUrlParams(f, q, v), { replace: true })
  }

  function handleFilterChange(f: WorkItemFilter) {
    setFilter(f)
    syncUrl(f, search, viewMode)
    // User manually changed filters — save preferences
    loadedFromUrlRef.current = false
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

  const [showCreate, setShowCreate] = useState(false)
  const [selected, setSelected] = useState<Set<number>>(new Set())

  const debouncedSearch = useDebounce(search, 300)

  const activeFilter = useMemo(() => ({
    ...filter,
    q: debouncedSearch || undefined,
    limit: viewMode === 'board' ? 200 : 50,
  }), [filter, debouncedSearch, viewMode])

  const { data: result, isLoading } = useWorkItems(projectKey ?? '', activeFilter)
  const createMutation = useCreateWorkItem(projectKey ?? '')
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
  }

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
      header: 'ID',
      className: 'w-28',
      render: (row) => <span className="font-mono text-gray-500">{row.display_id}</span>,
    },
    {
      key: 'type',
      header: 'Type',
      className: 'w-24',
      render: (row) => <TypeBadge type={row.type} />,
    },
    {
      key: 'title',
      header: 'Title',
      render: (row) => <span className="font-medium text-gray-900 truncate max-w-xs block">{row.title}</span>,
    },
    {
      key: 'status',
      header: 'Status',
      className: 'w-32',
      render: (row) => <StatusBadge status={row.status} statuses={statuses} />,
    },
    {
      key: 'priority',
      header: 'Priority',
      className: 'w-28',
      render: (row) => <PriorityBadge priority={row.priority} />,
    },
    {
      key: 'updated',
      header: 'Updated',
      className: 'w-36',
      render: (row) => <span className="text-gray-500">{new Date(row.updated_at).toLocaleDateString()}</span>,
    },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <h2 className="text-lg font-semibold text-gray-900">Work Items</h2>
        <div className="flex items-center gap-2">
          <div className="inline-flex rounded-md shadow-sm">
            <button
              className={`px-3 py-1.5 text-sm font-medium rounded-l-md border ${
                viewMode === 'list' ? 'bg-indigo-50 text-indigo-700 border-indigo-300' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
              }`}
              onClick={() => handleViewChange('list')}
            >
              List
            </button>
            <button
              className={`px-3 py-1.5 text-sm font-medium rounded-r-md border-t border-r border-b ${
                viewMode === 'board' ? 'bg-indigo-50 text-indigo-700 border-indigo-300' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
              }`}
              onClick={() => handleViewChange('board')}
            >
              Board
            </button>
          </div>
          <Button onClick={() => setShowCreate(true)}>New Item</Button>
        </div>
      </div>

      <WorkItemFilters
        filter={filter}
        onFilterChange={handleFilterChange}
        statuses={statuses}
        search={search}
        onSearchChange={handleSearchChange}
      />

      {/* Bulk action toolbar */}
      {selected.size > 0 && (
        <div className="flex items-center gap-3 rounded-md bg-indigo-50 px-4 py-2">
          <span className="text-sm font-medium text-indigo-700">{selected.size} selected</span>
          <div className="w-40">
            <Select onChange={(e) => handleBulkStatus(e.target.value)} value="">
              <option value="">Change status...</option>
              {statuses.map((s) => (
                <option key={s.name} value={s.name}>{s.display_name}</option>
              ))}
            </Select>
          </div>
          <div className="w-44">
            <Select onChange={(e) => handleBulkAssign(e.target.value)} value="">
              <option value="">Assign...</option>
              <option value="unassign">Unassign</option>
              {(members ?? []).map((m) => (
                <option key={m.user_id} value={m.user_id}>{m.display_name}</option>
              ))}
            </Select>
          </div>
          <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>Clear</Button>
          {bulkMutation.isPending && <Spinner size="sm" />}
        </div>
      )}

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Spinner size="lg" />
        </div>
      ) : viewMode === 'list' ? (
        <>
          <div className="border rounded-lg overflow-hidden">
            <div className="bg-gray-50 px-6 py-2 border-b">
              <label className="flex items-center gap-2 text-xs text-gray-500">
                <input
                  type="checkbox"
                  checked={items.length > 0 && selected.size === items.length}
                  onChange={toggleSelectAll}
                />
                Select all
              </label>
            </div>
            <DataTable
              columns={columns}
              data={allItems}
              onRowClick={(row) => navigate(`/projects/${projectKey}/items/${row.item_number}`)}
              emptyMessage="No work items found"
            />
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
                Load more
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

      <Modal open={showCreate} onClose={() => setShowCreate(false)} title="New Work Item">
        <WorkItemForm
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
    </div>
  )
}
