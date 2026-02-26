import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'

export type DiffLine = { type: 'same' | 'add' | 'remove'; text: string }

type WordSpan = { text: string; highlighted: boolean }

export type DiffPreviewTarget =
  | { kind: 'field'; fieldLabel: string; oldValue?: string; newValue?: string; diffLines?: DiffLine[] }
  | { kind: 'comment'; lines: DiffLine[] }

interface DiffPreviewModalProps {
  target: DiffPreviewTarget | null
  onClose: () => void
}

export function DiffPreviewModal({ target, onClose }: DiffPreviewModalProps) {
  const { t } = useTranslation()

  if (!target) return null

  const title = target.kind === 'field'
    ? t('activity.diff.title', { field: target.fieldLabel })
    : t('activity.diff.commentTitle')

  return (
    <Modal open={!!target} onClose={onClose} size="full">
      <div className="flex flex-col h-full">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200 dark:border-gray-700 shrink-0">
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100">{title}</span>
          <button
            className="p-1.5 text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200 rounded hover:bg-gray-100 dark:hover:bg-gray-700"
            onClick={onClose}
          >
            <svg className="w-4 h-4" fill="none" viewBox="0 0 16 16" stroke="currentColor" strokeWidth="1.5">
              <path strokeLinecap="round" strokeLinejoin="round" d="M4 4l8 8M12 4l-8 8" />
            </svg>
          </button>
        </div>

        {/* Content */}
        <div className="flex-1 overflow-auto p-4">
          {target.kind === 'field' && <FieldDiffContent oldValue={target.oldValue} newValue={target.newValue} diffLines={target.diffLines} />}
          {target.kind === 'comment' && <CommentDiffContent lines={target.lines} />}
        </div>
      </div>
    </Modal>
  )
}

function FieldDiffContent({ oldValue, newValue, diffLines }: { oldValue?: string; newValue?: string; diffLines?: DiffLine[] }) {
  const { t } = useTranslation()

  // When we have computed diff lines, render a proper line-level diff with word highlights
  if (diffLines) {
    const groups = pairDiffLines(diffLines)
    return (
      <div className="max-w-4xl mx-auto rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono overflow-hidden">
        {groups.map((group, gi) => {
          if (group.length === 2) {
            const { oldSpans, newSpans } = computeWordDiff(group[0].text, group[1].text)
            return (
              <div key={gi}>
                <ModalDiffLine line={group[0]} wordSpans={oldSpans} />
                <ModalDiffLine line={group[1]} wordSpans={newSpans} />
              </div>
            )
          }
          return <ModalDiffLine key={gi} line={group[0]} />
        })}
      </div>
    )
  }

  // Fallback for single-line field changes (no diffLines provided)
  return (
    <div className="max-w-4xl mx-auto rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono overflow-hidden">
      {oldValue && (
        <>
          <div className="px-4 py-1.5 bg-gray-100 dark:bg-gray-700 text-xs font-sans font-medium text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-600">
            {t('activity.diff.removed')}
          </div>
          <div className="px-4 py-3 bg-red-50 dark:bg-red-900/30 text-red-800 dark:text-red-300 whitespace-pre-wrap break-words border-b border-gray-200 dark:border-gray-700">
            {oldValue}
          </div>
        </>
      )}
      {newValue && (
        <>
          <div className="px-4 py-1.5 bg-gray-100 dark:bg-gray-700 text-xs font-sans font-medium text-gray-500 dark:text-gray-400 border-b border-gray-200 dark:border-gray-600">
            {t('activity.diff.added')}
          </div>
          <div className="px-4 py-3 bg-green-50 dark:bg-green-900/30 text-green-800 dark:text-green-300 whitespace-pre-wrap break-words">
            {newValue}
          </div>
        </>
      )}
    </div>
  )
}

function CommentDiffContent({ lines }: { lines: DiffLine[] }) {
  const groups = pairDiffLines(lines)
  return (
    <div className="max-w-4xl mx-auto rounded-md border border-gray-200 dark:border-gray-700 bg-gray-50 dark:bg-gray-800 text-sm font-mono overflow-hidden">
      {groups.map((group, gi) => {
        if (group.length === 2) {
          const { oldSpans, newSpans } = computeWordDiff(group[0].text, group[1].text)
          return (
            <div key={gi}>
              <ModalDiffLine line={group[0]} wordSpans={oldSpans} />
              <ModalDiffLine line={group[1]} wordSpans={newSpans} />
            </div>
          )
        }
        return <ModalDiffLine key={gi} line={group[0]} />
      })}
    </div>
  )
}

// --- Shared diff utilities for the modal ---

function computeWordDiff(oldLine: string, newLine: string): { oldSpans: WordSpan[]; newSpans: WordSpan[] } {
  const oldWords = oldLine.split(/(\s+)/)
  const newWords = newLine.split(/(\s+)/)
  const m = oldWords.length
  const n = newWords.length

  const dp: number[][] = Array.from({ length: m + 1 }, () => Array(n + 1).fill(0))
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[i][j] = oldWords[i - 1] === newWords[j - 1]
        ? dp[i - 1][j - 1] + 1
        : Math.max(dp[i - 1][j], dp[i][j - 1])
    }
  }

  const oldTags: boolean[] = Array(m).fill(true)
  const newTags: boolean[] = Array(n).fill(true)
  let i = m, j = n
  while (i > 0 && j > 0) {
    if (oldWords[i - 1] === newWords[j - 1]) {
      oldTags[i - 1] = false
      newTags[j - 1] = false
      i--; j--
    } else if (dp[i][j - 1] >= dp[i - 1][j]) {
      j--
    } else {
      i--
    }
  }

  function merge(words: string[], tags: boolean[]): WordSpan[] {
    const spans: WordSpan[] = []
    for (let k = 0; k < words.length; k++) {
      if (spans.length > 0 && spans[spans.length - 1].highlighted === tags[k]) {
        spans[spans.length - 1].text += words[k]
      } else {
        spans.push({ text: words[k], highlighted: tags[k] })
      }
    }
    return spans
  }

  return { oldSpans: merge(oldWords, oldTags), newSpans: merge(newWords, newTags) }
}

function pairDiffLines(lines: DiffLine[]): DiffLine[][] {
  const groups: DiffLine[][] = []
  let i = 0
  while (i < lines.length) {
    if (lines[i].type === 'remove') {
      const removes: DiffLine[] = []
      while (i < lines.length && lines[i].type === 'remove') { removes.push(lines[i]); i++ }
      const adds: DiffLine[] = []
      while (i < lines.length && lines[i].type === 'add') { adds.push(lines[i]); i++ }
      const paired = Math.min(removes.length, adds.length)
      for (let k = 0; k < paired; k++) groups.push([removes[k], adds[k]])
      for (let k = paired; k < removes.length; k++) groups.push([removes[k]])
      for (let k = paired; k < adds.length; k++) groups.push([adds[k]])
    } else {
      groups.push([lines[i]])
      i++
    }
  }
  return groups
}

function ModalDiffLine({ line, wordSpans }: { line: DiffLine; wordSpans?: WordSpan[] }) {
  const isRemove = line.type === 'remove'
  const isAdd = line.type === 'add'

  const bg = isRemove
    ? 'px-4 py-0.5 bg-red-50 dark:bg-red-900/30 text-red-800 dark:text-red-300'
    : isAdd
      ? 'px-4 py-0.5 bg-green-50 dark:bg-green-900/30 text-green-800 dark:text-green-300'
      : 'px-4 py-0.5 text-gray-600 dark:text-gray-400'

  const symbolColor = isRemove ? 'text-red-400' : isAdd ? 'text-green-400' : 'text-gray-400'
  const symbol = isRemove ? '\u2212' : isAdd ? '+' : ' '

  return (
    <div className={bg}>
      <span className={`select-none mr-2 ${symbolColor}`}>{symbol}</span>
      {wordSpans ? (
        wordSpans.map((span, si) =>
          span.highlighted ? (
            <span key={si} className={isRemove ? 'bg-red-200 dark:bg-red-800/60 rounded-sm' : 'bg-green-200 dark:bg-green-800/60 rounded-sm'}>
              {span.text}
            </span>
          ) : (
            <span key={si}>{span.text}</span>
          )
        )
      ) : (
        line.text || '\u00A0'
      )}
    </div>
  )
}
