import { useState, useMemo, useCallback, useRef } from 'react'
import { useNavigate, useParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useWorkItems, useBulkUpdateWorkItems, type BulkUpdateResult } from '@/hooks/useWorkItems'
import { useQueue } from '@/hooks/useQueues'
import { useMembers } from '@/hooks/useProjects'
import { useProjectWorkflow, useAvailableStatuses } from '@/hooks/useWorkflows'
import { useColumnWidths } from '@/hooks/useColumnWidths'
import { usePreference } from '@/hooks/usePreferences'
import { useDebounce } from '@/hooks/useDebounce'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { Button } from '@/components/ui/Button'
import { Select } from '@/components/ui/Select'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { Input } from '@/components/ui/Input'
import { Tooltip } from '@/components/ui/Tooltip'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { WorkItemFilters } from '@/components/workitems/WorkItemFilters'
import { BoardView } from '@/components/workitems/BoardView'
import { SLAIndicator } from '@/components/SLAIndicator'
import { WorkItemMobileCard } from '@/components/workitems/WorkItemMobileCard'
import { ArrowLeft, X, LayoutList, LayoutGrid } from 'lucide-react'
import { listWorkItems, type WorkItem, type WorkItemFilter } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

type ViewMode = 'list' | 'board'

const closedCategories = new Set(['done', 'cancelled'])

function isItemCompleted(status: string, statuses: WorkflowStatus[]): boolean {
  const category = statuses.find((s) => s.name === status)?.category
  return category === 'done' || category === 'cancelled'
}

function getDescriptionPreview(description: string): string {
  const line = description.split('\n').find(l => l.trim() !== '')
  if (!line) return ''
  return line.trim().replace(/^#+\s+/, '').replace(/[*_~`[\]]/g, '')
}

export function QueueWorkItemsPage() {
  const { t } = useTranslation()
  const { projectKey = '', queueId = '' } = useParams<{ projectKey: string; queueId: string }>()
  const navigate = useNavigate()
  const { p } = useNamespacePath()
  const { user } = useAuth()
  const { data: queue, isLoading: queueLoading } = useQueue(projectKey, queueId)
  const { statuses, transitionsMap } = useProjectWorkflow(projectKey)
  const { data: allStatuses } = useAvailableStatuses(projectKey)
  const { data: members } = useMembers(projectKey)

  const currentUserRole = members?.find((m) => m.user_id === user?.id)?.role ?? (user?.global_role === 'admin' ? 'owner' : null)
  const canEdit = user?.global_role === 'admin' || (currentUserRole != null && currentUserRole !== 'viewer')
  const readOnly = !canEdit

  // Column widths (shared key for queue items)
  const { columnWidths, onColumnResize, resetColumnWidth } = useColumnWidths('queueItems')

  // Strikethrough completed items preference
  const { data: strikethroughPref } = usePreference<boolean>('strikethrough_completed')
  const strikethroughEnabled = strikethroughPref ?? true

  // Filters & view state
  const [filter, setFilter] = useState<WorkItemFilter>({})
  const [search, setSearch] = useState('')
  const [viewMode, setViewMode] = useState<ViewMode>('list')
  const [sort, setSort] = useState('created_at')
  const [order, setOrder] = useState<'asc' | 'desc'>('desc')

  const debouncedSearch = useDebounce(search, 300)
  const searchRef = useRef<HTMLInputElement>(null)

  // Compute default open statuses
  const defaultOpenStatuses = useMemo(() => {
    if (!allStatuses?.length) return undefined
    const names = new Set(allStatuses.filter((s) => !closedCategories.has(s.category)).map((s) => s.name))
    return Array.from(names)
  }, [allStatuses])

  // Initialize status filter to open statuses once available
  const filterInitRef = useRef(false)
  if (!filterInitRef.current && defaultOpenStatuses && !filter.status) {
    filterInitRef.current = true
    setFilter((prev) => ({ ...prev, status: defaultOpenStatuses }))
  }

  const activeFilter = useMemo(() => ({
    ...filter,
    queue: queueId,
    q: debouncedSearch || undefined,
    sort,
    order,
    limit: viewMode === 'board' ? 200 : 50,
  }), [filter, queueId, debouncedSearch, viewMode, sort, order])

  const { data: result, isLoading } = useWorkItems(projectKey, activeFilter)
  const bulkMutation = useBulkUpdateWorkItems(projectKey)
  const items = result?.data ?? []

  // Selection & bulk actions
  const [selected, setSelected] = useState<Set<number>>(new Set())
  const [bulkError, setBulkError] = useState<string | null>(null)
  const [activeRow, setActiveRow] = useState(-1)

  // Pagination
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
    setSelected(new Set(result.failed.map((f) => f.itemNumber)))
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

  // Filter handlers
  function handleFilterChange(f: WorkItemFilter) {
    setFilter(f)
  }

  function handleSearchChange(q: string) {
    setSearch(q)
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
  }

  function handleOrderChange(newOrder: 'asc' | 'desc') {
    setOrder(newOrder)
  }

  function handleClearFilters() {
    const f: WorkItemFilter = defaultOpenStatuses ? { status: defaultOpenStatuses } : {}
    setFilter(f)
    setSearch('')
    setViewMode('list')
    setSort('created_at')
    setOrder('desc')
  }

  const navigateToItem = useCallback((item: WorkItem) => {
    navigate(p(`/projects/${projectKey}/items/${item.item_number}`))
  }, [navigate, p, projectKey])

  // Table columns (mirrors WorkItemListPage)
  const columns: Column<WorkItem>[] = [
    ...(!readOnly ? [{
      key: 'select',
      header: '',
      className: 'w-10',
      resizable: false,
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
      render: (row) => {
        const done = strikethroughEnabled && isItemCompleted(row.status, allStatuses ?? statuses)
        return <span className={`font-mono ${done ? 'text-gray-400 dark:text-gray-500' : 'text-gray-500 dark:text-gray-400'}`}>{row.display_id}</span>
      },
    },
    {
      key: 'type',
      header: t('workitems.table.type'),
      className: 'w-20',
      sortKey: 'type',
      render: (row) => {
        const done = strikethroughEnabled && isItemCompleted(row.status, allStatuses ?? statuses)
        return <span className={done ? 'opacity-40' : ''}><TypeBadge type={row.type} /></span>
      },
    },
    {
      key: 'title',
      header: t('workitems.table.title'),
      className: 'lg:!pr-2',
      resizable: false,
      sortKey: 'title',
      render: (row) => {
        const done = strikethroughEnabled && isItemCompleted(row.status, allStatuses ?? statuses)
        return (
          <div className="flex items-center gap-1 min-w-0">
            <Tooltip content={row.title} className="relative block min-w-0 flex-1">
              <span className={`truncate block ${!done && row.description ? 'text-gray-400 dark:text-gray-500' : ''}`}>
                <span className={done ? 'line-through text-gray-400 dark:text-gray-500' : 'font-medium text-gray-900 dark:text-gray-100'}>{row.title}</span>
                {row.description && !done && (
                  <span className="font-normal text-xs"> – {getDescriptionPreview(row.description)}</span>
                )}
              </span>
            </Tooltip>
          </div>
        )
      },
    },
    {
      key: 'status',
      header: t('workitems.table.status'),
      className: 'w-24 lg:!pl-3',
      sortKey: 'status',
      render: (row) => <StatusBadge status={row.status} statuses={allStatuses ?? statuses} />,
    },
    {
      key: 'priority',
      header: t('workitems.table.priority'),
      className: 'w-28',
      sortKey: 'priority',
      render: (row) => {
        const done = strikethroughEnabled && isItemCompleted(row.status, allStatuses ?? statuses)
        return <span className={done ? 'opacity-40' : ''}><PriorityBadge priority={row.priority} /></span>
      },
    },
    {
      key: 'sla',
      header: t('sla.columnHeader'),
      className: 'w-[110px]',
      sortKey: 'sla_target_at',
      render: (row) => {
        const done = strikethroughEnabled && isItemCompleted(row.status, allStatuses ?? statuses)
        return <span className={done ? 'opacity-40' : ''}><SLAIndicator sla={row.sla} /></span>
      },
    },
    {
      key: 'updated',
      header: t('workitems.table.updated'),
      className: 'w-[130px]',
      sortKey: 'updated_at',
      render: (row) => {
        const done = strikethroughEnabled && isItemCompleted(row.status, allStatuses ?? statuses)
        return <span className={done ? 'text-gray-300 dark:text-gray-600' : 'text-gray-500 dark:text-gray-400'}>{new Date(row.updated_at).toLocaleDateString()}</span>
      },
    },
  ]

  if (queueLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    )
  }

  return (
    <div className="space-y-4">
      {/* Header: back link + title + view toggle + search */}
      <div className="flex items-center gap-4">
        <Link
          to={p(`/projects/${projectKey}/queues`)}
          className="text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 shrink-0"
        >
          <ArrowLeft className="h-5 w-5" />
        </Link>
        <div className="flex items-center gap-2 shrink-0">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">
            {queue?.name ?? t('queues.workItems')}
          </h2>
          {queue && (
            <Badge color="gray">{t(`queues.types.${queue.queue_type}`)}</Badge>
          )}
        </div>
        <div className="grid grid-cols-2 rounded-md shadow-sm shrink-0">
          <button
            className={`flex items-center justify-center gap-1.5 px-3 py-2 text-sm font-medium rounded-l-md border ${
              viewMode === 'list' ? 'bg-indigo-50 text-indigo-700 border-indigo-300 dark:bg-indigo-900/30 dark:text-indigo-300 dark:border-indigo-700' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700'
            }`}
            onClick={() => setViewMode('list')}
          >
            <LayoutList className="h-4 w-4" />
            {t('workitems.view.list')}
          </button>
          <button
            className={`flex items-center justify-center gap-1.5 px-3 py-2 text-sm font-medium rounded-r-md border-t border-r border-b ${
              viewMode === 'board' ? 'bg-indigo-50 text-indigo-700 border-indigo-300 dark:bg-indigo-900/30 dark:text-indigo-300 dark:border-indigo-700' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700'
            }`}
            onClick={() => setViewMode('board')}
          >
            <LayoutGrid className="h-4 w-4" />
            {t('workitems.view.board')}
          </button>
        </div>
        <div className="flex items-center gap-2 flex-1 justify-end">
          <div className="hidden lg:block flex-1 min-w-0 max-w-lg">
            <div className="relative">
              <Input
                ref={searchRef}
                placeholder={t('workitems.filters.search')}
                value={search}
                onChange={(e) => handleSearchChange(e.target.value)}
                onKeyDown={(e) => { if (e.key === 'Escape') searchRef.current?.blur() }}
                className="pr-8"
              />
              {search && (
                <button
                  onClick={() => { handleSearchChange(''); searchRef.current?.focus() }}
                  className="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  aria-label={t('common.clear')}
                >
                  <X className="h-4 w-4" />
                </button>
              )}
            </div>
          </div>
        </div>
      </div>

      {/* Filters: priority, status, assignee only (no type, no milestone, no saved searches) */}
      <WorkItemFilters
        filter={filter}
        onFilterChange={handleFilterChange}
        statuses={allStatuses ?? statuses}
        members={members ?? []}
        search={search}
        onSearchChange={handleSearchChange}
        sort={sort}
        order={order}
        onSort={handleSort}
        onOrderChange={handleOrderChange}
        onClearFilters={handleClearFilters}
        hideTypeFilter
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
          <Button variant="ghost" size="sm" onClick={() => { setSelected(new Set()); setBulkError(null) }}>{t('common.clear')}</Button>
          {bulkMutation.isPending && <Spinner size="sm" />}
          {bulkError && (
            <span className="text-sm text-red-600 dark:text-red-400">{bulkError}</span>
          )}
        </div>
      )}

      {/* Content */}
      {isLoading ? (
        <div className="flex justify-center py-12">
          <Spinner size="lg" />
        </div>
      ) : viewMode === 'list' ? (
        <>
          {/* Desktop: table view */}
          <div className="hidden lg:block border dark:border-gray-600 rounded-lg overflow-hidden">
            {!readOnly && (
              <div className="bg-gray-50 dark:bg-gray-800 px-6 py-2 border-b dark:border-gray-600">
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
              onRowClick={navigateToItem}
              emptyMessage={t('queues.noItems')}
              sortBy={sort}
              sortOrder={order}
              onSort={handleSort}
              activeRowIndex={activeRow}
              resizable
              columnWidths={columnWidths}
              onColumnResize={onColumnResize}
              onColumnResetWidth={resetColumnWidth}
            />
          </div>

          {/* Mobile: card view */}
          <div className="lg:hidden space-y-2">
            {allItems.length === 0 ? (
              <p className="text-center text-sm text-gray-500 dark:text-gray-400 py-12">{t('queues.noItems')}</p>
            ) : (
              allItems.map((item) => {
                const assigneeName = item.assignee_id
                  ? members?.find(m => m.user_id === item.assignee_id)?.display_name ?? t('userPicker.unassigned')
                  : t('userPicker.unassigned')
                return (
                  <WorkItemMobileCard
                    key={item.id}
                    item={item}
                    statuses={allStatuses ?? statuses}
                    showDates
                    assigneeName={assigneeName}
                    isWatching={false}
                    isCompleted={strikethroughEnabled && isItemCompleted(item.status, allStatuses ?? statuses)}
                    onClick={() => navigateToItem(item)}
                  />
                )
              })
            )}
          </div>

          {/* Load more */}
          {result?.meta.has_more && (
            <div className="flex justify-center pt-2">
              <Button
                variant="secondary"
                onClick={async () => {
                  const lastItem = allItems[allItems.length - 1]
                  if (!lastItem) return
                  const next = await listWorkItems(projectKey, { ...activeFilter, cursor: lastItem.id })
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
          projectKey={projectKey}
          items={allItems}
          statuses={statuses}
          transitionsMap={transitionsMap}
          readOnly={readOnly}
          strikethroughCompleted={strikethroughEnabled}
          onItemClick={navigateToItem}
        />
      )}
    </div>
  )
}
