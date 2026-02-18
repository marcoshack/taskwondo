import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useProjects, useCreateProject } from '@/hooks/useProjects'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Spinner } from '@/components/ui/Spinner'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
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
    render: (p) => <span className="font-medium text-gray-900 dark:text-gray-100">{p.name}</span>,
  },
  {
    key: 'items',
    header: 'Items',
    render: (p) => <span className="text-gray-500 dark:text-gray-400">{p.item_counter}</span>,
  },
  {
    key: 'updated',
    header: 'Updated',
    render: (p) => (
      <span className="text-gray-500 dark:text-gray-400">{new Date(p.updated_at).toLocaleDateString()}</span>
    ),
  },
]

export function ProjectListPage() {
  const { data: projects, isLoading, error } = useProjects()
  const navigate = useNavigate()
  const createMutation = useCreateProject()

  const [showCreate, setShowCreate] = useState(false)
  const [name, setName] = useState('')
  const [key, setKey] = useState('')
  const [description, setDescription] = useState('')
  const [formError, setFormError] = useState('')

  function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!name.trim() || !key.trim()) return
    createMutation.mutate(
      { name: name.trim(), key: key.trim().toUpperCase(), description: description.trim() || undefined },
      {
        onSuccess: (project) => {
          setShowCreate(false)
          setName('')
          setKey('')
          setDescription('')
          navigate(`/projects/${project.key}`)
        },
        onError: (err) => {
          if (err && typeof err === 'object' && 'response' in err) {
            const axiosErr = err as { response?: { data?: { error?: { message?: string } } } }
            setFormError(axiosErr.response?.data?.error?.message ?? 'Failed to create project')
          } else {
            setFormError('Failed to create project')
          }
        },
      },
    )
  }

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
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">Projects</h1>
        <Button onClick={() => setShowCreate(true)}>New Project</Button>
      </div>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <DataTable
          columns={columns}
          data={projects ?? []}
          onRowClick={(p) => navigate(`/projects/${p.key}`)}
          emptyMessage="No projects yet. Create one to get started."
        />
      </div>

      <Modal open={showCreate} onClose={() => setShowCreate(false)} title="New Project">
        <form onSubmit={handleCreate} className="space-y-4">
          <Input
            label="Name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="My Project"
            required
          />
          <Input
            label="Key"
            value={key}
            onChange={(e) => setKey(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ''))}
            placeholder="PROJ"
            maxLength={10}
            required
          />
          <p className="text-xs text-gray-400 dark:text-gray-500 -mt-3">1-10 uppercase letters/digits, must start with a letter</p>
          <Input
            label="Description"
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder="Optional description"
          />
          {formError && <p className="text-sm text-red-600 dark:text-red-400">{formError}</p>}
          <div className="flex justify-end gap-3 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowCreate(false)}>Cancel</Button>
            <Button type="submit" disabled={createMutation.isPending || !name.trim() || !key.trim()}>
              {createMutation.isPending ? 'Creating...' : 'Create'}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
