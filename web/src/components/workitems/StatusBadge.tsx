import { Badge } from '@/components/ui/Badge'
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
  const ws = statuses?.find((s) => s.name === status)
  const category = ws?.category ?? 'todo'
  const color = categoryColors[category as keyof typeof categoryColors] ?? 'gray'
  const label = ws?.display_name ?? status
  return <Badge color={color}>{label}</Badge>
}
