import { useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Calendar, Check } from 'lucide-react'
import { Avatar } from '@/components/ui/Avatar'
import { Tooltip } from '@/components/ui/Tooltip'
import type { WorkItem } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

const PRIORITY_CHIP: Record<string, string> = {
  critical: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
  high: 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/40 dark:text-yellow-300',
  medium: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
  low: 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-300',
}

const STATUS_CATEGORY_CHIP: Record<string, string> = {
  todo: 'bg-gray-100 text-gray-700 dark:bg-gray-700 dark:text-gray-300',
  in_progress: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
  done: 'bg-green-100 text-green-700 dark:bg-green-900/40 dark:text-green-300',
  cancelled: 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300',
}

const LABEL_DOT_COLORS = [
  'bg-rose-500',
  'bg-orange-500',
  'bg-amber-500',
  'bg-lime-500',
  'bg-emerald-500',
  'bg-cyan-500',
  'bg-sky-500',
  'bg-indigo-500',
  'bg-violet-500',
  'bg-fuchsia-500',
]

function labelDotColor(label: string): string {
  let hash = 0
  for (let i = 0; i < label.length; i++) {
    hash = (hash * 31 + label.charCodeAt(i)) >>> 0
  }
  return LABEL_DOT_COLORS[hash % LABEL_DOT_COLORS.length]
}

function formatDueDate(due: string): string {
  const d = new Date(due)
  if (Number.isNaN(d.getTime())) return due
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
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
  assignee?: CompactRowAssignee | null
  isCompleted?: boolean
  readOnly?: boolean
  /** Statuses offered by the hover quick-action menu. */
  statusOptions?: WorkflowStatus[]
  onSelect: () => void
  onStatusChange?: (status: string) => void
}

/**
 * Narrow-width work-item row for split-pane list mode (~280–320px).
 * Title on line 1; status / priority / due chips + label dots on line 2.
 */
export function WorkItemCompactRow({
  item,
  statuses,
  active = false,
  selected = false,
  assignee,
  isCompleted = false,
  readOnly = false,
  statusOptions,
  onSelect,
  onStatusChange,
}: WorkItemCompactRowProps) {
  const { t } = useTranslation()
  const rowRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (selected || active) {
      rowRef.current?.scrollIntoView({ block: 'nearest' })
    }
  }, [selected, active])

  const ws = statuses.find((s) => s.name === item.status)
  const statusLabel = t(`workitems.statuses.${item.status}`, {
    defaultValue: ws?.display_name ?? item.status,
  })
  const statusChip =
    STATUS_CATEGORY_CHIP[ws?.category ?? 'todo'] ?? STATUS_CATEGORY_CHIP.todo

  const priorityKey = `workitems.priorities.${item.priority}`
  const priorityTranslated = t(priorityKey)
  const priorityLabel =
    priorityTranslated === priorityKey ? item.priority : priorityTranslated
  const priorityChip = PRIORITY_CHIP[item.priority] ?? PRIORITY_CHIP.low

  const options = (statusOptions ?? statuses).filter((s) => s.name !== item.status)
  const showQuickActions = !readOnly && !!onStatusChange && options.length > 0

  return (
    <div
      ref={rowRef}
      role="option"
      aria-selected={selected}
      tabIndex={-1}
      data-testid="work-item-compact-row"
      data-display-id={item.display_id}
      data-selected={selected || undefined}
      data-active={active || undefined}
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
            ? 'bg-gray-50 dark:bg-gray-800/80'
            : 'bg-white dark:bg-gray-900 hover:bg-gray-50 dark:hover:bg-gray-800/60',
      ].join(' ')}
    >
      <span
        aria-hidden="true"
        className={[
          'w-0.5 shrink-0 self-stretch transition-colors',
          selected ? 'bg-indigo-600 dark:bg-indigo-400' : 'bg-transparent group-hover:bg-indigo-300/60 dark:group-hover:bg-indigo-700/60',
        ].join(' ')}
      />
      <div className="min-w-0 flex-1 px-2.5 py-1.5">
        <div className="flex items-start gap-1.5 min-w-0">
          <div className="min-w-0 flex-1">
            <div className="flex items-baseline gap-1.5 min-w-0">
              <span
                className={`shrink-0 font-mono text-[11px] font-semibold ${
                  isCompleted
                    ? 'text-gray-400 dark:text-gray-500'
                    : 'text-gray-500 dark:text-gray-400'
                }`}
              >
                {item.display_id}
              </span>
              <span
                className={`truncate text-sm leading-snug ${
                  isCompleted
                    ? 'line-through text-gray-400 dark:text-gray-500'
                    : 'font-medium text-gray-900 dark:text-gray-100'
                }`}
                title={item.title}
              >
                {item.title}
              </span>
            </div>
            <div className="mt-0.5 flex flex-wrap items-center gap-1 min-w-0">
              <span
                className={`inline-flex max-w-[7.5rem] truncate rounded px-1.5 py-px text-[10px] font-medium leading-4 ${statusChip}`}
                title={statusLabel}
              >
                {statusLabel}
              </span>
              <span
                className={`inline-flex rounded px-1.5 py-px text-[10px] font-medium leading-4 uppercase tracking-wide ${priorityChip}`}
                title={priorityLabel}
              >
                {priorityLabel.charAt(0).toUpperCase()}
              </span>
              {item.due_date && (
                <span
                  className={`inline-flex items-center gap-0.5 rounded px-1 py-px text-[10px] leading-4 ${
                    isCompleted
                      ? 'text-gray-400 dark:text-gray-500'
                      : 'text-gray-500 dark:text-gray-400'
                  }`}
                  title={t('workitems.form.dueDate')}
                >
                  <Calendar className="h-2.5 w-2.5 shrink-0" aria-hidden="true" />
                  {formatDueDate(item.due_date)}
                </span>
              )}
              {item.labels?.length > 0 && (
                <span className="inline-flex items-center gap-0.5 ml-0.5" aria-label={t('workitems.form.labels')}>
                  {item.labels.slice(0, 5).map((label) => (
                    <Tooltip key={label} content={label}>
                      <span
                        className={`inline-block h-1.5 w-1.5 rounded-full ${labelDotColor(label)}`}
                        data-testid="label-dot"
                        data-label={label}
                      />
                    </Tooltip>
                  ))}
                  {item.labels.length > 5 && (
                    <span className="text-[10px] text-gray-400">+{item.labels.length - 5}</span>
                  )}
                </span>
              )}
            </div>
          </div>
          <div className="shrink-0 flex flex-col items-end gap-1 pt-0.5">
            {assignee ? (
              <Avatar name={assignee.name} avatarUrl={assignee.avatarUrl} size="xs" />
            ) : (
              <span className="h-4.5 w-4.5" aria-hidden="true" />
            )}
            {showQuickActions && (
              <div
                className="opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity"
                onClick={(e) => e.stopPropagation()}
              >
                <label className="sr-only" htmlFor={`compact-status-${item.id}`}>
                  {t('workitems.compactRow.changeStatus')}
                </label>
                <select
                  id={`compact-status-${item.id}`}
                  className="max-w-[7rem] rounded border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 text-[10px] py-0.5 px-1 text-gray-700 dark:text-gray-200"
                  value=""
                  aria-label={t('workitems.compactRow.changeStatus')}
                  onChange={(e) => {
                    const next = e.target.value
                    if (next) onStatusChange?.(next)
                    e.target.value = ''
                  }}
                >
                  <option value="">{t('workitems.compactRow.changeStatus')}</option>
                  {options.map((s) => (
                    <option key={s.name} value={s.name}>
                      {t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name })}
                    </option>
                  ))}
                </select>
              </div>
            )}
            {selected && (
              <Check
                className="h-3 w-3 text-indigo-600 dark:text-indigo-400 opacity-70 group-hover:opacity-0"
                aria-hidden="true"
              />
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
