import { useParams, Routes, Route } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { SquareStack } from 'lucide-react'
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

function QueuesPage() {
  const { t } = useTranslation()
  return (
    <div className="flex flex-col items-center justify-center h-64 text-gray-500 dark:text-gray-400">
      <SquareStack className="h-12 w-12 mb-4 opacity-30" />
      <p className="text-lg font-medium">{t('projects.queuesComingSoon')}</p>
    </div>
  )
}

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
