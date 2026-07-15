import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { RelationList } from '@/components/workitems/RelationList'
import { ParentEpicBadge, shouldShowParentEpic } from '@/components/workitems/ParentEpicBadge'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import type { WorkItem } from '@/api/workitems'
import type { Milestone } from '@/api/milestones'

export interface DetailExtrasColumnProps {
  projectKey: string
  itemNumber: number
  item: WorkItem
  milestones?: Milestone[]
  readOnly?: boolean
}

function displayIdHref(displayId: string, p: (path: string) => string): string {
  const idx = displayId.lastIndexOf('-')
  if (idx < 0) return '#'
  return p(`/projects/${displayId.slice(0, idx)}/items/${displayId.slice(idx + 1)}`)
}

/**
 * Wide-viewport (≥1600px) third column for work-item detail surfaces.
 * Shows relations (required) plus parent epic / milestone links when present.
 * Attachments, time, and activity stay on the main tab strip to avoid duplication.
 */
export function DetailExtrasColumn({
  projectKey,
  itemNumber,
  item,
  milestones = [],
  readOnly = false,
}: DetailExtrasColumnProps) {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const milestone = item.milestone_id
    ? milestones.find((m) => m.id === item.milestone_id)
    : undefined
  const showEpic = shouldShowParentEpic(item)

  return (
    <aside
      data-testid="detail-extras-column"
      className="flex h-full min-h-0 w-full flex-col"
      aria-label={t('workitems.detail.extrasColumn')}
    >
      <h3 className="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400 mb-3">
        {t('workitems.detail.extrasColumn')}
      </h3>

      {(showEpic || milestone) && (
        <div className="mb-4 space-y-2 text-sm border-b border-gray-100 dark:border-gray-800 pb-4">
          {showEpic && (
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs text-gray-500 dark:text-gray-400">
                {t('workitems.parentEpic')}
              </span>
              <Link to={displayIdHref(item.parent_epic_display_id!, p)} className="inline-flex">
                <ParentEpicBadge
                  displayId={item.parent_epic_display_id!}
                  title={item.parent_epic_title}
                />
              </Link>
            </div>
          )}
          {milestone && (
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-xs text-gray-500 dark:text-gray-400">
                {t('workitems.form.milestone')}
              </span>
              <Link
                to={p(`/projects/${projectKey}/milestones/${milestone.id}`)}
                className="text-sm text-indigo-600 dark:text-indigo-400 hover:underline truncate"
              >
                {milestone.name}
              </Link>
            </div>
          )}
        </div>
      )}

      <RelationList projectKey={projectKey} itemNumber={itemNumber} readOnly={readOnly} />
    </aside>
  )
}
