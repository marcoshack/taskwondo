import { Badge } from '@/components/ui/Badge'

const priorityColors = {
  critical: 'red',
  high: 'yellow',
  medium: 'blue',
  low: 'gray',
} as const

export function PriorityBadge({ priority }: { priority: string }) {
  const color = priorityColors[priority as keyof typeof priorityColors] ?? 'gray'
  return <Badge color={color}>{priority}</Badge>
}
