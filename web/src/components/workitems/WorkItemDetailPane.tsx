import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link } from 'react-router-dom'
import { ExternalLink, Settings2, X } from 'lucide-react'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { DetailSidebar } from '@/components/workitems/DetailSidebar'
import { CommentList } from '@/components/workitems/CommentList'
import { DescriptionWithInlineComments } from '@/components/workitems/DescriptionWithInlineComments'
import { RelationList } from '@/components/workitems/RelationList'
import { AttachmentList } from '@/components/workitems/AttachmentList'
import { TimeEntryList } from '@/components/workitems/TimeEntryList'
import { FilePreviewModal } from '@/components/workitems/FilePreviewModal'
import type { PreviewTarget } from '@/components/workitems/FilePreviewModal'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { ConfirmCheck } from '@/components/ui/ConfirmCheck'
import { Tooltip } from '@/components/ui/Tooltip'
import { MentionSearchModal } from '@/components/ui/MentionSearchModal'
import { CopyButton } from '@/components/ui/CopyButton'
import { useWorkItem, useUpdateWorkItem, useAttachments, useRelations } from '@/hooks/useWorkItems'
import { useProject, useMembers, useTypeWorkflows } from '@/hooks/useProjects'
import { useProjectWorkflow, useProjectWorkflows } from '@/hooks/useWorkflows'
import { useMilestones } from '@/hooks/useMilestones'
import { useAuth } from '@/contexts/AuthContext'
import { useConfirmFeedback } from '@/hooks/useConfirmFeedback'
import { usePasteUpload } from '@/hooks/usePasteUpload'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import type { WorkItem } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

type PaneTab = 'comments' | 'relations' | 'attachments' | 'time'

export interface WorkItemDetailPaneProps {
  projectKey: string
  itemNumber: number
  /** List-row snapshot shown while the full GET is in flight / as fallback. */
  listItem?: WorkItem | null
  /** Fallback statuses when type-specific workflow is not yet ready. */
  statuses: WorkflowStatus[]
  fullPageHref: string
  onClose: () => void
  readOnly?: boolean
}

function DetailSkeleton() {
  return (
    <div data-testid="work-item-detail-pane-skeleton" className="space-y-4 animate-pulse" aria-hidden="true">
      <div className="h-7 w-3/4 rounded bg-gray-200 dark:bg-gray-700" />
      <div className="space-y-2">
        <div className="h-3 w-24 rounded bg-gray-200 dark:bg-gray-700" />
        <div className="h-20 w-full rounded bg-gray-200 dark:bg-gray-700" />
      </div>
      <div className="grid grid-cols-2 gap-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="space-y-1">
            <div className="h-3 w-16 rounded bg-gray-200 dark:bg-gray-700" />
            <div className="h-9 w-full rounded bg-gray-200 dark:bg-gray-700" />
          </div>
        ))}
      </div>
      <div className="h-8 w-full rounded bg-gray-200 dark:bg-gray-700" />
      <div className="space-y-2">
        <div className="h-16 w-full rounded bg-gray-200 dark:bg-gray-700" />
        <div className="h-16 w-full rounded bg-gray-200 dark:bg-gray-700" />
      </div>
    </div>
  )
}

/**
 * Right-side detail panel (TASK-77 Option C / TASK-79).
 * Lazily fetches `GET …/items/:id` on selection, keeps the shell mounted across
 * row changes, and reuses the same editable surfaces as the full detail page.
 */
export function WorkItemDetailPane({
  projectKey,
  itemNumber,
  listItem = null,
  statuses: fallbackStatuses,
  fullPageHref,
  onClose,
  readOnly: readOnlyProp,
}: WorkItemDetailPaneProps) {
  const { t } = useTranslation()
  const { user } = useAuth()

  const { data: fetchedItem, isLoading, isFetching } = useWorkItem(projectKey, itemNumber, {
    retainInCache: true,
    staleTime: 30_000,
  })
  const item = fetchedItem ?? listItem ?? null

  const { statuses, transitionsMap } = useProjectWorkflow(projectKey, item?.type)
  const statusesForBadges = statuses.length ? statuses : fallbackStatuses
  const { data: project } = useProject(projectKey)
  const { data: members } = useMembers(projectKey)
  const { data: typeWorkflows } = useTypeWorkflows(projectKey)
  const { data: allWorkflows } = useProjectWorkflows(projectKey)
  const { data: milestones } = useMilestones(projectKey)
  const { data: relations } = useRelations(projectKey, itemNumber)
  const updateMutation = useUpdateWorkItem(projectKey)

  const currentUserRole =
    members?.find((m) => m.user_id === user?.id)?.role ?? (user?.global_role === 'admin' ? 'owner' : null)
  const canEdit = user?.global_role === 'admin' || (currentUserRole != null && currentUserRole !== 'viewer')
  const readOnly = readOnlyProp ?? !canEdit

  const allowed = item ? (transitionsMap?.[item.status]?.map((tr) => tr.to_status) ?? []) : []

  const currentDisplayId = item?.display_id ?? `${projectKey}-${itemNumber}`
  const childrenRelations = useMemo(() => {
    if (!relations) return []
    return relations.filter((r) => {
      const isSource = r.source_display_id === currentDisplayId
      return (isSource && r.relation_type === 'parent_of') || (!isSource && r.relation_type === 'child_of')
    })
  }, [relations, currentDisplayId])

  const [activeTab, setActiveTab] = useState<PaneTab>('comments')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const [commentDraft, setCommentDraft] = useState('')
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const [editingDesc, setEditingDesc] = useState(false)
  const [descDraft, setDescDraft] = useState('')
  const [showProperties, setShowProperties] = useState(false)
  const [openThreadRootId, setOpenThreadRootId] = useState<string | null>(null)
  const [previewTarget, setPreviewTarget] = useState<PreviewTarget | null>(null)
  const { confirmed: titleConfirmed, showConfirm: showTitleConfirm } = useConfirmFeedback()
  const { confirmed: descConfirmed, showConfirm: showDescConfirm } = useConfirmFeedback()

  // Reset ephemeral editor state when selection changes — shell stays mounted.
  useEffect(() => {
    setActiveTab('comments')
    setCommentDraft('')
    setEditingTitle(false)
    setTitleDraft('')
    setEditingDesc(false)
    setDescDraft('')
    setShowProperties(false)
    setOpenThreadRootId(null)
    setPreviewTarget(null)
  }, [itemNumber])

  const { data: allAttachments } = useAttachments(projectKey, itemNumber)
  const handleImageClick = useCallback(
    (src: string) => {
      const parts = src.split('/')
      const attIdx = parts.indexOf('attachments')
      const attId = attIdx >= 0 ? parts[attIdx + 1] : undefined
      const match = attId ? allAttachments?.find((a) => a.id === attId) : undefined
      setPreviewTarget({ kind: 'image', src, label: match?.filename, comment: match?.comment || undefined })
    },
    [allAttachments],
  )
  const handleAttachmentLinkClick = useCallback(
    (_href: string, attachmentId: string) => {
      const match = allAttachments?.find((a) => a.id === attachmentId)
      if (match) {
        setPreviewTarget({ kind: 'attachment', attachment: match, projectKey, itemNumber })
      }
    },
    [allAttachments, projectKey, itemNumber],
  )

  const { handlePaste: handleDescPaste, handleDrop: handleDescDrop, handleDragOver: handleDescDragOver } =
    usePasteUpload({
      projectKey,
      itemNumber,
      onTextChange: (updater) => setDescDraft(updater),
    })

  const descTextareaRef = useRef<HTMLTextAreaElement>(null)
  const descMention = useMentionAutocomplete({
    value: descDraft,
    onValueChange: setDescDraft,
    textareaRef: descTextareaRef,
  })

  // Prefer list snapshot for first paint; skeleton only when neither snapshot nor cache exists.
  const showSkeleton = !item && (isLoading || isFetching)
  const showStaleHint = Boolean(item && isFetching && !fetchedItem)

  const tabs: { key: PaneTab; label: string }[] = [
    { key: 'comments', label: t('tabs.comments') },
    { key: 'time', label: t('tabs.time') },
    { key: 'relations', label: t('tabs.relations') },
    { key: 'attachments', label: t('tabs.attachments') },
  ]

  return (
    <aside
      data-testid="work-item-detail-pane"
      data-item-number={itemNumber}
      className="flex h-full min-h-0 w-full flex-col border-l border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-900 max-lg:border-l-0 lg:animate-detail-pane-in"
      aria-label={item?.title ?? t('workitems.splitPane.panelLabel')}
      aria-busy={showSkeleton || undefined}
    >
      <header className="flex items-start gap-2 border-b border-gray-200 dark:border-gray-700 px-4 py-3">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-1.5 mb-1">
            <span className="font-mono text-xs font-semibold text-gray-500 dark:text-gray-400">
              {item?.display_id ?? `${projectKey}-${itemNumber}`}
            </span>
            {item && (
              <>
                <TypeBadge type={item.type} />
                <StatusBadge status={item.status} statuses={statusesForBadges} />
                <PriorityBadge priority={item.priority} />
              </>
            )}
            {showStaleHint && (
              <span className="text-[10px] uppercase tracking-wide text-gray-400 dark:text-gray-500">
                {t('workitems.splitPane.refreshing')}
              </span>
            )}
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-1">
          <button
            type="button"
            className="lg:hidden p-1.5 rounded-md text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
            onClick={() => setShowProperties(true)}
            aria-label={t('workitems.detail.properties')}
          >
            <Settings2 className="h-4 w-4" />
          </button>
          <Link
            to={fullPageHref}
            className="inline-flex items-center gap-1 rounded px-2 py-1 text-xs font-medium text-indigo-600 hover:bg-indigo-50 dark:text-indigo-400 dark:hover:bg-indigo-950/40"
          >
            <ExternalLink className="h-3.5 w-3.5" aria-hidden="true" />
            {t('workitems.splitPane.openFull')}
          </Link>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={onClose}
            aria-label={t('common.close')}
            title={t('workitems.splitPane.closePanel')}
          >
            <X className="h-4 w-4" />
          </Button>
        </div>
      </header>

      <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
        {showSkeleton ? (
          <DetailSkeleton />
        ) : (
          <div key={itemNumber} className="space-y-5">
            {/* Title (inline edit) */}
            {!readOnly && editingTitle ? (
              <div className="flex gap-2 items-center">
                <input
                  className="text-lg font-semibold text-gray-900 dark:text-gray-100 border-b border-indigo-500 outline-none flex-1 bg-transparent"
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
                <Button
                  size="sm"
                  onClick={() => {
                    updateMutation.mutate({ itemNumber, input: { title: titleDraft } })
                    setEditingTitle(false)
                    showTitleConfirm()
                  }}
                >
                  {t('common.save')}
                </Button>
              </div>
            ) : (
              <div className="flex items-center gap-2 -mt-1">
                <button
                  type="button"
                  className={`text-left text-lg font-semibold text-gray-900 dark:text-gray-100 rounded px-1 -mx-1 ${
                    readOnly
                      ? ''
                      : 'cursor-pointer border border-transparent hover:border-gray-300 dark:hover:border-gray-600'
                  }`}
                  onClick={
                    readOnly
                      ? undefined
                      : () => {
                          setTitleDraft(item!.title)
                          setEditingTitle(true)
                        }
                  }
                >
                  {item!.title}
                </button>
                <ConfirmCheck visible={titleConfirmed} />
              </div>
            )}

            {/* Description */}
            <div className="group/desc">
              <div className="flex items-center gap-1 mb-1">
                <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400">
                  {t('workitems.detail.description')}
                </h3>
                <ConfirmCheck visible={descConfirmed} />
                {!readOnly && !editingDesc && (
                  <button
                    type="button"
                    className="text-xs text-indigo-600 dark:text-indigo-400 hover:underline ml-auto"
                    onClick={() => {
                      setDescDraft(item!.description ?? '')
                      setEditingDesc(true)
                    }}
                  >
                    {t('common.edit')}
                  </button>
                )}
                {!editingDesc && item!.description && (
                  <CopyButton text={item!.description} className="opacity-0 group-hover/desc:opacity-100" />
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
                    <Button
                      size="sm"
                      onClick={() => {
                        updateMutation.mutate({ itemNumber, input: { description: descDraft || null } })
                        setEditingDesc(false)
                        showDescConfirm()
                      }}
                    >
                      {t('common.save')}
                    </Button>
                    <Button size="sm" variant="ghost" onClick={() => setEditingDesc(false)}>
                      {t('common.cancel')}
                    </Button>
                  </div>
                </div>
              ) : (
                <div
                  className="relative border border-transparent hover:border-gray-300 dark:hover:border-gray-600 rounded p-2 min-h-[2rem] cursor-pointer"
                  onClick={
                    readOnly
                      ? undefined
                      : () => {
                          setDescDraft(item!.description ?? '')
                          setEditingDesc(true)
                        }
                  }
                >
                  {item!.description ? (
                    <div onClick={(e) => e.stopPropagation()}>
                      <DescriptionWithInlineComments
                        projectKey={projectKey}
                        itemNumber={itemNumber}
                        description={item!.description}
                        readOnly={readOnly}
                        onImageClick={handleImageClick}
                        onAttachmentLinkClick={handleAttachmentLinkClick}
                        openThreadRootId={openThreadRootId}
                        onOpenThread={setOpenThreadRootId}
                      />
                    </div>
                  ) : (
                    <span className="text-sm text-gray-400 dark:text-gray-500 italic">
                      {t('workitems.detail.noDescription')}
                    </span>
                  )}
                </div>
              )}
            </div>

            {/* Properties — desktop inline; mobile via sheet button */}
            <div className="hidden lg:block border border-gray-100 dark:border-gray-800 rounded-lg p-3">
              <DetailSidebar
                item={item!}
                projectKey={projectKey}
                itemNumber={itemNumber}
                statuses={statusesForBadges}
                allowedTransitions={allowed}
                members={members ?? []}
                milestones={milestones}
                allowedComplexityValues={project?.allowed_complexity_values}
                typeWorkflows={typeWorkflows}
                allWorkflows={allWorkflows}
                onUpdate={(input) => updateMutation.mutate({ itemNumber, input })}
                readOnly={readOnly}
                updateError={updateMutation.isError}
              />
            </div>

            {childrenRelations.length > 0 && (
              <p className="text-xs text-gray-400 dark:text-gray-500">
                {t('relations.childrenProgress', {
                  completed: childrenRelations.filter((r) => {
                    const isSource = r.source_display_id === currentDisplayId
                    const category = isSource ? r.target_status_category : r.source_status_category
                    return category === 'done' || category === 'cancelled'
                  }).length,
                  total: childrenRelations.length,
                })}
              </p>
            )}

            {/* Comments / relations / time / attachments */}
            <div>
              <div className="border-b border-gray-200 dark:border-gray-700 mb-3 flex items-center justify-between">
                <nav className="flex gap-4 overflow-x-auto scrollbar-none pr-2">
                  {tabs.map((tab) => (
                    <button
                      key={tab.key}
                      type="button"
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
                {(activeTab === 'comments' || activeTab === 'attachments' || activeTab === 'time') && (
                  <Tooltip
                    content={
                      sortOrder === 'desc' ? t('common.showingNewestFirst') : t('common.showingOldestFirst')
                    }
                  >
                    <button
                      type="button"
                      className="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 pb-2"
                      onClick={() => setSortOrder((s) => (s === 'desc' ? 'asc' : 'desc'))}
                    >
                      {sortOrder === 'desc' ? '↓' : '↑'}
                    </button>
                  </Tooltip>
                )}
              </div>

              {activeTab === 'comments' && (
                <CommentList
                  projectKey={projectKey}
                  itemNumber={itemNumber}
                  sortOrder={sortOrder}
                  draft={commentDraft}
                  onDraftChange={setCommentDraft}
                  readOnly={readOnly}
                  itemVisibility={item?.visibility}
                  onImageClick={handleImageClick}
                  onAttachmentLinkClick={handleAttachmentLinkClick}
                  onViewInline={setOpenThreadRootId}
                />
              )}
              {activeTab === 'relations' && (
                <RelationList projectKey={projectKey} itemNumber={itemNumber} readOnly={readOnly} />
              )}
              {activeTab === 'attachments' && (
                <AttachmentList
                  projectKey={projectKey}
                  itemNumber={itemNumber}
                  sortOrder={sortOrder}
                  onPreview={(a) =>
                    setPreviewTarget({ kind: 'attachment', attachment: a, projectKey, itemNumber })
                  }
                  readOnly={readOnly}
                />
              )}
              {activeTab === 'time' && (
                <TimeEntryList
                  projectKey={projectKey}
                  itemNumber={itemNumber}
                  sortOrder={sortOrder}
                  readOnly={readOnly}
                />
              )}
            </div>
          </div>
        )}
      </div>

      <MentionSearchModal
        open={descMention.mentionModalOpen}
        position={descMention.dropdownPosition}
        onClose={descMention.onMentionClose}
        onSelect={descMention.onMentionSelect}
      />

      <Modal open={showProperties} onClose={() => setShowProperties(false)} title={t('workitems.detail.properties')}>
        {item && (
          <DetailSidebar
            item={item}
            projectKey={projectKey}
            itemNumber={itemNumber}
            statuses={statusesForBadges}
            allowedTransitions={allowed}
            members={members ?? []}
            milestones={milestones}
            allowedComplexityValues={project?.allowed_complexity_values}
            typeWorkflows={typeWorkflows}
            allWorkflows={allWorkflows}
            onUpdate={(input) => updateMutation.mutate({ itemNumber, input })}
            readOnly={readOnly}
            updateError={updateMutation.isError}
          />
        )}
      </Modal>

      <FilePreviewModal target={previewTarget} onClose={() => setPreviewTarget(null)} />
    </aside>
  )
}
