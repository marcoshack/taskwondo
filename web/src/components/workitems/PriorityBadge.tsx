import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/Badge'
import { Tooltip } from '@/components/ui/Tooltip'

const priorityColors = {
  critical: 'red',
  high: 'yellow',
  medium: 'blue',
  low: 'gray',
} as const

export function PriorityBadge({ priority }: { priority: string }) {
  const { t } = useTranslation()
  const color = priorityColors[priority as keyof typeof priorityColors] ?? 'gray'
  const key = `workitems.priorities.${priority}`
  const translated = t(key)
  return <Tooltip content={t('workitems.form.priority')}><Badge color={color}>{translated === key ? priority : translated}</Badge></Tooltip>
}
