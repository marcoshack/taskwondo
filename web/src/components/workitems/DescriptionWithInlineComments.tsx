import { useState, useMemo, useCallback, useRef, useEffect, useLayoutEffect } from 'react'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useTranslation } from 'react-i18next'
import { getMarkdownComponents } from '@/components/ui/markdownComponents'
import { useComments, useCreateInlineComment } from '@/hooks/useWorkItems'
import { useMembers } from '@/hooks/useProjects'
import { rehypeSourcePos, resolveSelection, findAnchorRange, findLineElement, type AnchorDraft } from '@/lib/inlineAnchor'
import { InlineCommentComposer } from './InlineCommentComposer'
import { InlineCommentThread } from './InlineCommentThread'
import type { Comment } from '@/api/workitems'

interface DescriptionWithInlineCommentsProps {
  projectKey: string
  itemNumber: number
  description: string
  readOnly?: boolean
  onImageClick?: (src: string) => void
  onAttachmentLinkClick?: (href: string, attachmentId: string) => void
  /** Root comment id whose thread is open; controlled by the parent so the
   *  comments feed's "View" link can drive it too. */
  openThreadRootId?: string | null
  onOpenThread?: (rootId: string | null) => void
}

interface GutterMarker {
  rootId: string
  top: number
  count: number
}

interface FloatingPos {
  top: number
  left: number
}

const HIGHLIGHT_NAME = 'inline-comment-anchor'

export function DescriptionWithInlineComments({
  projectKey,
  itemNumber,
  description,
  readOnly = false,
  onImageClick,
  onAttachmentLinkClick,
  openThreadRootId = null,
  onOpenThread,
}: DescriptionWithInlineCommentsProps) {
  const { t } = useTranslation()
  const wrapperRef = useRef<HTMLDivElement>(null)
  const proseRef = useRef<HTMLDivElement>(null)

  const { data: comments } = useComments(projectKey, itemNumber)
  const { data: members } = useMembers(projectKey)
  const createInline = useCreateInlineComment(projectKey, itemNumber)

  // Pending text selection (the floating "Comment" button) and the composer.
  const [pending, setPending] = useState<{ draft: AnchorDraft; pos: FloatingPos } | null>(null)
  const [composer, setComposer] = useState<{ draft: AnchorDraft; pos: FloatingPos } | null>(null)
  const [gutter, setGutter] = useState<GutterMarker[]>([])
  // Where the inline-thread overlay is pinned (below the anchored line).
  const [threadPos, setThreadPos] = useState<FloatingPos | null>(null)

  const baseComponents = useMemo(
    () => getMarkdownComponents({ onImageClick, onAttachmentLinkClick }),
    [onImageClick, onAttachmentLinkClick],
  )

  // Root inline comments (anchored, no parent) and a parent → replies index.
  const { roots, repliesByRoot } = useMemo(() => {
    const rootList: Comment[] = []
    const replies = new Map<string, Comment[]>()
    for (const c of comments ?? []) {
      if (c.parent_comment_id) {
        const list = replies.get(c.parent_comment_id) ?? []
        list.push(c)
        replies.set(c.parent_comment_id, list)
      } else if (c.anchor) {
        rootList.push(c)
      }
    }
    return { roots: rootList, repliesByRoot: replies }
  }, [comments])

  const openRoot = useMemo(
    () => roots.find((c) => c.id === openThreadRootId) ?? null,
    [roots, openThreadRootId],
  )

  // All anchored comments in reading order — by line, then column, then age.
  // Drives the prev/next navigation in the thread overlay, so comments that
  // share a line (same gutter marker) are all still reachable.
  const orderedRoots = useMemo(
    () =>
      [...roots].sort((a, b) => {
        const aa = a.anchor!
        const ba = b.anchor!
        if (aa.start_line !== ba.start_line) return aa.start_line - ba.start_line
        if (aa.start_col !== ba.start_col) return aa.start_col - ba.start_col
        return a.created_at.localeCompare(b.created_at)
      }),
    [roots],
  )
  const openIndex = useMemo(
    () => orderedRoots.findIndex((c) => c.id === openThreadRootId),
    [orderedRoots, openThreadRootId],
  )
  const navigateThread = useCallback(
    (delta: number) => {
      if (orderedRoots.length === 0) return
      const base = openIndex < 0 ? 0 : openIndex
      const next = (base + delta + orderedRoots.length) % orderedRoots.length
      onOpenThread?.(orderedRoots[next].id)
    },
    [orderedRoots, openIndex, onOpenThread],
  )

  // --- Gutter marker positioning -------------------------------------------
  const recomputeGutter = useCallback(() => {
    const prose = proseRef.current
    const wrapper = wrapperRef.current
    if (!prose || !wrapper) return
    const wrapperTop = wrapper.getBoundingClientRect().top
    // Group roots by source line so multiple comments on one line share a marker.
    const byLine = new Map<number, GutterMarker>()
    for (const c of roots) {
      if (!c.anchor || c.anchor.status !== 'active') continue
      const el = findLineElement(prose, c.anchor.start_line)
      if (!el) continue
      const top = el.getBoundingClientRect().top - wrapperTop
      const existing = byLine.get(c.anchor.start_line)
      if (existing) {
        existing.count += 1
      } else {
        byLine.set(c.anchor.start_line, { rootId: c.id, top, count: 1 })
      }
    }
    setGutter(Array.from(byLine.values()))
  }, [roots])

  useLayoutEffect(() => {
    recomputeGutter()
  }, [recomputeGutter, description])

  useEffect(() => {
    const prose = proseRef.current
    if (!prose) return
    const ro = new ResizeObserver(() => recomputeGutter())
    ro.observe(prose)
    window.addEventListener('resize', recomputeGutter)
    return () => {
      ro.disconnect()
      window.removeEventListener('resize', recomputeGutter)
    }
  }, [recomputeGutter])

  // --- Text selection → floating "Comment" button --------------------------
  const floatingPosFromSelection = useCallback((): FloatingPos | null => {
    const wrapper = wrapperRef.current
    const sel = window.getSelection()
    if (!wrapper || !sel || sel.rangeCount === 0) return null
    const rects = sel.getRangeAt(0).getClientRects()
    if (rects.length === 0) return null
    const last = rects[rects.length - 1]
    const wRect = wrapper.getBoundingClientRect()
    return { top: last.bottom - wRect.top + 6, left: Math.max(0, last.left - wRect.left) }
  }, [])

  useEffect(() => {
    if (readOnly) return
    const handler = () => {
      // While the composer is open the selection is frozen into it.
      if (composer) return
      const prose = proseRef.current
      if (!prose) return
      const draft = resolveSelection(prose, description)
      const pos = draft ? floatingPosFromSelection() : null
      setPending(draft && pos ? { draft, pos } : null)
    }
    document.addEventListener('selectionchange', handler)
    return () => document.removeEventListener('selectionchange', handler)
  }, [readOnly, composer, description, floatingPosFromSelection])

  // --- Anchor highlight while a thread is open -----------------------------
  useEffect(() => {
    const highlights = (CSS as unknown as { highlights?: Map<string, unknown> }).highlights
    const HighlightCtor = (window as unknown as { Highlight?: new (r: Range) => unknown }).Highlight
    if (!highlights || !HighlightCtor) return
    const prose = proseRef.current
    if (openRoot && prose && openRoot.anchor && openRoot.anchor.status === 'active') {
      const range = findAnchorRange(
        prose,
        openRoot.anchor.start_line,
        openRoot.anchor.start_col,
        openRoot.anchor.end_line,
        openRoot.anchor.end_col,
      )
      if (range) {
        highlights.set(HIGHLIGHT_NAME, new HighlightCtor(range))
        return () => {
          highlights.delete(HIGHLIGHT_NAME)
        }
      }
    }
    highlights.delete(HIGHLIGHT_NAME)
  }, [openRoot, description, comments])

  // --- Inline-thread overlay positioning -----------------------------------
  // The thread floats below the anchored line as a fixed-width overlay so it
  // never resizes the description body.
  useLayoutEffect(() => {
    if (!openRoot?.anchor) {
      setThreadPos(null)
      return
    }
    const prose = proseRef.current
    const wrapper = wrapperRef.current
    if (!prose || !wrapper) return
    const a = openRoot.anchor
    const wRect = wrapper.getBoundingClientRect()
    let rect: DOMRect | null = null
    const range =
      a.status === 'active'
        ? findAnchorRange(prose, a.start_line, a.start_col, a.end_line, a.end_col)
        : null
    if (range) {
      const rects = range.getClientRects()
      if (rects.length) rect = rects[rects.length - 1]
    }
    if (!rect) {
      const el = findLineElement(prose, a.end_line) ?? findLineElement(prose, a.start_line)
      if (el) rect = el.getBoundingClientRect()
    }
    // Left-align to the text column; vertical position tracks the line.
    setThreadPos({ top: rect ? rect.bottom - wRect.top + 6 : 0, left: 30 })
  }, [openRoot, description, comments])

  const threadRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (openRoot && threadRef.current) {
      threadRef.current.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
    }
  }, [openRoot, threadPos])

  const handleSubmit = useCallback(
    (body: string) => {
      if (!composer) return
      const { draft } = composer
      createInline.mutate(
        {
          body,
          anchor: {
            start_line: draft.startLine,
            start_col: draft.startCol,
            end_line: draft.endLine,
            end_col: draft.endCol,
            snippet: draft.snippet,
          },
        },
        {
          onSuccess: (created) => {
            setComposer(null)
            setPending(null)
            onOpenThread?.(created.id)
          },
        },
      )
    },
    [composer, createInline, onOpenThread],
  )

  return (
    // -ml-2 cancels the description box's left padding so the gutter sits
    // flush against the box edge.
    <div ref={wrapperRef} className="relative -ml-2">
      {/* Left gutter with a comment marker per anchored line. */}
      <div className="absolute left-0 top-0 bottom-0 w-6" aria-hidden={gutter.length === 0}>
        {gutter.map((m) => (
          <button
            key={m.rootId}
            type="button"
            data-testid="inline-comment-gutter-icon"
            title={t('inlineComments.gutterTooltip', { count: m.count })}
            className="absolute left-0 inline-flex items-center justify-center w-6 h-6 rounded-md text-indigo-500 hover:text-indigo-700 hover:bg-indigo-100 dark:hover:bg-indigo-900/40 transition-colors"
            style={{ top: m.top }}
            onClick={() => onOpenThread?.(openThreadRootId === m.rootId ? null : m.rootId)}
          >
            <svg className="w-4 h-4" viewBox="0 0 20 20" fill="currentColor">
              <path d="M2 4a2 2 0 012-2h12a2 2 0 012 2v8a2 2 0 01-2 2H7l-4 4v-4H4a2 2 0 01-2-2V4z" />
            </svg>
            {m.count > 1 && (
              <span
                data-testid="inline-comment-gutter-count"
                className="absolute -top-1 -right-1 inline-flex items-center justify-center min-w-[14px] h-[14px] px-0.5 rounded-full bg-indigo-600 text-white text-[9px] font-semibold leading-none"
              >
                {m.count}
              </span>
            )}
          </button>
        ))}
      </div>

      <div
        ref={proseRef}
        data-testid="description-body"
        className="prose prose-sm dark:prose-invert max-w-none text-gray-700 dark:text-gray-300 break-words pl-[30px]"
      >
        <Markdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeSourcePos]} components={baseComponents}>
          {description}
        </Markdown>
      </div>

      {/* Floating "Comment" button shown below an active text selection. */}
      {pending && !composer && (
        <button
          type="button"
          data-testid="inline-comment-add-button"
          // preventDefault on mousedown keeps the selection alive through the click.
          onMouseDown={(e) => e.preventDefault()}
          onClick={() => {
            setComposer(pending)
            setPending(null)
          }}
          className="absolute z-20 inline-flex items-center gap-1 px-2 py-1 rounded-md bg-indigo-600 hover:bg-indigo-700 text-white text-xs shadow-lg"
          style={{ top: pending.pos.top, left: pending.pos.left }}
        >
          <svg className="w-3.5 h-3.5" viewBox="0 0 20 20" fill="currentColor">
            <path d="M2 4a2 2 0 012-2h12a2 2 0 012 2v8a2 2 0 01-2 2H7l-4 4v-4H4a2 2 0 01-2-2V4z" />
          </svg>
          {t('inlineComments.commentButton')}
        </button>
      )}

      {/* Floating composer for a new inline comment. */}
      {composer && (
        <div className="absolute z-30" style={{ top: composer.pos.top, left: composer.pos.left }}>
          <InlineCommentComposer
            selectedText={composer.draft.selectedText}
            submitting={createInline.isPending}
            onSubmit={handleSubmit}
            onCancel={() => setComposer(null)}
          />
        </div>
      )}

      {/* Inline conversation thread — a floating overlay so it never resizes
          the description body. */}
      {openRoot && threadPos && (
        <div
          ref={threadRef}
          className="absolute z-30 w-[26rem] max-w-[calc(100%-2.5rem)] max-h-[75vh] overflow-y-auto"
          style={{ top: threadPos.top, left: threadPos.left }}
        >
          <InlineCommentThread
            projectKey={projectKey}
            itemNumber={itemNumber}
            root={openRoot}
            replies={repliesByRoot.get(openRoot.id) ?? []}
            members={members ?? []}
            readOnly={readOnly}
            onClose={() => onOpenThread?.(null)}
            position={openIndex + 1}
            total={orderedRoots.length}
            onPrev={() => navigateThread(-1)}
            onNext={() => navigateThread(1)}
          />
        </div>
      )}
    </div>
  )
}
