import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useWorkItem, useUpdateWorkItem, useDeleteWorkItem } from '@/hooks/useWorkItems'
import { useMembers } from '@/hooks/useProjects'
import { useProjectWorkflow } from '@/hooks/useWorkflows'
import { Spinner } from '@/components/ui/Spinner'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { DetailSidebar } from '@/components/workitems/DetailSidebar'
import { CommentList } from '@/components/workitems/CommentList'
import { ActivityTimeline } from '@/components/workitems/ActivityTimeline'
import { RelationList } from '@/components/workitems/RelationList'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'

type Tab = 'comments' | 'activity' | 'relations'

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
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const [editingDesc, setEditingDesc] = useState(false)
  const [descDraft, setDescDraft] = useState('')
  const [showDelete, setShowDelete] = useState(false)

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
  ]

  return (
    <div className="space-y-4">
      {/* Back link */}
      <button
        className="text-sm text-gray-400 hover:text-gray-600"
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
              <span className="text-sm font-mono text-gray-400">{item.display_id}</span>
              <TypeBadge type={item.type} />
              <StatusBadge status={item.status} statuses={statuses} />
            </div>

            {/* Title */}
            {editingTitle ? (
              <div className="flex gap-2 items-center">
                <input
                  className="text-xl font-semibold text-gray-900 border-b border-indigo-500 outline-none flex-1 bg-transparent"
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
                className="text-xl font-semibold text-gray-900 cursor-pointer hover:bg-gray-50 rounded px-1 -mx-1"
                onClick={() => { setTitleDraft(item.title); setEditingTitle(true) }}
              >
                {item.title}
              </h1>
            )}
          </div>

          {/* Description */}
          <div>
            <h3 className="text-sm font-medium text-gray-500 mb-1">Description</h3>
            {editingDesc ? (
              <div className="space-y-2">
                <textarea
                  className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  rows={6}
                  value={descDraft}
                  onChange={(e) => setDescDraft(e.target.value)}
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
                className="text-sm text-gray-700 whitespace-pre-wrap cursor-pointer hover:bg-gray-50 rounded p-2 -m-2 min-h-[2rem]"
                onClick={() => { setDescDraft(item.description ?? ''); setEditingDesc(true) }}
              >
                {item.description || <span className="text-gray-400 italic">No description. Click to add.</span>}
              </div>
            )}
          </div>

          {/* Tabs */}
          <div>
            <div className="border-b border-gray-200 mb-4">
              <nav className="flex gap-6">
                {tabs.map((tab) => (
                  <button
                    key={tab.key}
                    className={`pb-2 text-sm font-medium border-b-2 ${
                      activeTab === tab.key
                        ? 'border-indigo-500 text-indigo-600'
                        : 'border-transparent text-gray-500 hover:text-gray-700'
                    }`}
                    onClick={() => setActiveTab(tab.key)}
                  >
                    {tab.label}
                  </button>
                ))}
              </nav>
            </div>

            {activeTab === 'comments' && <CommentList projectKey={projectKey ?? ''} itemNumber={itemNumber} />}
            {activeTab === 'activity' && <ActivityTimeline projectKey={projectKey ?? ''} itemNumber={itemNumber} />}
            {activeTab === 'relations' && <RelationList projectKey={projectKey ?? ''} itemNumber={itemNumber} />}
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
          <div className="mt-6 pt-4 border-t border-gray-100">
            <Button variant="danger" size="sm" onClick={() => setShowDelete(true)}>Delete Item</Button>
          </div>
        </div>
      </div>

      {/* Delete confirmation */}
      <Modal open={showDelete} onClose={() => setShowDelete(false)} title="Delete Work Item">
        <p className="text-sm text-gray-600 mb-4">
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
