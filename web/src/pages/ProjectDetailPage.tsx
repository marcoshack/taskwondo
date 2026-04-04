import { useParams, Routes, Route } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useProject } from '@/hooks/useProjects'
import { AppSidebar } from '@/components/AppSidebar'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLayout } from '@/contexts/LayoutContext'
import { Spinner } from '@/components/ui/Spinner'
import { WorkItemListPage } from './WorkItemListPage'
import { WorkItemDetailPage } from './WorkItemDetailPage'
import { ProjectSettingsPage } from './ProjectSettingsPage'
import { ProjectOverviewPage } from './ProjectOverviewPage'
import { ProjectWorkflowsPage } from './ProjectWorkflowsPage'
import { MilestonesPage } from './MilestonesPage'
import { MilestoneDashboardPage } from './MilestoneDashboardPage'
import { QueuesPage } from './QueuesPage'
import { QueueSettingsPage } from './QueueSettingsPage'
import { TeamsPage } from './TeamsPage'
import { TeamDetailPage } from './TeamDetailPage'
import { QueueWorkItemsPage } from './QueueWorkItemsPage'

export function ProjectDetailPage() {
  const { t } = useTranslation()
  const { collapsed } = useSidebar('app')
  const { containerClass } = useLayout()
  const { projectKey } = useParams<{ projectKey: string }>()
  const { data: project, isLoading, error } = useProject(projectKey ?? '')

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Spinner size="lg" />
      </div>
    )
  }

  if (error || !project) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <p className="text-red-600">{t('projects.notFound')}</p>
      </div>
    )
  }

  return (
    <div className={`${containerClass(true)} py-6`}>
      <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <AppSidebar projectKey={project.key} />
        <div className="flex-1 min-w-0">
          <Routes>
            <Route index element={<ProjectOverviewPage />} />
            <Route path="items" element={<WorkItemListPage />} />
            <Route path="items/:itemNumber" element={<WorkItemDetailPage />} />
            <Route path="queues" element={<QueuesPage />} />
            <Route path="queues/:queueId" element={<QueueSettingsPage />} />
            <Route path="queues/:queueId/items" element={<QueueWorkItemsPage />} />
            <Route path="teams" element={<TeamsPage />} />
            <Route path="teams/:teamId" element={<TeamDetailPage />} />
            <Route path="milestones" element={<MilestonesPage />} />
            <Route path="milestones/:milestoneId" element={<MilestoneDashboardPage />} />
            <Route path="workflows" element={<ProjectWorkflowsPage />} />
            <Route path="settings" element={<ProjectSettingsPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}
