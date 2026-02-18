import { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { Trans, useTranslation } from 'react-i18next'
import { useProject, useUpdateProject, useDeleteProject } from '@/hooks/useProjects'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import type { AxiosError } from 'axios'

export function ProjectSettingsPage() {
  const { t } = useTranslation()
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
      setSaveError(t('projects.settings.nameRequired'))
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
        setSaveError(axiosErr.response?.data?.error?.message ?? t('projects.settings.updateError'))
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
        setSaveError(axiosErr.response?.data?.error?.message ?? t('projects.settings.deleteError'))
        setShowDeleteModal(false)
      },
    })
  }

  return (
    <div className="max-w-2xl space-y-8">
      <div>
        <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{t('projects.settings.general')}</h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('projects.settings.description')}</p>
      </div>

      <form onSubmit={handleSave} className="space-y-4">
        <Input label={t('projects.settings.projectKey')} value={project.key} disabled className="bg-gray-100 dark:bg-gray-700 text-gray-500 dark:text-gray-400 cursor-not-allowed" />

        <Input
          label={t('projects.settings.projectName')}
          value={currentName}
          onChange={(e) => setName(e.target.value)}
          required
        />

        <div>
          <label htmlFor="description" className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('common.description')}
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
        {saveSuccess && <p className="text-sm text-green-600 dark:text-green-400">{t('projects.settings.updateSuccess')}</p>}

        <div className="flex gap-2">
          <Button type="submit" disabled={!hasChanges || updateMutation.isPending}>
            {updateMutation.isPending ? t('common.saving') : t('projects.settings.saveChanges')}
          </Button>
        </div>
      </form>

      {/* Danger Zone */}
      <div className="border border-red-300 dark:border-red-800 rounded-lg mt-8">
        <div className="px-4 py-3 border-b border-red-300 dark:border-red-800">
          <h3 className="text-base font-semibold text-red-600">{t('projects.settings.dangerZone')}</h3>
        </div>
        <div className="p-4 flex items-center justify-between">
          <div>
            <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{t('projects.settings.deleteThisProject')}</p>
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {t('projects.settings.deleteWarning')}
            </p>
          </div>
          <Button variant="danger" size="sm" onClick={() => setShowDeleteModal(true)}>
            {t('projects.settings.deleteProject')}
          </Button>
        </div>
      </div>

      {/* Delete confirmation modal */}
      <Modal open={showDeleteModal} onClose={() => setShowDeleteModal(false)} title={t('projects.settings.deleteConfirmTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">
          <Trans i18nKey="projects.settings.deleteConfirmBody" values={{ projectKey: project.key }} components={{ bold: <strong /> }} />
        </p>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-2">
          <Trans i18nKey="projects.settings.deleteConfirmType" values={{ projectKey: project.key }} components={{ bold: <strong /> }} />
        </p>
        <Input
          value={deleteConfirmText}
          onChange={(e) => setDeleteConfirmText(e.target.value)}
          placeholder={project.key}
        />
        <div className="flex justify-end gap-2 mt-4">
          <Button variant="secondary" onClick={() => setShowDeleteModal(false)}>
            {t('common.cancel')}
          </Button>
          <Button
            variant="danger"
            disabled={deleteConfirmText !== project.key || deleteMutation.isPending}
            onClick={handleDelete}
          >
            {deleteMutation.isPending ? t('common.deleting') : t('projects.settings.deleteConfirmButton')}
          </Button>
        </div>
      </Modal>
    </div>
  )
}
