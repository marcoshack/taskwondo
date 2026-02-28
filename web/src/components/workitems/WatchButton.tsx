import { useState } from 'react'
import { Bell, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useToggleWatch } from '@/hooks/useWorkItems'
import { Tooltip } from '@/components/ui/Tooltip'

interface WatchButtonProps {
  projectKey: string
  itemNumber: number
  isWatching: boolean
  className?: string
}

export function WatchButton({ projectKey, itemNumber, isWatching, className = '' }: WatchButtonProps) {
  const { t } = useTranslation()
  const toggleMutation = useToggleWatch(projectKey, itemNumber)
  const [saved, setSaved] = useState(false)

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    e.preventDefault()
    toggleMutation.mutate(undefined, {
      onSuccess: () => {
        setSaved(true)
        setTimeout(() => setSaved(false), 2000)
      },
    })
  }

  if (saved) {
    return <Check className={`h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2] ${className}`} />
  }

  return (
    <Tooltip content={isWatching ? t('watchers.unwatch') : t('watchers.watch')}>
      <button
        onClick={handleClick}
        className={`${isWatching ? 'text-indigo-500 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300' : 'text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400'} transition-colors ${className}`}
        aria-label={isWatching ? t('watchers.unwatch') : t('watchers.watch')}
      >
        <Bell className="h-4 w-4" />
      </button>
    </Tooltip>
  )
}
