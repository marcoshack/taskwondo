import { useRef, useState } from 'react'
import { useTranslation, Trans } from 'react-i18next'
import { ScrollableRow } from '@/components/ui/ScrollableRow'
import { formatFileSize } from '@/utils/fileSize'
import type { StagedAttachment } from '@/utils/stagedAttachments'
import type { StagedUploadProgress } from '@/hooks/useStagedAttachments'

interface StagedAttachmentsFieldProps {
  staged: StagedAttachment[]
  onAdd: (files: File[]) => void
  onRemove: (id: string) => void
  /** Attachment size cap in bytes, shown as a hint when known. */
  maxUploadSize?: number
  error?: string | null
  /** Set while the staged files upload; the zone becomes a progress bar. */
  progress?: StagedUploadProgress | null
  disabled?: boolean
}

/** Short uppercase type tag for a filename, e.g. "crash.log.txt" -> "TXT". */
function fileKind(filename: string): string {
  const ext = filename.includes('.') ? filename.split('.').pop() ?? '' : ''
  return (ext || 'file').slice(0, 4).toUpperCase()
}

/**
 * Attachment picker for the create modal, sized to sit in the modal footer next
 * to the action buttons. Files are only staged here — the modal uploads them
 * once the work item exists.
 */
export function StagedAttachmentsField({ staged, onAdd, onRemove, maxUploadSize, error, progress, disabled }: StagedAttachmentsFieldProps) {
  const { t } = useTranslation()
  const inputRef = useRef<HTMLInputElement>(null)
  const [draggingOver, setDraggingOver] = useState(false)

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    e.stopPropagation()
    setDraggingOver(false)
    if (disabled) return
    const files = Array.from(e.dataTransfer?.files ?? [])
    if (files.length > 0) onAdd(files)
  }

  function handleDragOver(e: React.DragEvent) {
    if (disabled || !e.dataTransfer.types.includes('Files')) return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'copy'
    setDraggingOver(true)
  }

  function openPicker() {
    if (disabled) return
    inputRef.current?.click()
  }

  return (
    <div className="min-w-0 flex-1">
      <div
        aria-label={t('workitems.form.attachments')}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={() => setDraggingOver(false)}
        className={`flex min-h-[2.75rem] items-center rounded-md border border-dashed px-2 py-1.5 transition-colors ${
          draggingOver
            ? 'border-indigo-400 bg-indigo-50 dark:border-indigo-500 dark:bg-indigo-900/20'
            : 'border-gray-300 dark:border-gray-600'
        } ${disabled && !progress ? 'opacity-50' : ''}`}
      >
        {progress ? (
          <div className="w-full">
            <div className="mb-1 flex items-baseline justify-between gap-2 text-xs text-gray-500 dark:text-gray-400">
              <span className="truncate">{t('workitems.form.attachmentsUploading', { filename: progress.filename })}</span>
              <span className="shrink-0 tabular-nums">{Math.round(progress.ratio * 100)}%</span>
            </div>
            <div
              role="progressbar"
              aria-valuemin={0}
              aria-valuemax={100}
              aria-valuenow={Math.round(progress.ratio * 100)}
              className="h-1.5 w-full overflow-hidden rounded-full bg-gray-200 dark:bg-gray-700"
            >
              <div
                className="h-full rounded-full bg-indigo-600 transition-[width] duration-150 ease-out dark:bg-indigo-500"
                style={{ width: `${Math.round(progress.ratio * 100)}%` }}
              />
            </div>
          </div>
        ) : staged.length === 0 ? (
          <p className="w-full text-center text-sm text-gray-400 dark:text-gray-500">
            <Trans
              i18nKey="workitems.form.attachmentsDropHint"
              components={{
                browse: (
                  <button
                    type="button"
                    onClick={openPicker}
                    disabled={disabled}
                    className="font-medium text-indigo-600 hover:underline disabled:no-underline dark:text-indigo-400"
                  />
                ),
              }}
            />
            {maxUploadSize ? (
              <span className="text-xs"> &middot; {t('workitems.form.attachmentsMaxSize', { size: formatFileSize(maxUploadSize) })}</span>
            ) : null}
          </p>
        ) : (
          <ScrollableRow className="min-w-0 flex-1">
            {staged.map((s) => (
              <span
                key={s.id}
                className="inline-flex shrink-0 items-center gap-2 rounded-md border border-gray-200 dark:border-gray-600 bg-gray-50 dark:bg-gray-800 px-2 py-1"
              >
                <span className="rounded bg-gray-200 dark:bg-gray-700 px-1 text-[0.625rem] font-semibold tracking-wide text-gray-600 dark:text-gray-300">
                  {fileKind(s.file.name)}
                </span>
                <span className="max-w-[14rem] truncate text-sm text-gray-900 dark:text-gray-100">{s.file.name}</span>
                <span className="text-xs text-gray-400 dark:text-gray-500">{formatFileSize(s.file.size)}</span>
                <button
                  type="button"
                  onClick={() => onRemove(s.id)}
                  disabled={disabled}
                  aria-label={t('workitems.form.attachmentsRemove', { filename: s.file.name })}
                  className="text-gray-400 hover:text-gray-600 disabled:opacity-50 dark:hover:text-gray-300"
                >
                  &times;
                </button>
              </span>
            ))}
          </ScrollableRow>
        )}
      </div>

      {error && <p className="mt-1 truncate text-xs text-red-600 dark:text-red-400">{error}</p>}

      <input
        ref={inputRef}
        type="file"
        multiple
        className="hidden"
        onChange={(e) => {
          const files = Array.from(e.target.files ?? [])
          if (files.length > 0) onAdd(files)
          e.target.value = ''
        }}
      />
    </div>
  )
}
