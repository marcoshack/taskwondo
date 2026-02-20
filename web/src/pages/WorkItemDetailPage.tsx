import { useState, useCallback, useRef, useMemo, useEffect } from 'react'
import { useConfirmFeedback } from '@/hooks/useConfirmFeedback'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { ConfirmCheck } from '@/components/ui/ConfirmCheck'
import { useParams, useNavigate } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { useWorkItem, useUpdateWorkItem, useDeleteWorkItem, useUploadAttachment, useAttachments } from '@/hooks/useWorkItems'
import { useMembers, useTypeWorkflows } from '@/hooks/useProjects'
import { useProjectWorkflow, useWorkflows } from '@/hooks/useWorkflows'
import { Spinner } from '@/components/ui/Spinner'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { DetailSidebar } from '@/components/workitems/DetailSidebar'
import { CommentList } from '@/components/workitems/CommentList'
import { ActivityTimeline } from '@/components/workitems/ActivityTimeline'
import { RelationList } from '@/components/workitems/RelationList'
import { AttachmentList } from '@/components/workitems/AttachmentList'
import { FilePreviewModal } from '@/components/workitems/FilePreviewModal'
import type { PreviewTarget } from '@/components/workitems/FilePreviewModal'
import { usePasteUpload } from '@/hooks/usePasteUpload'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import { MentionModal } from '@/components/ui/MentionModal'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { CopyButton } from '@/components/ui/CopyButton'
import { Settings2, User, Calendar, Lock, Unlock, Globe } from 'lucide-react'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { getMarkdownComponents } from '@/components/ui/markdownComponents'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'

type Tab = 'comments' | 'activity' | 'relations' | 'attachments'

export function WorkItemDetailPage() {
  const { t } = useTranslation()
  const { projectKey, itemNumber: itemNumberParam } = useParams<{ projectKey: string; itemNumber: string }>()
  const navigate = useNavigate()
  const itemNumber = Number(itemNumberParam)

  const { data: item, isLoading } = useWorkItem(projectKey ?? '', itemNumber)
  const { statuses, transitionsMap } = useProjectWorkflow(projectKey ?? '', item?.type)
  const { data: members } = useMembers(projectKey ?? '')
  const { data: typeWorkflows } = useTypeWorkflows(projectKey ?? '')
  const { data: allWorkflows } = useWorkflows()
  const updateMutation = useUpdateWorkItem(projectKey ?? '')
  const deleteMutation = useDeleteWorkItem(projectKey ?? '')

  const [activeTab, setActiveTab] = useState<Tab>('comments')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const [commentDraft, setCommentDraft] = useState('')
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const [editingDesc, setEditingDesc] = useState(false)
  const [descDraft, setDescDraft] = useState('')
  const [showDelete, setShowDelete] = useState(false)
  const [showProperties, setShowProperties] = useState(false)
  const [draggingOver, setDraggingOver] = useState(false)
  const [toast, setToast] = useState<string | null>(null)
  const [highlightedAttachmentId, setHighlightedAttachmentId] = useState<string | null>(null)
  const [highlightedCommentId, setHighlightedCommentId] = useState<string | null>(null)
  const [previewTarget, setPreviewTarget] = useState<PreviewTarget | null>(null)
  const { confirmed: titleConfirmed, showConfirm: showTitleConfirm } = useConfirmFeedback()
  const { confirmed: descConfirmed, showConfirm: showDescConfirm } = useConfirmFeedback()
  useKeyboardShortcut({ key: '#' }, () => setShowDelete(true))

  // Navigation guard for unsaved comment draft
  const hasUnsavedComment = commentDraft.trim().length > 0
  const { guardRef, cancelCallbackRef, guardedNavigate, pendingPath, confirmNavigation, cancelNavigation } = useNavigationGuard()

  useEffect(() => {
    guardRef.current = () => commentDraft.trim().length > 0
    cancelCallbackRef.current = () => setActiveTab('comments')
    return () => {
      guardRef.current = null
      cancelCallbackRef.current = null
    }
  }, [commentDraft, guardRef, cancelCallbackRef])

  // Browser refresh/tab close warning
  useEffect(() => {
    if (!hasUnsavedComment) return
    const handler = (e: BeforeUnloadEvent) => { e.preventDefault() }
    window.addEventListener('beforeunload', handler)
    return () => window.removeEventListener('beforeunload', handler)
  }, [hasUnsavedComment])
  const dragCounter = useRef(0)
  const uploadMut = useUploadAttachment(projectKey ?? '', itemNumber)
  const { data: allAttachments } = useAttachments(projectKey ?? '', itemNumber)

  const handleImageClick = useCallback((src: string) => {
    // Try to resolve attachment filename from the URL (e.g. /api/v1/projects/X/items/N/attachments/{id})
    const parts = src.split('/')
    const attIdx = parts.indexOf('attachments')
    const attId = attIdx >= 0 ? parts[attIdx + 1] : undefined
    const match = attId ? allAttachments?.find((a) => a.id === attId) : undefined
    setPreviewTarget({ kind: 'image', src, label: match?.filename, comment: match?.comment || undefined })
  }, [allAttachments])

  const descMarkdownComponents = useMemo(() => getMarkdownComponents(handleImageClick), [handleImageClick])

  const handlePageDragEnter = useCallback((e: React.DragEvent) => {
    if (!e.dataTransfer.types.includes('Files')) return
    e.preventDefault()
    dragCounter.current++
    setDraggingOver(true)
  }, [])

  const handlePageDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault()
    dragCounter.current--
    if (dragCounter.current <= 0) {
      dragCounter.current = 0
      setDraggingOver(false)
    }
  }, [])

  const handlePageDragOver = useCallback((e: React.DragEvent) => {
    if (!e.dataTransfer.types.includes('Files')) return
    e.preventDefault()
    e.dataTransfer.dropEffect = 'copy'
  }, [])

  const handlePageDrop = useCallback(async (e: React.DragEvent) => {
    e.preventDefault()
    dragCounter.current = 0
    setDraggingOver(false)
    const files = e.dataTransfer?.files
    if (!files?.length) return
    for (const file of files) {
      uploadMut.mutate({ file }, {
        onSuccess: () => {
          setToast(t('workitems.attached', { filename: file.name }))
          setTimeout(() => setToast(null), 3000)
        },
      })
    }
  }, [uploadMut, t])

  const { handlePaste: handleDescPaste, handleDrop: handleDescDrop, handleDragOver: handleDescDragOver } = usePasteUpload({
    projectKey: projectKey ?? '',
    itemNumber,
    onTextChange: (updater) => setDescDraft(updater),
  })

  const descTextareaRef = useRef<HTMLTextAreaElement>(null)
  const descMention = useMentionAutocomplete({
    value: descDraft,
    onValueChange: setDescDraft,
    textareaRef: descTextareaRef,
  })

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!item) {
    return <p className="text-red-600">{t('workitems.notFound')}</p>
  }

  const allowed = transitionsMap?.[item.status]?.map((tr) => tr.to_status) ?? []

  const tabs: { key: Tab; label: string }[] = [
    { key: 'comments', label: t('tabs.comments') },
    { key: 'activity', label: t('tabs.activity') },
    { key: 'relations', label: t('tabs.relations') },
    { key: 'attachments', label: t('tabs.attachments') },
  ]

  return (
    <div
      className="space-y-4 relative"
      onDragEnter={handlePageDragEnter}
      onDragLeave={handlePageDragLeave}
      onDragOver={handlePageDragOver}
      onDrop={handlePageDrop}
    >
      {draggingOver && (
        <div className="fixed inset-0 z-50 bg-indigo-500/10 border-2 border-dashed border-indigo-400 flex items-center justify-center pointer-events-none">
          <span className="text-lg font-medium text-indigo-600 dark:text-indigo-400 bg-white dark:bg-gray-900 px-6 py-3 rounded-lg shadow-lg">
            {t('workitems.dropToAttach')}
          </span>
        </div>
      )}

      {toast && (
        <div className="fixed bottom-6 right-6 z-50 bg-green-600 text-white text-sm px-4 py-2 rounded-lg shadow-lg animate-fade-in">
          {toast}
        </div>
      )}

      {/* Back link */}
      <button
        className="text-sm text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
        onClick={() => guardedNavigate(`/projects/${projectKey}/items`)}
      >
        &larr; {t('workitems.backToItems')}
      </button>

      <div className="flex gap-6">
        {/* Left column */}
        <div className="flex-1 min-w-0 space-y-6">
          {/* Header */}
          <div>
            <div className="flex items-center gap-2 mb-1 group/header">
              <span className="text-base sm:text-sm font-bold sm:font-normal font-mono text-gray-600 sm:text-gray-400 dark:text-gray-400 dark:sm:text-gray-500">{item.display_id}</span>
              <TypeBadge type={item.type} />
              <StatusBadge status={item.status} statuses={statuses} />
              <PriorityBadge priority={item.priority} />
              <CopyButton
                text={[
                  '---',
                  `display_id: ${item.display_id}`,
                  `type: ${item.type}`,
                  `status: ${item.status}`,
                  '---',
                  '',
                  `# ${item.display_id} - ${item.title}`,
                  '',
                  item.description ?? '',
                ].join('\n')}
                tooltip={t('common.copyAsMarkdown')}
                className="opacity-0 group-hover/header:opacity-100"
              />
              <button
                onClick={() => setShowProperties(true)}
                className="sm:hidden ml-auto p-1.5 rounded-md text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
                aria-label={t('workitems.detail.properties')}
              >
                <Settings2 className="h-5 w-5" />
              </button>
            </div>

            {/* Mobile metadata line */}
            <button
              onClick={() => setShowProperties(true)}
              className="sm:hidden flex items-center gap-4 text-xs text-gray-400 dark:text-gray-500 mb-2 w-full text-left"
            >
              <span className="inline-flex items-center gap-1">
                <User className="h-3.5 w-3.5" />
                {item.assignee_id
                  ? members?.find(m => m.user_id === item.assignee_id)?.display_name ?? t('userPicker.unassigned')
                  : t('userPicker.unassigned')}
              </span>
              {item.due_date && (
                <span className="inline-flex items-center gap-1">
                  <Calendar className="h-3.5 w-3.5" />
                  {item.due_date}
                </span>
              )}
              <span className="inline-flex items-center gap-1">
                {item.visibility === 'internal' && <Lock className="h-3.5 w-3.5" />}
                {item.visibility === 'portal' && <Unlock className="h-3.5 w-3.5" />}
                {item.visibility === 'public' && <Globe className="h-3.5 w-3.5" />}
                {t(`workitems.visibilities.${item.visibility}`)}
              </span>
            </button>

            {/* Title */}
            {editingTitle ? (
              <div className="flex gap-2 items-center">
                <input
                  className="text-xl font-semibold text-gray-900 dark:text-gray-100 border-b border-indigo-500 outline-none flex-1 bg-transparent"
                  value={titleDraft}
                  onChange={(e) => setTitleDraft(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      updateMutation.mutate({ itemNumber, input: { title: titleDraft } })
                      setEditingTitle(false)
                      showTitleConfirm()
                    }
                    if (e.key === 'Escape') setEditingTitle(false)
                  }}
                  autoFocus
                />
                <Button size="sm" onClick={() => { updateMutation.mutate({ itemNumber, input: { title: titleDraft } }); setEditingTitle(false); showTitleConfirm() }}>{t('common.save')}</Button>
              </div>
            ) : (
              <div className="flex items-center gap-2">
                <h1
                  className="text-xl font-semibold text-gray-900 dark:text-gray-100 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 rounded px-1 -mx-1"
                  onClick={() => { setTitleDraft(item.title); setEditingTitle(true) }}
                >
                  {item.title}
                </h1>
                <ConfirmCheck visible={titleConfirmed} />
              </div>
            )}
          </div>

          {/* Description */}
          <div className="group/desc">
            <div className="flex items-center gap-1 mb-1">
              <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">{t('workitems.detail.description')}</h3>
              <ConfirmCheck visible={descConfirmed} />
              {!editingDesc && (
                <button
                  className="group/edit relative inline-flex items-center justify-center w-7 h-7 rounded-md text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:text-gray-500 dark:hover:text-gray-300 dark:hover:bg-gray-700 transition-colors opacity-0 group-hover/desc:opacity-100"
                  onClick={() => { setDescDraft(item.description ?? ''); setEditingDesc(true) }}
                >
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                    <path strokeLinecap="round" strokeLinejoin="round" d="M11.5 2.5a1.5 1.5 0 012.121 2.121L6.5 11.743l-2.5.757.757-2.5L11.5 2.5z" />
                  </svg>
                  <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 px-2 py-1 text-xs text-white bg-gray-900 dark:bg-gray-700 rounded whitespace-nowrap opacity-0 group-hover/edit:opacity-100 transition-opacity">
                    {t('common.edit')}
                  </span>
                </button>
              )}
              {!editingDesc && item.description && (
                <CopyButton text={item.description} className="opacity-0 group-hover/desc:opacity-100" />
              )}
            </div>
            {editingDesc ? (
              <div className="space-y-2">
                <textarea
                  ref={descTextareaRef}
                  className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm"
                  rows={6}
                  value={descDraft}
                  onChange={(e) => setDescDraft(e.target.value)}
                  onKeyDown={(e) => {
                    descMention.onMentionKeyDown(e)
                    if (e.defaultPrevented) return
                    if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                      e.preventDefault()
                      updateMutation.mutate({ itemNumber, input: { description: descDraft || null } })
                      setEditingDesc(false)
                      showDescConfirm()
                    }
                    if (e.key === 'Escape') setEditingDesc(false)
                  }}
                  onPaste={handleDescPaste}
                  onDrop={handleDescDrop}
                  onDragOver={handleDescDragOver}
                  autoFocus
                />
                <div className="flex gap-2">
                  <Button size="sm" onClick={() => {
                    updateMutation.mutate({ itemNumber, input: { description: descDraft || null } })
                    setEditingDesc(false)
                    showDescConfirm()
                  }}>{t('common.save')}</Button>
                  <Button size="sm" variant="ghost" onClick={() => setEditingDesc(false)}>{t('common.cancel')}</Button>
                </div>
              </div>
            ) : (
              <div
                className="hover:bg-gray-50 dark:hover:bg-gray-800 rounded p-2 -m-2 min-h-[2rem]"
                onDoubleClick={() => { setDescDraft(item.description ?? ''); setEditingDesc(true) }}
              >
                {item.description ? (
                  <div className="prose prose-sm dark:prose-invert max-w-none text-gray-700 dark:text-gray-300 break-words">
                    <Markdown remarkPlugins={[remarkGfm]} components={descMarkdownComponents}>{item.description}</Markdown>
                  </div>
                ) : (
                  <span className="text-sm text-gray-400 dark:text-gray-500 italic">{t('workitems.detail.noDescription')}</span>
                )}
              </div>
            )}
          </div>

          {/* Tabs */}
          <div>
            <div className="border-b border-gray-200 dark:border-gray-700 mb-4 flex items-center justify-between">
              <nav className="flex gap-6">
                {tabs.map((tab) => (
                  <button
                    key={tab.key}
                    className={`pb-2 text-sm font-medium border-b-2 ${
                      activeTab === tab.key
                        ? 'border-indigo-500 text-indigo-600 dark:text-indigo-400'
                        : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
                    }`}
                    onClick={() => setActiveTab(tab.key)}
                  >
                    {tab.label}
                  </button>
                ))}
              </nav>
              {(activeTab === 'comments' || activeTab === 'activity' || activeTab === 'attachments') && (
                <button
                  className="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 pb-2 flex items-center gap-1"
                  onClick={() => setSortOrder((s) => (s === 'desc' ? 'asc' : 'desc'))}
                  title={sortOrder === 'desc' ? t('common.showingNewestFirst') : t('common.showingOldestFirst')}
                >
                  <span className="text-base sm:text-xs">{sortOrder === 'desc' ? '\u2193' : '\u2191'}</span>
                  <span className="hidden sm:inline">{sortOrder === 'desc' ? t('common.newestFirst') : t('common.oldestFirst')}</span>
                </button>
              )}
            </div>

            {activeTab === 'comments' && <CommentList projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} highlightedCommentId={highlightedCommentId} onHighlightClear={() => setHighlightedCommentId(null)} onImageClick={handleImageClick} draft={commentDraft} onDraftChange={setCommentDraft} />}
            {activeTab === 'activity' && <ActivityTimeline projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} onAttachmentClick={(id) => { setActiveTab('attachments'); setHighlightedAttachmentId(id) }} onCommentClick={(id) => { setActiveTab('comments'); setHighlightedCommentId(id) }} />}
            {activeTab === 'relations' && <RelationList projectKey={projectKey ?? ''} itemNumber={itemNumber} />}
            {activeTab === 'attachments' && <AttachmentList projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} highlightedAttachmentId={highlightedAttachmentId} onHighlightClear={() => setHighlightedAttachmentId(null)} onPreview={(a) => setPreviewTarget({ kind: 'attachment', attachment: a, projectKey: projectKey ?? '', itemNumber })} />}
          </div>
        </div>

        {/* Right sidebar (desktop only) */}
        <div className="hidden sm:block w-52 shrink-0">
          <DetailSidebar
            item={item}
            statuses={statuses}
            allowedTransitions={allowed}
            members={members ?? []}
            typeWorkflows={typeWorkflows}
            allWorkflows={allWorkflows}
            onUpdate={(input) => updateMutation.mutate({ itemNumber, input })}
          />
          <div className="mt-6 pt-4 border-t border-gray-100 dark:border-gray-700">
            <Button variant="danger" size="sm" onClick={() => setShowDelete(true)}>{t('workitems.detail.deleteItem')}</Button>
          </div>
        </div>
      </div>

      <MentionModal
        open={descMention.mentionModalOpen}
        position={descMention.dropdownPosition}
        onClose={descMention.onMentionClose}
        onSelect={descMention.onMentionSelect}
        projectKey={projectKey ?? ''}
      />

      {/* Mobile properties modal */}
      <Modal open={showProperties} onClose={() => setShowProperties(false)} title={t('workitems.detail.properties')}>
        <DetailSidebar
          item={item}
          statuses={statuses}
          allowedTransitions={allowed}
          members={members ?? []}
          typeWorkflows={typeWorkflows}
          allWorkflows={allWorkflows}
          onUpdate={(input) => updateMutation.mutate({ itemNumber, input })}
        />
        <div className="mt-6 pt-4 border-t border-gray-100 dark:border-gray-700">
          <Button variant="danger" size="sm" onClick={() => { setShowProperties(false); setShowDelete(true) }}>{t('workitems.detail.deleteItem')}</Button>
        </div>
      </Modal>

      {/* Delete confirmation */}
      <Modal open={showDelete} onClose={() => setShowDelete(false)} title={t('workitems.detail.deleteTitle')}>
        <form onSubmit={(e) => {
          e.preventDefault()
          deleteMutation.mutate(itemNumber, {
            onSuccess: () => navigate(`/projects/${projectKey}/items`),
          })
        }}>
          <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
            <Trans i18nKey="workitems.detail.deleteConfirmBody" values={{ displayId: item.display_id }} components={{ bold: <strong /> }} />
          </p>
          <div className="flex justify-end gap-3">
            <Button type="button" variant="secondary" onClick={() => setShowDelete(false)}>{t('common.cancel')}</Button>
            <Button
              type="submit"
              variant="danger"
              autoFocus
              disabled={deleteMutation.isPending}
            >
              {deleteMutation.isPending ? t('common.deleting') : t('common.delete')}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Unsaved comment warning */}
      <Modal open={!!pendingPath} onClose={cancelNavigation} title={t('comments.unsavedTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          {t('comments.unsavedBody')}
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="secondary" onClick={cancelNavigation}>
            {t('comments.keepEditing')}
          </Button>
          <Button variant="danger" onClick={() => { setCommentDraft(''); confirmNavigation() }}>
            {t('comments.discard')}
          </Button>
        </div>
      </Modal>

      {/* File preview */}
      <FilePreviewModal target={previewTarget} onClose={() => setPreviewTarget(null)} />
    </div>
  )
}
