import { useNavigate } from 'react-router-dom'
import { useProjects } from '@/hooks/useProjects'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Spinner } from '@/components/ui/Spinner'
import type { Column } from '@/components/ui/DataTable'
import type { Project } from '@/api/projects'

const columns: Column<Project>[] = [
  {
    key: 'key',
    header: 'Key',
    render: (p) => <Badge color="indigo">{p.key}</Badge>,
  },
  {
    key: 'name',
    header: 'Name',
    render: (p) => <span className="font-medium text-gray-900">{p.name}</span>,
  },
  {
    key: 'items',
    header: 'Items',
    render: (p) => <span className="text-gray-500">{p.item_counter}</span>,
  },
  {
    key: 'updated',
    header: 'Updated',
    render: (p) => (
      <span className="text-gray-500">{new Date(p.updated_at).toLocaleDateString()}</span>
    ),
  },
]

export function ProjectListPage() {
  const { data: projects, isLoading, error } = useProjects()
  const navigate = useNavigate()

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-24">
        <Spinner size="lg" />
      </div>
    )
  }

  if (error) {
    return (
      <div className="max-w-7xl mx-auto px-4 py-8">
        <p className="text-red-600">Failed to load projects.</p>
      </div>
    )
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-gray-900">Projects</h1>
      </div>
      <div className="bg-white rounded-lg shadow overflow-hidden">
        <DataTable
          columns={columns}
          data={projects ?? []}
          onRowClick={(p) => navigate(`/projects/${p.key}`)}
          emptyMessage="No projects yet"
        />
      </div>
    </div>
  )
}
