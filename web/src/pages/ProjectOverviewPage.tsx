import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
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
  const { data: project } = useProject(projectKey ?? '')
  const { data: members, isLoading: membersLoading } = useMembers(projectKey ?? '')
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

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          {t('projects.overview.openWorkItems')}
        </h2>
        {loading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner />
          </div>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
            {panels.map((panel) => (
              <button
                key={panel.key}
                onClick={() => navigateToItems(panel.key)}
                className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4 shadow-sm text-left hover:border-indigo-300 dark:hover:border-indigo-600 hover:shadow-md transition-all cursor-pointer"
              >
                <div className="flex items-center gap-2 mb-2">
                  <span className={`inline-block h-2.5 w-2.5 rounded-full ${panel.iconBg}`} />
                  <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                    {panel.label}
                  </span>
                </div>
                <p className={`text-3xl font-bold ${panel.color}`}>{counts.byKey[panel.key]}</p>
              </button>
            ))}
          </div>
        )}
      </div>

      <div className="flex gap-6">
        <div className="w-4/5 min-w-0">
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            {t('projects.overview.about')}
          </h2>
          <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-5 shadow-sm">
            {project?.description ? (
              <div className="prose prose-sm dark:prose-invert max-w-none text-gray-700 dark:text-gray-300">
                <Markdown remarkPlugins={[remarkGfm]}>{project.description}</Markdown>
              </div>
            ) : (
              <p className="text-sm text-gray-400 dark:text-gray-500 italic">{t('projects.overview.noDescription')}</p>
            )}
          </div>
        </div>

        <div className="w-1/5 min-w-[160px]">
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            {t('projects.overview.members')}
          </h2>
          <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4 shadow-sm">
            {membersLoading ? (
              <Spinner />
            ) : !members || members.length === 0 ? (
              <p className="text-sm text-gray-400 dark:text-gray-500">{t('projects.overview.noMembers')}</p>
            ) : (
              <ul className="space-y-3">
                {members.map((member) => (
                  <li key={member.user_id} className="flex items-center gap-2">
                    <Avatar name={member.display_name} size="sm" />
                    <div className="min-w-0">
                      <p className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate">
                        {member.display_name}
                      </p>
                      <p className="text-xs text-gray-500 dark:text-gray-400 capitalize">{member.role}</p>
                    </div>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
