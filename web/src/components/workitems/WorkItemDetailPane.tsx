import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { ExternalLink, X } from 'lucide-react'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { Button } from '@/components/ui/Button'
import type { WorkItem } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

export interface WorkItemDetailPaneProps {
  item: WorkItem
  statuses: WorkflowStatus[]
  fullPageHref: string
  onClose: () => void
  /** Optional richer body (TASK-79); defaults to list-item summary. */
  children?: ReactNode
}

/**
 * Right-side panel shell opened by tap-to-expand (TASK-77 Option C).
 * Receives the already-loaded list item — no refetch on open.
 * Full editable detail content lands in TASK-79.
 */
export function WorkItemDetailPane({
  item,
  statuses,
  fullPageHref,
  onClose,
  children,
}: WorkItemDetailPaneProps) {
  const { t } = useTranslation()

  return (
    <aside
      data-testid="work-item-detail-pane"
      className="flex h-full min-h-0 flex-col border-l border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900"
      aria-label={item.title}
    >
      <header className="flex items-start gap-2 border-b border-gray-200 dark:border-gray-700 px-4 py-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-1.5 mb-1">
            <span className="font-mono text-xs font-semibold text-gray-500 dark:text-gray-400">
              {item.display_id}
            </span>
            <TypeBadge type={item.type} />
            <StatusBadge status={item.status} statuses={statuses} />
            <PriorityBadge priority={item.priority} />
          </div>
          <h2 className="text-base font-semibold text-gray-900 dark:text-gray-100 leading-snug break-words">
            {item.title}
          </h2>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <Link
            to={fullPageHref}
            className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-indigo-600 hover:bg-indigo-50 dark:text-indigo-400 dark:hover:bg-indigo-950/40"
          >
            <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
            {t('workitems.splitPane.openFull')}
          </Link>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onClose}
            aria-label={t('common.close')}
            title={t('workitems.splitPane.closePanel')}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </header>
      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
        {children ?? (
          item.description ? (
            <pre className="whitespace-pre-wrap font-sans text-sm leading-relaxed">
              {item.description}
            </pre>
          ) : (
            <p className="text-gray-400 dark:text-gray-500 italic">
              {t('workitems.splitPane.emptyDescription')}
            </p>
          )
        )}
      </div>
    </aside>
  )
}
