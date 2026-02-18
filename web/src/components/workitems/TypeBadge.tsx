import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/Badge'

const typeColors = {
  bug: 'red',
  task: 'blue',
  ticket: 'indigo',
  feedback: 'yellow',
  epic: 'green',
} as const

export function TypeBadge({ type }: { type: string }) {
  const { t } = useTranslation()
  const color = typeColors[type as keyof typeof typeColors] ?? 'gray'
  const key = `workitems.types.${type}`
  const translated = t(key)
  return <Badge color={color}>{translated === key ? type : translated}</Badge>
}
