import { useState, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePortalTicket, usePortalComments, useAddPortalComment, useUpdatePortalTicket, usePortalAttachments, useUploadPortalAttachment, useUpdatePortalAttachmentComment, useDeletePortalAttachment } from '@/hooks/usePortal'
import { Button } from '@/components/ui/Button'
import { Spinner } from '@/components/ui/Spinner'
import { Tooltip } from '@/components/ui/Tooltip'
import { Modal } from '@/components/ui/Modal'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { FilePreviewModal, isPreviewable } from '@/components/workitems/FilePreviewModal'
import type { PreviewTarget } from '@/components/workitems/FilePreviewModal'
import { ScrollableRow } from '@/components/ui/ScrollableRow'
import { ArrowLeft, CalendarPlus, Check, History, Pencil } from 'lucide-react'
import { getToken } from '@/api/client'
import type { PortalComment, PortalAttachment } from '@/api/portal'
import type { Attachment } from '@/api/workitems'
import { formatFileSize } from '@/utils/fileSize'

type Tab = 'comments' | 'attachments'

export function PortalTicketDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { namespace = 'default', projectKey = '', itemNumber } = useParams<{
    namespace: string
    projectKey: string
    itemNumber: string
  }>()
  const num = parseInt(itemNumber ?? '0', 10)

  const { data: ticket, isLoading } = usePortalTicket(namespace, projectKey, num)
  const { data: comments } = usePortalComments(namespace, projectKey, num)
  const addComment = useAddPortalComment(namespace, projectKey, num)
  const updateTicket = useUpdatePortalTicket(namespace, projectKey, num)
  const { data: attachments } = usePortalAttachments(namespace, projectKey, num)
  const uploadMutation = useUploadPortalAttachment(namespace, projectKey, num)
  const updateCommentMutation = useUpdatePortalAttachmentComment(namespace, projectKey, num)
  const deleteMutation = useDeletePortalAttachment(namespace, projectKey, num)

  const [activeTab, setActiveTab] = useState<Tab>('comments')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const [commentBody, setCommentBody] = useState('')

  // Title editing
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const [titleSaved, setTitleSaved] = useState(false)

  // Description editing
  const [editingDesc, setEditingDesc] = useState(false)
  const [descDraft, setDescDraft] = useState('')
  const [descSaved, setDescSaved] = useState(false)

  // Attachments & preview
  const [previewTarget, setPreviewTarget] = useState<PreviewTarget | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [uploadComment, setUploadComment] = useState('')
  const [editingCommentId, setEditingCommentId] = useState<string | null>(null)
  const [editCommentDraft, setEditCommentDraft] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<PortalAttachment | null>(null)

  function startEditTitle() {
    setTitleDraft(ticket?.title ?? '')
    setEditingTitle(true)
  }

  async function saveTitle() {
    const value = titleDraft.trim()
    if (!value) return
    try {
      await updateTicket.mutateAsync({ title: value })
      setEditingTitle(false)
      setTitleSaved(true)
      setTimeout(() => setTitleSaved(false), 2000)
    } catch { /* mutation state shows failure */ }
  }

  function startEditDesc() {
    setDescDraft(ticket?.description ?? '')
    setEditingDesc(true)
  }

  async function saveDesc() {
    const value = descDraft.trim() || null
    try {
      await updateTicket.mutateAsync({ description: value })
      setEditingDesc(false)
      setDescSaved(true)
      setTimeout(() => setDescSaved(false), 2000)
    } catch { /* mutation state shows failure */ }
  }

  const handleAddComment = async () => {
    if (!commentBody.trim()) return
    try {
      await addComment.mutateAsync(commentBody.trim())
      setCommentBody('')
    } catch { /* mutation state shows failure */ }
  }

  function handleFileSelect(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    if (!file) return
    uploadMutation.mutate(
      { file, comment: uploadComment || undefined },
      {
        onSuccess: () => {
          setUploadComment('')
          if (fileInputRef.current) fileInputRef.current.value = ''
        },
      }
    )
  }

  const [dragging, setDragging] = useState(false)

  function handleDrop(e: React.DragEvent) {
    e.preventDefault()
    setDragging(false)
    const files = e.dataTransfer?.files
    if (!files?.length) return
    for (const file of files) {
      uploadMutation.mutate({ file })
    }
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault()
    e.dataTransfer.dropEffect = 'copy'
    setDragging(true)
  }

  function handleDragLeave(e: React.DragEvent) {
    e.preventDefault()
    setDragging(false)
  }

  function toAttachment(a: PortalAttachment): Attachment {
    return { id: a.id, uploader_id: a.uploader_id, filename: a.filename, content_type: a.content_type, size_bytes: a.size_bytes, comment: a.comment, download_url: a.download_url, created_at: a.created_at }
  }

  async function handleDownload(url: string, filename: string) {
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

  function handleAttachmentClick(a: PortalAttachment) {
    const att = toAttachment(a)
    if (isPreviewable(att)) {
      setPreviewTarget({ kind: 'attachment', attachment: att, projectKey, itemNumber: num, downloadUrl: a.download_url })
    } else {
      handleDownload(a.download_url, a.filename)
    }
  }

  function handleDeleteAttachment() {
    if (!deleteTarget) return
    deleteMutation.mutate(deleteTarget.id, {
      onSuccess: () => setDeleteTarget(null),
    })
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!ticket) {
    return <p className="text-[var(--danger)] py-8 text-center">{t('workitems.notFound')}</p>
  }

  const tabs: { key: Tab; label: string }[] = [
    { key: 'comments', label: `${t('portal.comments')}${comments ? ` (${comments.length})` : ''}` },
    { key: 'attachments', label: `${t('tabs.attachments')}${attachments ? ` (${attachments.length})` : ''}` },
  ]

  return (
    <div>
      <button
        onClick={() => navigate(-1)}
        className="flex items-center gap-1.5 text-sm text-[var(--foreground-secondary)] hover:text-[var(--foreground)] mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        {t('portal.backToTickets')}
      </button>

      {/* Header row */}
      <div className="mb-6">
        <div className="flex items-center gap-3 mb-2">
          <span className="text-base font-bold font-mono text-[var(--foreground-secondary)]">{ticket.display_id}</span>
          <StatusBadge status={ticket.status} />
          <PriorityBadge priority={ticket.priority} />
          <Tooltip content={t('portal.created')}>
            <span className="hidden sm:inline-flex items-center gap-1 text-xs text-[var(--foreground-muted)] ml-auto shrink-0">
              <CalendarPlus className="h-3.5 w-3.5" />
              {new Date(ticket.created_at).toLocaleString()}
            </span>
          </Tooltip>
          <Tooltip content={t('portal.updated')}>
            <span className="hidden sm:inline-flex items-center gap-1 text-xs text-[var(--foreground-muted)] shrink-0">
              <History className="h-3.5 w-3.5" />
              {new Date(ticket.updated_at).toLocaleString()}
            </span>
          </Tooltip>
        </div>
        {/* Mobile metadata line */}
        <ScrollableRow className="sm:hidden mb-2" contentClassName="gap-3 text-xs text-[var(--foreground-muted)]" gradientFrom="from-gray-50 dark:from-gray-900">
          <span className="inline-flex items-center gap-1 shrink-0">
            <CalendarPlus className="h-3.5 w-3.5" />
            {new Date(ticket.created_at).toLocaleString()}
          </span>
          <span className="inline-flex items-center gap-1 shrink-0">
            <History className="h-3.5 w-3.5" />
            {new Date(ticket.updated_at).toLocaleString()}
          </span>
        </ScrollableRow>

        {/* Editable title */}
        {editingTitle ? (
          <div className="space-y-2">
            <input
              type="text"
              value={titleDraft}
              onChange={(e) => setTitleDraft(e.target.value)}
              className="w-full text-xl font-semibold rounded-md border border-[var(--border)] px-2 py-1 focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] bg-[var(--surface)] text-[var(--foreground)]"
              autoFocus
              onKeyDown={(e) => {
                if (e.key === 'Enter') saveTitle()
                if (e.key === 'Escape') setEditingTitle(false)
              }}
            />
            <div className="flex gap-2">
              <Button size="sm" onClick={saveTitle} disabled={updateTicket.isPending}>
                {updateTicket.isPending ? t('common.saving') : t('common.save')}
              </Button>
              <Button size="sm" variant="secondary" onClick={() => setEditingTitle(false)}>
                {t('common.cancel')}
              </Button>
            </div>
          </div>
        ) : (
          <h1
            className="text-xl font-semibold text-[var(--foreground)] rounded px-1 -mx-1 cursor-pointer border border-transparent hover:border-[var(--border)] dark:hover:border-[var(--border)]"
            onClick={startEditTitle}
            onDoubleClick={startEditTitle}
          >
            {ticket.title}
            {titleSaved && <Check className="inline h-4 w-4 text-green-500 ml-2" />}
          </h1>
        )}

        {/* Editable description */}
        <div className="mt-3 group/desc">
          <div className="flex items-center gap-1 mb-1">
            <h3 className="text-sm font-medium text-[var(--foreground-secondary)] cursor-pointer" onClick={startEditDesc}>{t('workitems.detail.description')}</h3>
            {descSaved && <Check className="h-4 w-4 text-green-500" />}
            {!editingDesc && (
              <button
                className="inline-flex items-center justify-center w-7 h-7 rounded-md text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] hover:bg-[var(--surface-tertiary)] hover:bg-[var(--surface-hover)] transition-colors opacity-0 group-hover/desc:opacity-100"
                onClick={startEditDesc}
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
            )}
          </div>
          {editingDesc ? (
            <div className="space-y-2">
              <textarea
                value={descDraft}
                onChange={(e) => setDescDraft(e.target.value)}
                rows={4}
                className="block w-full rounded-md border border-[var(--border)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)] bg-[var(--surface)] text-[var(--foreground)]"
                autoFocus
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) saveDesc()
                  if (e.key === 'Escape') setEditingDesc(false)
                }}
              />
              <div className="flex gap-2">
                <Button size="sm" onClick={saveDesc} disabled={updateTicket.isPending}>
                  {updateTicket.isPending ? t('common.saving') : t('common.save')}
                </Button>
                <Button size="sm" variant="secondary" onClick={() => setEditingDesc(false)}>
                  {t('common.cancel')}
                </Button>
              </div>
            </div>
          ) : (
            <div
              className="border border-transparent hover:border-[var(--border)] dark:hover:border-[var(--border)] rounded p-2 min-h-[2rem] cursor-pointer"
              onClick={startEditDesc}
            >
              {ticket.description ? (
                <p className="text-sm text-[var(--foreground)] whitespace-pre-wrap">
                  {ticket.description}
                </p>
              ) : (
                <p className="text-sm text-[var(--foreground-muted)] italic">
                  {t('portal.noDescription')}
                </p>
              )}
            </div>
          )}
        </div>
      </div>

      {/* Tabs */}
      <div className="border-b border-[var(--border)] mb-4 flex items-center justify-between">
        <nav className="flex gap-6 pr-8 overflow-x-auto scrollbar-none">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              className={`pb-2 text-sm font-medium border-b-2 whitespace-nowrap ${
                activeTab === tab.key
                  ? 'border-[var(--primary)] text-[var(--primary)]'
                  : 'border-transparent text-[var(--foreground-secondary)] hover:text-[var(--foreground)] text-[var(--foreground-muted)] dark:hover:text-[var(--foreground-muted)]'
              }`}
              onClick={() => setActiveTab(tab.key)}
            >
              {tab.label}
            </button>
          ))}
        </nav>
        <Tooltip content={sortOrder === 'desc' ? t('common.showingNewestFirst') : t('common.showingOldestFirst')}>
          <button
            className="text-xs text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)] pb-2 flex items-center gap-1"
            onClick={() => setSortOrder((s) => (s === 'desc' ? 'asc' : 'desc'))}
          >
            <span className="text-base lg:text-xs">{sortOrder === 'desc' ? '\u2193' : '\u2191'}</span>
            <span className="hidden lg:inline">{sortOrder === 'desc' ? t('common.newestFirst') : t('common.oldestFirst')}</span>
          </button>
        </Tooltip>
      </div>

      {/* Comments tab */}
      {activeTab === 'comments' && (
        <div>
          <div className="mb-6">
            <textarea
              value={commentBody}
              onChange={(e) => setCommentBody(e.target.value)}
              rows={3}
              placeholder={t('portal.commentPlaceholder')}
              className="block w-full rounded-md border border-[var(--border)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)] bg-[var(--surface)] text-[var(--foreground)] placeholder-[var(--foreground-muted)]"
              onKeyDown={(e) => {
                if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleAddComment()
              }}
            />
            <div className="mt-2">
              <Button size="sm" onClick={handleAddComment} disabled={addComment.isPending || !commentBody.trim()}>
                {addComment.isPending ? t('comments.adding') : t('comments.add')}
              </Button>
            </div>
          </div>

          {(!comments || comments.length === 0) ? (
            <p className="text-sm text-[var(--foreground-secondary)] py-4 text-center">
              {t('portal.noComments')}
            </p>
          ) : (
            <div className="space-y-4">
              {(sortOrder === 'desc' ? [...comments].reverse() : comments).map((c: PortalComment) => (
                <div key={c.id} className="rounded-lg border border-[var(--border)] bg-[var(--surface)] p-4">
                  <div className="flex items-center gap-2 mb-2">
                    <span className="text-sm font-medium text-[var(--foreground)]">{c.author_name}</span>
                    <span className="text-xs text-[var(--foreground-muted)]">{new Date(c.created_at).toLocaleString()}</span>
                  </div>
                  <p className="text-sm text-[var(--foreground)] whitespace-pre-wrap">{c.body}</p>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* Attachments tab */}
      {activeTab === 'attachments' && (
        <div
          onDrop={handleDrop}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          className={`space-y-4 rounded-lg transition-colors ${dragging ? 'bg-[var(--primary-muted)] ring-2 ring-indigo-400 ring-dashed' : ''}`}
        >
          {/* Upload form — matches AttachmentList */}
          <div className="space-y-2 pb-3 border-b border-[var(--border)]">
            <input
              type="text"
              className="block w-full rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-3 py-1.5 text-sm"
              placeholder={t('attachments.commentPlaceholder')}
              value={uploadComment}
              onChange={(e) => setUploadComment(e.target.value)}
            />
            <div className="flex items-center gap-2">
              <input
                ref={fileInputRef}
                type="file"
                className="text-sm text-[var(--foreground-secondary)] file:mr-2 file:py-1 file:px-3 file:rounded-md file:border-0 file:text-sm file:bg-[var(--primary-muted)] file:text-[var(--primary)]  dark:file:text-[var(--primary)] hover:file:bg-[var(--primary-muted)]"
                onChange={handleFileSelect}
                disabled={uploadMutation.isPending}
              />
              {uploadMutation.isPending && <Spinner size="sm" />}
            </div>
            {uploadMutation.isError && (
              <p className="text-xs text-[var(--danger)]">{t('attachments.uploadFailed')}</p>
            )}
          </div>

          {/* Attachment list — matches AttachmentList */}
          {(sortOrder === 'desc' ? [...(attachments ?? [])].reverse() : (attachments ?? [])).map((a: PortalAttachment) => (
            <div key={a.id} className="flex items-start gap-3 border-b border-[var(--border)] pb-3">
              <div className="flex-1 min-w-0">
                <button
                  onClick={() => handleAttachmentClick(a)}
                  className="text-sm font-medium text-[var(--primary)] hover:underline whitespace-nowrap text-left cursor-pointer"
                >
                  {a.filename}
                </button>
                {editingCommentId === a.id ? (
                  <div className="flex items-center gap-1 mt-0.5">
                    <input
                      type="text"
                      className="text-xs border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] rounded px-1.5 py-0.5 flex-1"
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
                    <button className="text-xs text-[var(--primary)] hover:underline" onClick={() => { updateCommentMutation.mutate({ attachmentId: a.id, comment: editCommentDraft }); setEditingCommentId(null) }}>
                      {t('common.save')}
                    </button>
                    <button className="text-xs text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)]" onClick={() => setEditingCommentId(null)}>
                      {t('common.cancel')}
                    </button>
                  </div>
                ) : a.comment ? (
                  <p
                    className="text-xs text-[var(--foreground-secondary)] mt-0.5 rounded hover:bg-[var(--surface-hover)] cursor-default"
                    onDoubleClick={() => {
                      {
                        setEditingCommentId(a.id)
                        setEditCommentDraft(a.comment)
                      }
                    }}
                  >
                    {a.comment}
                  </p>
                ) : null}
                <p className="text-xs text-[var(--foreground-muted)] mt-0.5">
                  {formatFileSize(a.size_bytes)} &middot; {new Date(a.created_at).toLocaleString()}
                </p>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <Tooltip content={t('attachments.editDescription')}>
                  <button
                    className="p-1 text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)] rounded hover:bg-[var(--surface-hover)]"
                    onClick={() => { setEditingCommentId(a.id); setEditCommentDraft(a.comment ?? '') }}
                  >
                    <Pencil className="h-4 w-4" />
                  </button>
                </Tooltip>
                <Tooltip content={t('preview.download')}>
                  <button
                    className="p-1 text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] dark:hover:text-[var(--foreground-muted)] rounded hover:bg-[var(--surface-hover)]"
                    onClick={() => handleDownload(a.download_url, a.filename)}
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M8 2v8m0 0l-3-3m3 3l3-3M3 12h10" />
                    </svg>
                  </button>
                </Tooltip>
                <Tooltip content={t('common.delete')}>
                  <button
                    className="p-1 text-red-400 hover:text-[var(--danger)] dark:hover:text-red-300 rounded hover:bg-[var(--surface-hover)]"
                    onClick={() => setDeleteTarget(a)}
                  >
                    <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                      <path strokeLinecap="round" strokeLinejoin="round" d="M3 4h10M6 4V3a1 1 0 011-1h2a1 1 0 011 1v1m2 0v9a1 1 0 01-1 1H5a1 1 0 01-1-1V4h8zM7 7v4M9 7v4" />
                    </svg>
                  </button>
                </Tooltip>
              </div>
            </div>
          ))}

          {(attachments ?? []).length === 0 && (
            <p className="text-sm text-[var(--foreground-muted)] italic">{t('attachments.noAttachments')}</p>
          )}

          {/* Delete confirmation modal */}
          <Modal open={!!deleteTarget} onClose={() => setDeleteTarget(null)} title={t('attachments.deleteTitle')}>
            <p className="text-sm text-[var(--foreground-secondary)] mb-4">
              {t('attachments.deleteConfirm')} <strong>{deleteTarget?.filename}</strong>
            </p>
            <div className="flex justify-end gap-3">
              <Button variant="secondary" onClick={() => setDeleteTarget(null)}>{t('common.cancel')}</Button>
              <Button variant="danger" autoFocus disabled={deleteMutation.isPending} onClick={handleDeleteAttachment}>
                {deleteMutation.isPending ? t('common.deleting') : t('common.delete')}
              </Button>
            </div>
          </Modal>
        </div>
      )}
      <FilePreviewModal target={previewTarget} onClose={() => setPreviewTarget(null)} />
    </div>
  )
}
