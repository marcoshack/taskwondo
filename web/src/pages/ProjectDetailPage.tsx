import { useParams, Routes, Route, Navigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useProject } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
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
import { PortalTicketListPage } from './PortalTicketListPage'
import { PortalTicketDetailPage } from './PortalTicketDetailPage'

export function ProjectDetailPage() {
  const { t } = useTranslation()
  const { collapsed } = useSidebar('app')
  const { containerClass } = useLayout()
  const { projectKey } = useParams<{ projectKey: string }>()
  const { user } = useAuth()

  // Determine if the user is a customer in this project
  const isCustomerProject = user?.global_role !== 'admin'
    && (user?.portal_projects ?? []).some((p) => p.project_key === projectKey)

  // Skip the regular project API call for customer projects (ExcludeCustomer middleware blocks it)
  const { data: project, isLoading, error } = useProject(isCustomerProject ? '' : (projectKey ?? ''))

  if (isCustomerProject) {
    return (
      <div className={`${containerClass(true)} py-6`}>
        <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
          <AppSidebar projectKey={projectKey} customerProject />
          <div className="flex-1 min-w-0">
            <Routes>
              <Route index element={<Navigate to="support" replace />} />
              <Route path="support" element={<PortalTicketListPage />} />
              <Route path="support/:itemNumber" element={<PortalTicketDetailPage />} />
              <Route path="*" element={<Navigate to="support" replace />} />
            </Routes>
          </div>
        </div>
      </div>
    )
  }

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
