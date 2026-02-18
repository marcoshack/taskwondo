import { useState, useRef, useEffect } from 'react'
import { useComments, useCreateComment, useUpdateComment, useDeleteComment } from '@/hooks/useWorkItems'
import { useAuth } from '@/contexts/AuthContext'
import { useMembers } from '@/hooks/useProjects'
import { usePasteUpload } from '@/hooks/usePasteUpload'
import { Button } from '@/components/ui/Button'
import { Avatar } from '@/components/ui/Avatar'
import { Spinner } from '@/components/ui/Spinner'
import { CopyButton } from '@/components/ui/CopyButton'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { markdownComponents } from '@/components/ui/markdownComponents'

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

  const addCommentRef = useRef<HTMLDivElement>(null)
  const [addCommentVisible, setAddCommentVisible] = useState(true)

  useEffect(() => {
    const el = addCommentRef.current
    if (!el) return
    const observer = new IntersectionObserver(
      ([entry]) => setAddCommentVisible(entry.isIntersecting),
      { threshold: 0.1 },
    )
    observer.observe(el)
    return () => observer.disconnect()
  }, [])

  const { handlePaste, handleDrop, handleDragOver } = usePasteUpload({
    projectKey,
    itemNumber,
    onTextChange: (updater) => setNewBody(updater),
  })

  function authorName(authorId: string | null): string {
    if (!authorId) return 'Unknown'
    const member = members?.find((m) => m.user_id === authorId)
    return member?.display_name ?? 'Unknown'
  }

  if (isLoading) return <Spinner size="sm" />

  return (
    <div className="space-y-4">
      <div ref={addCommentRef} className="space-y-2 pb-3 border-b border-gray-100 dark:border-gray-700">
        <textarea
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm"
          rows={3}
          placeholder="Add a comment... (paste or drag files to attach)"
          value={newBody}
          onChange={(e) => setNewBody(e.target.value)}
          onPaste={handlePaste}
          onDrop={handleDrop}
          onDragOver={handleDragOver}
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

      {(sortOrder === 'desc' ? [...(comments ?? [])].reverse() : (comments ?? [])).map((c) => (
        <div key={c.id} className="group/comment border-b border-gray-100 dark:border-gray-700 pb-3">
          {!addCommentVisible && (
            <div className="flex justify-end mb-1">
              <button
                className="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 flex items-center gap-1"
                onClick={() => addCommentRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })}
              >
                &uarr; Top
              </button>
            </div>
          )}
          {editingId === c.id ? (
            <div className="space-y-2">
              <textarea
                className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm"
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
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{authorName(c.author_id)}</span>
                <span className="text-xs text-gray-400 dark:text-gray-500">{new Date(c.created_at).toLocaleString()}</span>
                <CopyButton text={c.body} className="opacity-0 group-hover/comment:opacity-100" />
              </div>
              <div className="prose prose-sm dark:prose-invert max-w-none text-gray-900 dark:text-gray-100 pl-8">
                <Markdown remarkPlugins={[remarkGfm]} components={markdownComponents}>{c.body}</Markdown>
              </div>
              <div className="flex items-center gap-3 mt-1 pl-8">
                {c.visibility !== 'internal' && (
                  <span className="text-xs text-indigo-500">{c.visibility}</span>
                )}
                {user && c.author_id === user.id && (
                  <>
                    <button
                      className="text-xs text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
                      onClick={() => { setEditingId(c.id); setEditBody(c.body) }}
                    >
                      Edit
                    </button>
                    <button
                      className="text-xs text-red-400 hover:text-red-600 dark:hover:text-red-300"
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
    </div>
  )
}
