import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { useProjects, useCreateProject } from '@/hooks/useProjects'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Spinner } from '@/components/ui/Spinner'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import type { Column } from '@/components/ui/DataTable'
import type { Project } from '@/api/projects'

export function ProjectListPage() {
  const { t } = useTranslation()
  const { data: projects, isLoading, error } = useProjects()
  const navigate = useNavigate()
  const createMutation = useCreateProject()

  const [showCreate, setShowCreate] = useState(false)
  const [activeRow, setActiveRow] = useState(-1)

  useKeyboardShortcut({ key: 'n' }, () => setShowCreate(true))

  const projectList = projects ?? []
  useKeyboardShortcut([{ key: 'ArrowDown' }, { key: 'j' }], () => setActiveRow((prev) => Math.min(prev + 1, projectList.length - 1)))
  useKeyboardShortcut([{ key: 'ArrowUp' }, { key: 'k' }], () => setActiveRow((prev) => Math.max(prev - 1, 0)))
  useKeyboardShortcut([{ key: 'Enter' }, { key: 'o' }], () => {
    if (activeRow >= 0 && activeRow < projectList.length) {
      navigate(`/projects/${projectList[activeRow].key}`)
    }
  }, activeRow >= 0)
  useKeyboardShortcut({ key: 'Escape' }, () => setActiveRow(-1), activeRow >= 0)

  const [name, setName] = useState('')
  const [key, setKey] = useState('')
  const [description, setDescription] = useState('')
  const [formError, setFormError] = useState('')

  const columns: Column<Project>[] = [
    {
      key: 'key',
      header: t('projects.table.key'),
      render: (p) => <Badge color="indigo">{p.key}</Badge>,
    },
    {
      key: 'name',
      header: t('projects.table.name'),
      render: (p) => <span className="font-medium text-gray-900 dark:text-gray-100">{p.name}</span>,
    },
    {
      key: 'items',
      header: t('projects.table.items'),
      render: (p) => <span className="text-gray-500 dark:text-gray-400">{p.item_counter}</span>,
    },
    {
      key: 'updated',
      header: t('projects.table.updated'),
      render: (p) => (
        <span className="text-gray-500 dark:text-gray-400">{new Date(p.updated_at).toLocaleDateString()}</span>
      ),
    },
  ]

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
            setFormError(axiosErr.response?.data?.error?.message ?? t('projects.createError'))
          } else {
            setFormError(t('projects.createError'))
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
        <p className="text-red-600">{t('projects.loadError')}</p>
      </div>
    )
  }

  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{t('projects.title')}</h1>
        <Button onClick={() => setShowCreate(true)}>{t('projects.new')}</Button>
      </div>
      <div className="bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
        <DataTable
          columns={columns}
          data={projectList}
          onRowClick={(p) => navigate(`/projects/${p.key}`)}
          emptyMessage={t('projects.empty')}
          activeRowIndex={activeRow}
        />
      </div>

      <Modal open={showCreate} onClose={() => setShowCreate(false)} title={t('projects.create.title')}>
        <form onSubmit={handleCreate} className="space-y-4">
          <Input
            label={t('projects.create.name')}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('projects.create.namePlaceholder')}
            required
          />
          <Input
            label={t('projects.create.key')}
            value={key}
            onChange={(e) => setKey(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ''))}
            placeholder={t('projects.create.keyPlaceholder')}
            maxLength={10}
            required
          />
          <p className="text-xs text-gray-400 dark:text-gray-500 -mt-3">{t('projects.create.keyHint')}</p>
          <Input
            label={t('projects.create.description')}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t('projects.create.descriptionPlaceholder')}
          />
          {formError && <p className="text-sm text-red-600 dark:text-red-400">{formError}</p>}
          <div className="flex justify-end gap-3 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowCreate(false)}>{t('common.cancel')}</Button>
            <Button type="submit" disabled={createMutation.isPending || !name.trim() || !key.trim()}>
              {createMutation.isPending ? t('common.creating') : t('common.create')}
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  )
}
