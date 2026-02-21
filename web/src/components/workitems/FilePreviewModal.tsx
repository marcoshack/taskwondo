import { useState, useEffect, useRef } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Tooltip } from '@/components/ui/Tooltip'
import { getToken } from '@/api/client'
import { getAttachmentDownloadURL } from '@/api/workitems'
import type { Attachment } from '@/api/workitems'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { markdownComponents } from '@/components/ui/markdownComponents'

export type PreviewTarget =
  | { kind: 'attachment'; attachment: Attachment; projectKey: string; itemNumber: number }
  | { kind: 'image'; src: string; label?: string; comment?: string }

interface FilePreviewModalProps {
  target: PreviewTarget | null
  onClose: () => void
}

type PreviewType = 'image' | 'pdf' | 'markdown' | 'text'

const MARKDOWN_EXTENSIONS = ['.md', '.markdown', '.mdx']
const TEXT_EXTENSIONS = ['.txt', '.log', '.csv', '.json', '.yaml', '.yml', '.xml', '.toml', '.ini', '.cfg', '.conf', '.sh', '.bash', '.zsh']

function getFileExtension(filename: string): string {
  const idx = filename.lastIndexOf('.')
  return idx >= 0 ? filename.slice(idx).toLowerCase() : ''
}

function getPreviewType(contentType: string, filename: string): PreviewType | null {
  const ext = getFileExtension(filename)
  if (contentType.startsWith('image/')) return 'image'
  if (contentType === 'application/pdf') return 'pdf'
  if (contentType.includes('markdown') || MARKDOWN_EXTENSIONS.includes(ext)) return 'markdown'
  if (contentType.startsWith('text/') || TEXT_EXTENSIONS.includes(ext)) return 'text'
  return null
}

export function isPreviewable(attachment: Attachment): boolean {
  return getPreviewType(attachment.content_type, attachment.filename) !== null
}

function getTargetInfo(target: PreviewTarget): { url: string; filename: string; comment?: string; previewType: PreviewType } {
  if (target.kind === 'image') {
    const filename = target.label ?? target.src.split('/').pop() ?? 'image'
    return { url: target.src, filename, comment: target.comment, previewType: 'image' }
  }
  const { attachment, projectKey, itemNumber } = target
  const url = getAttachmentDownloadURL(projectKey, itemNumber, attachment.id)
  const previewType = getPreviewType(attachment.content_type, attachment.filename) ?? 'image'
  return { url, filename: attachment.filename, comment: attachment.comment || undefined, previewType }
}

export function FilePreviewModal({ target, onClose }: FilePreviewModalProps) {
  const { t } = useTranslation()
  const containerRef = useRef<HTMLDivElement>(null)
  const [blobUrl, setBlobUrl] = useState<string | null>(null)
  const [textContent, setTextContent] = useState<string | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState(false)

  const info = target ? getTargetInfo(target) : null

  useEffect(() => {
    if (target) containerRef.current?.focus()
  }, [target])

  useEffect(() => {
    if (!info) return

    let revoked = false
    setLoading(true)
    setError(false)
    setBlobUrl(null)
    setTextContent(null)

    const token = getToken()

    fetch(info.url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        if (info.previewType === 'markdown' || info.previewType === 'text') {
          return res.text().then((text) => {
            if (!revoked) setTextContent(text)
          })
        }
        return res.blob().then((blob) => {
          if (!revoked) setBlobUrl(URL.createObjectURL(blob))
        })
      })
      .catch(() => {
        if (!revoked) setError(true)
      })
      .finally(() => {
        if (!revoked) setLoading(false)
      })

    return () => {
      revoked = true
      setBlobUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev)
        return null
      })
    }
  }, [target]) // eslint-disable-line react-hooks/exhaustive-deps

  function handleDownload() {
    if (!info) return
    const token = getToken()
    fetch(info.url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
      .then((res) => {
        if (!res.ok) return
        return res.blob()
      })
      .then((blob) => {
        if (!blob) return
        const url = URL.createObjectURL(blob)
        const a = document.createElement('a')
        a.href = url
        a.download = info.filename
        document.body.appendChild(a)
        a.click()
        a.remove()
        URL.revokeObjectURL(url)
      })
  }

  return (
    <Modal open={!!target} onClose={onClose} size="full">
     <div ref={containerRef} tabIndex={-1} className="flex flex-col h-full outline-none">
      {/* Header */}
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 shrink-0" onMouseEnter={() => containerRef.current?.focus()}>
        <div className="flex items-center gap-2 min-w-0 mr-4">
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100 shrink-0">
            {info?.filename}
          </span>
          {info?.comment && (
            <Tooltip content={info.comment}>
              <span className="text-sm text-gray-500 dark:text-gray-400 truncate">
                &mdash; {info.comment}
              </span>
            </Tooltip>
          )}
        </div>
        <div className="flex items-center gap-2">
          <Tooltip content={t('preview.download')}>
            <button
              className="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
              onClick={handleDownload}
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                <path strokeLinecap="round" strokeLinejoin="round" d="M8 2v8m0 0l-3-3m3 3l3-3M3 12h10" />
              </svg>
            </button>
          </Tooltip>
          <Tooltip content={t('preview.close')}>
            <button
              className="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
              onClick={onClose}
            >
              <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                <path strokeLinecap="round" strokeLinejoin="round" d="M4 4l8 8M12 4l-8 8" />
              </svg>
            </button>
          </Tooltip>
        </div>
      </div>

      {/* Content */}
      <div className="flex-1 overflow-auto flex items-center justify-center p-4">
        {loading && <Spinner size="lg" />}

        {error && (
          <p className="text-sm text-red-500">{t('preview.loadError')}</p>
        )}

        {!loading && !error && info?.previewType === 'image' && blobUrl && (
          <img
            src={blobUrl}
            alt={info.filename}
            className="max-w-full max-h-full object-contain"
          />
        )}

        {!loading && !error && info?.previewType === 'pdf' && blobUrl && (
          <iframe
            src={blobUrl}
            title={info.filename}
            className="w-full h-full border-0"
          />
        )}

        {!loading && !error && info?.previewType === 'markdown' && textContent !== null && (
          <div className="prose prose-sm dark:prose-invert max-w-3xl w-full self-start">
            <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>{textContent}</Markdown>
          </div>
        )}

        {!loading && !error && info?.previewType === 'text' && textContent !== null && (
          <pre className="text-sm text-gray-800 dark:text-gray-200 whitespace-pre-wrap font-mono bg-gray-50 dark:bg-gray-900 rounded p-4 max-w-4xl w-full self-start">
            {textContent}
          </pre>
        )}
      </div>
     </div>
    </Modal>
  )
}
