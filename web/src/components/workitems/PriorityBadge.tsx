import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/Badge'
import { Tooltip } from '@/components/ui/Tooltip'

const priorityColors = {
  critical: 'red',
  high: 'yellow',
  medium: 'blue',
  low: 'gray',
} as const

const PRIORITY_KEYS = Object.keys(priorityColors) as (keyof typeof priorityColors)[]

interface PriorityBadgeProps {
  priority: string
  /**
   * 'list' renders an equal-width badge (sized to the widest priority label) so
   * titles after it align across rows, and collapses to the label's first
   * letter on small screens.
   */
  variant?: 'default' | 'list'
}

export function PriorityBadge({ priority, variant = 'default' }: PriorityBadgeProps) {
  const { t } = useTranslation()
  const color = priorityColors[priority as keyof typeof priorityColors] ?? 'gray'
  const key = `workitems.priorities.${priority}`
  const translated = t(key)
  const label = translated === key ? priority : translated

  if (variant === 'list') {
    return (
      <Tooltip content={t('workitems.form.priority')}>
        <Badge color={color}>
          <span className="inline-grid justify-items-center" data-testid="priority-badge" data-priority={priority}>
            {/* Invisible labels size every badge to the widest priority */}
            {PRIORITY_KEYS.map((k) => (
              <span key={k} aria-hidden="true" className="invisible col-start-1 row-start-1">
                <span className="hidden sm:inline">{t(`workitems.priorities.${k}`)}</span>
                <span className="sm:hidden">{t(`workitems.priorities.${k}`).charAt(0).toUpperCase()}</span>
              </span>
            ))}
            <span className="col-start-1 row-start-1">
              <span className="hidden sm:inline">{label}</span>
              <span className="sm:hidden">{label.charAt(0).toUpperCase()}</span>
            </span>
          </span>
        </Badge>
      </Tooltip>
    )
  }

  return <Tooltip content={t('workitems.form.priority')}><Badge color={color}>{label}</Badge></Tooltip>
}
