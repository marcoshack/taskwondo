import { useMemo } from 'react'
import { NavLink, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import {
  LayoutDashboard,
  ClipboardList,
  Inbox,
  Target,
  Route,
  Settings,
} from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { useProject, useMembers } from '@/hooks/useProjects'
import { useWorkItems } from '@/hooks/useWorkItems'
import { useProjectWorkflow } from '@/hooks/useWorkflows'
import { Avatar } from '@/components/ui/Avatar'
import { Spinner } from '@/components/ui/Spinner'

const closedCategories = new Set(['done', 'cancelled'])
const primaryTypes = new Set(['task', 'ticket', 'bug'])

export function ProjectOverviewPage() {
  const { t } = useTranslation()
  const { projectKey } = useParams<{ projectKey: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const { guardRef, guardedNavigate } = useNavigationGuard()
  const { data: project } = useProject(projectKey ?? '')
  const { data: members, totalCount: membersTotalCount, isLoading: membersLoading } = useMembers(projectKey ?? '')
  const { statuses } = useProjectWorkflow(projectKey ?? '')

  const panels = [
    { key: 'task', label: t('projects.overview.tasks'), color: 'text-blue-700 dark:text-blue-400', iconBg: 'bg-blue-100 dark:bg-blue-900/30' },
    { key: 'ticket', label: t('projects.overview.tickets'), color: 'text-indigo-700 dark:text-indigo-300', iconBg: 'bg-indigo-100 dark:bg-indigo-900/30' },
    { key: 'bug', label: t('projects.overview.bugs'), color: 'text-red-700 dark:text-red-400', iconBg: 'bg-red-100 dark:bg-red-900/30' },
    { key: 'other', label: t('projects.overview.other'), color: 'text-yellow-700 dark:text-yellow-400', iconBg: 'bg-yellow-100 dark:bg-yellow-900/30' },
    { key: 'total', label: t('projects.overview.total'), color: 'text-gray-900 dark:text-gray-100', iconBg: 'bg-gray-200 dark:bg-gray-600' },
  ]

  const openStatusNames = useMemo(
    () => statuses.filter((s) => !closedCategories.has(s.category)).map((s) => s.name),
    [statuses],
  )

  const { data: itemsData, isLoading: itemsLoading } = useWorkItems(projectKey ?? '', {
    limit: 500,
    status: openStatusNames.length ? openStatusNames : undefined,
  })

  const counts = useMemo(() => {
    const byKey: Record<string, number> = { task: 0, ticket: 0, bug: 0, other: 0, total: 0 }
    const otherTypes: string[] = []
    if (itemsData?.data) {
      for (const item of itemsData.data) {
        if (primaryTypes.has(item.type)) {
          byKey[item.type]++
        } else {
          byKey.other++
          if (!otherTypes.includes(item.type)) otherTypes.push(item.type)
        }
        byKey.total++
      }
    }
    return { byKey, otherTypes }
  }, [itemsData])

  const allTypes = ['task', 'ticket', 'bug', 'feedback', 'epic']
  const otherTypes = allTypes.filter((tp) => !primaryTypes.has(tp))

  function navigateToItems(panelKey: string) {
    const params = new URLSearchParams()
    if (openStatusNames.length) params.set('status', openStatusNames.join(','))
    if (primaryTypes.has(panelKey)) {
      params.set('type', panelKey)
    } else if (panelKey === 'other') {
      params.set('type', otherTypes.join(','))
    } else {
      params.set('type', allTypes.join(','))
    }
    navigate(`/projects/${projectKey}/items?${params.toString()}`)
  }

  const loading = itemsLoading || !statuses.length

  const base = `/projects/${projectKey}`
  const navItems = [
    { to: '', label: t('sidebar.overview'), icon: LayoutDashboard, end: true },
    { to: 'items', label: t('sidebar.items'), icon: ClipboardList, end: false },
    { to: 'queues', label: t('sidebar.queues'), icon: Inbox, end: false },
    { to: 'milestones', label: t('sidebar.milestones'), icon: Target, end: false },
    { to: 'workflows', label: t('sidebar.workflows'), icon: Route, end: false },
    { to: 'settings', label: t('sidebar.settings'), icon: Settings, end: false },
  ]

  return (
    <div className="max-w-3xl space-y-6">
      {/* Mobile top bar with navigation icons */}
      <nav className="flex sm:hidden overflow-hidden">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={`${base}/${item.to}`}
            end={item.end}
            onClick={(e) => {
              if (guardRef.current?.()) {
                e.preventDefault()
                guardedNavigate(`${base}/${item.to}`)
              }
            }}
            className={({ isActive }) =>
              `flex flex-1 flex-col items-center gap-1 py-3 text-xs font-medium transition-colors ${
                isActive
                  ? 'bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300'
                  : 'text-gray-500 hover:bg-gray-50 dark:text-gray-400 dark:hover:bg-gray-800'
              }`
            }
          >
            <item.icon className="h-5 w-5" />
            <span className="truncate max-w-full px-1">{item.label}</span>
          </NavLink>
        ))}
      </nav>

      <div>
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('projects.overview.openWorkItems')}
        </h2>
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner />
          </div>
        ) : (
          <div className="grid grid-cols-3 sm:grid-cols-5 gap-3">
            {panels.map((panel) => (
              <button
                key={panel.key}
                onClick={() => navigateToItems(panel.key)}
                className="rounded-lg border border-gray-200 dark:border-gray-700 p-3 text-left hover:border-indigo-300 dark:hover:border-indigo-600 hover:shadow-md transition-all cursor-pointer"
              >
                <div className="flex items-center gap-1.5 mb-1">
                  <span className={`inline-block h-2 w-2 rounded-full ${panel.iconBg}`} />
                  <span className="text-xs font-medium text-gray-600 dark:text-gray-400">
                    {panel.label}
                  </span>
                </div>
                <p className={`text-2xl font-bold ${panel.color}`}>{counts.byKey[panel.key]}</p>
              </button>
            ))}
          </div>
        )}
      </div>

      <div>
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('projects.overview.about')}
        </h2>
        <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-5">
          {project?.description ? (
            <div className="prose prose-sm dark:prose-invert max-w-none text-gray-700 dark:text-gray-300 break-words">
              <Markdown remarkPlugins={[remarkGfm]}>{project.description}</Markdown>
            </div>
          ) : (
            <p className="text-sm text-gray-400 dark:text-gray-500 italic">{t('projects.overview.noDescription')}</p>
          )}
        </div>
      </div>

      <div>
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('projects.overview.members')}
        </h2>
        <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-4">
          {membersLoading ? (
            <Spinner />
          ) : !members || members.length === 0 ? (
            <p className="text-sm text-gray-400 dark:text-gray-500">{t('projects.overview.noMembers')}</p>
          ) : (
            <div className="flex flex-wrap gap-3 items-center">
              {members.map((member) => (
                <div key={member.user_id} className="flex items-center gap-2">
                  <Avatar name={member.display_name} size="sm" />
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                      {member.display_name}
                      {member.user_id === user?.id && <span className="ml-1 text-xs text-gray-400">({t('common.you')})</span>}
                    </p>
                    <p className="text-xs text-gray-500 dark:text-gray-400 capitalize">{member.role}</p>
                  </div>
                </div>
              ))}
              {membersTotalCount != null && membersTotalCount > members.length && (
                <span className="text-sm text-gray-500 dark:text-gray-400">
                  {t('projects.settings.hiddenMembers', { count: membersTotalCount - members.length })}
                </span>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
