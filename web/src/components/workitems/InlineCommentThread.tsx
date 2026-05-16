import { useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { getMarkdownComponents } from '@/components/ui/markdownComponents'
import { Avatar } from '@/components/ui/Avatar'
import { Button } from '@/components/ui/Button'
import { ScrollableDate } from '@/components/ui/ScrollableDate'
import { useCreateInlineComment } from '@/hooks/useWorkItems'
import type { Comment } from '@/api/workitems'

interface Member {
  user_id: string
  display_name: string
  avatar_url?: string
}

interface InlineCommentThreadProps {
  projectKey: string
  itemNumber: number
  /** The thread's root inline comment. */
  root: Comment
  /** Replies to the root, oldest first. */
  replies: Comment[]
  members: Member[]
  readOnly?: boolean
  onClose: () => void
}

/**
 * A conversation box anchored to a region of the description. Shows the root
 * inline comment, its replies, and — unless read-only — a reply box.
 */
export function InlineCommentThread({
  projectKey,
  itemNumber,
  root,
  replies,
  members,
  readOnly = false,
  onClose,
}: InlineCommentThreadProps) {
  const { t } = useTranslation()
  const [replyBody, setReplyBody] = useState('')
  const replyRef = useRef<HTMLTextAreaElement>(null)
  const createReply = useCreateInlineComment(projectKey, itemNumber)
  const mdComponents = useMemo(() => getMarkdownComponents(), [])

  const outdated = root.anchor?.status === 'outdated'

  function authorName(authorId: string | null): string {
    if (!authorId) return t('common.unknown')
    return members.find((m) => m.user_id === authorId)?.display_name ?? t('common.unknown')
  }
  function authorAvatar(authorId: string | null): string | undefined {
    if (!authorId) return undefined
    return members.find((m) => m.user_id === authorId)?.avatar_url
  }

  function submitReply() {
    if (!replyBody.trim()) return
    createReply.mutate(
      { body: replyBody, parent_comment_id: root.id, visibility: root.visibility },
      { onSuccess: () => setReplyBody('') },
    )
  }

  const thread = [root, ...replies]

  return (
    <div
      className="rounded-md border border-indigo-300 dark:border-indigo-600 bg-white dark:bg-gray-800 shadow-lg"
      data-testid="inline-comment-thread"
    >
      <div className="flex items-center gap-2 px-3 py-2 border-b border-indigo-200 dark:border-indigo-700">
        <span className="text-xs font-medium text-indigo-700 dark:text-indigo-300">
          {t('inlineComments.threadHeader')}
        </span>
        {outdated && (
          <span
            data-testid="inline-comment-outdated-badge"
            className="inline-flex items-center px-1.5 py-0.5 rounded bg-amber-100 dark:bg-amber-900/40 text-amber-700 dark:text-amber-300 text-[10px] uppercase tracking-wide"
          >
            {t('inlineComments.outdated')}
          </span>
        )}
        <button
          type="button"
          aria-label={t('common.close')}
          data-testid="inline-comment-thread-close"
          className="ml-auto text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
          onClick={onClose}
        >
          <svg className="w-4 h-4" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="2">
            <path strokeLinecap="round" d="M4 4l8 8M12 4l-8 8" />
          </svg>
        </button>
      </div>

      <div className="divide-y divide-indigo-100 dark:divide-indigo-800">
        {thread.map((c) => (
          <div key={c.id} className="px-3 py-2" data-testid="inline-comment-thread-item">
            <div className="flex items-center gap-2 mb-1">
              <Avatar name={authorName(c.author_id)} avatarUrl={authorAvatar(c.author_id)} size="xs" />
              <span className="text-sm font-medium text-gray-700 dark:text-gray-300">
                {authorName(c.author_id)}
              </span>
              <ScrollableDate date={c.created_at} />
            </div>
            <div className="prose prose-sm dark:prose-invert max-w-none text-gray-900 dark:text-gray-100 break-words pl-7">
              <Markdown remarkPlugins={[remarkGfm]} components={mdComponents}>
                {c.body}
              </Markdown>
            </div>
          </div>
        ))}
      </div>

      {!readOnly && (
        <div className="px-3 py-2 border-t border-indigo-200 dark:border-indigo-700">
          <textarea
            ref={replyRef}
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-900 dark:text-gray-100 px-3 py-2 text-sm"
            rows={2}
            placeholder={t('inlineComments.replyPlaceholder')}
            value={replyBody}
            onChange={(e) => setReplyBody(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && (e.ctrlKey || e.metaKey) && replyBody.trim()) {
                e.preventDefault()
                submitReply()
              }
            }}
          />
          <div className="mt-2">
            <Button
              size="sm"
              onClick={submitReply}
              disabled={!replyBody.trim() || createReply.isPending}
            >
              {createReply.isPending ? t('inlineComments.submitting') : t('inlineComments.reply')}
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
