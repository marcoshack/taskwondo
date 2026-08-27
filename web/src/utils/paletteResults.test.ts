import { describe, it, expect } from 'vitest'
import en from '@/i18n/en.json'
import { buildNavigationCatalog } from './navigationCatalog'
import type { CatalogNamespace, NavigationEntry } from './navigationCatalog'
import type { NavUser } from './sidebarNav'
import {
  composePalette,
  dedupeSemanticResults,
  entityRowId,
  groupEntityResults,
  groupNavigationEntries,
  isPaletteToggleEvent,
  moveSelection,
  navRowId,
  rowIndex,
} from './paletteResults'
import type { PaletteRow } from './paletteResults'
import type { SearchResult } from '@/api/search'

/**
 * Real English copy, so the acceptance cases ("milestone", "appearance",
 * "api keys") exercise the strings a user actually types against.
 */
const strings = en as Record<string, string>
function t(key: string, params?: Record<string, unknown>): string {
  const raw = strings[key] ?? key
  if (!params) return raw
  return raw.replace(/\{\{(\w+)\}\}/g, (_, name) => String(params[name] ?? ''))
}

const MEMBER: NavUser = { global_role: 'member', total_project_count: 3 }
const ADMIN: NavUser = { global_role: 'admin', total_project_count: 3 }

const ACME: CatalogNamespace = {
  slug: 'acme',
  display_name: 'Acme',
  icon: 'rocket',
  color: 'indigo',
  is_default: false,
}

function catalog(
  overrides: { user?: NavUser | null; activeProjectKey?: string; namespaces?: CatalogNamespace[] } = {},
): NavigationEntry[] {
  return buildNavigationCatalog({
    t,
    p: (path) => `/d${path}`,
    toSegment: (slug) => (slug === 'default' ? 'd' : slug),
    user: overrides.user === undefined ? MEMBER : overrides.user,
    activeProjectKey: overrides.activeProjectKey,
    namespaces: overrides.namespaces ?? [],
  })
}

let nextId = 0
function result(entityType: string, snippet: string, over: Partial<SearchResult> = {}): SearchResult {
  nextId += 1
  return {
    entity_type: entityType,
    entity_id: `id-${nextId}`,
    project_id: 'p1',
    score: 0.5,
    snippet,
    ...over,
  }
}

const rowLabel = (row: PaletteRow) =>
  row.kind === 'nav' ? row.entry.label : row.result.snippet

const rowLabels = (rows: readonly PaletteRow[]) => rows.map(rowLabel)

describe('groupNavigationEntries', () => {
  it('splits the catalog into contiguous runs, one per group', () => {
    const sections = groupNavigationEntries(catalog({ activeProjectKey: 'TF', namespaces: [ACME] }))
    expect(sections.map((s) => s.group)).toEqual(['project', 'user', 'preferences', 'namespace'])
    expect(sections[0].entries.every((e) => e.group === 'project')).toBe(true)
  })

  it('keeps the admin run for an admin and drops it for a member', () => {
    expect(groupNavigationEntries(catalog({ user: ADMIN })).map((s) => s.group))
      .toEqual(['user', 'preferences', 'admin'])
    expect(groupNavigationEntries(catalog({ user: MEMBER })).map((s) => s.group))
      .toEqual(['user', 'preferences'])
  })

  it('returns nothing for an empty match', () => {
    expect(groupNavigationEntries([])).toEqual([])
  })
})

describe('groupEntityResults', () => {
  it('orders sections by ENTITY_TYPE_ORDER regardless of arrival order', () => {
    const sections = groupEntityResults([
      result('comment', 'Comment:\n\nlate'),
      result('project', 'Alpha'),
      result('work_item', '[task] Something'),
    ])
    expect(sections.map((s) => s.entityType)).toEqual(['project', 'work_item', 'comment'])
  })

  it('keeps every hit of a type together, in arrival order', () => {
    const sections = groupEntityResults([
      result('work_item', 'first'),
      result('project', 'Alpha'),
      result('work_item', 'second'),
    ])
    expect(sections[1].results.map((r) => r.snippet)).toEqual(['first', 'second'])
  })

  it('drops entity types the palette does not render', () => {
    expect(groupEntityResults([result('wormhole', 'nope')])).toEqual([])
  })
})

describe('dedupeSemanticResults', () => {
  it('drops semantic hits the exact half already showed', () => {
    const shared = result('work_item', '[task] Shared')
    const only = result('work_item', '[task] Only semantic')
    expect(dedupeSemanticResults([shared], [shared, only]).map((r) => r.snippet))
      .toEqual(['[task] Only semantic'])
  })

  it('keeps a same-id hit of a different entity type', () => {
    const a = result('work_item', 'wi')
    const b = { ...result('milestone', 'ms'), entity_id: a.entity_id }
    expect(dedupeSemanticResults([a], [b])).toHaveLength(1)
  })
})

describe('composePalette — result composition', () => {
  it('puts navigation above entity results', () => {
    const { rows } = composePalette({
      navEntries: catalog({ activeProjectKey: 'TF' }),
      query: 'milestone',
      ftsResults: [result('milestone', 'Q3 milestone')],
    })
    expect(rowLabels(rows)).toEqual(['Milestones', 'Q3 milestone'])
    expect(rows[0].kind).toBe('nav')
    expect(rows[1].kind).toBe('entity')
  })

  it('offers the active project\'s Milestones section for "milestone"', () => {
    const { navSections } = composePalette({
      navEntries: catalog({ activeProjectKey: 'TF' }),
      query: 'milestone',
    })
    expect(navSections).toHaveLength(1)
    expect(navSections[0].group).toBe('project')
    expect(navSections[0].entries[0].target).toEqual({
      kind: 'route',
      to: '/d/projects/TF/milestones',
    })
  })

  it('offers Preferences → Appearance for "appearance"', () => {
    const { navSections } = composePalette({ navEntries: catalog(), query: 'appearance' })
    expect(navSections.map((s) => s.group)).toEqual(['preferences'])
    expect(navSections[0].entries.map((e) => e.target)).toEqual([
      { kind: 'route', to: '/preferences/appearance' },
    ])
  })

  it('offers System Settings → API Keys for an admin and nothing for a member', () => {
    const asAdmin = composePalette({ navEntries: catalog({ user: ADMIN }), query: 'api keys' })
    expect(asAdmin.navSections.map((s) => s.group)).toEqual(['admin'])
    expect(asAdmin.navSections[0].entries[0].target).toEqual({ kind: 'route', to: '/admin/api-keys' })

    const asMember = composePalette({ navEntries: catalog({ user: MEMBER }), query: 'api keys' })
    expect(asMember.navSections).toEqual([])
    expect(asMember.rows).toEqual([])
  })

  it('matches navigation from the very first character, below the entity floor', () => {
    const { navSections, ftsSections } = composePalette({
      navEntries: catalog({ activeProjectKey: 'TF' }),
      query: 'm',
      // useSearch returns nothing below two characters, so nothing is passed.
    })
    expect(navSections.flatMap((s) => s.entries.map((e) => e.label))).toContain('Milestones')
    expect(ftsSections).toEqual([])
  })

  it('shows the whole catalog and no entity results for an empty query', () => {
    const entries = catalog({ user: ADMIN, activeProjectKey: 'TF', namespaces: [ACME] })
    const { rows, navSections, ftsSections, semanticSections } = composePalette({
      navEntries: entries,
      query: '',
    })
    expect(rows).toHaveLength(entries.length)
    expect(rows.every((r) => r.kind === 'nav')).toBe(true)
    expect(navSections.map((s) => s.group)).toEqual([
      'project', 'user', 'preferences', 'admin', 'namespace',
    ])
    expect(ftsSections).toEqual([])
    expect(semanticSections).toEqual([])
  })

  it('keeps a whitespace-only query browsable rather than treating it as a search', () => {
    const entries = catalog()
    expect(composePalette({ navEntries: entries, query: '   ' }).rows).toHaveLength(entries.length)
  })

  it('lays out exact-match hits before semantic hits, each in entity-type order', () => {
    const shared = result('work_item', '[task] Shared')
    const { rows } = composePalette({
      navEntries: [],
      query: 'zzz-matches-no-navigation',
      ftsResults: [result('comment', 'Comment:\n\nc1'), shared],
      semanticResults: [shared, result('project', 'Alpha')],
    })
    expect(rowLabels(rows)).toEqual(['[task] Shared', 'Comment:\n\nc1', 'Alpha'])
    expect(rows.map((r) => (r.kind === 'entity' ? r.section : 'nav')))
      .toEqual(['fts', 'fts', 'semantic'])
  })

  it('gives every row a unique, stable id', () => {
    const { rows } = composePalette({
      navEntries: catalog({ user: ADMIN, activeProjectKey: 'TF', namespaces: [ACME] }),
      query: '',
      ftsResults: [result('work_item', 'a')],
    })
    expect(new Set(rows.map((r) => r.id)).size).toBe(rows.length)
  })

  it('derives the same ids the renderer uses to light the selection', () => {
    const entries = catalog()
    const hit = result('work_item', '[task] Thing')
    const { rows } = composePalette({ navEntries: entries, query: '', ftsResults: [hit] })
    expect(rowIndex(rows, navRowId(entries[0]))).toBe(0)
    expect(rowIndex(rows, entityRowId('fts', hit))).toBe(rows.length - 1)
    expect(rowIndex(rows, 'nav:nope')).toBe(-1)
  })
})

describe('composePalette — the project scope carve-out', () => {
  it('shows project hits alongside scoped ones, so another project stays reachable', () => {
    // TF-432 keeps `project` hits global while everything else is scoped; the
    // palette does not re-filter what the backend sent.
    const { rows } = composePalette({
      navEntries: [],
      query: 'alpha',
      ftsResults: [
        result('project', 'Alpha', { project_key: 'AL' }),
        result('work_item', '[task] Alpha thing', { project_key: 'TF' }),
      ],
    })
    expect(rows.map((r) => (r.kind === 'entity' ? r.result.project_key : null))).toEqual(['AL', 'TF'])
  })
})

describe('moveSelection — one selection across the group boundary', () => {
  const rows = composePalette({
    navEntries: catalog({ activeProjectKey: 'TF' }),
    query: 'milestone',
    ftsResults: [result('milestone', 'Q3 milestone'), result('milestone', 'Q4 milestone')],
  }).rows

  it('walks off the last navigation row into the first entity row', () => {
    expect(rows).toHaveLength(3)
    expect(rows[0].kind).toBe('nav')
    expect(moveSelection(0, 1, rows.length)).toBe(1)
    expect(rows[1].kind).toBe('entity')
  })

  it('walks back from the first entity row onto the navigation row', () => {
    expect(moveSelection(1, -1, rows.length)).toBe(0)
  })

  it('clamps at both ends instead of wrapping', () => {
    expect(moveSelection(0, -1, rows.length)).toBe(0)
    expect(moveSelection(rows.length - 1, 1, rows.length)).toBe(rows.length - 1)
  })

  it('stays at zero when there is nothing to select', () => {
    expect(moveSelection(0, 1, 0)).toBe(0)
    expect(moveSelection(5, -1, 0)).toBe(0)
  })

  it('never points past the end after the list shrinks', () => {
    expect(moveSelection(9, 0, 3)).toBe(2)
  })

  it('opens what the arrows landed on', () => {
    const landed = rows[moveSelection(0, 1, rows.length)]
    expect(landed.kind).toBe('entity')
    expect(landed.kind === 'entity' && landed.result.snippet).toBe('Q3 milestone')
  })
})

describe('isPaletteToggleEvent — close on a second Cmd/Ctrl+K', () => {
  it('matches Cmd+K and Ctrl+K, in either case', () => {
    expect(isPaletteToggleEvent({ key: 'k', metaKey: true })).toBe(true)
    expect(isPaletteToggleEvent({ key: 'k', ctrlKey: true })).toBe(true)
    expect(isPaletteToggleEvent({ key: 'K', metaKey: true })).toBe(true)
  })

  it('ignores an unmodified k, so typing "k" still searches', () => {
    expect(isPaletteToggleEvent({ key: 'k' })).toBe(false)
    expect(isPaletteToggleEvent({ key: 'k', metaKey: false, ctrlKey: false })).toBe(false)
  })

  it('ignores other modified keys', () => {
    expect(isPaletteToggleEvent({ key: 'j', metaKey: true })).toBe(false)
    expect(isPaletteToggleEvent({ key: 'Enter', ctrlKey: true })).toBe(false)
    expect(isPaletteToggleEvent({ key: 'Escape' })).toBe(false)
  })
})
