/**
 * Result composition and keyboard maths for the command palette.
 *
 * The palette shows two halves at once — the in-memory navigation catalog
 * (`@/utils/navigationCatalog`) on top and entity hits from the search API
 * below — under a single selection that moves across the boundary. All of that
 * bookkeeping lives here as pure functions so it can be unit-tested without a
 * DOM: `CommandPalette` only renders what `composePalette` returns and asks
 * `moveSelection` where the arrow keys go.
 *
 * This module must stay free of `@/api/client` — it is imported by units that
 * run under Vitest without a DOM.
 */
import type { NavigationEntry, NavigationGroupId } from '@/utils/navigationCatalog'
import { matchNavigationItems } from '@/utils/navigationSearch'
import type { SearchResult } from '@/api/search'

/**
 * Order the entity sections are shown in, unchanged from the search modal this
 * palette replaces. `project` leads because it is how the user jumps to another
 * project now that entity search is scoped to the active one.
 */
export const ENTITY_TYPE_ORDER = [
  'project',
  'work_item',
  'team',
  'milestone',
  'queue',
  'comment',
  'attachment',
] as const

/** Which half of the entity results a row came from. */
export type EntitySectionId = 'fts' | 'semantic'

export type PaletteRow =
  | { kind: 'nav'; id: string; entry: NavigationEntry }
  | { kind: 'entity'; id: string; section: EntitySectionId; result: SearchResult }

/** A run of catalog entries sharing a group, in catalog order. */
export interface PaletteNavSection {
  group: NavigationGroupId
  entries: NavigationEntry[]
}

/** A run of entity hits sharing an entity type. */
export interface PaletteEntitySection {
  entityType: string
  results: SearchResult[]
}

export interface PaletteComposition {
  navSections: PaletteNavSection[]
  ftsSections: PaletteEntitySection[]
  semanticSections: PaletteEntitySection[]
  /** Every selectable row, in the order they are painted. */
  rows: PaletteRow[]
}

/** Stable React key / DOM id for a navigation row. */
export function navRowId(entry: NavigationEntry): string {
  return `nav:${entry.id}`
}

/** Stable React key / DOM id for an entity row. */
export function entityRowId(section: EntitySectionId, result: SearchResult): string {
  return `${section}:${result.entity_type}:${result.entity_id}`
}

/**
 * Split matched catalog entries into contiguous runs by group. The catalog
 * guarantees entries are already grouped, so a single scan is enough and the
 * palette's headings follow the catalog's own order.
 */
export function groupNavigationEntries(
  entries: readonly NavigationEntry[],
): PaletteNavSection[] {
  const sections: PaletteNavSection[] = []
  for (const entry of entries) {
    const last = sections[sections.length - 1]
    if (last && last.group === entry.group) last.entries.push(entry)
    else sections.push({ group: entry.group, entries: [entry] })
  }
  return sections
}

/** Bucket entity hits by type, in `ENTITY_TYPE_ORDER`; unknown types are dropped. */
export function groupEntityResults(results: readonly SearchResult[]): PaletteEntitySection[] {
  const map = new Map<string, SearchResult[]>()
  for (const r of results) {
    const list = map.get(r.entity_type)
    if (list) list.push(r)
    else map.set(r.entity_type, [r])
  }
  return ENTITY_TYPE_ORDER.filter((type) => map.has(type)).map((type) => ({
    entityType: type,
    results: map.get(type)!,
  }))
}

/** Drop semantic hits the exact-match half already showed. */
export function dedupeSemanticResults(
  ftsResults: readonly SearchResult[],
  semanticResults: readonly SearchResult[],
): SearchResult[] {
  const seen = new Set(ftsResults.map((r) => `${r.entity_type}:${r.entity_id}`))
  return semanticResults.filter((r) => !seen.has(`${r.entity_type}:${r.entity_id}`))
}

export interface ComposePaletteInput {
  /** The full catalog for this user; matched here, not by the caller. */
  navEntries: readonly NavigationEntry[]
  /** The live, undebounced query — navigation matches from the first character. */
  query: string
  /** Exact-match hits. Already empty below the API's two-character floor. */
  ftsResults?: readonly SearchResult[]
  /** Semantic hits, deduped against `ftsResults` here. */
  semanticResults?: readonly SearchResult[]
}

/**
 * Compose the whole palette: navigation on top, entity hits below, flattened
 * into one selectable row list in painting order.
 */
export function composePalette(input: ComposePaletteInput): PaletteComposition {
  const { navEntries, query, ftsResults = [], semanticResults = [] } = input

  const navSections = groupNavigationEntries(matchNavigationItems(navEntries, query))
  const ftsSections = groupEntityResults(ftsResults)
  const semanticSections = groupEntityResults(dedupeSemanticResults(ftsResults, semanticResults))

  const rows: PaletteRow[] = []
  for (const section of navSections) {
    for (const entry of section.entries) {
      rows.push({ kind: 'nav', id: navRowId(entry), entry })
    }
  }
  for (const [sectionId, sections] of [
    ['fts', ftsSections],
    ['semantic', semanticSections],
  ] as const) {
    for (const section of sections) {
      for (const result of section.results) {
        rows.push({ kind: 'entity', id: entityRowId(sectionId, result), section: sectionId, result })
      }
    }
  }

  return { navSections, ftsSections, semanticSections, rows }
}

/** Index of a row in the flattened list, or -1. Used to light the selection. */
export function rowIndex(rows: readonly PaletteRow[], id: string): number {
  return rows.findIndex((row) => row.id === id)
}

/**
 * Where ArrowUp/ArrowDown land. The list does not wrap — it clamps at both
 * ends, so holding ArrowDown walks off the navigation half into the entity
 * half and stops at the last hit rather than jumping back to the top.
 */
export function moveSelection(current: number, delta: number, count: number): number {
  if (count <= 0) return 0
  return Math.min(Math.max(current + delta, 0), count - 1)
}

/** The subset of a keyboard event the toggle check needs. */
export interface PaletteKeyEvent {
  key: string
  metaKey?: boolean
  ctrlKey?: boolean
}

/**
 * True for the Cmd/Ctrl+K that closes an open palette.
 *
 * This has to be answered by the palette's own input: `KeyboardShortcutContext`
 * bails out of the global handler both while a modal is open and whenever the
 * event target is an INPUT, so the global Cmd/Ctrl+K never fires a second time.
 */
export function isPaletteToggleEvent(e: PaletteKeyEvent): boolean {
  return (e.key === 'k' || e.key === 'K') && (e.metaKey === true || e.ctrlKey === true)
}
