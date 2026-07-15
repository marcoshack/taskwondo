import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Flag } from 'lucide-react'
import { ParentEpicBadge, shouldShowParentEpic } from '@/components/workitems/ParentEpicBadge'
import type { WorkItem } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

/** Left accent bar colors by workflow category (TASK-113 nav rail). */
const STATUS_CATEGORY_BAR: Record<string, string> = {
  todo: 'bg-gray-400 dark:bg-gray-500',
  in_progress: 'bg-blue-500 dark:bg-blue-400',
  done: 'bg-green-500 dark:bg-green-400',
  cancelled: 'bg-red-500 dark:bg-red-400',
}

const PRIORITY_FLAG_CLASS: Record<string, string> = {
  critical: 'text-red-600 dark:text-red-400 fill-red-600 dark:fill-red-400',
  high: 'text-amber-600 dark:text-amber-400 fill-amber-600 dark:fill-amber-400',
}

function showPriorityFlag(priority: string): boolean {
  return priority === 'critical' || priority === 'high'
}

export interface CompactRowAssignee {
  name: string
  avatarUrl?: string
}

export interface WorkItemCompactRowProps {
  item: WorkItem
  statuses: WorkflowStatus[]
  /** Keyboard focus highlight (↑↓ navigation). */
  active?: boolean
  /** Open in the split-pane detail panel. */
  selected?: boolean
  /** Unused in rail mode (TASK-113); kept for call-site compatibility. */
  assignee?: CompactRowAssignee | null
  isCompleted?: boolean
  readOnly?: boolean
  /** Unused in rail mode — status edits live in the detail pane. */
  statusOptions?: WorkflowStatus[]
  onSelect: () => void
  /** Unused in rail mode — hover status `<select>` removed (operator decision). */
  onStatusChange?: (status: string) => void
}

/**
 * Navigation-rail work-item row for split-pane list mode (~280–320px).
 * Identity + pick-next only: mono ID, truncated title, category accent bar,
 * lucide flag for high/critical. Chips / dots / avatar / due / status select omitted
 * (TASK-113 / TASK-114).
 */
export function WorkItemCompactRow({
  item,
  statuses,
  active = false,
  selected = false,
  isCompleted = false,
  onSelect,
}: WorkItemCompactRowProps) {
  const { t } = useTranslation()
  const rowRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (selected || active) {
      rowRef.current?.scrollIntoView({ block: 'nearest' })
    }
  }, [selected, active])

  const ws = statuses.find((s) => s.name === item.status)
  const category = ws?.category ?? 'todo'
  const statusLabel = t(`workitems.statuses.${item.status}`, {
    defaultValue: ws?.display_name ?? item.status,
  })
  const statusBar =
    STATUS_CATEGORY_BAR[category] ?? STATUS_CATEGORY_BAR.todo

  const priorityKey = `workitems.priorities.${item.priority}`
  const priorityTranslated = t(priorityKey)
  const priorityLabel =
    priorityTranslated === priorityKey ? item.priority : priorityTranslated
  const markPriority = showPriorityFlag(item.priority)
  const flagClass = PRIORITY_FLAG_CLASS[item.priority] ?? PRIORITY_FLAG_CLASS.high

  const ariaLabel = markPriority
    ? `${item.display_id}, ${statusLabel}, ${priorityLabel} priority: ${item.title}`
    : `${item.display_id}, ${statusLabel}: ${item.title}`

  return (
    <div
      ref={rowRef}
      role="option"
      aria-selected={selected}
      aria-label={ariaLabel}
      tabIndex={-1}
      data-testid="work-item-compact-row"
      data-display-id={item.display_id}
      data-selected={selected || undefined}
      data-active={active || undefined}
      data-status-category={category}
      onClick={onSelect}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onSelect()
        }
      }}
      className={[
        'group relative flex gap-0 cursor-pointer border-b border-gray-100 dark:border-gray-800',
        'outline-none transition-colors',
        selected
          ? 'bg-indigo-50 dark:bg-indigo-950/40'
          : active
            ? 'bg-gray-50 dark:bg-gray-800/80 ring-1 ring-inset ring-gray-200/80 dark:ring-gray-700/80'
            : 'bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800/60',
      ].join(' ')}
    >
      {/* Dual bars: status category + selection indicator (operator decision) */}
      <span
        aria-hidden="true"
        data-testid="status-category-bar"
        className={`w-[3px] shrink-0 self-stretch transition-colors ${statusBar}`}
      />
      <span
        aria-hidden="true"
        data-testid="selection-bar"
        className={[
          'w-0.5 shrink-0 self-stretch transition-colors',
          selected
            ? 'bg-indigo-600 dark:bg-indigo-400'
            : 'bg-transparent group-hover:bg-indigo-300/50 dark:group-hover:bg-indigo-700/50',
        ].join(' ')}
      />
      <div className="min-w-0 flex-1 px-2.5 py-2">
        <div className="flex items-center gap-1.5 min-w-0">
          <span
            className={`shrink-0 font-mono text-[11px] font-semibold leading-4 ${
              isCompleted
                ? 'text-gray-400 dark:text-gray-500'
                : 'text-gray-600 dark:text-gray-300'
            }`}
          >
            {item.display_id}
          </span>
          {shouldShowParentEpic(item) && (
            <ParentEpicBadge
              displayId={item.parent_epic_display_id!}
              title={item.parent_epic_title}
              size="xs"
            />
          )}
          {markPriority && (
            <Flag
              className={`ml-auto h-3 w-3 shrink-0 ${flagClass}`}
              aria-hidden="true"
              data-testid="priority-flag"
              data-priority={item.priority}
            />
          )}
        </div>
        <div
          className={`mt-0.5 truncate text-sm leading-snug ${
            isCompleted
              ? 'line-through text-gray-400 dark:text-gray-500'
              : 'text-gray-900 dark:text-gray-100'
          }`}
          title={item.title}
        >
          {item.title}
        </div>
      </div>
    </div>
  )
}
