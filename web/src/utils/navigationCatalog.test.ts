import { describe, it, expect } from 'vitest'
import { buildNavigationCatalog } from './navigationCatalog'
import type { CatalogNamespace, NavigationCatalogInput, NavigationEntry } from './navigationCatalog'
import { matchNavigationItems } from './navigationSearch'
import * as sidebarNav from './sidebarNav'
import {
  preferencesNavItems,
  projectNavItems,
  systemSettingsNavItems,
  userNavItems,
} from './sidebarNav'
import type { NavUser, SidebarNavItem } from './sidebarNav'

/** Stand-in for i18next's `t`: echoes the key so tests assert on keys, not copy. */
function keyEcho(key: string, params?: Record<string, unknown>): string {
  return params && 'name' in params ? `${key}(${String(params.name)})` : key
}

/** Stand-in locale, to prove labels really come from `t`. */
const FRENCH: Record<string, string> = {
  'sidebar.overview': "Vue d'ensemble",
  'sidebar.items': 'Éléments',
  'preferences.sidebar.profile': 'Profil',
  'user.sidebar.inbox': 'Boîte de réception',
}

function frenchT(key: string, params?: Record<string, unknown>): string {
  return FRENCH[key] ?? keyEcho(key, params)
}

const DEFAULT_NS: CatalogNamespace = {
  slug: 'default',
  display_name: 'Public',
  icon: 'building2',
  color: 'slate',
  is_default: true,
}

const TEAM_NS: CatalogNamespace = {
  slug: 'acme',
  display_name: 'Acme',
  icon: 'rocket',
  color: 'indigo',
  is_default: false,
}

const MEMBER: NavUser = { global_role: 'member', total_project_count: 3 }
const ADMIN: NavUser = { global_role: 'admin', total_project_count: 3 }

function build(overrides: Partial<NavigationCatalogInput> = {}): NavigationEntry[] {
  return buildNavigationCatalog({
    t: keyEcho,
    p: (path) => `/d${path}`,
    toSegment: (slug) => (slug === 'default' ? 'd' : slug),
    user: MEMBER,
    namespaces: [],
    ...overrides,
  })
}

const labels = (entries: NavigationEntry[]) => entries.map((e) => e.label)
const inGroup = (entries: NavigationEntry[], group: NavigationEntry['group']) =>
  entries.filter((e) => e.group === group)

describe('buildNavigationCatalog — active project', () => {
  it('lists the project sections for the active project', () => {
    const project = inGroup(build({ activeProjectKey: 'TF' }), 'project')
    expect(labels(project)).toEqual([
      'sidebar.overview',
      'sidebar.items',
      'sidebar.milestones',
      'sidebar.queues',
      'sidebar.workflows',
      'sidebar.settings',
    ])
    expect(project[0].target).toEqual({ kind: 'route', to: '/d/projects/TF/' })
    expect(project[1].target).toEqual({ kind: 'route', to: '/d/projects/TF/items' })
  })

  it('omits the project group entirely when no project is active', () => {
    const entries = build()
    expect(inGroup(entries, 'project')).toEqual([])
    // The rest of the catalog is unaffected.
    expect(inGroup(entries, 'preferences')).toHaveLength(5)
  })

  it('gives a customer in that project Support and nothing else', () => {
    const customer: NavUser = {
      global_role: 'member',
      total_project_count: 5,
      portal_projects: [{ project_key: 'TF' }],
    }
    const project = inGroup(build({ user: customer, activeProjectKey: 'TF' }), 'project')
    expect(labels(project)).toEqual(['sidebar.support'])
    expect(project[0].target).toEqual({ kind: 'route', to: '/d/projects/TF/support' })
  })

  it('gives a system admin the full sections even for a portal project', () => {
    const adminCustomer: NavUser = {
      global_role: 'admin',
      total_project_count: 5,
      portal_projects: [{ project_key: 'TF' }],
    }
    const project = inGroup(build({ user: adminCustomer, activeProjectKey: 'TF' }), 'project')
    expect(labels(project)).toContain('sidebar.overview')
    expect(labels(project)).not.toContain('sidebar.support')
  })

  it('carries the sidebar icon for every entry', () => {
    for (const entry of build({ activeProjectKey: 'TF', namespaces: [DEFAULT_NS, TEAM_NS] })) {
      expect(entry.icon).toBeTruthy()
    }
  })

  it('gives every entry a unique id', () => {
    const entries = build({
      user: ADMIN,
      activeProjectKey: 'TF',
      namespaces: [DEFAULT_NS, TEAM_NS],
    })
    expect(new Set(entries.map((e) => e.id)).size).toBe(entries.length)
  })
})

describe('buildNavigationCatalog — user pages', () => {
  it('lists Projects, Inbox, Watchlist and Feed', () => {
    expect(labels(inGroup(build(), 'user'))).toEqual([
      'sidebar.projects',
      'user.sidebar.inbox',
      'user.sidebar.watchlist',
      'user.sidebar.feed',
    ])
  })

  it('hides the personal pages for a user who is a customer in every project', () => {
    const customerOnly: NavUser = {
      global_role: 'member',
      total_project_count: 1,
      portal_projects: [{ project_key: 'TF' }],
    }
    // Projects survives the gate — AppSidebar links it for customers too.
    expect(labels(inGroup(build({ user: customerOnly }), 'user'))).toEqual(['sidebar.projects'])
  })
})

describe('buildNavigationCatalog — the project list', () => {
  const projectsRow = (entries: NavigationEntry[]) =>
    entries.find((e) => e.label === 'sidebar.projects')

  it('routes to the namespace-prefixed project list', () => {
    expect(projectsRow(build())?.target).toEqual({ kind: 'route', to: '/d/projects' })
  })

  it('is present when no project is active — the palette is the only way there', () => {
    const entries = build()
    expect(inGroup(entries, 'project')).toEqual([])
    expect(projectsRow(entries)).toBeDefined()
  })

  it('is present when a project is active, and is not one of that project\'s sections', () => {
    const entries = build({ activeProjectKey: 'TF' })
    expect(projectsRow(entries)?.group).toBe('user')
    expect(labels(inGroup(entries, 'project'))).not.toContain('sidebar.projects')
  })

  it('is matchable by name from the very first character', () => {
    expect(labels(matchNavigationItems(build(), 'sidebar.proj'))).toEqual(['sidebar.projects'])
  })
})

describe('buildNavigationCatalog — preferences', () => {
  it('lists the five preferences pages for every user', () => {
    expect(labels(inGroup(build(), 'preferences'))).toEqual([
      'preferences.sidebar.profile',
      'preferences.sidebar.general',
      'preferences.sidebar.appearance',
      'preferences.sidebar.notifications',
      'preferences.sidebar.authentication',
    ])
  })
})

describe('buildNavigationCatalog — system admin gate', () => {
  it('includes System Settings for a system admin', () => {
    const admin = inGroup(build({ user: ADMIN }), 'admin')
    expect(labels(admin)).toEqual([
      'admin.sidebar.general',
      'admin.sidebar.directory',
      'admin.sidebar.workflows',
      'admin.sidebar.authentication',
      'admin.sidebar.apiKeys',
      'admin.sidebar.integrations',
      'admin.sidebar.features',
    ])
    expect(admin[4].target).toEqual({ kind: 'route', to: '/admin/api-keys' })
  })

  it('omits System Settings for a non-admin, and for no user at all', () => {
    expect(inGroup(build({ user: MEMBER }), 'admin')).toEqual([])
    expect(inGroup(build({ user: null }), 'admin')).toEqual([])
    expect(inGroup(build({ user: undefined }), 'admin')).toEqual([])
  })

  it('never leaks an /admin route into a member catalog', () => {
    const entries = build({ user: MEMBER, activeProjectKey: 'TF', namespaces: [DEFAULT_NS, TEAM_NS] })
    const routes = entries.flatMap((e) => (e.target.kind === 'route' ? [e.target.to] : []))
    expect(routes.filter((r) => r.startsWith('/admin'))).toEqual([])
  })
})

describe('buildNavigationCatalog — namespaces', () => {
  it('adds a switch row per namespace that calls setActiveNamespace', () => {
    const ns = inGroup(build({ namespaces: [DEFAULT_NS, TEAM_NS] }), 'namespace')
    expect(ns[0].label).toBe('palette.nav.switchToNamespace(Public)')
    expect(ns[0].target).toEqual({ kind: 'namespace', slug: 'default' })
    expect(ns[0].namespace).toEqual({ slug: 'default', icon: 'building2', color: 'slate' })

    const acmeSwitch = ns.find((e) => e.target.kind === 'namespace' && e.target.slug === 'acme')
    expect(acmeSwitch?.label).toBe('palette.nav.switchToNamespace(Acme)')
  })

  it('adds a Settings row for a non-default namespace', () => {
    const ns = inGroup(build({ namespaces: [TEAM_NS] }), 'namespace')
    expect(labels(ns)).toEqual([
      'palette.nav.switchToNamespace(Acme)',
      'palette.nav.namespaceSettings(Acme)',
    ])
    expect(ns[1].target).toEqual({ kind: 'route', to: '/acme/settings' })
  })

  it('omits the Settings row for the default namespace', () => {
    const ns = inGroup(build({ namespaces: [DEFAULT_NS] }), 'namespace')
    expect(labels(ns)).toEqual(['palette.nav.switchToNamespace(Public)'])
    expect(ns.every((e) => e.target.kind === 'namespace')).toBe(true)
  })

  it('adds nothing when the user has no namespaces', () => {
    expect(inGroup(build({ namespaces: [] }), 'namespace')).toEqual([])
  })
})

describe('buildNavigationCatalog — translation', () => {
  it('takes labels from t, so a locale switch changes what matches', () => {
    const english = build({ activeProjectKey: 'TF' })
    const french = build({ t: frenchT, activeProjectKey: 'TF' })

    expect(labels(french)).toContain("Vue d'ensemble")
    expect(labels(french)).toContain('Éléments')
    expect(labels(french)).toContain('Profil')

    // "Éléments" is only reachable in French; "sidebar.items" only in English.
    expect(matchNavigationItems(french, 'elem')).toHaveLength(1)
    expect(matchNavigationItems(english, 'elem')).toHaveLength(0)
    expect(matchNavigationItems(english, 'sidebar.items')).toHaveLength(1)
  })

  it('matches accented translated labels without the accents', () => {
    const french = build({ t: frenchT })
    expect(labels(matchNavigationItems(french, 'boite'))).toEqual(['Boîte de réception'])
  })
})

/**
 * The rule from AGENTS.md — every sidebar destination is reachable from the
 * palette — holds by construction *within* a builder: `buildNavigationCatalog`
 * maps the very array the sidebar renders, so a page added to `projectNavItems`
 * cannot go missing. What is not automatic is the wiring: a brand-new sidebar
 * section means a brand-new builder in `sidebarNav`, and nothing stops it from
 * being rendered by a sidebar and never handed to the catalog. That is the gap
 * these two tests close.
 */
describe('every shared sidebar builder is wired into the catalog', () => {
  /**
   * How to call each builder with the same arguments the catalog uses. A
   * builder missing from here fails the first test — which is the point: you
   * cannot add a sidebar section without deciding whether the palette lists it.
   */
  const builderCalls: Record<string, () => SidebarNavItem[]> = {
    projectNavItems: () => projectNavItems(keyEcho, '/d/projects/TF', false),
    userNavItems: () => userNavItems(keyEcho),
    preferencesNavItems: () => preferencesNavItems(keyEcho),
    systemSettingsNavItems: () => systemSettingsNavItems(keyEcho),
  }

  it('covers every *NavItems builder sidebarNav exports', () => {
    const exported = Object.entries(sidebarNav)
      .filter(([name, value]) => name.endsWith('NavItems') && typeof value === 'function')
      .map(([name]) => name)
    expect(exported.sort()).toEqual(Object.keys(builderCalls).sort())
  })

  it('lists every destination those builders produce', () => {
    // An admin with an active project sees every group at once.
    const routes = build({ user: ADMIN, activeProjectKey: 'TF' }).flatMap((e) =>
      e.target.kind === 'route' ? [e.target.to] : [],
    )
    for (const [name, call] of Object.entries(builderCalls)) {
      for (const item of call()) {
        expect(routes, `${name} → ${item.to} is missing from the catalog`).toContain(item.to)
      }
    }
  })
})
