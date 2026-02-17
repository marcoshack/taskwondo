import { useParams, Routes, Route } from 'react-router-dom'
import { useProject } from '@/hooks/useProjects'
import { Sidebar } from '@/components/Sidebar'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { WorkItemListPage } from './WorkItemListPage'
import { WorkItemDetailPage } from './WorkItemDetailPage'
import { ProjectSettingsPage } from './ProjectSettingsPage'
import { ProjectOverviewPage } from './ProjectOverviewPage'

function PlaceholderPage({ title }: { title: string }) {
  return <p className="text-gray-500">{title} view coming soon.</p>
}

export function ProjectDetailPage() {
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
        <p className="text-red-600">Project not found.</p>
      </div>
    )
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="mb-6">
        <div className="flex items-center gap-3">
          <Badge color="indigo">{project.key}</Badge>
          <h1 className="text-xl font-semibold text-gray-900">{project.name}</h1>
        </div>
      </div>
      <div className="flex gap-8">
        <Sidebar projectKey={project.key} />
        <div className="flex-1 min-w-0">
          <Routes>
            <Route index element={<ProjectOverviewPage />} />
            <Route path="items" element={<WorkItemListPage />} />
            <Route path="items/:itemNumber" element={<WorkItemDetailPage />} />
            <Route path="queues" element={<PlaceholderPage title="Queues" />} />
            <Route path="milestones" element={<PlaceholderPage title="Milestones" />} />
            <Route path="settings" element={<ProjectSettingsPage />} />
          </Routes>
        </div>
      </div>
    </div>
  )
}
