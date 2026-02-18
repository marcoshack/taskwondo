import { useState, useRef, useEffect, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
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
  highlightedCommentId?: string | null
  onHighlightClear?: () => void
}

export function CommentList({ projectKey, itemNumber, sortOrder = 'desc', highlightedCommentId, onHighlightClear }: CommentListProps) {
  const { t } = useTranslation()
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

  const highlightNodeRef = useRef<HTMLDivElement | null>(null)

  const highlightRef = useCallback((node: HTMLDivElement | null) => {
    highlightNodeRef.current = node
  }, [highlightedCommentId]) // eslint-disable-line react-hooks/exhaustive-deps

  useEffect(() => {
    const node = highlightNodeRef.current
    if (!node || !highlightedCommentId) return

    node.scrollIntoView({ behavior: 'smooth', block: 'center' })

    // Re-scroll when images in sibling comments load and shift layout.
    // Uses capture phase because load events don't bubble.
    const list = node.parentElement
    if (!list) return

    const rescroll = () => node.scrollIntoView({ behavior: 'smooth', block: 'center' })
    list.addEventListener('load', rescroll, true)

    const highlightTimer = setTimeout(() => onHighlightClear?.(), 2000)
    const cleanupTimer = setTimeout(() => list.removeEventListener('load', rescroll, true), 5000)

    return () => {
      list.removeEventListener('load', rescroll, true)
      clearTimeout(highlightTimer)
      clearTimeout(cleanupTimer)
    }
  }, [highlightedCommentId, onHighlightClear])

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
    if (!authorId) return t('common.unknown')
    const member = members?.find((m) => m.user_id === authorId)
    return member?.display_name ?? t('common.unknown')
  }

  if (isLoading) return <Spinner size="sm" />

  return (
    <div className="space-y-4">
      <div ref={addCommentRef} className="space-y-2 pb-3 border-b border-gray-100 dark:border-gray-700">
        <textarea
          className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm"
          rows={3}
          placeholder={t('comments.placeholder')}
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
          {createMutation.isPending ? t('comments.adding') : t('comments.add')}
        </Button>
      </div>

      {(sortOrder === 'desc' ? [...(comments ?? [])].reverse() : (comments ?? [])).map((c) => (
        <div
          key={c.id}
          ref={c.id === highlightedCommentId ? highlightRef : undefined}
          className={`group/comment border-b border-gray-100 dark:border-gray-700 pb-3 rounded-md transition-colors duration-700 ${
            c.id === highlightedCommentId ? 'bg-indigo-50 dark:bg-indigo-900/30 ring-1 ring-indigo-300 dark:ring-indigo-600 px-2 py-2 -mx-2' : ''
          }`}
        >
          {!addCommentVisible && (
            <div className="flex justify-end mb-1">
              <button
                className="text-xs text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 flex items-center gap-1"
                onClick={() => addCommentRef.current?.scrollIntoView({ behavior: 'smooth', block: 'start' })}
              >
                &uarr; {t('common.top')}
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
                  {t('common.save')}
                </Button>
                <Button size="sm" variant="ghost" onClick={() => setEditingId(null)}>{t('common.cancel')}</Button>
              </div>
            </div>
          ) : (
            <>
              <div className="flex items-center gap-2 mb-1">
                <Avatar name={authorName(c.author_id)} size="xs" />
                <span className="text-sm font-medium text-gray-700 dark:text-gray-300">{authorName(c.author_id)}</span>
                <span className="text-xs text-gray-400 dark:text-gray-500">
                  {new Date(c.created_at).toLocaleString()}
                  {c.edit_count > 0 && <span className="ml-1 italic">{t('comments.editCount', { count: c.edit_count })}</span>}
                </span>
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
                      {t('common.edit')}
                    </button>
                    <button
                      className="text-xs text-red-400 hover:text-red-600 dark:hover:text-red-300"
                      onClick={() => { if (confirm(t('comments.deleteConfirm'))) deleteMutation.mutate(c.id) }}
                    >
                      {t('common.delete')}
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
