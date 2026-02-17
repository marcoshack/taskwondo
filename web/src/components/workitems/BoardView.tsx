import { useState } from 'react'
import { useUpdateWorkItem } from '@/hooks/useWorkItems'
import { PriorityBadge } from './PriorityBadge'
import { TypeBadge } from './TypeBadge'
import type { WorkItem } from '@/api/workitems'
import type { WorkflowStatus, WorkflowTransition } from '@/api/workflows'

interface BoardViewProps {
  projectKey: string
  items: WorkItem[]
  statuses: WorkflowStatus[]
  transitionsMap?: Record<string, WorkflowTransition[]>
  onItemClick: (item: WorkItem) => void
}

export function BoardView({ projectKey, items, statuses, transitionsMap, onItemClick }: BoardViewProps) {
  const updateMutation = useUpdateWorkItem(projectKey)

  const sortedStatuses = [...statuses].sort((a, b) => a.position - b.position)

  const itemsByStatus = new Map<string, WorkItem[]>()
  for (const status of sortedStatuses) {
    itemsByStatus.set(status.name, [])
  }
  for (const item of items) {
    const list = itemsByStatus.get(item.status)
    if (list) list.push(item)
  }

  const categoryDot: Record<string, string> = {
    todo: 'bg-gray-400',
    in_progress: 'bg-blue-400',
    done: 'bg-green-400',
    cancelled: 'bg-red-400',
  }

  return (
    <div className="flex gap-4 overflow-x-auto pb-4">
      {sortedStatuses.map((status) => (
        <div key={status.name} className="min-w-[280px] w-72 shrink-0">
          <div className="flex items-center gap-2 mb-3 px-1">
            <span className={`w-2.5 h-2.5 rounded-full ${categoryDot[status.category] ?? 'bg-gray-400'}`} />
            <h3 className="text-sm font-medium text-gray-700">{status.display_name}</h3>
            <span className="text-xs text-gray-400">{itemsByStatus.get(status.name)?.length ?? 0}</span>
          </div>
          <div className="space-y-2">
            {(itemsByStatus.get(status.name) ?? []).map((item) => (
              <BoardCard
                key={item.id}
                item={item}
                transitionsMap={transitionsMap}
                statuses={statuses}
                onClick={() => onItemClick(item)}
                onStatusChange={(newStatus) => {
                  updateMutation.mutate({ itemNumber: item.item_number, input: { status: newStatus } })
                }}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

function BoardCard({
  item,
  transitionsMap,
  statuses,
  onClick,
  onStatusChange,
}: {
  item: WorkItem
  transitionsMap?: Record<string, { to_status: string }[]>
  statuses: WorkflowStatus[]
  onClick: () => void
  onStatusChange: (status: string) => void
}) {
  const [showMenu, setShowMenu] = useState(false)
  const allowed = transitionsMap?.[item.status]?.map((t) => t.to_status) ?? []

  return (
    <div
      className="bg-white rounded-lg border border-gray-200 p-3 shadow-sm hover:shadow-md cursor-pointer relative"
      onClick={onClick}
    >
      <div className="flex items-center justify-between mb-1">
        <span className="text-xs font-mono text-gray-400">{item.display_id}</span>
        {allowed.length > 0 && (
          <div className="relative">
            <button
              className="text-gray-400 hover:text-gray-600 text-xs px-1"
              onClick={(e) => { e.stopPropagation(); setShowMenu(!showMenu) }}
              title="Move to..."
            >
              &hellip;
            </button>
            {showMenu && (
              <div className="absolute right-0 top-5 z-10 bg-white border border-gray-200 rounded-md shadow-lg py-1 min-w-[140px]">
                {allowed.map((toStatus) => {
                  const ws = statuses.find((s) => s.name === toStatus)
                  return (
                    <button
                      key={toStatus}
                      className="block w-full text-left px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
                      onClick={(e) => {
                        e.stopPropagation()
                        onStatusChange(toStatus)
                        setShowMenu(false)
                      }}
                    >
                      {ws?.display_name ?? toStatus}
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        )}
      </div>
      <p className="text-sm font-medium text-gray-900 mb-2 line-clamp-2">{item.title}</p>
      <div className="flex items-center gap-2">
        <TypeBadge type={item.type} />
        <PriorityBadge priority={item.priority} />
      </div>
    </div>
  )
}
