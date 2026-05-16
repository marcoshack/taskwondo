/**
 * Source-position machinery for inline description comments (TF-350).
 *
 * `rehypeSourcePos` stamps every rendered text run with the markdown source
 * line/column range it came from. The resolver functions then map a DOM
 * selection back to 1-based source coordinates, and — for the reverse — turn a
 * stored anchor back into a DOM Range so the anchored text can be highlighted.
 */

/** A text selection resolved to 1-based markdown source coordinates. */
export interface AnchorDraft {
  startLine: number
  startCol: number
  endLine: number
  endCol: number
  /** Whole source lines [startLine, endLine] — the unit the API re-anchors on. */
  snippet: string
  /** Exact text the user selected, for display in the composer. */
  selectedText: string
}

// Text inside these tags is left unwrapped: fenced code blocks are read back
// as plain strings by the markdown `code` renderer (e.g. for mermaid).
const SKIP_TAGS = new Set(['pre', 'code'])

interface HastNode {
  type: string
  tagName?: string
  value?: string
  position?: { start: { line: number; column: number }; end: { line: number; column: number } }
  properties?: Record<string, unknown>
  children?: HastNode[]
}

/**
 * rehypeSourcePos wraps every text node in a <span> carrying the source
 * line/column range it came from (data-src-line / data-src-col and the
 * matching end attributes). Whitespace-only nodes and code blocks are skipped.
 */
export function rehypeSourcePos() {
  return (tree: HastNode) => walk(tree, false)
}

function walk(node: HastNode, inCode: boolean): void {
  if (!node.children) return
  node.children = node.children.map((child) => {
    if (child.type === 'element') {
      walk(child, inCode || SKIP_TAGS.has(child.tagName ?? ''))
      return child
    }
    if (
      child.type === 'text' &&
      !inCode &&
      child.position &&
      child.value &&
      child.value.trim() !== ''
    ) {
      const { start, end } = child.position
      return {
        type: 'element',
        tagName: 'span',
        properties: {
          dataSrcLine: start.line,
          dataSrcCol: start.column,
          dataSrcEndLine: end.line,
          dataSrcEndCol: end.column,
        },
        children: [child],
      } as HastNode
    }
    return child
  })
}

/** (l1,c1) strictly precedes (l2,c2) in document order. */
function before(l1: number, c1: number, l2: number, c2: number): boolean {
  return l1 < l2 || (l1 === l2 && c1 < c2)
}

function firstTextNode(node: Node | null): Text | null {
  if (!node) return null
  if (node.nodeType === Node.TEXT_NODE) return node as Text
  for (const child of Array.from(node.childNodes)) {
    const found = firstTextNode(child)
    if (found) return found
  }
  return null
}

function stampedAncestor(node: Node | null): HTMLElement | null {
  let el: HTMLElement | null =
    node instanceof HTMLElement ? node : (node?.parentElement ?? null)
  while (el) {
    if (el.hasAttribute('data-src-line')) return el
    el = el.parentElement
  }
  return null
}

interface SourcePoint {
  line: number
  col: number
}

/** Maps a DOM (node, offset) pair to a 1-based source line/column. */
function domPointToSource(node: Node, offset: number): SourcePoint | null {
  let textNode: Text | null = null
  let textOffset = 0
  if (node.nodeType === Node.TEXT_NODE) {
    textNode = node as Text
    textOffset = offset
  } else {
    // An element container: the offset indexes child nodes. Descend to text.
    const child = node.childNodes[offset] ?? node.childNodes[node.childNodes.length - 1]
    textNode = firstTextNode(child ?? null)
    textOffset = 0
  }
  if (!textNode) return null

  const stamped = stampedAncestor(textNode)
  if (!stamped) return null
  const baseLine = Number(stamped.getAttribute('data-src-line'))
  const baseCol = Number(stamped.getAttribute('data-src-col'))

  const value = textNode.textContent ?? ''
  const prefix = value.slice(0, textOffset)
  const newlines = (prefix.match(/\n/g) ?? []).length
  if (newlines === 0) {
    return { line: baseLine, col: baseCol + textOffset }
  }
  // Soft-wrapped text node spanning multiple source lines.
  const lastNL = prefix.lastIndexOf('\n')
  return { line: baseLine + newlines, col: textOffset - lastNL }
}

/**
 * resolveSelection turns the current document selection — when it lies wholly
 * inside `container` — into an AnchorDraft. Returns null when there is no
 * usable selection or it cannot be mapped to source positions.
 */
export function resolveSelection(container: HTMLElement, description: string): AnchorDraft | null {
  const sel = window.getSelection()
  if (!sel || sel.rangeCount === 0 || sel.isCollapsed) return null

  const range = sel.getRangeAt(0)
  if (!container.contains(range.startContainer) || !container.contains(range.endContainer)) {
    return null
  }

  const start = domPointToSource(range.startContainer, range.startOffset)
  const end = domPointToSource(range.endContainer, range.endOffset)
  if (!start || !end) return null
  if (!before(start.line, start.col, end.line, end.col)) return null

  const lines = description.split('\n')
  if (start.line < 1 || end.line > lines.length) return null

  const snippet = lines.slice(start.line - 1, end.line).join('\n')
  if (snippet.trim() === '') return null

  return {
    startLine: start.line,
    startCol: start.col,
    endLine: end.line,
    endCol: end.col,
    snippet,
    selectedText: range.toString(),
  }
}

interface DomLocation {
  node: Text
  offset: number
}

/** Finds the DOM text node + offset for a 1-based source (line, col). */
function locateSource(container: HTMLElement, line: number, col: number): DomLocation | null {
  if (col < 1) return null
  const spans = container.querySelectorAll<HTMLElement>('[data-src-line]')
  for (const span of Array.from(spans)) {
    const bL = Number(span.getAttribute('data-src-line'))
    const bC = Number(span.getAttribute('data-src-col'))
    const eL = Number(span.getAttribute('data-src-end-line'))
    const eC = Number(span.getAttribute('data-src-end-col'))
    if (before(line, col, bL, bC)) continue
    if (before(eL, eC, line, col)) continue

    const textNode = span.firstChild
    if (!textNode || textNode.nodeType !== Node.TEXT_NODE) continue
    const value = textNode.textContent ?? ''
    let curL = bL
    let curC = bC
    for (let offset = 0; offset <= value.length; offset++) {
      if (curL === line && curC === col) return { node: textNode as Text, offset }
      if (value[offset] === '\n') {
        curL++
        curC = 1
      } else {
        curC++
      }
    }
  }
  return null
}

/**
 * findAnchorRange reconstructs a DOM Range covering the anchored text. Returns
 * null when the precise positions can't be located (e.g. a whole-block anchor
 * with no columns, or an outdated comment whose text has moved).
 */
export function findAnchorRange(
  container: HTMLElement,
  startLine: number,
  startCol: number,
  endLine: number,
  endCol: number,
): Range | null {
  const start = locateSource(container, startLine, startCol)
  const end = locateSource(container, endLine, endCol)
  if (!start || !end) return null
  try {
    const range = document.createRange()
    range.setStart(start.node, start.offset)
    range.setEnd(end.node, end.offset)
    return range
  } catch {
    return null
  }
}

/**
 * findLineElement returns the first stamped element covering `line`, used to
 * position a gutter marker vertically against that source line.
 */
export function findLineElement(container: HTMLElement, line: number): HTMLElement | null {
  const spans = container.querySelectorAll<HTMLElement>('[data-src-line]')
  for (const span of Array.from(spans)) {
    const bL = Number(span.getAttribute('data-src-line'))
    const eL = Number(span.getAttribute('data-src-end-line'))
    if (line >= bL && line <= eL) return span
  }
  return null
}
