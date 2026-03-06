import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronLeft, ChevronRight } from 'lucide-react'
import { useUpdateWorkItem } from '@/hooks/useWorkItems'
import { PriorityBadge } from './PriorityBadge'
import { TypeBadge } from './TypeBadge'
import { SLAIndicator } from '@/components/SLAIndicator'
import { Tooltip } from '@/components/ui/Tooltip'
import type { WorkItem } from '@/api/workitems'
import type { WorkflowStatus, WorkflowTransition } from '@/api/workflows'

const COLUMN_WIDTH = 304 // 288px (w-72) + 16px gap (gap-4 = 1rem)

interface DraggedItem {
  itemNumber: number
  fromStatus: string
}

interface BoardViewProps {
  projectKey: string
  items: WorkItem[]
  statuses: WorkflowStatus[]
  transitionsMap?: Record<string, WorkflowTransition[]>
  onItemClick: (item: WorkItem) => void
  readOnly?: boolean
}

export function BoardView({ projectKey, items, statuses, transitionsMap, onItemClick, readOnly = false }: BoardViewProps) {
  const { t } = useTranslation()
  const updateMutation = useUpdateWorkItem(projectKey)
  const [draggedItem, setDraggedItem] = useState<DraggedItem | null>(null)
  const [hoveredColumn, setHoveredColumn] = useState<string | null>(null)
  const dragCounterRef = useRef(new Map<string, number>())

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

  const canDropOnStatus = useCallback((targetStatus: string) => {
    if (!draggedItem || !transitionsMap) return false
    if (draggedItem.fromStatus === targetStatus) return false
    const allowed = transitionsMap[draggedItem.fromStatus]?.map((tr) => tr.to_status) ?? []
    return allowed.includes(targetStatus)
  }, [draggedItem, transitionsMap])

  const handleColumnDragOver = useCallback((e: React.DragEvent, statusName: string) => {
    if (canDropOnStatus(statusName)) {
      e.preventDefault()
      e.dataTransfer.dropEffect = 'move'
    }
  }, [canDropOnStatus])

  const handleColumnDragEnter = useCallback((e: React.DragEvent, statusName: string) => {
    e.preventDefault()
    const counters = dragCounterRef.current
    counters.set(statusName, (counters.get(statusName) ?? 0) + 1)
    if (canDropOnStatus(statusName)) {
      setHoveredColumn(statusName)
    }
  }, [canDropOnStatus])

  const handleColumnDragLeave = useCallback((_e: React.DragEvent, statusName: string) => {
    const counters = dragCounterRef.current
    const next = (counters.get(statusName) ?? 1) - 1
    counters.set(statusName, next)
    if (next <= 0) {
      counters.delete(statusName)
      if (hoveredColumn === statusName) setHoveredColumn(null)
    }
  }, [hoveredColumn])

  const handleColumnDrop = useCallback((e: React.DragEvent, statusName: string) => {
    e.preventDefault()
    dragCounterRef.current.clear()
    setHoveredColumn(null)
    if (!draggedItem || !canDropOnStatus(statusName)) return
    updateMutation.mutate({ itemNumber: draggedItem.itemNumber, input: { status: statusName } })
    setDraggedItem(null)
  }, [draggedItem, canDropOnStatus, updateMutation])

  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(false)

  const updateScrollIndicators = useCallback(() => {
    const el = scrollContainerRef.current
    if (!el) return
    setCanScrollLeft(el.scrollLeft > 0)
    setCanScrollRight(el.scrollLeft + el.clientWidth < el.scrollWidth - 1)
  }, [])

  useEffect(() => {
    const el = scrollContainerRef.current
    if (!el) return
    updateScrollIndicators()
    el.addEventListener('scroll', updateScrollIndicators, { passive: true })
    const ro = new ResizeObserver(updateScrollIndicators)
    ro.observe(el)
    return () => {
      el.removeEventListener('scroll', updateScrollIndicators)
      ro.disconnect()
    }
  }, [updateScrollIndicators, sortedStatuses.length])

  const scrollBy = useCallback((direction: 'left' | 'right') => {
    const el = scrollContainerRef.current
    if (!el) return
    const currentScroll = el.scrollLeft
    if (direction === 'right') {
      // Snap to the start of the next hidden column
      const nextColumnStart = (Math.floor(currentScroll / COLUMN_WIDTH) + 1) * COLUMN_WIDTH
      el.scrollTo({ left: nextColumnStart, behavior: 'smooth' })
    } else {
      // Snap to the start of the previous column
      const prevColumnStart = (Math.ceil(currentScroll / COLUMN_WIDTH) - 1) * COLUMN_WIDTH
      el.scrollTo({ left: Math.max(0, prevColumnStart), behavior: 'smooth' })
    }
  }, [])

  return (
    <div className="relative">
      {canScrollLeft && (
        <div className="absolute left-0 top-0 bottom-0 z-20 pointer-events-none" style={{ width: 0 }}>
          <button
            type="button"
            aria-label={t('workitems.board.scrollLeft')}
            className="pointer-events-auto sticky top-1/2 -translate-y-1/2 ml-1 flex items-center justify-center w-8 h-8 rounded-full bg-white/90 dark:bg-gray-800/90 border border-gray-200 dark:border-gray-600 shadow-md hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-600 dark:text-gray-300 transition-colors"
            onClick={() => scrollBy('left')}
          >
            <ChevronLeft size={20} />
          </button>
        </div>
      )}
      {canScrollRight && (
        <div className="absolute right-0 top-0 bottom-0 z-20 pointer-events-none" style={{ width: 0 }}>
          <button
            type="button"
            aria-label={t('workitems.board.scrollRight')}
            className="pointer-events-auto sticky top-1/2 -translate-y-1/2 -ml-9 flex items-center justify-center w-8 h-8 rounded-full bg-white/90 dark:bg-gray-800/90 border border-gray-200 dark:border-gray-600 shadow-md hover:bg-gray-100 dark:hover:bg-gray-700 text-gray-600 dark:text-gray-300 transition-colors"
            onClick={() => scrollBy('right')}
          >
            <ChevronRight size={20} />
          </button>
        </div>
      )}
      <div ref={scrollContainerRef} className="flex gap-4 overflow-x-auto pb-4">
        {sortedStatuses.map((status) => {
          const isValidTarget = draggedItem !== null && canDropOnStatus(status.name)
          const isHovered = hoveredColumn === status.name
          const columnClasses = isHovered
            ? 'border-2 border-dashed border-indigo-400 dark:border-indigo-500 bg-indigo-50/50 dark:bg-indigo-900/20 rounded-lg'
            : isValidTarget
              ? 'border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg'
              : draggedItem !== null
                ? 'border-2 border-transparent rounded-lg opacity-60'
                : ''

          return (
            <div
              key={status.name}
              className={`min-w-[280px] w-72 shrink-0 p-2 transition-colors ${columnClasses}`}
              onDragOver={(e) => handleColumnDragOver(e, status.name)}
              onDragEnter={(e) => handleColumnDragEnter(e, status.name)}
              onDragLeave={(e) => handleColumnDragLeave(e, status.name)}
              onDrop={(e) => handleColumnDrop(e, status.name)}
            >
              <div className="flex items-center gap-2 mb-3 px-1">
                <span className={`w-2.5 h-2.5 rounded-full ${categoryDot[status.category] ?? 'bg-gray-400'}`} />
                <h3 className="text-sm font-medium text-gray-700 dark:text-gray-300">{t(`workitems.statuses.${status.name}`, { defaultValue: status.display_name })}</h3>
                <span className="text-xs text-gray-400 dark:text-gray-500">{itemsByStatus.get(status.name)?.length ?? 0}</span>
              </div>
              <div className="space-y-2">
                {(itemsByStatus.get(status.name) ?? []).map((item) => (
                  <BoardCard
                    key={item.id}
                    item={item}
                    transitionsMap={transitionsMap}
                    statuses={statuses}
                    isDragging={draggedItem?.itemNumber === item.item_number}
                    readOnly={readOnly}
                    onClick={() => onItemClick(item)}
                    onStatusChange={(newStatus) => {
                      updateMutation.mutate({ itemNumber: item.item_number, input: { status: newStatus } })
                    }}
                    onDragStart={() => setDraggedItem({ itemNumber: item.item_number, fromStatus: item.status })}
                    onDragEnd={() => {
                      setDraggedItem(null)
                      setHoveredColumn(null)
                      dragCounterRef.current.clear()
                    }}
                  />
                ))}
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function BoardCard({
  item,
  transitionsMap,
  statuses,
  isDragging,
  readOnly,
  onClick,
  onStatusChange,
  onDragStart,
  onDragEnd,
}: {
  item: WorkItem
  transitionsMap?: Record<string, { to_status: string }[]>
  statuses: WorkflowStatus[]
  isDragging?: boolean
  readOnly?: boolean
  onClick: () => void
  onStatusChange: (status: string) => void
  onDragStart: () => void
  onDragEnd: () => void
}) {
  const { t } = useTranslation()
  const [showMenu, setShowMenu] = useState(false)
  const allowed = transitionsMap?.[item.status]?.map((tr) => tr.to_status) ?? []

  return (
    <div
      className={`bg-white dark:bg-gray-800 rounded-lg border border-gray-200 dark:border-gray-700 p-3 shadow-sm hover:shadow-md cursor-pointer relative ${isDragging ? 'opacity-50' : ''}`}
      draggable={!readOnly}
      onClick={onClick}
      onDragStart={(e) => {
        e.dataTransfer.effectAllowed = 'move'
        e.dataTransfer.setData('text/plain', String(item.item_number))
        onDragStart()
      }}
      onDragEnd={onDragEnd}
    >
      <div className="flex items-center gap-1.5 mb-1">
        <span className="text-xs font-bold font-mono text-gray-600 dark:text-gray-400">{item.display_id}</span>
        <TypeBadge type={item.type} />
        <PriorityBadge priority={item.priority} />
        {allowed.length > 0 && !readOnly && (
          <div className="relative ml-auto">
            <Tooltip content={t('workitems.view.moveTo')}>
              <button
                className="text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300 text-xs px-1"
                onClick={(e) => { e.stopPropagation(); setShowMenu(!showMenu) }}
              >
              &hellip;
              </button>
            </Tooltip>
            {showMenu && (
              <div className="absolute right-0 top-5 z-10 bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-md shadow-lg py-1 min-w-[140px]">
                {allowed.map((toStatus) => {
                  const ws = statuses.find((s) => s.name === toStatus)
                  return (
                    <button
                      key={toStatus}
                      className="block w-full text-left px-3 py-1.5 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-50 dark:hover:bg-gray-700"
                      onClick={(e) => {
                        e.stopPropagation()
                        onStatusChange(toStatus)
                        setShowMenu(false)
                      }}
                    >
                      {t(`workitems.statuses.${toStatus}`, { defaultValue: ws?.display_name ?? toStatus })}
                    </button>
                  )
                })}
              </div>
            )}
          </div>
        )}
      </div>
      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 line-clamp-2">{item.title}</p>
      {item.description && (
        <p className="text-xs text-gray-500 dark:text-gray-400 line-clamp-1 mt-0.5">{item.description}</p>
      )}
      {item.sla && (
        <div className="mt-1.5">
          <SLAIndicator sla={item.sla} compact />
        </div>
      )}
    </div>
  )
}
