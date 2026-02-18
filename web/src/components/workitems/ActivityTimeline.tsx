import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { useEvents } from '@/hooks/useWorkItems'
import { useMembers } from '@/hooks/useProjects'
import { Spinner } from '@/components/ui/Spinner'
import type { WorkItemEvent } from '@/api/workitems'
import type { ProjectMember } from '@/api/projects'

/** Fields whose values are user UUIDs that should be resolved to display names */
const userFields = new Set(['assignee_id', 'reporter_id'])

interface ActivityTimelineProps {
  projectKey: string
  itemNumber: number
  sortOrder?: 'asc' | 'desc'
  onAttachmentClick?: (attachmentId: string) => void
  onCommentClick?: (commentId: string) => void
}

function fieldLabel(name: string, t: TFunction): string {
  const key = `activity.fields.${name}`
  const translated = t(key, { defaultValue: '' })
  return translated || name.replace(/_/g, ' ')
}

function formatEventLabel(event: WorkItemEvent, t: TFunction): string {
  if (event.event_type === 'created') return t('activity.createdItem')
  if (event.event_type === 'comment_added') return t('activity.addedComment')
  if (event.event_type === 'comment_updated') return t('activity.editedComment')
  if (event.event_type === 'comment_deleted') return t('activity.deletedComment')
  if (event.field_name) {
    const field = fieldLabel(event.field_name, t)
    if (event.old_value && event.new_value) {
      return t('activity.changed', { field })
    }
    if (event.new_value) return t('activity.set', { field })
    if (event.old_value) return t('activity.cleared', { field })
  }
  return event.event_type.replace(/_/g, ' ')
}

export function ActivityTimeline({ projectKey, itemNumber, sortOrder = 'desc', onAttachmentClick, onCommentClick }: ActivityTimelineProps) {
  const { t } = useTranslation()
  const { data: events, isLoading } = useEvents(projectKey, itemNumber)
  const { data: members } = useMembers(projectKey)

  if (isLoading) return <Spinner size="sm" />

  if (!events?.length) {
    return <p className="text-sm text-gray-400 dark:text-gray-500">{t('activity.noActivity')}</p>
  }

  const sorted = sortOrder === 'desc' ? [...events].reverse() : events

  return (
    <div className="border-l-2 border-gray-200 dark:border-gray-700 pl-4 space-y-4">
      {sorted.map((event) => (
        <div key={event.id} className="relative">
          <div className="absolute -left-[21px] top-1.5 w-2.5 h-2.5 rounded-full bg-gray-300 dark:bg-gray-600 border-2 border-white dark:border-gray-900" />
          <div className="text-sm">
            <span className="font-medium text-gray-700 dark:text-gray-300">{event.actor?.display_name ?? t('common.system')}</span>
            {' '}
            <span className="text-gray-500 dark:text-gray-400">{formatEventLabel(event, t)}</span>
            <CommentLink event={event} onClick={onCommentClick} t={t} />
            <AttachmentLink event={event} onClick={onAttachmentClick} />
          </div>
          <FieldChangeDiff event={event} members={members} />
          <CommentPreview event={event} t={t} />
          <span className="text-xs text-gray-400 dark:text-gray-500">{new Date(event.created_at).toLocaleString()}</span>
        </div>
      ))}
    </div>
  )
}

function truncate(value: string, max: number = 120): string {
  return value.length > max ? value.slice(0, max) + '\u2026' : value
}

function resolveValue(fieldName: string, value: string, members?: ProjectMember[]): string {
  if (userFields.has(fieldName) && members) {
    const member = members.find((m) => m.user_id === value)
    if (member) return member.display_name
  }
  return value
}

function FieldChangeDiff({ event, members }: { event: WorkItemEvent; members?: ProjectMember[] }) {
  if (!event.field_name) return null
  if (!event.old_value && !event.new_value) return null

  return (
    <div className="mt-1 mb-1 rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-xs font-mono overflow-hidden">
      {event.old_value && (
        <div className="px-3 py-1.5 bg-red-50 dark:bg-red-900/30 text-red-800 dark:text-red-300 border-b border-gray-200 dark:border-gray-700">
          <span className="select-none text-red-400 mr-2">&minus;</span>
          {truncate(resolveValue(event.field_name, event.old_value, members))}
        </div>
      )}
      {event.new_value && (
        <div className="px-3 py-1.5 bg-green-50 dark:bg-green-900/30 text-green-800 dark:text-green-300">
          <span className="select-none text-green-400 mr-2">+</span>
          {truncate(resolveValue(event.field_name, event.new_value, members))}
        </div>
      )}
    </div>
  )
}

// --- Simple line-level diff (LCS-based) ---

type DiffLine = { type: 'same' | 'add' | 'remove'; text: string }

function computeDiff(oldText: string, newText: string): DiffLine[] {
  const oldLines = oldText.split('\n')
  const newLines = newText.split('\n')
  const m = oldLines.length
  const n = newLines.length

  // Build LCS table
  const dp: number[][] = Array.from({ length: m + 1 }, () => Array(n + 1).fill(0))
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[i][j] = oldLines[i - 1] === newLines[j - 1]
        ? dp[i - 1][j - 1] + 1
        : Math.max(dp[i - 1][j], dp[i][j - 1])
    }
  }

  // Backtrack to produce diff
  const result: DiffLine[] = []
  let i = m, j = n
  while (i > 0 || j > 0) {
    if (i > 0 && j > 0 && oldLines[i - 1] === newLines[j - 1]) {
      result.push({ type: 'same', text: oldLines[i - 1] })
      i--; j--
    } else if (j > 0 && (i === 0 || dp[i][j - 1] >= dp[i - 1][j])) {
      result.push({ type: 'add', text: newLines[j - 1] })
      j--
    } else {
      result.push({ type: 'remove', text: oldLines[i - 1] })
      i--
    }
  }
  return result.reverse()
}

/** Get first N lines of a string */
function firstLines(text: string, n: number): string {
  const lines = text.split('\n')
  if (lines.length <= n) return text
  return lines.slice(0, n).join('\n')
}

const COLLAPSED_LINES = 4

function CommentPreview({ event, t }: { event: WorkItemEvent; t: TFunction }) {
  const [expanded, setExpanded] = useState(false)

  const isCommentEvent = event.event_type === 'comment_added' || event.event_type === 'comment_updated'
  if (!isCommentEvent) return null

  const preview = (event.metadata?.preview as string) ?? null
  const oldPreview = (event.metadata?.old_preview as string) ?? null

  if (!preview && !oldPreview) return null

  // For comment_updated, show a proper line-level diff
  if (event.event_type === 'comment_updated' && oldPreview && preview) {
    const allLines = computeDiff(oldPreview, preview)
    // Only show changed lines (and context)
    const changedLines = allLines.filter((l) => l.type !== 'same')

    if (changedLines.length === 0) return null

    const displayLines = expanded ? changedLines : changedLines.slice(0, COLLAPSED_LINES)
    const hasMore = changedLines.length > COLLAPSED_LINES

    return (
      <div className="mt-1 mb-1 rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-xs font-mono overflow-hidden">
        {displayLines.map((line, idx) => (
          <div
            key={idx}
            className={
              line.type === 'remove'
                ? 'px-3 py-0.5 bg-red-50 dark:bg-red-900/30 text-red-800 dark:text-red-300'
                : 'px-3 py-0.5 bg-green-50 dark:bg-green-900/30 text-green-800 dark:text-green-300'
            }
          >
            <span className={`select-none mr-2 ${line.type === 'remove' ? 'text-red-400' : 'text-green-400'}`}>
              {line.type === 'remove' ? '\u2212' : '+'}
            </span>
            {line.text || '\u00A0'}
          </div>
        ))}
        {hasMore && (
          <button
            className="w-full px-3 py-1 text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 text-left"
            onClick={() => setExpanded(!expanded)}
          >
            {expanded ? t('common.showLess') : t('common.showMoreLines', { count: changedLines.length - COLLAPSED_LINES })}
          </button>
        )}
      </div>
    )
  }

  // For comment_added, show the preview (first 2 lines)
  if (!preview) return null

  const needsExpand = preview.split('\n').length > 2
  const displayText = expanded ? preview : firstLines(preview, 2)

  return (
    <div className="mt-1 mb-1 rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-xs overflow-hidden">
      <div className="px-3 py-1.5 text-gray-600 dark:text-gray-400 whitespace-pre-wrap">{displayText}</div>
      {needsExpand && (
        <button
          className="w-full px-3 py-1 text-xs text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 text-left"
          onClick={() => setExpanded(!expanded)}
        >
          {expanded ? t('common.showLess') : t('common.showMore')}
        </button>
      )}
    </div>
  )
}

function CommentLink({ event, onClick, t }: { event: WorkItemEvent; onClick?: (commentId: string) => void; t: TFunction }) {
  const isCommentEvent = event.event_type === 'comment_added' || event.event_type === 'comment_updated' || event.event_type === 'comment_deleted'
  if (!isCommentEvent) return null
  const commentId = event.metadata?.comment_id as string | undefined

  if (commentId && onClick && event.event_type !== 'comment_deleted') {
    return (
      <>
        {' '}
        <button
          className="text-indigo-600 dark:text-indigo-400 hover:underline"
          onClick={() => onClick(commentId)}
        >
          {t('activity.aComment')}
        </button>
      </>
    )
  }

  return <span className="text-gray-500 dark:text-gray-400"> {t('activity.aComment')}</span>
}

function AttachmentLink({ event, onClick }: { event: WorkItemEvent; onClick?: (attachmentId: string) => void }) {
  if (event.event_type !== 'attachment_added' && event.event_type !== 'attachment_removed') return null
  const attachmentId = event.metadata?.attachment_id as string | undefined
  const filename = event.metadata?.filename as string | undefined
  if (!filename) return null

  if (event.event_type === 'attachment_added' && attachmentId && onClick) {
    return (
      <>
        {' '}
        <button
          className="text-indigo-600 dark:text-indigo-400 hover:underline"
          onClick={() => onClick(attachmentId)}
        >
          {filename}
        </button>
      </>
    )
  }

  return <span className="text-gray-600 dark:text-gray-300"> {filename}</span>
}
