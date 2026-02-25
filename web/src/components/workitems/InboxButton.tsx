import { useState } from 'react'
import { Inbox, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAddToInbox, useRemoveFromInbox } from '@/hooks/useInbox'

interface InboxButtonProps {
  workItemId: string
  inboxItemId?: string
  className?: string
}

export function InboxButton({ workItemId, inboxItemId, className = '' }: InboxButtonProps) {
  const { t } = useTranslation()
  const addToInbox = useAddToInbox()
  const removeFromInbox = useRemoveFromInbox()
  const [saved, setSaved] = useState(false)

  const isInInbox = !!inboxItemId

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    e.preventDefault()
    if (isInInbox) {
      removeFromInbox.mutate(inboxItemId, {
        onSuccess: () => {
          setSaved(true)
          setTimeout(() => setSaved(false), 2000)
        },
      })
    } else {
      addToInbox.mutate(workItemId, {
        onSuccess: () => {
          setSaved(true)
          setTimeout(() => setSaved(false), 2000)
        },
      })
    }
  }

  if (saved) {
    return <Check className={`h-4 w-4 text-green-500 animate-[pulse_0.6s_ease-in-out_2] ${className}`} />
  }

  return (
    <button
      onClick={handleClick}
      className={`${isInInbox ? 'text-indigo-500 dark:text-indigo-400 hover:text-indigo-700 dark:hover:text-indigo-300' : 'text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400'} transition-colors ${className}`}
      aria-label={isInInbox ? t('inbox.removeFromInbox') : t('inbox.sendToInbox')}
      title={isInInbox ? t('inbox.removeFromInbox') : t('inbox.sendToInbox')}
    >
      <Inbox className="h-4 w-4" />
    </button>
  )
}
