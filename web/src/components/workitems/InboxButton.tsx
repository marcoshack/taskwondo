import { useState } from 'react'
import { Inbox, Check } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAddToInbox } from '@/hooks/useInbox'

interface InboxButtonProps {
  workItemId: string
  className?: string
}

export function InboxButton({ workItemId, className = '' }: InboxButtonProps) {
  const { t } = useTranslation()
  const addToInbox = useAddToInbox()
  const [saved, setSaved] = useState(false)

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation()
    e.preventDefault()
    addToInbox.mutate(workItemId, {
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
    <button
      onClick={handleClick}
      className={`text-gray-400 hover:text-indigo-600 dark:hover:text-indigo-400 transition-colors ${className}`}
      aria-label={t('inbox.sendToInbox')}
      title={t('inbox.sendToInbox')}
    >
      <Inbox className="h-4 w-4" />
    </button>
  )
}
