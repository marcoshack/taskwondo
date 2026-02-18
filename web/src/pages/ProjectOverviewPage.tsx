import { useParams } from 'react-router-dom'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { useProject, useMembers } from '@/hooks/useProjects'
import { useWorkItems } from '@/hooks/useWorkItems'
import { Avatar } from '@/components/ui/Avatar'
import { Spinner } from '@/components/ui/Spinner'

const typeConfig: Record<string, { label: string; color: string; iconBg: string }> = {
  bug: { label: 'Bugs', color: 'text-red-700 dark:text-red-400', iconBg: 'bg-red-100 dark:bg-red-900/30' },
  task: { label: 'Tasks', color: 'text-blue-700 dark:text-blue-400', iconBg: 'bg-blue-100 dark:bg-blue-900/30' },
  ticket: { label: 'Tickets', color: 'text-indigo-700 dark:text-indigo-300', iconBg: 'bg-indigo-100 dark:bg-indigo-900/30' },
  feedback: { label: 'Feedback', color: 'text-yellow-700 dark:text-yellow-400', iconBg: 'bg-yellow-100 dark:bg-yellow-900/30' },
  epic: { label: 'Epics', color: 'text-green-700 dark:text-green-400', iconBg: 'bg-green-100 dark:bg-green-900/30' },
}

function getTypeLabel(type: string): string {
  return typeConfig[type]?.label ?? type.charAt(0).toUpperCase() + type.slice(1)
}

function getTypeStyle(type: string) {
  return typeConfig[type] ?? { label: type, color: 'text-gray-700 dark:text-gray-300', iconBg: 'bg-gray-100 dark:bg-gray-800' }
}

export function ProjectOverviewPage() {
  const { projectKey } = useParams<{ projectKey: string }>()
  const { data: project } = useProject(projectKey ?? '')
  const { data: members, isLoading: membersLoading } = useMembers(projectKey ?? '')
  const { data: itemsData, isLoading: itemsLoading } = useWorkItems(projectKey ?? '', { limit: 500 })

  // Count non-closed (unresolved) items by type
  const openByType: Record<string, number> = {}
  let totalOpen = 0
  if (itemsData?.data) {
    for (const item of itemsData.data) {
      if (!item.resolved_at) {
        openByType[item.type] = (openByType[item.type] ?? 0) + 1
        totalOpen++
      }
    }
  }

  // Sort types: known types first in defined order, then any extras alphabetically
  const knownOrder = Object.keys(typeConfig)
  const allTypes = Object.keys(openByType)
  const sortedTypes = [
    ...knownOrder.filter((t) => allTypes.includes(t)),
    ...allTypes.filter((t) => !knownOrder.includes(t)).sort(),
  ]

  return (
    <div className="space-y-6">
      {/* Stats row */}
      <div>
        <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
          Open Work Items
        </h2>
        {itemsLoading ? (
          <div className="flex items-center justify-center py-8">
            <Spinner />
          </div>
        ) : sortedTypes.length === 0 ? (
          <p className="text-sm text-gray-400 dark:text-gray-500 py-4">No open work items.</p>
        ) : (
          <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-4">
            {sortedTypes.map((type) => {
              const style = getTypeStyle(type)
              return (
                <div
                  key={type}
                  className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4 shadow-sm"
                >
                  <div className="flex items-center gap-2 mb-2">
                    <span className={`inline-block h-2.5 w-2.5 rounded-full ${style.iconBg}`} />
                    <span className="text-sm font-medium text-gray-600 dark:text-gray-400">
                      {getTypeLabel(type)}
                    </span>
                  </div>
                  <p className={`text-3xl font-bold ${style.color}`}>{openByType[type]}</p>
                </div>
              )
            })}
            <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4 shadow-sm">
              <div className="flex items-center gap-2 mb-2">
                <span className="inline-block h-2.5 w-2.5 rounded-full bg-gray-200 dark:bg-gray-600" />
                <span className="text-sm font-medium text-gray-600 dark:text-gray-400">Total</span>
              </div>
              <p className="text-3xl font-bold text-gray-900 dark:text-gray-100">{totalOpen}</p>
            </div>
          </div>
        )}
      </div>

      {/* Description + Members */}
      <div className="flex gap-6">
        {/* Description — left 80% */}
        <div className="w-4/5 min-w-0">
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            About
          </h2>
          <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-5 shadow-sm">
            {project?.description ? (
              <div className="prose prose-sm dark:prose-invert max-w-none text-gray-700 dark:text-gray-300">
                <Markdown remarkPlugins={[remarkGfm]}>{project.description}</Markdown>
              </div>
            ) : (
              <p className="text-sm text-gray-400 dark:text-gray-500 italic">No description provided.</p>
            )}
          </div>
        </div>

        {/* Members — right 20% */}
        <div className="w-1/5 min-w-[160px]">
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            Members
          </h2>
          <div className="rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4 shadow-sm">
            {membersLoading ? (
              <Spinner />
            ) : !members || members.length === 0 ? (
              <p className="text-sm text-gray-400 dark:text-gray-500">No members.</p>
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
