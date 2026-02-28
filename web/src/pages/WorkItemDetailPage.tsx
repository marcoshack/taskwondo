import { useState, useCallback, useRef, useMemo, useEffect } from 'react'
import { useConfirmFeedback } from '@/hooks/useConfirmFeedback'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { ConfirmCheck } from '@/components/ui/ConfirmCheck'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { useWorkItem, useUpdateWorkItem, useDeleteWorkItem, useUploadAttachment, useAttachments } from '@/hooks/useWorkItems'
import { useProject, useMembers, useTypeWorkflows } from '@/hooks/useProjects'
import { useProjectWorkflow, useWorkflows } from '@/hooks/useWorkflows'
import { useMilestones } from '@/hooks/useMilestones'
import { Spinner } from '@/components/ui/Spinner'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { DetailSidebar } from '@/components/workitems/DetailSidebar'
import { CommentList } from '@/components/workitems/CommentList'
import { ActivityTimeline } from '@/components/workitems/ActivityTimeline'
import { RelationList } from '@/components/workitems/RelationList'
import { AttachmentList } from '@/components/workitems/AttachmentList'
import { TimeEntryList } from '@/components/workitems/TimeEntryList'
import { FilePreviewModal } from '@/components/workitems/FilePreviewModal'
import type { PreviewTarget } from '@/components/workitems/FilePreviewModal'
import { usePasteUpload } from '@/hooks/usePasteUpload'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import { MentionModal } from '@/components/ui/MentionModal'
import { Tooltip } from '@/components/ui/Tooltip'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { CopyButton } from '@/components/ui/CopyButton'
import { SLAIndicator } from '@/components/SLAIndicator'
import { useAuth } from '@/contexts/AuthContext'
import { Settings2, User, Calendar, CalendarPlus, History, Lock, Unlock, Globe } from 'lucide-react'
import { InboxButton } from '@/components/workitems/InboxButton'
import { WatchButton } from '@/components/workitems/WatchButton'
import { WatcherList } from '@/components/workitems/WatcherList'
import { useInboxItems } from '@/hooks/useInbox'
import { useWatchers } from '@/hooks/useWorkItems'
import { formatRelativeTime } from '@/utils/duration'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { getMarkdownComponents } from '@/components/ui/markdownComponents'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'

type Tab = 'comments' | 'activity' | 'relations' | 'attachments' | 'time' | 'watchers'

export function WorkItemDetailPage() {
  const { t } = useTranslation()
  const { projectKey, itemNumber: itemNumberParam } = useParams<{ projectKey: string; itemNumber: string }>()
  const navigate = useNavigate()
  const itemNumber = Number(itemNumberParam)

  const { data: item, isLoading } = useWorkItem(projectKey ?? '', itemNumber)
  const { statuses, transitionsMap } = useProjectWorkflow(projectKey ?? '', item?.type)
  const { data: project } = useProject(projectKey ?? '')
  const { data: members } = useMembers(projectKey ?? '')
  const { data: typeWorkflows } = useTypeWorkflows(projectKey ?? '')
  const { data: allWorkflows } = useWorkflows()
  const { data: milestones } = useMilestones(projectKey ?? '')
  const { user } = useAuth()
  const updateMutation = useUpdateWorkItem(projectKey ?? '')
  const deleteMutation = useDeleteWorkItem(projectKey ?? '')

  // Inbox: find if this work item is in the user's inbox
  const { data: inboxData } = useInboxItems()
  const inboxItemId = useMemo(() => {
    if (!item || !inboxData?.items) return undefined
    return inboxData.items.find((i) => i.work_item_id === item.id)?.id
  }, [item, inboxData])

  // Watchers: check if current user is watching
  const { data: watcherData } = useWatchers(projectKey ?? '', itemNumber)
  const isWatching = useMemo(() => {
    if (!watcherData || !user) return false
    if (Array.isArray(watcherData)) {
      return watcherData.some((w) => w.user_id === user.id)
    }
    // ViewerWatcherResponse
    return !!(watcherData as { me: unknown }).me
  }, [watcherData, user])

  const currentUserRole = members?.find((m) => m.user_id === user?.id)?.role ?? (user?.global_role === 'admin' ? 'owner' : null)
  const canEdit = user?.global_role === 'admin' || (currentUserRole != null && currentUserRole !== 'viewer')
  const readOnly = !canEdit

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
  const [copiedId, setCopiedId] = useState(false)
  const [highlightedAttachmentId, setHighlightedAttachmentId] = useState<string | null>(null)
  const [highlightedCommentId, setHighlightedCommentId] = useState<string | null>(null)
  const [previewTarget, setPreviewTarget] = useState<PreviewTarget | null>(null)
  const { confirmed: titleConfirmed, showConfirm: showTitleConfirm } = useConfirmFeedback()
  const { confirmed: descConfirmed, showConfirm: showDescConfirm } = useConfirmFeedback()

  // Build back URL — inbox or project items list
  const location = useLocation()
  const fromInbox = (location.state as { from?: string } | null)?.from === 'inbox'
  const backToListUrl = useMemo(() => {
    if (fromInbox) return '/user/inbox'
    const base = `/projects/${projectKey}/items`
    const stored = sessionStorage.getItem(`taskwondo_listParams_${projectKey}`)
    return stored ? `${base}?${stored}` : base
  }, [projectKey, fromInbox])
  useKeyboardShortcut({ key: '#' }, () => setShowDelete(true), canEdit)

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

  // Reset drag overlay on any drop, even when stopPropagation prevents bubbling
  // (e.g. drops on textarea handled by usePasteUpload)
  useEffect(() => {
    const handler = () => {
      dragCounter.current = 0
      setDraggingOver(false)
    }
    document.addEventListener('drop', handler, true) // capture phase
    return () => document.removeEventListener('drop', handler, true)
  }, [])

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

  const handleAttachmentLinkClick = useCallback((_href: string, attachmentId: string) => {
    const match = allAttachments?.find((a) => a.id === attachmentId)
    if (match) {
      setPreviewTarget({ kind: 'attachment', attachment: match, projectKey: projectKey ?? '', itemNumber })
    }
  }, [allAttachments, projectKey, itemNumber])

  const descMarkdownComponents = useMemo(() => getMarkdownComponents({ onImageClick: handleImageClick, onAttachmentLinkClick: handleAttachmentLinkClick }), [handleImageClick, handleAttachmentLinkClick])

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
        onSuccess: (attachment) => {
          setActiveTab('attachments')
          setHighlightedAttachmentId(attachment.id)
        },
      })
    }
  }, [uploadMut])

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
    { key: 'time', label: t('tabs.time') },
    { key: 'relations', label: t('tabs.relations') },
    { key: 'attachments', label: t('tabs.attachments') },
    { key: 'watchers', label: t('watchers.watchersTab') },
  ]

  return (
    <div
      className="space-y-4 relative overflow-x-hidden"
      onDragEnter={readOnly ? undefined : handlePageDragEnter}
      onDragLeave={readOnly ? undefined : handlePageDragLeave}
      onDragOver={readOnly ? undefined : handlePageDragOver}
      onDrop={readOnly ? undefined : handlePageDrop}
    >
      {draggingOver && (
        <div className="fixed inset-0 z-50 bg-indigo-500/10 border-2 border-dashed border-indigo-400 flex items-center justify-center pointer-events-none">
          <span className="text-lg font-medium text-indigo-600 dark:text-indigo-400 bg-white dark:bg-gray-900 px-6 py-3 rounded-lg shadow-lg">
            {t('workitems.dropToAttach')}
          </span>
        </div>
      )}

      {/* Back link + mobile properties button */}
      <div className="flex items-center">
        <button
          className="text-sm text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          onClick={() => guardedNavigate(backToListUrl)}
        >
          &larr; {t(fromInbox ? 'workitems.backToInbox' : 'workitems.backToItems')}
        </button>
        <span className="lg:hidden ml-auto flex items-center gap-1">
          {item && <InboxButton workItemId={item.id} inboxItemId={inboxItemId} className="p-1.5" />}
          {item && <WatchButton projectKey={projectKey ?? ''} itemNumber={itemNumber} isWatching={isWatching} className="p-1.5" />}
          <button
            onClick={() => setShowProperties(true)}
            className="p-1.5 rounded-md text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
            aria-label={t('workitems.detail.properties')}
          >
            <Settings2 className="h-5 w-5" />
          </button>
        </span>
      </div>

      <div className="flex gap-6">
        {/* Left column */}
        <div className="flex-1 min-w-0 space-y-6">
          {/* Header */}
          <div>
            <div className="flex items-center gap-2 mb-1 group/header flex-wrap">
              <Tooltip content={copiedId ? t('common.copied') : t('common.clickToCopy')}>
                <button
                  type="button"
                  className="text-base lg:text-base font-bold lg:font-semibold font-mono text-gray-600 lg:text-gray-500 dark:text-gray-400 dark:lg:text-gray-400 hover:text-gray-900 dark:hover:text-gray-200 cursor-pointer transition-colors"
                  onClick={async (e) => {
                    e.stopPropagation()
                    try {
                      await navigator.clipboard.writeText(item.display_id)
                    } catch {
                      const ta = document.createElement('textarea')
                      ta.value = item.display_id
                      ta.style.position = 'fixed'
                      ta.style.opacity = '0'
                      document.body.appendChild(ta)
                      ta.select()
                      document.execCommand('copy')
                      document.body.removeChild(ta)
                    }
                    setCopiedId(true)
                    setTimeout(() => setCopiedId(false), 2000)
                  }}
                >
                  {item.display_id}
                </button>
              </Tooltip>
              <TypeBadge type={item.type} />
              <StatusBadge status={item.status} statuses={statuses} />
              <PriorityBadge priority={item.priority} />
              <span className="hidden lg:inline-flex">
                <Tooltip content={t('workitems.form.assignee')}>
                  <span className="inline-flex items-center gap-1 text-xs text-gray-400 dark:text-gray-500">
                    <User className="h-3.5 w-3.5" />
                    {item.assignee_id
                      ? members?.find(m => m.user_id === item.assignee_id)?.display_name ?? t('userPicker.unassigned')
                      : t('userPicker.unassigned')}
                  </span>
                </Tooltip>
              </span>
              {item.due_date && (
                <span className="hidden lg:inline-flex items-center gap-1 text-xs text-gray-400 dark:text-gray-500">
                  <Calendar className="h-3.5 w-3.5" />
                  {item.due_date}
                </span>
              )}
              <span className="hidden lg:inline-flex">
                <Tooltip content={t(`workitems.visibilities.${item.visibility}.description`)}>
                  <span className={`inline-flex items-center gap-1 text-xs ${
                    item.visibility === 'portal' ? 'text-yellow-500 dark:text-yellow-400' :
                    item.visibility === 'public' ? 'text-red-500 dark:text-red-400' :
                    'text-gray-400 dark:text-gray-500'
                  }`}>
                    {item.visibility === 'internal' && <Lock className="h-3.5 w-3.5" />}
                    {item.visibility === 'portal' && <Unlock className="h-3.5 w-3.5" />}
                    {item.visibility === 'public' && <Globe className="h-3.5 w-3.5" />}
                    {t(`workitems.visibilities.${item.visibility}`)}
                  </span>
                </Tooltip>
              </span>
              <span className="hidden lg:inline-flex"><SLAIndicator sla={item.sla} /></span>
              <CopyButton
                text={[
                  '---',
                  `id: ${item.display_id}`,
                  `title: ${item.title}`,
                  `type: ${item.type}`,
                  `status: ${item.status}`,
                  `assignee: ${item.assignee_id ? (members?.find(m => m.user_id === item.assignee_id)?.display_name ?? '') : ''}`,
                  '---',
                  '',
                  `# ${item.display_id} - ${item.title}`,
                  '',
                  item.description ?? '',
                ].join('\n')}
                tooltip={t('common.copyAsMarkdown')}
                className=""
              />
              <span className="hidden lg:inline-flex items-center gap-1">
                <InboxButton workItemId={item.id} inboxItemId={inboxItemId} className="p-1" />
                <WatchButton projectKey={projectKey ?? ''} itemNumber={itemNumber} isWatching={isWatching} className="p-1" />
              </span>
            </div>

            {/* Mobile metadata line */}
            <div className="lg:hidden flex items-center gap-3 text-xs text-gray-400 dark:text-gray-500 mb-2 flex-wrap">
              <span className="inline-flex items-center gap-1">
                <User className="h-3.5 w-3.5" />
                {item.assignee_id
                  ? members?.find(m => m.user_id === item.assignee_id)?.display_name ?? t('userPicker.unassigned')
                  : t('userPicker.unassigned')}
              </span>
              <span className="inline-flex items-center gap-1">
                <CalendarPlus className="h-3.5 w-3.5" />
                {formatRelativeTime(item.created_at)}
              </span>
              <span className="inline-flex items-center gap-1">
                <History className="h-3.5 w-3.5" />
                {formatRelativeTime(item.updated_at)}
              </span>
              {item.sla && <SLAIndicator sla={item.sla} />}
            </div>

            {/* Title */}
            {!readOnly && editingTitle ? (
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
                  className={`text-xl font-semibold text-gray-900 dark:text-gray-100 rounded px-1 -mx-1 ${readOnly ? '' : 'cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800'}`}
                  onClick={readOnly ? undefined : () => { setTitleDraft(item.title); setEditingTitle(true) }}
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
              {!readOnly && !editingDesc && (
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
            {!readOnly && editingDesc ? (
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
                onDoubleClick={readOnly ? undefined : () => { setDescDraft(item.description ?? ''); setEditingDesc(true) }}
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
              <nav className="flex gap-6 pr-8 overflow-x-auto scrollbar-none">
                {tabs.map((tab) => (
                  <button
                    key={tab.key}
                    className={`pb-2 text-sm font-medium border-b-2 whitespace-nowrap ${
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
              {(activeTab === 'comments' || activeTab === 'activity' || activeTab === 'attachments' || activeTab === 'time') && (
                <Tooltip content={sortOrder === 'desc' ? t('common.showingNewestFirst') : t('common.showingOldestFirst')}>
                  <button
                    className="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 pb-2 flex items-center gap-1"
                    onClick={() => setSortOrder((s) => (s === 'desc' ? 'asc' : 'desc'))}
                  >
                  <span className="text-base lg:text-xs">{sortOrder === 'desc' ? '\u2193' : '\u2191'}</span>
                  <span className="hidden lg:inline">{sortOrder === 'desc' ? t('common.newestFirst') : t('common.oldestFirst')}</span>
                  </button>
                </Tooltip>
              )}
            </div>

            {activeTab === 'comments' && <CommentList projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} highlightedCommentId={highlightedCommentId} onHighlightClear={() => setHighlightedCommentId(null)} onImageClick={handleImageClick} onAttachmentLinkClick={handleAttachmentLinkClick} draft={commentDraft} onDraftChange={setCommentDraft} readOnly={readOnly} />}
            {activeTab === 'activity' && <ActivityTimeline projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} onAttachmentClick={(id) => { setActiveTab('attachments'); setHighlightedAttachmentId(id) }} onCommentClick={(id) => { setActiveTab('comments'); setHighlightedCommentId(id) }} />}
            {activeTab === 'relations' && <RelationList projectKey={projectKey ?? ''} itemNumber={itemNumber} readOnly={readOnly} />}
            {activeTab === 'attachments' && <AttachmentList projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} highlightedAttachmentId={highlightedAttachmentId} onHighlightClear={() => setHighlightedAttachmentId(null)} onPreview={(a) => setPreviewTarget({ kind: 'attachment', attachment: a, projectKey: projectKey ?? '', itemNumber })} readOnly={readOnly} />}
            {activeTab === 'time' && <TimeEntryList projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} readOnly={readOnly} />}
            {activeTab === 'watchers' && <WatcherList projectKey={projectKey ?? ''} itemNumber={itemNumber} members={members ?? []} currentUserRole={currentUserRole} />}
          </div>
        </div>

        {/* Right sidebar (desktop only) */}
        <div className="hidden lg:block w-52 shrink-0">
          <DetailSidebar
            item={item}
            projectKey={projectKey ?? ''}
            itemNumber={itemNumber}
            statuses={statuses}
            allowedTransitions={allowed}
            members={members ?? []}
            milestones={milestones}
            allowedComplexityValues={project?.allowed_complexity_values}
            typeWorkflows={typeWorkflows}
            allWorkflows={allWorkflows}
            onUpdate={(input) => updateMutation.mutate({ itemNumber, input })}
            onDelete={() => setShowDelete(true)}
            readOnly={readOnly}
            updateError={updateMutation.isError}
          />
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
          projectKey={projectKey ?? ''}
          itemNumber={itemNumber}
          statuses={statuses}
          allowedTransitions={allowed}
          members={members ?? []}
          milestones={milestones}
          allowedComplexityValues={project?.allowed_complexity_values}
          typeWorkflows={typeWorkflows}
          allWorkflows={allWorkflows}
          onUpdate={(input) => updateMutation.mutate({ itemNumber, input })}
          onDelete={() => { setShowProperties(false); setShowDelete(true) }}
          readOnly={readOnly}
        />
      </Modal>

      {/* Delete confirmation */}
      <Modal open={showDelete} onClose={() => setShowDelete(false)} title={t('workitems.detail.deleteTitle')}>
        <form onSubmit={(e) => {
          e.preventDefault()
          deleteMutation.mutate(itemNumber, {
            onSuccess: () => navigate(backToListUrl),
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
