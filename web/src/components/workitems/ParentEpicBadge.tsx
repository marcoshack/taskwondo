import { useTranslation } from 'react-i18next'

interface ParentEpicBadgeProps {
  displayId: string
  title?: string | null
  /** Denser styling for compact / mobile rows. */
  size?: 'sm' | 'xs'
  className?: string
}

/**
 * Shows the parent epic display id on non-epic summary rows.
 * Hidden when the item itself is an epic (caller must gate).
 */
export function ParentEpicBadge({
  displayId,
  title,
  size = 'sm',
  className = '',
}: ParentEpicBadgeProps) {
  const { t } = useTranslation()
  const label = title
    ? `${t('workitems.parentEpic')}: ${displayId} — ${title}`
    : `${t('workitems.parentEpic')}: ${displayId}`
  const sizeClass =
    size === 'xs'
      ? 'text-[10px] leading-4 px-1 py-px'
      : 'text-xs leading-4 px-1.5 py-0.5'

  return (
    <span
      data-testid="parent-epic-badge"
      data-epic-id={displayId}
      title={label}
      className={[
        'inline-flex shrink-0 max-w-[7.5rem] truncate rounded font-mono font-medium',
        'bg-green-50 text-green-800 dark:bg-green-900/40 dark:text-green-300',
        sizeClass,
        className,
      ].join(' ')}
    >
      {displayId}
    </span>
  )
}

/** True when a list/get item should show a parent epic badge. */
export function shouldShowParentEpic(item: {
  type: string
  parent_epic_display_id?: string | null
}): item is { type: string; parent_epic_display_id: string } {
  return item.type !== 'epic' && !!item.parent_epic_display_id
}
