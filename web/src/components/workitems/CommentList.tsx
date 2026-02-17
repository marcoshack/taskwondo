import { useState } from 'react'
import { useComments, useCreateComment, useUpdateComment, useDeleteComment } from '@/hooks/useWorkItems'
import { useAuth } from '@/contexts/AuthContext'
import { useMembers } from '@/hooks/useProjects'
import { Button } from '@/components/ui/Button'
import { Avatar } from '@/components/ui/Avatar'
import { Spinner } from '@/components/ui/Spinner'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface CommentListProps {
  projectKey: string
  itemNumber: number
  sortOrder?: 'asc' | 'desc'
}

export function CommentList({ projectKey, itemNumber, sortOrder = 'desc' }: CommentListProps) {
  const { user } = useAuth()
  const { data: comments, isLoading } = useComments(projectKey, itemNumber)
  const { data: members } = useMembers(projectKey)
  const createMutation = useCreateComment(projectKey, itemNumber)
  const updateMutation = useUpdateComment(projectKey, itemNumber)
  const deleteMutation = useDeleteComment(projectKey, itemNumber)

  const [newBody, setNewBody] = useState('')
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editBody, setEditBody] = useState('')

  function authorName(authorId: string | null): string {
    if (!authorId) return 'Unknown'
    const member = members?.find((m) => m.user_id === authorId)
    return member?.display_name ?? 'Unknown'
  }

  if (isLoading) return <Spinner size="sm" />

  return (
    <div className="space-y-4">
      {(sortOrder === 'desc' ? [...(comments ?? [])].reverse() : (comments ?? [])).map((c) => (
        <div key={c.id} className="border-b border-gray-100 pb-3">
          {editingId === c.id ? (
            <div className="space-y-2">
              <textarea
                className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                rows={3}
                value={editBody}
                onChange={(e) => setEditBody(e.target.value)}
              />
              <div className="flex gap-2">
                <Button
                  size="sm"
                  onClick={() => {
                    updateMutation.mutate({ commentId: c.id, body: editBody }, {
                      onSuccess: () => setEditingId(null),
                    })
                  }}
                  disabled={updateMutation.isPending}
                >
                  Save
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setEditingId(null)}>Cancel</Button>
              </div>
            </div>
          ) : (
            <>
              <div className="flex items-center gap-2 mb-1">
                <Avatar name={authorName(c.author_id)} size="xs" />
                <span className="text-sm font-medium text-gray-700">{authorName(c.author_id)}</span>
                <span className="text-xs text-gray-400">{new Date(c.created_at).toLocaleString()}</span>
              </div>
              <div className="prose prose-sm max-w-none text-gray-900 pl-8">
                <Markdown remarkPlugins={[remarkGfm]}>{c.body}</Markdown>
              </div>
              <div className="flex items-center gap-3 mt-1 pl-8">
                {c.visibility !== 'internal' && (
                  <span className="text-xs text-indigo-500">{c.visibility}</span>
                )}
                {user && c.author_id === user.id && (
                  <>
                    <button
                      className="text-xs text-gray-400 hover:text-gray-600"
                      onClick={() => { setEditingId(c.id); setEditBody(c.body) }}
                    >
                      Edit
                    </button>
                    <button
                      className="text-xs text-red-400 hover:text-red-600"
                      onClick={() => { if (confirm('Delete this comment?')) deleteMutation.mutate(c.id) }}
                    >
                      Delete
                    </button>
                  </>
                )}
              </div>
            </>
          )}
        </div>
      ))}

      <div className="space-y-2 pt-2">
        <textarea
          className="block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
          rows={3}
          placeholder="Add a comment..."
          value={newBody}
          onChange={(e) => setNewBody(e.target.value)}
        />
        <Button
          size="sm"
          onClick={() => {
            createMutation.mutate({ body: newBody }, {
              onSuccess: () => setNewBody(''),
            })
          }}
          disabled={!newBody.trim() || createMutation.isPending}
        >
          {createMutation.isPending ? 'Adding...' : 'Add Comment'}
        </Button>
      </div>
    </div>
  )
}
