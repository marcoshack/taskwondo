import { useState, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useWorkItem, useUpdateWorkItem, useDeleteWorkItem, useUploadAttachment } from '@/hooks/useWorkItems'
import { useMembers } from '@/hooks/useProjects'
import { useProjectWorkflow } from '@/hooks/useWorkflows'
import { Spinner } from '@/components/ui/Spinner'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { DetailSidebar } from '@/components/workitems/DetailSidebar'
import { CommentList } from '@/components/workitems/CommentList'
import { ActivityTimeline } from '@/components/workitems/ActivityTimeline'
import { RelationList } from '@/components/workitems/RelationList'
import { AttachmentList } from '@/components/workitems/AttachmentList'
import { usePasteUpload } from '@/hooks/usePasteUpload'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { markdownComponents } from '@/components/ui/markdownComponents'

type Tab = 'comments' | 'activity' | 'relations' | 'attachments'

export function WorkItemDetailPage() {
  const { projectKey, itemNumber: itemNumberParam } = useParams<{ projectKey: string; itemNumber: string }>()
  const navigate = useNavigate()
  const itemNumber = Number(itemNumberParam)

  const { data: item, isLoading } = useWorkItem(projectKey ?? '', itemNumber)
  const { statuses, transitionsMap } = useProjectWorkflow(projectKey ?? '')
  const { data: members } = useMembers(projectKey ?? '')
  const updateMutation = useUpdateWorkItem(projectKey ?? '')
  const deleteMutation = useDeleteWorkItem(projectKey ?? '')

  const [activeTab, setActiveTab] = useState<Tab>('comments')
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc')
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const [editingDesc, setEditingDesc] = useState(false)
  const [descDraft, setDescDraft] = useState('')
  const [showDelete, setShowDelete] = useState(false)
  const [draggingOver, setDraggingOver] = useState(false)
  const [toast, setToast] = useState<string | null>(null)
  const [highlightedAttachmentId, setHighlightedAttachmentId] = useState<string | null>(null)
  const dragCounter = useRef(0)
  const uploadMut = useUploadAttachment(projectKey ?? '', itemNumber)

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
          setToast(`Attached "${file.name}"`)
          setTimeout(() => setToast(null), 3000)
        },
      })
    }
  }, [uploadMut])

  const { handlePaste: handleDescPaste, handleDrop: handleDescDrop, handleDragOver: handleDescDragOver } = usePasteUpload({
    projectKey: projectKey ?? '',
    itemNumber,
    onTextChange: (updater) => setDescDraft(updater),
  })

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!item) {
    return <p className="text-red-600">Work item not found.</p>
  }

  const allowed = transitionsMap?.[item.status]?.map((t) => t.to_status) ?? []

  const tabs: { key: Tab; label: string }[] = [
    { key: 'comments', label: 'Comments' },
    { key: 'activity', label: 'Activity' },
    { key: 'relations', label: 'Relations' },
    { key: 'attachments', label: 'Attachments' },
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
            Drop files to attach
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
        onClick={() => navigate(`/projects/${projectKey}/items`)}
      >
        &larr; Back to items
      </button>

      <div className="flex gap-6">
        {/* Left column */}
        <div className="flex-1 min-w-0 space-y-6">
          {/* Header */}
          <div>
            <div className="flex items-center gap-2 mb-1">
              <span className="text-sm font-mono text-gray-400 dark:text-gray-500">{item.display_id}</span>
              <TypeBadge type={item.type} />
              <StatusBadge status={item.status} statuses={statuses} />
            </div>

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
                    }
                    if (e.key === 'Escape') setEditingTitle(false)
                  }}
                  autoFocus
                />
                <Button size="sm" onClick={() => { updateMutation.mutate({ itemNumber, input: { title: titleDraft } }); setEditingTitle(false) }}>Save</Button>
              </div>
            ) : (
              <h1
                className="text-xl font-semibold text-gray-900 dark:text-gray-100 cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 rounded px-1 -mx-1"
                onClick={() => { setTitleDraft(item.title); setEditingTitle(true) }}
              >
                {item.title}
              </h1>
            )}
          </div>

          {/* Description */}
          <div>
            <h3 className="text-sm font-medium text-gray-500 dark:text-gray-400 mb-1">Description</h3>
            {editingDesc ? (
              <div className="space-y-2">
                <textarea
                  className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm"
                  rows={6}
                  value={descDraft}
                  onChange={(e) => setDescDraft(e.target.value)}
                  onPaste={handleDescPaste}
                  onDrop={handleDescDrop}
                  onDragOver={handleDescDragOver}
                  autoFocus
                />
                <div className="flex gap-2">
                  <Button size="sm" onClick={() => {
                    updateMutation.mutate({ itemNumber, input: { description: descDraft || null } })
                    setEditingDesc(false)
                  }}>Save</Button>
                  <Button size="sm" variant="ghost" onClick={() => setEditingDesc(false)}>Cancel</Button>
                </div>
              </div>
            ) : (
              <div
                className="cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-800 rounded p-2 -m-2 min-h-[2rem]"
                onClick={() => { setDescDraft(item.description ?? ''); setEditingDesc(true) }}
              >
                {item.description ? (
                  <div className="prose prose-sm dark:prose-invert max-w-none text-gray-700 dark:text-gray-300">
                    <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>{item.description}</Markdown>
                  </div>
                ) : (
                  <span className="text-sm text-gray-400 dark:text-gray-500 italic">No description. Click to add.</span>
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
                  title={sortOrder === 'desc' ? 'Showing newest first' : 'Showing oldest first'}
                >
                  <span>{sortOrder === 'desc' ? '\u2193' : '\u2191'}</span>
                  {sortOrder === 'desc' ? 'Newest first' : 'Oldest first'}
                </button>
              )}
            </div>

            {activeTab === 'comments' && <CommentList projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} />}
            {activeTab === 'activity' && <ActivityTimeline projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} onAttachmentClick={(id) => { setActiveTab('attachments'); setHighlightedAttachmentId(id) }} />}
            {activeTab === 'relations' && <RelationList projectKey={projectKey ?? ''} itemNumber={itemNumber} />}
            {activeTab === 'attachments' && <AttachmentList projectKey={projectKey ?? ''} itemNumber={itemNumber} sortOrder={sortOrder} highlightedAttachmentId={highlightedAttachmentId} onHighlightClear={() => setHighlightedAttachmentId(null)} />}
          </div>
        </div>

        {/* Right sidebar */}
        <div className="w-64 shrink-0">
          <DetailSidebar
            item={item}
            statuses={statuses}
            allowedTransitions={allowed}
            members={members ?? []}
            onUpdate={(input) => updateMutation.mutate({ itemNumber, input })}
          />
          <div className="mt-6 pt-4 border-t border-gray-100 dark:border-gray-700">
            <Button variant="danger" size="sm" onClick={() => setShowDelete(true)}>Delete Item</Button>
          </div>
        </div>
      </div>

      {/* Delete confirmation */}
      <Modal open={showDelete} onClose={() => setShowDelete(false)} title="Delete Work Item">
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          Are you sure you want to delete <strong>{item.display_id}</strong>? This action cannot be undone.
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="secondary" onClick={() => setShowDelete(false)}>Cancel</Button>
          <Button
            variant="danger"
            onClick={() => {
              deleteMutation.mutate(itemNumber, {
                onSuccess: () => navigate(`/projects/${projectKey}/items`),
              })
            }}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? 'Deleting...' : 'Delete'}
          </Button>
        </div>
      </Modal>
    </div>
  )
}
