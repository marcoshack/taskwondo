import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { useLastProjectKey } from '@/hooks/useLastProjectKey'
import { toUrlSegment } from '@/utils/namespaceUrl'
import { buildNavigationCatalog } from '@/utils/navigationCatalog'
import type { NavigationEntry } from '@/utils/navigationCatalog'

/**
 * The command palette's navigation catalog for the current user: gated,
 * translated and aware of the active project.
 *
 * All the logic lives in `buildNavigationCatalog` (`@/utils/navigationCatalog`),
 * which is pure and unit-tested; this hook only gathers its inputs from
 * context. The active project is the one `AppSidebar` shows sections for — the
 * last project the user opened.
 *
 * @param projectKey Overrides the remembered project (e.g. on a project page).
 */
export function useNavigationCatalog(projectKey?: string): NavigationEntry[] {
  const { t } = useTranslation()
  const { p } = useNamespacePath()
  const { user } = useAuth()
  const { namespaces } = useNamespaceContext()
  const storedLastProjectKey = useLastProjectKey()

  const activeProjectKey = projectKey ?? storedLastProjectKey ?? undefined

  return useMemo(
    () =>
      buildNavigationCatalog({
        t,
        p,
        toSegment: toUrlSegment,
        user,
        activeProjectKey,
        namespaces,
      }),
    [t, p, user, activeProjectKey, namespaces],
  )
}
