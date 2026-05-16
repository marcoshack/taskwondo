import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/Button'

interface InlineCommentComposerProps {
  /** The exact text the user selected, shown as a quote above the input. */
  selectedText: string
  submitting: boolean
  onSubmit: (body: string) => void
  onCancel: () => void
}

/**
 * Composer card for a brand-new inline comment. It is positioned by its
 * parent (anchored to the text selection); this component only renders the
 * card body.
 */
export function InlineCommentComposer({
  selectedText,
  submitting,
  onSubmit,
  onCancel,
}: InlineCommentComposerProps) {
  const { t } = useTranslation()
  const [body, setBody] = useState('')
  const ref = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    ref.current?.focus()
  }, [])

  return (
    <div
      className="w-80 max-w-[90vw] rounded-md border border-indigo-300 dark:border-indigo-600 bg-white dark:bg-gray-800 shadow-lg p-3"
      data-testid="inline-comment-composer"
    >
      <div className="text-xs text-gray-500 dark:text-gray-400 mb-2">
        {t('inlineComments.composerHeader')}
      </div>
      <blockquote className="text-xs text-gray-600 dark:text-gray-300 border-l-2 border-indigo-300 dark:border-indigo-600 pl-2 mb-2 line-clamp-3 whitespace-pre-wrap break-words">
        {selectedText}
      </blockquote>
      <textarea
        ref={ref}
        className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 px-3 py-2 text-sm"
        rows={3}
        placeholder={t('inlineComments.placeholder')}
        value={body}
        onChange={(e) => setBody(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === 'Escape') {
            e.preventDefault()
            onCancel()
          }
          if (e.key === 'Enter' && (e.ctrlKey || e.metaKey) && body.trim()) {
            e.preventDefault()
            onSubmit(body)
          }
        }}
      />
      <div className="mt-2 flex items-center gap-2">
        <Button size="sm" onClick={() => onSubmit(body)} disabled={!body.trim() || submitting}>
          {submitting ? t('inlineComments.submitting') : t('inlineComments.submit')}
        </Button>
        <Button size="sm" variant="ghost" onClick={onCancel}>
          {t('common.cancel')}
        </Button>
      </div>
    </div>
  )
}
