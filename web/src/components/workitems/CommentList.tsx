import { useState, useRef, useEffect, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useComments, useCreateComment, useUpdateComment, useDeleteComment } from '@/hooks/useWorkItems'
import { useAuth } from '@/contexts/AuthContext'
import { useMembers } from '@/hooks/useProjects'
import { usePasteUpload } from '@/hooks/usePasteUpload'
import { useMentionAutocomplete } from '@/hooks/useMentionAutocomplete'
import { Button } from '@/components/ui/Button'
import { Modal } from '@/components/ui/Modal'
import { MentionModal } from '@/components/ui/MentionModal'
import { Avatar } from '@/components/ui/Avatar'
import { Spinner } from '@/components/ui/Spinner'
import { ScrollableDate } from '@/components/ui/ScrollableDate'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { getMarkdownComponents } from '@/components/ui/markdownComponents'

interface CommentListProps {
  projectKey: string
  itemNumber: number
  sortOrder?: 'asc' | 'desc'
  highlightedCommentId?: string | null
  onHighlightClear?: () => void
  onImageClick?: (src: string) => void
  onAttachmentLinkClick?: (href: string, attachmentId: string) => void
  draft?: string
  onDraftChange?: (value: string) => void
  readOnly?: boolean
}

export function CommentList({ projectKey, itemNumber, sortOrder = 'desc', highlightedCommentId, onHighlightClear, onImageClick, onAttachmentLinkClick, draft, onDraftChange, readOnly = false }: CommentListProps) {
  const { t } = useTranslation()
  const { user } = useAuth()
  const { data: comments, isLoading } = useComments(projectKey, itemNumber)
  const { data: members } = useMembers(projectKey)
  const createMutation = useCreateComment(projectKey, itemNumber)
  const updateMutation = useUpdateComment(projectKey, itemNumber)
  const deleteMutation = useDeleteComment(projectKey, itemNumber)

  // Use lifted state from parent if provided, otherwise fall back to local state
  const [localBody, setLocalBody] = useState('')
  const newBody = draft ?? localBody
  // Ref keeps the latest value so async callbacks (paste-upload) never read a stale draft.
  const draftRef = useRef(newBody)
  draftRef.current = newBody
  const setNewBody = useCallback((value: string | ((prev: string) => string)) => {
    if (onDraftChange) {
      if (typeof value === 'function') {
        onDraftChange(value(draftRef.current))
      } else {
        onDraftChange(value)
      }
    } else {
      setLocalBody(value)
    }
  }, [onDraftChange])
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editBody, setEditBody] = useState('')
  const [deletingId, setDeletingId] = useState<string | null>(null)

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
  }, [isLoading])

  const { handlePaste, handleDrop, handleDragOver } = usePasteUpload({
    projectKey,
    itemNumber,
    onTextChange: (updater) => setNewBody(updater),
  })

  const editBodyRef = useRef(editBody)
  editBodyRef.current = editBody
  const editPaste = usePasteUpload({
    projectKey,
    itemNumber,
    onTextChange: (updater) => setEditBody(updater(editBodyRef.current)),
  })

  const newCommentRef = useRef<HTMLTextAreaElement>(null)
  const newMention = useMentionAutocomplete({
    value: newBody,
    onValueChange: (v) => setNewBody(v),
    textareaRef: newCommentRef,
  })

  const editCommentRef = useRef<HTMLTextAreaElement>(null)
  const editMention = useMentionAutocomplete({
    value: editBody,
    onValueChange: setEditBody,
    textareaRef: editCommentRef,
  })

  const mdComponents = useMemo(() => getMarkdownComponents({ onImageClick, onAttachmentLinkClick }), [onImageClick, onAttachmentLinkClick])

  function authorName(authorId: string | null): string {
    if (!authorId) return t('common.unknown')
    const member = members?.find((m) => m.user_id === authorId)
    return member?.display_name ?? t('common.unknown')
  }

  function authorAvatarUrl(authorId: string | null): string | undefined {
    if (!authorId) return undefined
    return members?.find((m) => m.user_id === authorId)?.avatar_url
  }

  if (isLoading) return <Spinner size="sm" />

  return (
    <div className="space-y-4">
      {!readOnly && (
        <div ref={addCommentRef} className="space-y-2 pb-3 border-b border-gray-100 dark:border-gray-700">
          <textarea
            ref={newCommentRef}
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm"
            rows={3}
            placeholder={t('comments.placeholder')}
            value={newBody}
            onChange={(e) => setNewBody(e.target.value)}
            onKeyDown={(e) => {
              newMention.onMentionKeyDown(e)
              if (e.defaultPrevented) return
              if (e.key === 'Enter' && (e.ctrlKey || e.metaKey) && newBody.trim()) {
                e.preventDefault()
                createMutation.mutate({ body: newBody }, { onSuccess: () => setNewBody('') })
              }
            }}
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
      )}

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
                ref={editCommentRef}
                className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm"
                rows={3}
                value={editBody}
                onChange={(e) => setEditBody(e.target.value)}
                onKeyDown={(e) => {
                  editMention.onMentionKeyDown(e)
                  if (e.defaultPrevented) return
                  if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
                    e.preventDefault()
                    updateMutation.mutate({ commentId: c.id, body: editBody }, { onSuccess: () => setEditingId(null) })
                  }
                  if (e.key === 'Escape') setEditingId(null)
                }}
                onPaste={editPaste.handlePaste}
                onDrop={editPaste.handleDrop}
                onDragOver={editPaste.handleDragOver}
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
              <div className="flex items-center justify-between gap-2 mb-1">
                <div className="flex items-center gap-2 min-w-0">
                  <Avatar name={authorName(c.author_id)} avatarUrl={authorAvatarUrl(c.author_id)} size="xs" />
                  <span className="text-sm font-medium text-gray-700 dark:text-gray-300 shrink-0">{authorName(c.author_id)}</span>
                  <ScrollableDate date={c.created_at} />
                  {c.edit_count > 0 && <span className="text-xs text-gray-400 dark:text-gray-500 italic shrink-0">{t('comments.editCount', { count: c.edit_count })}</span>}
                  {c.visibility !== 'internal' && (
                    <span className="text-xs text-indigo-500 shrink-0">{c.visibility}</span>
                  )}
                </div>
                <div className="flex items-center shrink-0">
                  {user && c.author_id === user.id && (
                    <button
                      className="group/edit relative inline-flex items-center justify-center w-7 h-7 rounded-md text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:text-gray-500 dark:hover:text-gray-300 dark:hover:bg-gray-700 transition-colors sm:opacity-0 sm:group-hover/comment:opacity-100"
                      onClick={() => { setEditingId(c.id); setEditBody(c.body) }}
                    >
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                        <path strokeLinecap="round" strokeLinejoin="round" d="M11.5 2.5a1.5 1.5 0 012.121 2.121L6.5 11.743l-2.5.757.757-2.5L11.5 2.5z" />
                      </svg>
                      <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 px-2 py-1 text-xs text-white bg-gray-900 dark:bg-gray-700 rounded whitespace-nowrap opacity-0 group-hover/edit:opacity-100 transition-opacity">
                        {t('common.edit')}
                      </span>
                    </button>
                  )}
                  {user && c.author_id === user.id && (
                    <button
                      className="group/del relative inline-flex items-center justify-center w-7 h-7 rounded-md text-red-400 hover:text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:text-red-300 dark:hover:bg-red-900/30 transition-colors sm:opacity-0 sm:group-hover/comment:opacity-100"
                      onClick={() => setDeletingId(c.id)}
                    >
                      <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
                        <path strokeLinecap="round" strokeLinejoin="round" d="M3 4.5h10M6.5 4.5V3a1 1 0 011-1h1a1 1 0 011 1v1.5M5 4.5v8a1 1 0 001 1h4a1 1 0 001-1v-8" />
                      </svg>
                      <span className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-1.5 px-2 py-1 text-xs text-white bg-gray-900 dark:bg-gray-700 rounded whitespace-nowrap opacity-0 group-hover/del:opacity-100 transition-opacity">
                        {t('common.delete')}
                      </span>
                    </button>
                  )}
                </div>
              </div>
              <div
                className={`prose prose-sm dark:prose-invert max-w-none text-gray-900 dark:text-gray-100 pl-8 rounded break-words ${user && c.author_id === user.id ? 'hover:bg-gray-50 dark:hover:bg-gray-800' : ''}`}
                onDoubleClick={() => {
                  if (user && c.author_id === user.id) {
                    setEditingId(c.id)
                    setEditBody(c.body)
                  }
                }}
              >
                <Markdown remarkPlugins={[remarkGfm]} components={mdComponents}>{c.body}</Markdown>
              </div>
            </>
          )}
        </div>
      ))}

      <MentionModal
        open={newMention.mentionModalOpen}
        position={newMention.dropdownPosition}
        onClose={newMention.onMentionClose}
        onSelect={newMention.onMentionSelect}
        projectKey={projectKey}
      />
      <MentionModal
        open={editMention.mentionModalOpen}
        position={editMention.dropdownPosition}
        onClose={editMention.onMentionClose}
        onSelect={editMention.onMentionSelect}
        projectKey={projectKey}
      />

      <Modal open={!!deletingId} onClose={() => setDeletingId(null)} title={t('comments.deleteConfirm')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          {t('comments.deleteBody')}
        </p>
        <div className="flex justify-end gap-3">
          <Button variant="secondary" onClick={() => setDeletingId(null)}>{t('common.cancel')}</Button>
          <Button
            variant="danger"
            onClick={() => {
              if (deletingId) {
                deleteMutation.mutate(deletingId, {
                  onSuccess: () => setDeletingId(null),
                })
              }
            }}
            disabled={deleteMutation.isPending}
          >
            {deleteMutation.isPending ? t('common.deleting') : t('common.delete')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}
