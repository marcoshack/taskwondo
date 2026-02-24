import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'

type DiffLine = { type: 'same' | 'add' | 'remove'; text: string }

export type DiffPreviewTarget =
  | { kind: 'field'; fieldLabel: string; oldValue?: string; newValue?: string }
  | { kind: 'comment'; lines: DiffLine[] }

interface DiffPreviewModalProps {
  target: DiffPreviewTarget | null
  onClose: () => void
}

export function DiffPreviewModal({ target, onClose }: DiffPreviewModalProps) {
  const { t } = useTranslation()

  if (!target) return null

  const title = target.kind === 'field'
    ? t('activity.diff.title', { field: target.fieldLabel })
    : t('activity.diff.commentTitle')

  return (
    <Modal open={!!target} onClose={onClose} size="full">
      <div className="flex flex-col h-full">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 shrink-0">
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{title}</span>
          <button
            className="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
            onClick={onClose}
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
              <path strokeLinecap="round" strokeLinejoin="round" d="M4 4l8 8M12 4l-8 8" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-4">
          {target.kind === 'field' && <FieldDiffContent oldValue={target.oldValue} newValue={target.newValue} />}
          {target.kind === 'comment' && <CommentDiffContent lines={target.lines} />}
        </div>
      </div>
    </Modal>
  )
}

function FieldDiffContent({ oldValue, newValue }: { oldValue?: string; newValue?: string }) {
  const { t } = useTranslation()

  return (
    <div className="max-w-4xl mx-auto rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono overflow-hidden">
      {oldValue && (
        <>
          <div className="px-4 py-1.5 bg-gray-100 dark:bg-gray-700 text-xs font-sans font-medium text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-600">
            {t('activity.diff.removed')}
          </div>
          <div className="px-4 py-3 bg-red-50 dark:bg-red-900/30 text-red-800 dark:text-red-300 whitespace-pre-wrap break-words border-b border-gray-200 dark:border-gray-700">
            {oldValue}
          </div>
        </>
      )}
      {newValue && (
        <>
          <div className="px-4 py-1.5 bg-gray-100 dark:bg-gray-700 text-xs font-sans font-medium text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-600">
            {t('activity.diff.added')}
          </div>
          <div className="px-4 py-3 bg-green-50 dark:bg-green-900/30 text-green-800 dark:text-green-300 whitespace-pre-wrap break-words">
            {newValue}
          </div>
        </>
      )}
    </div>
  )
}

function CommentDiffContent({ lines }: { lines: DiffLine[] }) {
  return (
    <div className="max-w-4xl mx-auto rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono overflow-hidden">
      {lines.map((line, idx) => (
        <div
          key={idx}
          className={
            line.type === 'remove'
              ? 'px-4 py-0.5 bg-red-50 dark:bg-red-900/30 text-red-800 dark:text-red-300'
              : line.type === 'add'
                ? 'px-4 py-0.5 bg-green-50 dark:bg-green-900/30 text-green-800 dark:text-green-300'
                : 'px-4 py-0.5 text-gray-600 dark:text-gray-400'
          }
        >
          <span className={`select-none mr-2 ${line.type === 'remove' ? 'text-red-400' : line.type === 'add' ? 'text-green-400' : 'text-gray-400'}`}>
            {line.type === 'remove' ? '\u2212' : line.type === 'add' ? '+' : ' '}
          </span>
          {line.text || '\u00A0'}
        </div>
      ))}
    </div>
  )
}
