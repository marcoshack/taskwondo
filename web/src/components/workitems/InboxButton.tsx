import { useState } from 'react'
import { Inbox, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAddToInbox, useRemoveFromInbox } from '@/hooks/useInbox'
import { Tooltip } from '@/components/ui/Tooltip'

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
    <Tooltip content={isInInbox ? t('inbox.removeFromInbox') : t('inbox.sendToInbox')}>
      <button
        onClick={handleClick}
        className={`${isInInbox ? 'text-[var(--primary)] dark:text-[var(--primary)] hover:text-[var(--primary)] dark:hover:text-[var(--primary)]' : 'text-[var(--foreground-muted)] hover:text-[var(--primary)] dark:hover:text-[var(--primary)]'} transition-colors ${className}`}
        aria-label={isInInbox ? t('inbox.removeFromInbox') : t('inbox.sendToInbox')}
      >
        <Inbox className="h-4 w-4" />
      </button>
    </Tooltip>
  )
}
