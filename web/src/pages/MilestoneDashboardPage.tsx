import { useState, useRef, useEffect, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ChevronRight, Pencil, Clock } from 'lucide-react'
import { useMilestone, useMilestoneStats, useUpdateMilestone } from '@/hooks/useMilestones'
import { useWorkItems } from '@/hooks/useWorkItems'
import { useMembers } from '@/hooks/useProjects'
import { useProjectWorkflow } from '@/hooks/useWorkflows'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { Spinner } from '@/components/ui/Spinner'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { Avatar } from '@/components/ui/Avatar'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { ScrollableRow } from '@/components/ui/ScrollableRow'
import { formatDuration } from '@/utils/duration'
import type { Milestone, UpdateMilestoneInput } from '@/api/milestones'
import type { WorkItem } from '@/api/workitems'
import type { ProjectMember } from '@/api/projects'
import type { AxiosError } from 'axios'

// Color maps matching badge components
const typeBarColors: Record<string, string> = {
  bug: 'bg-red-500',
  task: 'bg-blue-500',
  ticket: 'bg-indigo-500',
  feedback: 'bg-yellow-500',
  epic: 'bg-green-500',
}

const priorityBarColors: Record<string, string> = {
  critical: 'bg-red-500',
  high: 'bg-yellow-500',
  medium: 'bg-blue-500',
  low: 'bg-gray-400',
}

const statusCategoryOrder: Record<string, number> = {
  in_progress: 0,
  todo: 1,
  done: 2,
  cancelled: 3,
}

const priorityOrder: Record<string, number> = {
  critical: 0,
  high: 1,
  medium: 2,
  low: 3,
}

export function MilestoneDashboardPage() {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const { projectKey, milestoneId } = useParams<{ projectKey: string; milestoneId: string }>()
  const { user } = useAuth()
  const { data: milestone, isLoading: milestoneLoading } = useMilestone(projectKey ?? '', milestoneId ?? '')
  const { data: stats, isLoading: statsLoading } = useMilestoneStats(projectKey ?? '', milestoneId ?? '')
  const { data: workItemsData, isLoading: itemsLoading } = useWorkItems(projectKey ?? '', {
    milestone: milestoneId ? [milestoneId] : [],
    limit: 200,
  })
  const { data: members } = useMembers(projectKey ?? '')
  const { workflow } = useProjectWorkflow(projectKey ?? '')
  const updateMutation = useUpdateMilestone(projectKey ?? '')

  const [editOpen, setEditOpen] = useState(false)
  const [error, setError] = useState('')

  const currentUserMember = members?.find((m) => m.user_id === user?.id)
  const currentUserRole = currentUserMember?.role ?? (user?.global_role === 'admin' ? 'owner' : null)
  const canManage = currentUserRole === 'owner' || currentUserRole === 'admin' || user?.global_role === 'admin'

  if (milestoneLoading || statsLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    )
  }

  if (!milestone) {
    return <p className="text-red-600">{t('common.notFound')}</p>
  }

  const percent = milestone.total_count > 0 ? Math.round((milestone.closed_count / milestone.total_count) * 100) : 0
  const workItems = workItemsData?.data ?? []
  const memberMap = new Map((members ?? []).map((m) => [m.user_id, m]))

  // Sort work items by status category (in_progress first), then priority, then number desc
  const sortedItems = [...workItems].sort((a, b) => {
    const aCat = getStatusCategory(a.status)
    const bCat = getStatusCategory(b.status)
    const catDiff = (statusCategoryOrder[aCat] ?? 9) - (statusCategoryOrder[bCat] ?? 9)
    if (catDiff !== 0) return catDiff
    const priDiff = (priorityOrder[a.priority] ?? 9) - (priorityOrder[b.priority] ?? 9)
    if (priDiff !== 0) return priDiff
    return b.item_number - a.item_number
  })

  function getStatusCategory(status: string): string {
    const ws = workflow?.statuses?.find((s) => s.name === status)
    return ws?.category ?? 'todo'
  }

  function handleSave(input: UpdateMilestoneInput) {
    setError('')
    updateMutation.mutate(
      { milestoneId: milestone!.id, input },
      {
        onSuccess: () => setEditOpen(false),
        onError: (err) => {
          const axiosErr = err as AxiosError<{ error?: { message?: string } }>
          setError(axiosErr.response?.data?.error?.message ?? t('milestones.updateError'))
        },
      },
    )
  }

  return (
    <div className="max-w-4xl space-y-6">
      {/* Breadcrumb */}
      <nav className="flex items-center gap-1 text-sm text-gray-500 dark:text-gray-400">
        <Link to={p(`/projects/${projectKey}/milestones`)} className="hover:text-gray-700 dark:hover:text-gray-300">
          {t('milestone.dashboard.backToMilestones')}
        </Link>
        <ChevronRight className="h-3.5 w-3.5" />
        <span className="text-gray-900 dark:text-gray-100">{milestone.name}</span>
      </nav>

      {/* Header */}
      <HeaderSection
        milestone={milestone}
        percent={percent}
        canManage={canManage}
        onEdit={() => setEditOpen(true)}
      />

      {/* Summary Counters */}
      <SummaryCounters milestone={milestone} stats={stats} />

      {/* Work Item Breakdown Charts */}
      {stats && (Object.keys(stats.by_type).length > 0 || Object.keys(stats.by_priority).length > 0) && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {Object.keys(stats.by_type).length > 0 && (
            <BreakdownChart
              title={t('milestone.dashboard.byType')}
              data={stats.by_type}
              colorMap={typeBarColors}
              projectKey={projectKey ?? ''}
              milestoneId={milestoneId ?? ''}
              filterKey="type"
            />
          )}
          {Object.keys(stats.by_priority).length > 0 && (
            <BreakdownChart
              title={t('milestone.dashboard.byPriority')}
              data={stats.by_priority}
              colorMap={priorityBarColors}
              projectKey={projectKey ?? ''}
              milestoneId={milestoneId ?? ''}
              filterKey="priority"
            />
          )}
        </div>
      )}

      {/* Label Breakdown */}
      {stats && Object.keys(stats.by_label).length > 0 && (
        <LabelBreakdown labels={stats.by_label} />
      )}

      {/* Time Tracking */}
      {(milestone.total_estimated_seconds > 0 || milestone.total_spent_seconds > 0) && (
        <TimeTrackingSection milestone={milestone} />
      )}

      {/* Work Items Table */}
      <WorkItemsTable
        items={sortedItems}
        memberMap={memberMap}
        projectKey={projectKey ?? ''}
        milestoneId={milestoneId ?? ''}
        isLoading={itemsLoading}
        hasMore={(workItemsData?.meta?.has_more) ?? false}
        statuses={workflow?.statuses}
      />

      {/* Edit Modal */}
      <Modal open={editOpen} onClose={() => setEditOpen(false)} title={t('milestones.editMilestone')}>
        <MilestoneEditForm
          milestone={milestone}
          onSubmit={handleSave}
          onCancel={() => setEditOpen(false)}
          isPending={updateMutation.isPending}
          error={error}
        />
      </Modal>
    </div>
  )
}

// --- Header Section ---

function HeaderSection({
  milestone,
  percent,
  canManage,
  onEdit,
}: {
  milestone: Milestone
  percent: number
  canManage: boolean
  onEdit: () => void
}) {
  const { t } = useTranslation()
  const isClosed = milestone.status === 'closed'

  return (
    <div>
      <div className="flex items-start justify-between gap-2">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-3 flex-wrap">
            <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{milestone.name}</h2>
            <span
              className={`inline-flex items-center px-2 py-0.5 text-xs font-medium rounded-full ${
                isClosed
                  ? 'bg-gray-100 text-gray-600 dark:bg-gray-700 dark:text-gray-400'
                  : 'bg-green-100 text-green-700 dark:bg-green-900 dark:text-green-300'
              }`}
            >
              {isClosed ? t('milestones.statusClosed') : t('milestones.statusOpen')}
            </span>
            {milestone.due_date && <DueDateLabel dueDate={milestone.due_date} isClosed={isClosed} />}
          </div>
          {milestone.description && (
            <p className="text-sm text-gray-500 dark:text-gray-400 mt-1">{milestone.description}</p>
          )}
        </div>
        {canManage && (
          <Button variant="ghost" size="sm" onClick={onEdit}>
            <Pencil className="h-4 w-4" />
          </Button>
        )}
      </div>
      {/* Progress bar */}
      <div className="mt-4">
        <div className="flex items-center justify-between text-sm text-gray-500 dark:text-gray-400 mb-1">
          <span>
            {milestone.total_count > 0
              ? t('milestones.progress', { closed: milestone.closed_count, total: milestone.total_count })
              : t('milestones.progressNone')}
          </span>
          {milestone.total_count > 0 && <span>{percent}%</span>}
        </div>
        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-3">
          <div
            className="bg-green-500 dark:bg-green-400 h-3 rounded-full transition-all"
            style={{ width: `${percent}%` }}
          />
        </div>
      </div>
    </div>
  )
}

// --- Summary Counters ---

function SummaryCounters({
  milestone,
  stats,
}: {
  milestone: Milestone
  stats: import('@/api/milestones').MilestoneStats | undefined
}) {
  const { t } = useTranslation()

  const counters = [
    { label: t('milestone.dashboard.openItems'), value: milestone.open_count, color: 'text-blue-600 dark:text-blue-400' },
    { label: t('milestone.dashboard.closedItems'), value: milestone.closed_count, color: 'text-green-600 dark:text-green-400' },
    { label: t('milestone.dashboard.totalItems'), value: milestone.total_count, color: 'text-gray-900 dark:text-gray-100' },
  ]

  // Add type counts
  if (stats) {
    for (const [type, counts] of Object.entries(stats.by_type)) {
      counters.push({
        label: t(`workitems.types.${type}`, { defaultValue: type }),
        value: counts.open + counts.closed,
        color: 'text-gray-700 dark:text-gray-300',
      })
    }
  }

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">{t('milestone.dashboard.summary')}</h3>
      <div className="grid grid-cols-3 sm:grid-cols-4 md:grid-cols-6 gap-3">
        {counters.map((c) => (
          <div key={c.label} className="rounded-lg border border-gray-200 dark:border-gray-700 p-3 text-center">
            <div className={`text-2xl font-bold ${c.color}`}>{c.value}</div>
            <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{c.label}</div>
          </div>
        ))}
      </div>
    </div>
  )
}

// --- Breakdown Chart ---

function BreakdownChart({
  title,
  data,
  colorMap,
  projectKey,
  milestoneId,
  filterKey,
}: {
  title: string
  data: Record<string, import('@/api/milestones').StatusCount>
  colorMap: Record<string, string>
  projectKey: string
  milestoneId: string
  filterKey: string
}) {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const entries = Object.entries(data).sort((a, b) => (b[1].open + b[1].closed) - (a[1].open + a[1].closed))
  const maxCount = Math.max(...entries.map(([, v]) => v.open + v.closed), 1)

  return (
    <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-4">
      <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3">{title}</h3>
      <div className="space-y-2">
        {entries.map(([key, counts]) => {
          const total = counts.open + counts.closed
          const pct = Math.round((total / maxCount) * 100)
          const i18nPrefix = filterKey === 'type' ? 'workitems.types' : 'workitems.priorities'
          const label = t(`${i18nPrefix}.${key}`, { defaultValue: key })
          const filterParam = filterKey === 'type' ? `type=${key}` : `priority=${key}`
          return (
            <Link
              key={key}
              to={p(`/projects/${projectKey}/items?milestones=${milestoneId}&${filterParam}`)}
              className="block group"
            >
              <div className="flex items-center justify-between text-xs mb-0.5">
                <span className="text-gray-700 dark:text-gray-300 group-hover:text-indigo-600 dark:group-hover:text-indigo-400">{label}</span>
                <span className="text-gray-500 dark:text-gray-400">{total}</span>
              </div>
              <div className="w-full bg-gray-100 dark:bg-gray-800 rounded-full h-2">
                <div
                  className={`h-2 rounded-full transition-all ${colorMap[key] ?? 'bg-gray-400'}`}
                  style={{ width: `${pct}%` }}
                />
              </div>
            </Link>
          )
        })}
      </div>
    </div>
  )
}

// --- Label Breakdown ---

function LabelBreakdown({ labels }: { labels: Record<string, number> }) {
  const { t } = useTranslation()
  const entries = Object.entries(labels).sort((a, b) => b[1] - a[1])

  return (
    <div>
      <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-3">{t('milestone.dashboard.byLabel')}</h3>
      <div className="flex flex-wrap gap-2">
        {entries.map(([label, count]) => (
          <span
            key={label}
            className="inline-flex items-center gap-1.5 rounded-full border border-gray-200 dark:border-gray-700 px-3 py-1 text-xs text-gray-700 dark:text-gray-300"
          >
            {label}
            <span className="font-medium text-gray-900 dark:text-gray-100">{count}</span>
          </span>
        ))}
      </div>
    </div>
  )
}

// --- Time Tracking Section ---

function TimeTrackingSection({ milestone }: { milestone: Milestone }) {
  const { t } = useTranslation()
  const estimated = milestone.total_estimated_seconds
  const spent = milestone.total_spent_seconds
  const remaining = estimated - spent
  const isOver = spent > estimated

  const timePercent = estimated > 0 ? Math.min(Math.round((spent / estimated) * 100), 100) : 0

  return (
    <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-4">
      <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100 mb-3 flex items-center gap-2">
        <Clock className="h-4 w-4" />
        {t('milestone.dashboard.timeTracking')}
      </h3>
      <div className="grid grid-cols-3 gap-4 text-center mb-3">
        {estimated > 0 && (
          <div>
            <div className="text-lg font-semibold text-gray-900 dark:text-gray-100">{formatDuration(estimated)}</div>
            <div className="text-xs text-gray-500 dark:text-gray-400">{t('milestone.dashboard.estimated')}</div>
          </div>
        )}
        {spent > 0 && (
          <div>
            <div className="text-lg font-semibold text-gray-900 dark:text-gray-100">{formatDuration(spent)}</div>
            <div className="text-xs text-gray-500 dark:text-gray-400">{t('milestone.dashboard.spent')}</div>
          </div>
        )}
        {estimated > 0 && (
          <div>
            <div className={`text-lg font-semibold ${isOver ? 'text-red-500' : 'text-gray-900 dark:text-gray-100'}`}>
              {isOver ? t('milestone.dashboard.overBy', { time: formatDuration(Math.abs(remaining)) }) : formatDuration(remaining)}
            </div>
            <div className="text-xs text-gray-500 dark:text-gray-400">{t('milestone.dashboard.remaining')}</div>
          </div>
        )}
      </div>
      {estimated > 0 && (
        <div className="w-full bg-gray-200 dark:bg-gray-700 rounded-full h-2">
          <div
            className={`h-2 rounded-full transition-all ${isOver ? 'bg-red-500' : 'bg-blue-500'}`}
            style={{ width: `${timePercent}%` }}
          />
        </div>
      )}
    </div>
  )
}

// --- Work Items Table ---

const ITEMS_PREVIEW_LIMIT = 15

function WorkItemsTable({
  items,
  memberMap,
  projectKey,
  milestoneId,
  isLoading,
  hasMore,
  statuses,
}: {
  items: WorkItem[]
  memberMap: Map<string, ProjectMember>
  projectKey: string
  milestoneId: string
  isLoading: boolean
  hasMore: boolean
  statuses?: import('@/api/workflows').WorkflowStatus[]
}) {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const [expanded, setExpanded] = useState(false)
  const [activeItemNumber, setActiveItemNumber] = useState(-1)
  const restoredRef = useRef(false)
  const activeRowStorageKey = `taskwondo_activeRow_milestone_${milestoneId}`

  const visibleItems = expanded ? items : items.slice(0, ITEMS_PREVIEW_LIMIT)
  const hasHiddenItems = items.length > ITEMS_PREVIEW_LIMIT

  // Restore active row from sessionStorage after navigating back
  useEffect(() => {
    if (restoredRef.current || items.length === 0) return
    const stored = sessionStorage.getItem(activeRowStorageKey)
    if (stored) {
      const itemNumber = parseInt(stored, 10)
      if (items.some((i) => i.item_number === itemNumber)) {
        setActiveItemNumber(itemNumber)
        // Auto-expand if the item is beyond the preview limit
        const idx = items.findIndex((i) => i.item_number === itemNumber)
        if (idx >= ITEMS_PREVIEW_LIMIT) setExpanded(true)
      }
      sessionStorage.removeItem(activeRowStorageKey)
    }
    restoredRef.current = true
  }, [items, activeRowStorageKey])

  const handleItemClick = useCallback((itemNumber: number) => {
    sessionStorage.setItem(activeRowStorageKey, String(itemNumber))
  }, [activeRowStorageKey])

  return (
    <div>
      <div className="flex items-center justify-between mb-3">
        <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('milestone.dashboard.workItems')}</h3>
        <Link
          to={p(`/projects/${projectKey}/items?milestone=${milestoneId}`)}
          className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline"
        >
          {t('milestone.dashboard.viewAll')}
        </Link>
      </div>

      {isLoading ? (
        <div className="flex justify-center py-8">
          <Spinner />
        </div>
      ) : items.length === 0 ? (
        <div className="border border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
          <p className="text-sm text-gray-500 dark:text-gray-400">{t('milestone.dashboard.noItems')}</p>
        </div>
      ) : (
        <div className="border border-gray-200 dark:border-gray-700 rounded-lg overflow-hidden divide-y divide-gray-200 dark:divide-gray-700 text-sm">
          {visibleItems.map((item) => (
            <WorkItemRow
              key={item.id}
              item={item}
              member={item.assignee_id ? memberMap.get(item.assignee_id) ?? null : null}
              projectKey={projectKey}
              milestoneId={milestoneId}
              isActive={item.item_number === activeItemNumber}
              onClick={handleItemClick}
              statuses={statuses}
            />
          ))}
          {(hasHiddenItems || hasMore) && (
            <div className="border-t border-gray-200 dark:border-gray-700 p-2 text-center">
              {hasHiddenItems && !expanded ? (
                <button
                  onClick={() => setExpanded(true)}
                  className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline"
                >
                  {t('milestone.dashboard.showAll', { count: items.length })}
                </button>
              ) : hasMore ? (
                <Link
                  to={p(`/projects/${projectKey}/items?milestone=${milestoneId}`)}
                  className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline"
                >
                  {t('milestone.dashboard.viewAll')}
                </Link>
              ) : null}
            </div>
          )}
        </div>
      )}
    </div>
  )
}

// --- Work Item Row (uses ScrollableRow for per-row horizontal scroll on mobile) ---

function WorkItemRow({
  item,
  member,
  projectKey,
  milestoneId,
  isActive,
  onClick,
  statuses,
}: {
  item: WorkItem
  member: ProjectMember | null
  projectKey: string
  milestoneId: string
  isActive?: boolean
  onClick?: (itemNumber: number) => void
  statuses?: import('@/api/workflows').WorkflowStatus[]
}) {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const linkState = { state: { from: 'milestone', backUrl: p(`/projects/${projectKey}/milestones/${milestoneId}`) } }
  const rowRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (isActive) rowRef.current?.scrollIntoView({ block: 'nearest' })
  }, [isActive])

  return (
    <ScrollableRow
      ref={rowRef}
      className={isActive ? 'bg-indigo-50 dark:bg-indigo-900/20' : 'hover:bg-gray-50 dark:hover:bg-gray-800/50'}
      gradientFrom="from-white dark:from-gray-900"
    >
      <div className="px-3 py-2 shrink-0">
        <Link
          to={p(`/projects/${projectKey}/items/${item.item_number}`)}
          {...linkState}
          onClick={() => onClick?.(item.item_number)}
          className="text-xs text-gray-500 dark:text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 font-mono"
        >
          {item.display_id}
        </Link>
      </div>
      <div className="px-3 py-2 shrink-0 sm:flex-1 sm:min-w-0">
        <Link
          to={p(`/projects/${projectKey}/items/${item.item_number}`)}
          {...linkState}
          onClick={() => onClick?.(item.item_number)}
          className="text-gray-900 dark:text-gray-100 hover:text-indigo-600 dark:hover:text-indigo-400 whitespace-nowrap sm:truncate sm:block"
        >
          {item.title}
        </Link>
      </div>
      <div className="px-2 py-2 shrink-0">
        <TypeBadge type={item.type} />
      </div>
      <div className="px-2 py-2 shrink-0">
        <PriorityBadge priority={item.priority} />
      </div>
      <div className="px-2 py-2 shrink-0">
        <StatusBadge status={item.status} statuses={statuses} />
      </div>
      <div className="px-3 py-2 shrink-0">
        {member ? (
          <Avatar name={member.display_name} avatarUrl={member.avatar_url} size="xs" />
        ) : (
          <span className="text-xs text-gray-400 whitespace-nowrap">{t('milestone.dashboard.unassigned')}</span>
        )}
      </div>
    </ScrollableRow>
  )
}

// --- Due Date Label (reused from MilestonesPage) ---

function DueDateLabel({ dueDate, isClosed }: { dueDate: string; isClosed: boolean }) {
  const { t } = useTranslation()
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  const due = new Date(dueDate + 'T00:00:00')
  const diffDays = Math.round((due.getTime() - today.getTime()) / (1000 * 60 * 60 * 24))

  let text: string
  let colorClass: string

  if (isClosed) {
    text = dueDate
    colorClass = 'text-gray-400 dark:text-gray-500'
  } else if (diffDays < 0) {
    text = t('milestones.overdue', { days: Math.abs(diffDays) })
    colorClass = 'text-red-600 dark:text-red-400'
  } else if (diffDays === 0) {
    text = t('milestones.dueToday')
    colorClass = 'text-yellow-600 dark:text-yellow-400'
  } else if (diffDays <= 7) {
    text = t('milestones.dueIn', { days: diffDays })
    colorClass = 'text-yellow-600 dark:text-yellow-400'
  } else {
    text = dueDate
    colorClass = 'text-gray-400 dark:text-gray-500'
  }

  return <span className={`text-sm ${colorClass}`}>{text}</span>
}

// --- Milestone Edit Form (simplified from MilestonesPage) ---

function MilestoneEditForm({
  milestone,
  onSubmit,
  onCancel,
  isPending,
  error,
}: {
  milestone: Milestone
  onSubmit: (input: UpdateMilestoneInput) => void
  onCancel: () => void
  isPending: boolean
  error: string
}) {
  const { t } = useTranslation()
  const [name, setName] = useState(milestone.name)
  const [description, setDescription] = useState(milestone.description ?? '')
  const [dueDate, setDueDate] = useState(milestone.due_date ?? '')
  const [status, setStatus] = useState(milestone.status)
  const [validationError, setValidationError] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setValidationError('')

    if (!name.trim()) {
      setValidationError(t('milestones.nameRequired'))
      return
    }

    const input: UpdateMilestoneInput = {}
    if (name.trim() !== milestone.name) input.name = name.trim()
    if (description !== (milestone.description ?? '')) input.description = description || null
    if (dueDate !== (milestone.due_date ?? '')) input.due_date = dueDate || null
    if (status !== milestone.status) input.status = status as 'open' | 'closed'
    onSubmit(input)
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {(validationError || error) && (
        <p className="text-sm text-red-600 dark:text-red-400">{validationError || error}</p>
      )}
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('milestones.name')}</label>
        <input
          type="text"
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          autoFocus
        />
      </div>
      <div>
        <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('common.description')}</label>
        <textarea
          rows={3}
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <div className="flex gap-4">
        <div className="w-48">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('milestones.dueDate')}</label>
          <input
            type="date"
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            value={dueDate}
            onChange={(e) => setDueDate(e.target.value)}
          />
        </div>
        <div className="w-48">
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.form.status')}</label>
          <select
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            value={status}
            onChange={(e) => setStatus(e.target.value as 'open' | 'closed')}
          >
            <option value="open">{t('milestones.statusOpen')}</option>
            <option value="closed">{t('milestones.statusClosed')}</option>
          </select>
        </div>
      </div>
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="secondary" onClick={onCancel}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={isPending}>
          {isPending ? t('common.saving') : t('common.save')}
        </Button>
      </div>
    </form>
  )
}
