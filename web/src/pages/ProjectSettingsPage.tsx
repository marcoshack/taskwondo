import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useProject, useUpdateProject, useDeleteProject } from '@/hooks/useProjects'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import type { AxiosError } from 'axios'

export function ProjectSettingsPage() {
  const { projectKey } = useParams<{ projectKey: string }>()
  const navigate = useNavigate()
  const { data: project, isLoading } = useProject(projectKey ?? '')
  const updateMutation = useUpdateProject(projectKey ?? '')
  const deleteMutation = useDeleteProject(projectKey ?? '')

  const [name, setName] = useState<string | null>(null)
  const [description, setDescription] = useState<string | null>(null)
  const [saveError, setSaveError] = useState('')
  const [saveSuccess, setSaveSuccess] = useState(false)

  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [deleteConfirmText, setDeleteConfirmText] = useState('')

  if (isLoading || !project) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    )
  }

  const currentName = name ?? project.name
  const currentDescription = description ?? project.description ?? ''

  const hasChanges =
    (name !== null && name !== project.name) ||
    (description !== null && description !== (project.description ?? ''))

  function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setSaveError('')
    setSaveSuccess(false)

    if (!currentName.trim()) {
      setSaveError('Project name is required.')
      return
    }

    const input: Record<string, string | null> = {}
    if (name !== null && name !== project!.name) input.name = name.trim()
    if (description !== null && description !== (project!.description ?? '')) {
      input.description = description.trim() || null
    }

    if (Object.keys(input).length === 0) return

    updateMutation.mutate(input, {
      onSuccess: () => {
        setSaveSuccess(true)
        setName(null)
        setDescription(null)
        setTimeout(() => setSaveSuccess(false), 3000)
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setSaveError(axiosErr.response?.data?.error?.message ?? 'Failed to update project.')
      },
    })
  }

  function handleDelete() {
    deleteMutation.mutate(undefined, {
      onSuccess: () => {
        navigate('/projects', { replace: true })
      },
      onError: (err) => {
        const axiosErr = err as AxiosError<{ error?: { message?: string } }>
        setSaveError(axiosErr.response?.data?.error?.message ?? 'Failed to delete project.')
        setShowDeleteModal(false)
      },
    })
  }

  return (
    <div className="max-w-2xl space-y-8">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">General</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">Manage your project settings.</p>
      </div>

      <form onSubmit={handleSave} className="space-y-4">
        <Input label="Project key" value={project.key} disabled className="bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed" />

        <Input
          label="Project name"
          value={currentName}
          onChange={(e) => setName(e.target.value)}
          required
        />

        <div>
          <label htmlFor="description" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            Description
          </label>
          <textarea
            id="description"
            rows={12}
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
            value={currentDescription}
            onChange={(e) => setDescription(e.target.value)}
          />
        </div>

        {saveError && <p className="text-sm text-red-600 dark:text-red-400">{saveError}</p>}
        {saveSuccess && <p className="text-sm text-green-600 dark:text-green-400">Project updated successfully.</p>}

        <div className="flex gap-2">
          <Button type="submit" disabled={!hasChanges || updateMutation.isPending}>
            {updateMutation.isPending ? 'Saving...' : 'Save changes'}
          </Button>
        </div>
      </form>

      {/* Danger Zone */}
      <div className="border border-red-300 dark:border-red-800 rounded-lg mt-8">
        <div className="px-4 py-3 border-b border-red-300 dark:border-red-800">
          <h3 className="text-base font-semibold text-red-600">Danger Zone</h3>
        </div>
        <div className="p-4 flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">Delete this project</p>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              All work items, comments, and relations in this project will be permanently deleted.
            </p>
          </div>
          <Button variant="danger" size="sm" onClick={() => setShowDeleteModal(true)}>
            Delete project
          </Button>
        </div>
      </div>

      {/* Delete confirmation modal */}
      <Modal open={showDeleteModal} onClose={() => setShowDeleteModal(false)} title="Delete project">
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          This action <strong>cannot be undone</strong>. This will permanently delete the{' '}
          <strong>{project.key}</strong> project and all of its work items, comments, and relations.
        </p>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-2">
          Please type <strong>{project.key}</strong> to confirm.
        </p>
        <Input
          value={deleteConfirmText}
          onChange={(e) => setDeleteConfirmText(e.target.value)}
          placeholder={project.key}
        />
        <div className="flex justify-end gap-2 mt-4">
          <Button variant="secondary" onClick={() => setShowDeleteModal(false)}>
            Cancel
          </Button>
          <Button
            variant="danger"
            disabled={deleteConfirmText !== project.key || deleteMutation.isPending}
            onClick={handleDelete}
          >
            {deleteMutation.isPending ? 'Deleting...' : 'I understand, delete this project'}
          </Button>
        </div>
      </Modal>
    </div>
  )
}
