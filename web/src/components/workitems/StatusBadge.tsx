import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/Badge'
import { Tooltip } from '@/components/ui/Tooltip'
import type { WorkflowStatus } from '@/api/workflows'

const categoryColors = {
  todo: 'gray',
  in_progress: 'blue',
  done: 'green',
  cancelled: 'red',
} as const

interface StatusBadgeProps {
  status: string
  statuses?: WorkflowStatus[]
}

export function StatusBadge({ status, statuses }: StatusBadgeProps) {
  const { t } = useTranslation()
  const ws = statuses?.find((s) => s.name === status)
  const category = ws?.category ?? 'todo'
  const color = categoryColors[category as keyof typeof categoryColors] ?? 'gray'
  const label = t(`workitems.statuses.${status}`, { defaultValue: ws?.display_name ?? status })
  return <Tooltip content={t('workitems.form.status')}><Badge color={color}>{label}</Badge></Tooltip>
}
