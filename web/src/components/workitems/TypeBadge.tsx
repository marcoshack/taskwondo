import { useTranslation } from 'react-i18next'
import { Badge } from '@/components/ui/Badge'
import { Tooltip } from '@/components/ui/Tooltip'

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
  return <Tooltip content={t('workitems.form.type')}><Badge color={color}>{translated === key ? type : translated}</Badge></Tooltip>
}
