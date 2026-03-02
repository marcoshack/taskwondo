import { useState, useRef, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useAttachments, useUploadAttachment, useUpdateAttachmentComment, useDeleteAttachment } from '@/hooks/useWorkItems'
import { useAuth } from '@/contexts/AuthContext'
import { useMembers } from '@/hooks/useProjects'
import { getAttachmentDownloadURL } from '@/api/workitems'
import type { Attachment } from '@/api/workitems'
import { getToken } from '@/api/client'
import { Spinner } from '@/components/ui/Spinner'
import { Tooltip } from '@/components/ui/Tooltip'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { ScrollableRow } from '@/components/ui/ScrollableRow'
import { isPreviewable } from './FilePreviewModal'

interface AttachmentListProps {
  projectKey: string
  itemNumber: number
  sortOrder?: 'asc' | 'desc'
  highlightedAttachmentId?: string | null
  onHighlightClear?: () => void
  onPreview?: (attachment: Attachment) => void
  readOnly?: boolean
}

function formatFileSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

export function AttachmentList({ projectKey, itemNumber, sortOrder = 'desc', highlightedAttachmentId, onHighlightClear, onPreview, readOnly = false }: AttachmentListProps) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const { data: attachments, isLoading } = useAttachments(projectKey, itemNumber)
  const { data: members } = useMembers(projectKey)
  const uploadMutation = useUploadAttachment(projectKey, itemNumber)
  const updateCommentMutation = useUpdateAttachmentComment(projectKey, itemNumber)
  const deleteMutation = useDeleteAttachment(projectKey, itemNumber)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [comment, setComment] = useState('')
  const [editingCommentId, setEditingCommentId] = useState<string | null>(null)
  const [editCommentDraft, setEditCommentDraft] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<Attachment | null>(null)
  const highlightRef = useCallback((node: HTMLDivElement | null) => {
    if (node) {
      node.scrollIntoView({ behavior: 'smooth', block: 'center' })
      if (onHighlightClear) {
        const timer = setTimeout(onHighlightClear, 2000)
        return () => clearTimeout(timer)
      }
    }
  }, [highlightedAttachmentId, onHighlightClear]) // eslint-disable-line react-hooks/exhaustive-deps

  function uploaderName(uploaderId: string): string {
    const member = members?.find((m) => m.user_id === uploaderId)
    return member?.display_name ?? t('common.unknown')
  }

  async function handleDownload(attachmentId: string, filename: string) {
    const url = getAttachmentDownloadURL(projectKey, itemNumber, attachmentId)
    const token = getToken()
    const res = await fetch(url, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
    if (!res.ok) return
    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = blobUrl
    a.download = filename
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(blobUrl)
  }

  function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    uploadMutation.mutate(
      { file, comment: comment || undefined },
      {
        onSuccess: () => {
          setComment('')
          if (fileInputRef.current) fileInputRef.current.value = ''
        },
      }
    )
  }

  if (isLoading) return <Spinner size="sm" />

  return (
    <div className="space-y-4">
      {/* Upload form */}
      {!readOnly && (
        <div className="space-y-2 pb-3 border-b border-gray-100 dark:border-gray-700">
          <input
            type="text"
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-1.5 text-sm"
            placeholder={t('attachments.commentPlaceholder')}
            value={comment}
            onChange={(e) => setComment(e.target.value)}
          />
          <div className="flex items-center gap-2">
            <input
              ref={fileInputRef}
              type="file"
              className="text-sm text-gray-500 file:mr-2 file:py-1 file:px-3 file:rounded-md file:border-0 file:text-sm file:bg-indigo-50 file:text-indigo-600 dark:file:bg-indigo-900/30 dark:file:text-indigo-400 hover:file:bg-indigo-100"
              onChange={handleFileSelect}
              disabled={uploadMutation.isPending}
            />
            {uploadMutation.isPending && <Spinner size="sm" />}
          </div>
          {uploadMutation.isError && (
            <p className="text-xs text-red-500">{t('attachments.uploadFailed')}</p>
          )}
        </div>
      )}

      {/* Attachment list */}
      {(sortOrder === 'desc' ? [...(attachments ?? [])].reverse() : (attachments ?? [])).map((a) => (
        <div
          key={a.id}
          ref={a.id === highlightedAttachmentId ? highlightRef : undefined}
          className={`flex items-start gap-3 border-b border-gray-100 dark:border-gray-700 pb-3 rounded-md transition-colors duration-700 ${
            a.id === highlightedAttachmentId ? 'bg-indigo-50 dark:bg-indigo-900/30 ring-1 ring-indigo-300 dark:ring-indigo-600 px-2 py-2 -mx-2' : ''
          }`}
        >
          <div className="flex-1 min-w-0">
            <ScrollableRow gradientFrom="from-white dark:from-gray-900">
              <button
                onClick={() => {
                  if (onPreview && isPreviewable(a)) {
                    onPreview(a)
                  } else {
                    handleDownload(a.id, a.filename)
                  }
                }}
                className="text-sm font-medium text-indigo-600 dark:text-indigo-400 hover:underline whitespace-nowrap text-left cursor-pointer"
              >
                {a.filename}
              </button>
            </ScrollableRow>
            {editingCommentId === a.id ? (
              <div className="flex items-center gap-1 mt-0.5">
                <input
                  type="text"
                  className="text-xs border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 rounded px-1.5 py-0.5 flex-1"
                  value={editCommentDraft}
                  onChange={(e) => setEditCommentDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      updateCommentMutation.mutate({ attachmentId: a.id, comment: editCommentDraft })
                      setEditingCommentId(null)
                    }
                    if (e.key === 'Escape') setEditingCommentId(null)
                  }}
                  autoFocus
                />
                <button
                  className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline"
                  onClick={() => {
                    updateCommentMutation.mutate({ attachmentId: a.id, comment: editCommentDraft })
                    setEditingCommentId(null)
                  }}
                >{t('common.save')}</button>
                <button
                  className="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                  onClick={() => setEditingCommentId(null)}
                >{t('common.cancel')}</button>
              </div>
            ) : a.comment ? (
              <p
                className={`text-xs text-gray-500 dark:text-gray-400 mt-0.5 rounded ${user && a.uploader_id === user.id ? 'hover:bg-gray-100 dark:hover:bg-gray-700 cursor-default' : ''}`}
                onDoubleClick={() => {
                  if (user && a.uploader_id === user.id) {
                    setEditingCommentId(a.id)
                    setEditCommentDraft(a.comment)
                  }
                }}
              >{a.comment}</p>
            ) : null}
            <p className="text-xs text-gray-400 dark:text-gray-500 mt-0.5">
              {formatFileSize(a.size_bytes)} &middot; {uploaderName(a.uploader_id)} &middot; {new Date(a.created_at).toLocaleString()}
            </p>
          </div>
          <div className="flex items-center gap-1 shrink-0">
            {user && a.uploader_id === user.id && (
              <Tooltip content={t('attachments.editDescription')}>
                <button
                  className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
                  onClick={() => {
                    setEditingCommentId(a.id)
                    setEditCommentDraft(a.comment ?? '')
                  }}
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M11.5 2.5a1.5 1.5 0 012.121 2.121L6.5 11.743l-2.5.757.757-2.5L11.5 2.5z" />
                  </svg>
                </button>
              </Tooltip>
            )}
            <Tooltip content={t('preview.download')}>
              <button
                className="p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
                onClick={() => handleDownload(a.id, a.filename)}
              >
                <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                  <path strokeLinecap="round" strokeLinejoin="round" d="M8 2v8m0 0l-3-3m3 3l3-3M3 12h10" />
                </svg>
              </button>
            </Tooltip>
            {user && a.uploader_id === user.id && (
              <Tooltip content={t('common.delete')}>
                <button
                  className="p-1 text-red-400 hover:text-red-600 dark:hover:text-red-300 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
                  onClick={() => setDeleteTarget(a)}
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M3 4h10M6 4V3a1 1 0 011-1h2a1 1 0 011 1v1m2 0v9a1 1 0 01-1 1H5a1 1 0 01-1-1V4h8zM7 7v4M9 7v4" />
                  </svg>
                </button>
              </Tooltip>
            )}
          </div>
        </div>
      ))}

      {(attachments ?? []).length === 0 && (
        <p className="text-sm text-gray-400 italic">{t('attachments.noAttachments')}</p>
      )}

      {/* Delete confirmation modal */}
      <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('attachments.deleteTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          {t('attachments.deleteConfirm')} <strong>{deleteTarget?.filename}</strong>
        </p>
        <div className="flex justify-end gap-3">
          <Button type="button" variant="secondary" onClick={() => setDeleteTarget(null)}>{t('common.cancel')}</Button>
          <Button
            type="button"
            variant="danger"
            autoFocus
            disabled={deleteMutation.isPending}
            onClick={() => {
              if (deleteTarget) {
                deleteMutation.mutate(deleteTarget.id, { onSuccess: () => setDeleteTarget(null) })
              }
            }}
          >
            {deleteMutation.isPending ? t('common.deleting') : t('common.delete')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}
