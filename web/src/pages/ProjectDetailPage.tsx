import { useParams, Routes, Route } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useProject } from '@/hooks/useProjects'
import { Sidebar } from '@/components/Sidebar'
import { useSidebar } from '@/contexts/SidebarContext'
import { Spinner } from '@/components/ui/Spinner'
import { WorkItemListPage } from './WorkItemListPage'
import { WorkItemDetailPage } from './WorkItemDetailPage'
import { ProjectSettingsPage } from './ProjectSettingsPage'
import { ProjectOverviewPage } from './ProjectOverviewPage'
import { ProjectWorkflowsPage } from './ProjectWorkflowsPage'
import { MilestonesPage } from './MilestonesPage'

function PlaceholderPage({ title }: { title: string }) {
  const { t } = useTranslation()
  return <p className="text-gray-500 dark:text-gray-400">{t('projects.placeholder.comingSoon', { title })}</p>
}

export function ProjectDetailPage() {
  const { t } = useTranslation()
  const { collapsed } = useSidebar()
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
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6">
      <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <Sidebar projectKey={project.key} />
        <div className="flex-1 min-w-0">
          <Routes>
            <Route index element={<ProjectOverviewPage />} />
            <Route path="items" element={<WorkItemListPage />} />
            <Route path="items/:itemNumber" element={<WorkItemDetailPage />} />
            <Route path="queues" element={<PlaceholderPage title={t('sidebar.queues')} />} />
            <Route path="milestones" element={<MilestonesPage />} />
            <Route path="workflows" element={<ProjectWorkflowsPage />} />
            <Route path="settings" element={<ProjectSettingsPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}
