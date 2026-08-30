import { User, History } from 'lucide-react'
import { ScrollableRow } from '@/components/ui/ScrollableRow'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { InboxButton } from '@/components/workitems/InboxButton'
import { WatchButton } from '@/components/workitems/WatchButton'
import { SLAIndicator } from '@/components/SLAIndicator'
import { formatRelativeTime } from '@/utils/duration'
import type { WorkItem } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

function getDescriptionPreview(description: string): string {
  const line = description.split('\n').find(l => l.trim() !== '')
  if (!line) return ''
  return line.trim().replace(/^#+\s+/, '').replace(/[*_~`[\]]/g, '')
}

interface WorkItemMobileCardProps {
  item: WorkItem
  statuses: WorkflowStatus[]
  showDates: boolean
  assigneeName: string
  inboxItemId?: string
  isWatching: boolean
  isCompleted?: boolean
  onClick: () => void
}

export function WorkItemMobileCard({ item, statuses, showDates, assigneeName, inboxItemId, isWatching, isCompleted = false, onClick }: WorkItemMobileCardProps) {
  return (
    <div
      className="relative w-full text-left rounded-lg border border-[var(--border)] bg-[var(--surface)] p-3 hover:border-[var(--primary-border)] dark:hover:border-[var(--primary-border)] transition-colors cursor-pointer"
      onClick={onClick}
      role="button"
      tabIndex={0}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault()
          onClick()
        }
      }}
    >
      <span className="absolute top-2 right-2 flex flex-col items-center gap-2" onClick={(e) => e.stopPropagation()}>
        <InboxButton workItemId={item.id} inboxItemId={inboxItemId} />
        <WatchButton projectKey={item.project_key} itemNumber={item.item_number} isWatching={isWatching} />
      </span>
      <ScrollableRow className="mr-5">
        <span className={`shrink-0 font-mono text-sm font-semibold ${isCompleted ? 'text-[var(--foreground-muted)]' : 'text-[var(--foreground)]'}`}>{item.display_id}</span>
        <span className={`shrink-0 inline-flex ${isCompleted ? 'opacity-40' : ''}`}><TypeBadge type={item.type} /></span>
        <span className="shrink-0 inline-flex"><StatusBadge status={item.status} statuses={statuses} /></span>
        <span className={`shrink-0 inline-flex ${isCompleted ? 'opacity-40' : ''}`}><PriorityBadge priority={item.priority} /></span>
        {!isCompleted && !showDates && item.sla && (
          <span className="shrink-0 inline-flex ml-auto"><SLAIndicator sla={item.sla} compact /></span>
        )}
      </ScrollableRow>
      {showDates && (
        <ScrollableRow className={`mr-5 mt-1.5 text-xs ${isCompleted ? 'text-[var(--foreground-secondary)]' : 'text-[var(--foreground-muted)]'}`} contentClassName="gap-3">
          <span className="shrink-0 inline-flex items-center gap-1">
            <User className="h-3 w-3" />
            <span className="truncate max-w-[8rem]">{assigneeName}</span>
          </span>
          <span className="shrink-0 inline-flex items-center gap-1">
            <History className="h-3 w-3" />
            {formatRelativeTime(item.updated_at)}
          </span>
          {!isCompleted && item.sla && <span className="shrink-0 inline-flex"><SLAIndicator sla={item.sla} /></span>}
        </ScrollableRow>
      )}
      <p className={`mt-1.5 text-sm font-medium truncate ${isCompleted ? 'line-through text-[var(--foreground-muted)]' : 'text-[var(--foreground)]'}`}>{item.title}</p>
      {item.description && (
        <p className={`mt-0.5 text-xs truncate ${isCompleted ? 'text-[var(--foreground-secondary)]' : 'text-[var(--foreground-secondary)]'}`}>{getDescriptionPreview(item.description)}</p>
      )}
    </div>
  )
}
