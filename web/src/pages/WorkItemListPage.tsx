import { useState, useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useWorkItems, useCreateWorkItem, useBulkUpdateWorkItems } from '@/hooks/useWorkItems'
import { useMembers } from '@/hooks/useProjects'
import { useProjectWorkflow } from '@/hooks/useWorkflows'
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

export function WorkItemListPage() {
  const { projectKey } = useParams<{ projectKey: string }>()
  const navigate = useNavigate()

  const [filter, setFilter] = useState<WorkItemFilter>({})
  const [search, setSearch] = useState('')
  const [viewMode, setViewMode] = useState<ViewMode>('list')
  const [showCreate, setShowCreate] = useState(false)
  const [selected, setSelected] = useState<Set<number>>(new Set())

  const debouncedSearch = useDebounce(search, 300)

  const activeFilter = useMemo(() => ({
    ...filter,
    q: debouncedSearch || undefined,
    limit: viewMode === 'board' ? 200 : 50,
  }), [filter, debouncedSearch, viewMode])

  const { data: result, isLoading } = useWorkItems(projectKey ?? '', activeFilter)
  const { statuses, transitionsMap } = useProjectWorkflow(projectKey ?? '')
  const { data: members } = useMembers(projectKey ?? '')
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

  // Combine first-page data with loaded extra pages
  const allItems = useMemo(() => {
    if (loadedPages.length === 0) return items
    return [...items, ...loadedPages.flat()]
  }, [items, loadedPages])

  // Reset extra pages when filter changes
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
              onClick={() => setViewMode('list')}
            >
              List
            </button>
            <button
              className={`px-3 py-1.5 text-sm font-medium rounded-r-md border-t border-r border-b ${
                viewMode === 'board' ? 'bg-indigo-50 text-indigo-700 border-indigo-300' : 'bg-white text-gray-700 border-gray-300 hover:bg-gray-50'
              }`}
              onClick={() => setViewMode('board')}
            >
              Board
            </button>
          </div>
          <Button onClick={() => setShowCreate(true)}>New Item</Button>
        </div>
      </div>

      <WorkItemFilters
        filter={filter}
        onFilterChange={setFilter}
        statuses={statuses}
        search={search}
        onSearchChange={setSearch}
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
            {/* Select-all checkbox in header */}
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

      {/* Create modal */}
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
