import { useState, useMemo, useCallback, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { X, Search, Bookmark, LayoutList, LayoutGrid, FolderKanban, Eraser } from 'lucide-react'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { Input } from '@/components/ui/Input'
import { Spinner } from '@/components/ui/Spinner'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { WorkItemFilters } from '@/components/workitems/WorkItemFilters'
import { BoardView } from '@/components/workitems/BoardView'
import { InboxButton } from '@/components/workitems/InboxButton'
import { WatchButton } from '@/components/workitems/WatchButton'
import { WorkItemMobileCard } from '@/components/workitems/WorkItemMobileCard'
import { DataTable, type Column } from '@/components/ui/DataTable'
import { MultiSelect, type MultiSelectOption } from '@/components/ui/MultiSelect'
import { Tooltip } from '@/components/ui/Tooltip'
import { SLAIndicator } from '@/components/SLAIndicator'
import { useInboxItems } from '@/hooks/useInbox'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
import { useColumnWidths } from '@/hooks/useColumnWidths'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { toUrlSegment } from '@/hooks/useNamespacePath'
import { useDebounce } from '@/hooks/useDebounce'
import { useProjects, useMembers } from '@/hooks/useProjects'
import { useWatchedItems, useWatchedItemIDs } from '@/hooks/useWorkItems'
import { useProjectWorkflow, useAvailableStatuses } from '@/hooks/useWorkflows'
import { useMilestones } from '@/hooks/useMilestones'
import { listWatchedItems, type WorkItem, type WorkItemFilter } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

function isItemCompleted(status: string, statuses: WorkflowStatus[]): boolean {
  const category = statuses.find((s) => s.name === status)?.category
  return category === 'done' || category === 'cancelled'
}

function getDescriptionPreview(description: string): string {
  const line = description.split('\n').find(l => l.trim() !== '')
  if (!line) return ''
  return line.trim().replace(/^#+\s+/, '').replace(/[*_~`[\]]/g, '')
}

type WatchlistViewMode = 'list' | 'board'

function WatchlistSearchBar({ search, onSearchChange }: { search: string; onSearchChange: (v: string) => void }) {
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
        className="pr-8"
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

export default function WatchlistPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const itemUrl = (item: WorkItem) =>
    `/${toUrlSegment(item.namespace_slug ?? 'default')}/projects/${item.project_key}/items/${item.item_number}`
  const { data: projects } = useProjects()
  const setPreferenceMutation = useSetPreference()

  // Persisted watchlist filters
  interface WatchlistPrefs {
    projects?: string[]
    filter?: WorkItemFilter
    sort?: string
    order?: 'asc' | 'desc'
    viewMode?: WatchlistViewMode
  }
  const { data: savedPrefs } = usePreference<WatchlistPrefs>('watchlistFilters')
  const [prefsLoaded, setPrefsLoaded] = useState(false)
  const [selectedProjects, setSelectedProjects] = useState<string[]>([])
  const [filter, setFilter] = useState<WorkItemFilter>({})
  const [sort, setSort] = useState('created_at')
  const [order, setOrder] = useState<'asc' | 'desc'>('desc')
  const [viewMode, setViewMode] = useState<WatchlistViewMode>('list')
  const [search, setSearch] = useState('')
  const [activeRow, setActiveRow] = useState(-1)
  const activeRowStorageKey = 'taskwondo_activeRow_watchlist'
  const restoredRef = useRef(false)
  const [projectModalOpen, setProjectModalOpen] = useState(false)
  const [projectSearch, setProjectSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)

  // Load saved preferences once
  useEffect(() => {
    if (prefsLoaded || savedPrefs === undefined) return
    if (savedPrefs && typeof savedPrefs === 'object') {
      if (Array.isArray(savedPrefs.projects)) setSelectedProjects(savedPrefs.projects)
      if (savedPrefs.filter) setFilter(savedPrefs.filter)
      if (savedPrefs.sort) setSort(savedPrefs.sort)
      if (savedPrefs.order) setOrder(savedPrefs.order)
      if (savedPrefs.viewMode) setViewMode(savedPrefs.viewMode)
    }
    setPrefsLoaded(true)
  }, [savedPrefs, prefsLoaded])

  // Persist filter changes (debounced via mutate coalescing)
  const savePrefs = useCallback((updates: Partial<WatchlistPrefs>) => {
    const current: WatchlistPrefs = {
      projects: selectedProjects,
      filter,
      sort,
      order,
      viewMode,
    }
    setPreferenceMutation.mutate({ key: 'watchlistFilters', value: { ...current, ...updates } })
  }, [selectedProjects, filter, sort, order, viewMode, setPreferenceMutation])

  function handleProjectsChange(keys: string[]) {
    setSelectedProjects(keys)
    savePrefs({ projects: keys })
  }

  const projectOptions: MultiSelectOption[] = useMemo(() =>
    (projects ?? []).map((p) => ({ value: p.key, label: p.name })),
  [projects])

  // Use first selected project (or first available) for project-scoped data (statuses, members, milestones)
  const primaryProject = selectedProjects[0] || (projects?.[0]?.key ?? '')
  const isSingleProject = selectedProjects.length === 1

  // Project-scoped data
  const { statuses, transitionsMap } = useProjectWorkflow(primaryProject)
  const { data: allStatuses } = useAvailableStatuses(primaryProject)
  const { data: members } = useMembers(primaryProject)
  const { data: milestones } = useMilestones(primaryProject)

  // Show dates preference
  const { data: showDatesPref } = usePreference<boolean>('showDates')
  const showDates = showDatesPref ?? true

  // Column widths
  const { columnWidths, onColumnResize, resetColumnWidth } = useColumnWidths()

  const activeFilter = useMemo(() => ({
    ...filter,
    q: debouncedSearch || undefined,
    sort,
    order,
    limit: viewMode === 'board' ? 200 : 50,
  }), [filter, debouncedSearch, viewMode, sort, order])

  const { data: result, isLoading } = useWatchedItems(selectedProjects, activeFilter)
  const items = result?.data ?? []

  // Pagination
  const [loadedPages, setLoadedPages] = useState<WorkItem[][]>([])
  const allItems = useMemo(() => {
    if (loadedPages.length === 0) return items
    return [...items, ...loadedPages.flat()]
  }, [items, loadedPages])

  // Reset pagination on filter change
  const filterKey = JSON.stringify(activeFilter)
  const [prevFilterKey, setPrevFilterKey] = useState(filterKey)
  if (filterKey !== prevFilterKey) {
    setPrevFilterKey(filterKey)
    setLoadedPages([])
    setActiveRow(-1)
  }

  // Inbox data for InboxButton
  const { data: inboxData } = useInboxItems()
  const inboxByWorkItemId = useMemo(() => {
    const map = new Map<string, string>()
    if (inboxData?.items) {
      for (const item of inboxData.items) map.set(item.work_item_id, item.id)
    }
    return map
  }, [inboxData])

  // Watched item IDs for WatchButton
  const { data: watchedIds } = useWatchedItemIDs()
  const watchedItemIdSet = useMemo(() => new Set(watchedIds ?? []), [watchedIds])

  // Restore active row from sessionStorage after navigating back
  useEffect(() => {
    if (restoredRef.current || allItems.length === 0) return
    const stored = sessionStorage.getItem(activeRowStorageKey)
    if (stored) {
      const id = stored
      const idx = allItems.findIndex((i) => i.id === id)
      if (idx >= 0) setActiveRow(idx)
      sessionStorage.removeItem(activeRowStorageKey)
    }
    restoredRef.current = true
  }, [allItems])

  // Keyboard navigation
  useKeyboardShortcut([{ key: 'ArrowDown' }, { key: 'j' }], () => setActiveRow((prev) => Math.min(prev + 1, allItems.length - 1)), viewMode === 'list')
  useKeyboardShortcut([{ key: 'ArrowUp' }, { key: 'k' }], () => setActiveRow((prev) => Math.max(prev - 1, 0)), viewMode === 'list')
  useKeyboardShortcut([{ key: 'Enter' }, { key: 'o' }], () => {
    if (activeRow >= 0 && activeRow < allItems.length) {
      const item = allItems[activeRow]
      sessionStorage.setItem(activeRowStorageKey, item.id)
      navigate(itemUrl(item), { state: { from: 'watchlist' } })
    }
  }, activeRow >= 0)
  useKeyboardShortcut({ key: 'Escape' }, () => setActiveRow(-1), activeRow >= 0)

  function handleFilterChange(f: WorkItemFilter) {
    setFilter(f)
    savePrefs({ filter: f })
  }

  function handleSearchChange(q: string) {
    setSearch(q)
  }

  function handleViewChange(v: WatchlistViewMode) {
    setViewMode(v)
    savePrefs({ viewMode: v })
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
    savePrefs({ sort: sortKey, order: newOrder })
  }

  function handleClearFilters() {
    setSelectedProjects([])
    const f: WorkItemFilter = {}
    setFilter(f)
    setSearch('')
    setViewMode('list')
    setSort('created_at')
    setOrder('desc')
    setPreferenceMutation.mutate({ key: 'watchlistFilters', value: { projects: [], filter: f, sort: 'created_at', order: 'desc', viewMode: 'list' } })
  }

  const columns: Column<WorkItem>[] = [
    {
      key: 'display_id',
      header: t('workitems.table.id'),
      className: 'w-[102px]',
      sortKey: 'item_number',
      render: (row) => {
        const done = isItemCompleted(row.status, allStatuses ?? statuses)
        return <span className={`font-mono ${done ? 'text-gray-400 dark:text-gray-500' : 'text-gray-500 dark:text-gray-400'}`}>{row.display_id}</span>
      },
    },
    {
      key: 'type',
      header: t('workitems.table.type'),
      className: 'w-20',
      sortKey: 'type',
      render: (row) => {
        const done = isItemCompleted(row.status, allStatuses ?? statuses)
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
        const done = isItemCompleted(row.status, allStatuses ?? statuses)
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
            <span className="shrink-0 ml-auto flex items-center gap-1" onClick={(e) => e.stopPropagation()}>
              <span className={`${watchedItemIdSet.has(row.id) ? 'opacity-100' : 'lg:opacity-0 lg:group-hover:opacity-100'} transition-opacity`}>
                <WatchButton projectKey={row.project_key} itemNumber={row.item_number} isWatching={watchedItemIdSet.has(row.id)} />
              </span>
              <span className={`${inboxByWorkItemId.get(row.id) ? 'opacity-100' : 'lg:opacity-0 lg:group-hover:opacity-100'} transition-opacity`}>
                <InboxButton workItemId={row.id} inboxItemId={inboxByWorkItemId.get(row.id)} />
              </span>
            </span>
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
        const done = isItemCompleted(row.status, allStatuses ?? statuses)
        return <span className={done ? 'opacity-40' : ''}><PriorityBadge priority={row.priority} /></span>
      },
    },
    {
      key: 'sla',
      header: t('sla.columnHeader'),
      className: 'w-[110px]',
      sortKey: 'sla_target_at',
      render: (row) => {
        const done = isItemCompleted(row.status, allStatuses ?? statuses)
        return <span className={done ? 'opacity-40' : ''}><SLAIndicator sla={row.sla} /></span>
      },
    },
    {
      key: 'updated',
      header: t('workitems.table.updated'),
      className: 'w-[130px]',
      sortKey: 'updated_at',
      render: (row) => {
        const done = isItemCompleted(row.status, allStatuses ?? statuses)
        return <span className={done ? 'text-gray-300 dark:text-gray-600' : 'text-gray-500 dark:text-gray-400'}>{new Date(row.updated_at).toLocaleDateString()}</span>
      },
    },
  ]

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-4">
        <div className="flex items-center gap-3 min-w-0 shrink lg:shrink-0">
          <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 truncate">
            <span className="lg:hidden">{t('watchlist.titleShort')}</span>
            <span className="hidden lg:inline">{t('watchlist.title')}</span>
          </h2>
          {result && (
            <span className="text-sm text-gray-500 dark:text-gray-400">
              ({result.meta.total})
            </span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <div className="hidden lg:block flex-1 min-w-0 max-w-md">
            <WatchlistSearchBar search={search} onSearchChange={handleSearchChange} />
          </div>
          <div className="inline-flex rounded-md shadow-sm">
            <button
              className={`flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-l-md border ${
                viewMode === 'list' ? 'bg-indigo-50 text-indigo-700 border-indigo-300 dark:bg-indigo-900/30 dark:text-indigo-300 dark:border-indigo-700' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700'
              }`}
              onClick={() => handleViewChange('list')}
            >
              <LayoutList className="h-4 w-4" />
              {t('workitems.view.list')}
            </button>
            <button
              className={`flex items-center gap-1.5 px-3 py-2 text-sm font-medium rounded-r-md border-t border-r border-b ${
                viewMode === 'board' ? 'bg-indigo-50 text-indigo-700 border-indigo-300 dark:bg-indigo-900/30 dark:text-indigo-300 dark:border-indigo-700' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50 dark:bg-gray-800 dark:text-gray-300 dark:border-gray-600 dark:hover:bg-gray-700'
              }`}
              onClick={() => handleViewChange('board')}
            >
              <LayoutGrid className="h-4 w-4" />
              {t('workitems.view.board')}
            </button>
          </div>
        </div>
      </div>

      <WorkItemFilters
        filter={filter}
        onFilterChange={handleFilterChange}
        statuses={allStatuses ?? statuses}
        milestones={isSingleProject ? (milestones ?? []) : []}
        members={isSingleProject ? (members ?? []) : []}
        search={search}
        onSearchChange={handleSearchChange}
        sort={sort}
        order={order}
        onSort={handleSort}
        onOrderChange={(newOrder) => setOrder(newOrder)}
        showDates={showDates}
        onShowDatesChange={(v) => setPreferenceMutation.mutate({ key: 'showDates', value: v })}
        onClearFilters={handleClearFilters}
        leadingContent={
          <MultiSelect
            options={projectOptions}
            selected={selectedProjects}
            onChange={handleProjectsChange}
            placeholder={t('watchlist.allProjects')}
            searchable
          />
        }
        leadingContentMobileButton={
          <>
            <button
              onClick={() => setProjectModalOpen(true)}
              className="relative shrink-0 p-2.5 rounded-md border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
              aria-label={t('watchlist.allProjects')}
            >
              <FolderKanban className="h-5 w-5" />
              {selectedProjects.length > 0 && (
                <span className="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-indigo-600 text-[10px] font-bold text-white">
                  {selectedProjects.length}
                </span>
              )}
            </button>
            <Modal open={projectModalOpen} onClose={() => { setProjectModalOpen(false); setProjectSearch('') }} position="top" containerClassName="!pt-[10.3rem]" className="!h-[60vh] !flex !flex-col !overflow-hidden" title={
              <span className="flex items-center flex-1 min-w-0">
                <span className="truncate">{t('watchlist.filterByProject')}</span>
                <span className="flex items-center ml-auto shrink-0">
                  <button
                    onClick={() => handleProjectsChange([])}
                    className="p-2.5 rounded-md border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-gray-100 dark:hover:bg-gray-700"
                    aria-label={t('workitems.filters.clearAll')}
                  >
                    <Eraser className="h-5 w-5" />
                  </button>
                </span>
              </span>
            }>
              <div className="flex flex-col flex-1 min-h-0">
                <div className="pb-2">
                  <div className="relative">
                    <Search className="absolute left-2.5 top-1/2 -translate-y-1/2 h-3.5 w-3.5 text-gray-400" />
                    <input
                      type="text"
                      value={projectSearch}
                      onChange={(e) => setProjectSearch(e.target.value)}
                      placeholder={t('common.search')}
                      className="w-full pl-8 pr-3 py-1.5 text-sm rounded border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-700 text-gray-900 dark:text-gray-100 focus:outline-none focus:ring-1 focus:ring-indigo-500"
                    />
                  </div>
                </div>
                <div className="flex-1 overflow-y-auto space-y-1">
                  {projectOptions
                    .filter((opt) => !projectSearch || opt.label.toLowerCase().includes(projectSearch.toLowerCase()))
                    .map((opt) => {
                      const checked = selectedProjects.includes(opt.value)
                      return (
                        <label key={opt.value} className="flex items-center gap-3 px-3 py-2.5 rounded-md hover:bg-gray-50 dark:hover:bg-gray-700 cursor-pointer text-sm text-gray-700 dark:text-gray-300">
                          <input
                            type="checkbox"
                            checked={checked}
                            onChange={() => {
                              const next = checked ? selectedProjects.filter((k) => k !== opt.value) : [...selectedProjects, opt.value]
                              handleProjectsChange(next)
                            }}
                            className="rounded border-gray-300 text-indigo-600 focus:ring-indigo-500"
                          />
                          {opt.label}
                        </label>
                      )
                    })}
                </div>
              </div>
            </Modal>
          </>
        }
      />

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Spinner size="lg" />
        </div>
      ) : allItems.length === 0 && !debouncedSearch && !filter.type?.length && !filter.priority?.length &&
          !filter.status?.length && !filter.assignee?.length && !filter.milestone?.length &&
          selectedProjects.length === 0 ? (
        <div className="flex flex-col items-center justify-center h-64 text-gray-500 dark:text-gray-400">
          <Bookmark className="h-12 w-12 mb-4 opacity-30" />
          <p className="text-lg font-medium">{t('watchlist.empty')}</p>
          <p className="text-sm mt-1">{t('watchlist.emptyHint')}</p>
        </div>
      ) : viewMode === 'list' ? (
        <>
          {/* Desktop: table view */}
          <div className="hidden lg:block border dark:border-gray-700 rounded-lg overflow-hidden">
            <DataTable
              columns={columns}
              data={allItems}
              onRowClick={(row) => { sessionStorage.setItem(activeRowStorageKey, row.id); navigate(itemUrl(row), { state: { from: 'watchlist' } }) }}
              emptyMessage={t('workitems.empty')}
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
            {allItems.map((item) => {
              const assigneeName = item.assignee_id
                ? members?.find(m => m.user_id === item.assignee_id)?.display_name ?? t('userPicker.unassigned')
                : t('userPicker.unassigned')
              return (
                <WorkItemMobileCard
                  key={item.id}
                  item={item}
                  statuses={allStatuses ?? statuses}
                  showDates={showDates}
                  assigneeName={assigneeName}
                  inboxItemId={inboxByWorkItemId.get(item.id)}
                  isWatching={watchedItemIdSet.has(item.id)}
                  isCompleted={isItemCompleted(item.status, allStatuses ?? statuses)}
                  onClick={() => { sessionStorage.setItem(activeRowStorageKey, item.id); navigate(itemUrl(item), { state: { from: 'watchlist' } }) }}
                />
              )
            })}
          </div>

          {result?.meta.has_more && (
            <div className="flex justify-center pt-2">
              <Button
                variant="secondary"
                onClick={async () => {
                  const lastItem = allItems[allItems.length - 1]
                  if (!lastItem) return
                  const next = await listWatchedItems(selectedProjects, { ...activeFilter, cursor: lastItem.id })
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
          projectKey={primaryProject}
          items={allItems}
          statuses={statuses}
          transitionsMap={transitionsMap}
          readOnly
          onItemClick={(item) => { sessionStorage.setItem(activeRowStorageKey, item.id); navigate(itemUrl(item), { state: { from: 'watchlist' } }) }}
        />
      )}
    </div>
  )
}
