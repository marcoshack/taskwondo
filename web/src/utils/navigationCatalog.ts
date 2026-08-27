/**
 * The command palette's navigation catalog.
 *
 * A flat, gated, already-translated list of everywhere the signed-in user can
 * go, derived from the shared sidebar definitions in `@/utils/sidebarNav` so a
 * page added to a sidebar shows up here without a second edit.
 *
 * `buildNavigationCatalog` is a pure function: every input is passed in
 * explicitly, so all the gating can be unit-tested without rendering React.
 * `useNavigationCatalog` (`@/hooks/useNavigationCatalog`) is the thin wrapper
 * that gathers those inputs from context.
 *
 * Gating is real, not cosmetic: the admin group only exists for system admins
 * and a namespace's Settings row only for a non-default namespace, so the
 * catalog can never contain a row that would 403 on click.
 *
 * This module must stay free of `@/api/client` — see `@/utils/sidebarNav`.
 */
import { Building2, Settings } from 'lucide-react'
import type { LucideIcon } from 'lucide-react'
import {
  isCustomerOnly,
  isCustomerProject,
  isSystemAdmin,
  preferencesNavItems,
  projectNavItems,
  systemSettingsNavItems,
  userNavItems,
} from '@/utils/sidebarNav'
import type { NavUser, SidebarNavItem, Translate } from '@/utils/sidebarNav'

/** Which section of the palette an entry belongs to. */
export type NavigationGroupId = 'project' | 'user' | 'preferences' | 'admin' | 'namespace'

/**
 * What activating an entry does. `route` is an ordinary client-side navigation;
 * `namespace` must call `setActiveNamespace(slug)` instead — switching
 * namespaces is context state, not a URL.
 */
export type NavigationTarget =
  | { kind: 'route'; to: string }
  | { kind: 'namespace'; slug: string }

export interface NavigationEntry {
  /** Stable identity, unique across the catalog. Use as the React key. */
  id: string
  /** Translated, user-visible label — the only field the matcher looks at. */
  label: string
  /** Lucide icon mirroring the sidebar the entry came from. */
  icon: LucideIcon
  group: NavigationGroupId
  target: NavigationTarget
  /**
   * Present on namespace rows so the palette can render `<NamespaceIcon>` with
   * the namespace's own icon and colour instead of the generic `icon`.
   */
  namespace?: { slug: string; icon: string; color: string }
}

/** The subset of `Namespace` (`@/api/namespaces`) the catalog needs. */
export interface CatalogNamespace {
  slug: string
  display_name: string
  icon: string
  color: string
  is_default: boolean
}

export interface NavigationCatalogInput {
  t: Translate
  /** `p` from `useNamespacePath` — prefixes a path with the active namespace segment. */
  p: (path: string) => string
  /** Maps a namespace slug to its URL segment (`toUrlSegment`). */
  toSegment: (slug: string) => string
  user: NavUser | null | undefined
  /** Project whose sections to list; omit when no project is active. */
  activeProjectKey?: string
  /** Namespaces the user has access to. */
  namespaces: readonly CatalogNamespace[]
}

function fromSidebar(group: NavigationGroupId, items: SidebarNavItem[]): NavigationEntry[] {
  return items.map((item) => ({
    id: `${group}:${item.to}`,
    label: item.label,
    icon: item.icon,
    group,
    target: { kind: 'route', to: item.to },
  }))
}

/**
 * Build the full navigation catalog for a user. Order is stable: active project
 * sections, personal pages, preferences, system settings, namespaces.
 */
export function buildNavigationCatalog(input: NavigationCatalogInput): NavigationEntry[] {
  const { t, p, toSegment, user, activeProjectKey, namespaces } = input
  const entries: NavigationEntry[] = []

  // Active project sections — absent entirely when no project is active.
  // A customer in that project gets Support and nothing else.
  if (activeProjectKey) {
    const base = p(`/projects/${activeProjectKey}`)
    entries.push(
      ...fromSidebar('project', projectNavItems(t, base, isCustomerProject(user, activeProjectKey))),
    )
  }

  // Personal pages — hidden for users who are customers in every project,
  // matching what AppSidebar does.
  if (!isCustomerOnly(user)) {
    entries.push(...fromSidebar('user', userNavItems(t)))
  }

  entries.push(...fromSidebar('preferences', preferencesNavItems(t)))

  // System Settings — system admins only.
  if (isSystemAdmin(user)) {
    entries.push(...fromSidebar('admin', systemSettingsNavItems(t)))
  }

  for (const ns of namespaces) {
    entries.push({
      id: `namespace:${ns.slug}`,
      label: t('palette.nav.switchToNamespace', { name: ns.display_name }),
      icon: Building2,
      group: 'namespace',
      target: { kind: 'namespace', slug: ns.slug },
      namespace: { slug: ns.slug, icon: ns.icon, color: ns.color },
    })

    // Settings under exactly the condition AppSidebar puts on its settings
    // gear: the default namespace has no settings page.
    if (!ns.is_default) {
      entries.push({
        id: `namespace:${ns.slug}:settings`,
        label: t('palette.nav.namespaceSettings', { name: ns.display_name }),
        icon: Settings,
        group: 'namespace',
        target: { kind: 'route', to: `/${toSegment(ns.slug)}/settings` },
        namespace: { slug: ns.slug, icon: ns.icon, color: ns.color },
      })
    }
  }

  return entries
}
