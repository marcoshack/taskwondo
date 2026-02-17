import { Badge } from '@/components/ui/Badge'

const typeColors = {
  bug: 'red',
  task: 'blue',
  ticket: 'indigo',
  feedback: 'yellow',
  epic: 'green',
} as const

export function TypeBadge({ type }: { type: string }) {
  const color = typeColors[type as keyof typeof typeColors] ?? 'gray'
  return <Badge color={color}>{type}</Badge>
}
