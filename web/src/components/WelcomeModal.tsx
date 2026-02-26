import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FolderKanban, ClipboardList, SquareStack, Target, Route, Inbox, Rss, Bookmark, ChevronLeft, ChevronRight } from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'

interface WelcomeModalProps {
  open: boolean
  onClose: () => void
  onDismiss: () => void
  alreadyDismissed?: boolean
}

const SLIDE_ICONS = [FolderKanban, ClipboardList, SquareStack, Target, Route, Inbox, Rss, Bookmark]
const SLIDE_KEYS = ['projects', 'workItems', 'queues', 'milestones', 'workflows', 'inbox', 'feed', 'watchlist'] as const

export function WelcomeModal({ open, onClose, onDismiss, alreadyDismissed }: WelcomeModalProps) {
  const { t } = useTranslation()
  const [current, setCurrent] = useState(0)
  const [dontShow, setDontShow] = useState(false)
  const isLast = current === SLIDE_KEYS.length - 1

  const handleClose = () => {
    if (dontShow) onDismiss()
    setCurrent(0)
    onClose()
  }

  const handleGetStarted = () => {
    if (!alreadyDismissed) onDismiss()
    setCurrent(0)
    onClose()
  }

  const handleNext = () => {
    if (isLast) {
      handleGetStarted()
    } else {
      setCurrent(current + 1)
    }
  }

  const handlePrev = () => {
    if (current > 0) setCurrent(current - 1)
  }

  const Icon = SLIDE_ICONS[current]
  const slideKey = SLIDE_KEYS[current]

  return (
    <Modal open={open} onClose={handleClose} title={t('welcome.title')} className="!max-w-xl !overflow-hidden">
      <div className="flex flex-col items-center text-center px-2 h-[420px]">
        {/* Icon — fixed */}
        <div className="w-16 h-16 rounded-full bg-indigo-100 dark:bg-indigo-900/40 flex items-center justify-center mb-4 shrink-0">
          <Icon className="w-8 h-8 text-indigo-600 dark:text-indigo-400" />
        </div>

        {/* Title — fixed */}
        <h3 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-2 shrink-0">
          {t(`welcome.${slideKey}.title`)}
        </h3>

        {/* Description — scrollable */}
        <div className="flex-1 min-h-0 w-full mb-4 overflow-y-auto overscroll-contain">
          <p className="text-sm text-gray-600 dark:text-gray-400 leading-relaxed px-2">
            {t(`welcome.${slideKey}.description`)}
          </p>
        </div>

        {/* Dot indicators — fixed */}
        <div className="flex gap-2 mb-5 shrink-0">
          {SLIDE_KEYS.map((_, i) => (
            <button
              key={i}
              onClick={() => setCurrent(i)}
              className={`w-2 h-2 rounded-full transition-colors ${
                i === current
                  ? 'bg-indigo-600 dark:bg-indigo-400'
                  : 'bg-gray-300 dark:bg-gray-600 hover:bg-gray-400 dark:hover:bg-gray-500'
              }`}
              aria-label={`${i + 1} / ${SLIDE_KEYS.length}`}
            />
          ))}
        </div>

        {/* Navigation — fixed */}
        <div className="flex items-center justify-between w-full shrink-0">
          <Button
            variant="secondary"
            size="sm"
            onClick={handlePrev}
            disabled={current === 0}
            className={current === 0 ? 'invisible' : ''}
          >
            <ChevronLeft className="w-4 h-4 mr-1" />
            {t('welcome.previous')}
          </Button>
          <span className="text-xs text-gray-500 dark:text-gray-400">
            {current + 1} / {SLIDE_KEYS.length}
          </span>
          <Button
            variant="primary"
            size="sm"
            onClick={handleNext}
          >
            {isLast ? t('welcome.getStarted') : t('welcome.next')}
            {!isLast && <ChevronRight className="w-4 h-4 ml-1" />}
          </Button>
        </div>

        {/* Don't show again — hidden when already dismissed */}
        {!alreadyDismissed && (
          <label className="flex items-center gap-2 mt-4 text-xs text-gray-500 dark:text-gray-400 cursor-pointer select-none shrink-0">
            <input
              type="checkbox"
              checked={dontShow}
              onChange={(e) => setDontShow(e.target.checked)}
              className="rounded border-gray-300 dark:border-gray-600 text-indigo-600 focus:ring-indigo-500"
            />
            {t('welcome.dontShowAgain')}
          </label>
        )}
      </div>
    </Modal>
  )
}
