import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { useProjects, useCreateProject } from '@/hooks/useProjects'
import { DataTable } from '@/components/ui/DataTable'
import { Spinner } from '@/components/ui/Spinner'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { Tooltip } from '@/components/ui/Tooltip'
import type { Column } from '@/components/ui/DataTable'
import type { Project } from '@/api/projects'

function IconOpen({ className }: { className?: string }) {
  // Inbox/tray icon — items waiting to be picked up
  return (
    <svg className={className} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M2 9h3.5l1.5 2h2l1.5-2H14" />
      <path d="M3.5 3h9l1.5 6v4H2V9z" />
    </svg>
  )
}

function IconInProgress({ className }: { className?: string }) {
  // Wrench icon — work in progress
  return (
    <svg className={className} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M10.5 2.5a4 4 0 0 0-4.2 6.4L2.5 12.7a1 1 0 0 0 0 1.4l.4.4a1 1 0 0 0 1.4 0l3.8-3.8A4 4 0 0 0 14.5 6.5l-2.5 2.5-1.5-1.5L13 5a4 4 0 0 0-2.5-2.5z" />
    </svg>
  )
}

function IconTotal({ className }: { className?: string }) {
  // Sigma/sum icon
  return (
    <svg className={className} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 3H4.5l4 5-4 5H12" />
    </svg>
  )
}

function IconMembers({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round">
      <circle cx="6" cy="5.5" r="2.5" />
      <path d="M1.5 14c0-2.5 2-4 4.5-4s4.5 1.5 4.5 4" />
      <circle cx="11.5" cy="5.5" r="2" />
      <path d="M14.5 14c0-2 -1.5-3.5-3-3.5" />
    </svg>
  )
}

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
      width: '110px',
      render: (p) => (
        <ProjectKeyBadge>{p.key}</ProjectKeyBadge>
      ),
    },
    {
      key: 'name',
      header: t('projects.table.name'),
      render: (p) => <span className="font-medium text-gray-900 dark:text-gray-100 truncate block">{p.name}</span>,
    },
    {
      key: 'open',
      header: t('projects.table.open'),
      width: '12%',
      className: 'text-right',
      render: (p) => <span className="text-gray-500 dark:text-gray-400">{p.open_count}</span>,
    },
    {
      key: 'in_progress',
      header: t('projects.table.inProgress'),
      width: '12%',
      className: 'text-right',
      render: (p) => <span className="text-gray-500 dark:text-gray-400">{p.in_progress_count}</span>,
    },
    {
      key: 'total',
      header: t('projects.table.total'),
      width: '12%',
      className: 'text-right',
      render: (p) => <span className="text-gray-500 dark:text-gray-400">{p.item_counter}</span>,
    },
    {
      key: 'members',
      header: t('projects.table.members'),
      width: '12%',
      className: 'text-right',
      render: (p) => <span className="text-gray-500 dark:text-gray-400">{p.member_count}</span>,
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

      {/* Mobile card view */}
      <div className="sm:hidden space-y-3">
        {projectList.length === 0 ? (
          <p className="text-center text-sm text-gray-500 dark:text-gray-400 py-12">{t('projects.empty')}</p>
        ) : (
          projectList.map((p) => (
            <button
              key={p.key}
              onClick={() => navigate(`/projects/${p.key}`)}
              className="w-full text-left bg-white dark:bg-gray-800 rounded-lg shadow p-4 active:bg-gray-50 dark:active:bg-gray-700 transition-colors"
            >
              <div className="flex items-center gap-3">
                <ProjectKeyBadge>{p.key}</ProjectKeyBadge>
                <span className="font-medium text-gray-900 dark:text-gray-100 truncate">{p.name}</span>
              </div>
              <div className="flex items-center gap-4 mt-2.5 ml-1 text-xs text-gray-500 dark:text-gray-400">
                <Tooltip content={t('projects.table.open')}>
                  <span className="inline-flex items-center gap-1">
                    <IconOpen className="w-3.5 h-3.5" />
                    {p.open_count}
                  </span>
                </Tooltip>
                <Tooltip content={t('projects.table.inProgress')}>
                  <span className="inline-flex items-center gap-1">
                    <IconInProgress className="w-3.5 h-3.5" />
                    {p.in_progress_count}
                  </span>
                </Tooltip>
                <Tooltip content={t('projects.table.total')}>
                  <span className="inline-flex items-center gap-1">
                    <IconTotal className="w-3.5 h-3.5" />
                    {p.item_counter}
                  </span>
                </Tooltip>
                <Tooltip content={t('projects.table.members')}>
                  <span className="inline-flex items-center gap-1">
                    <IconMembers className="w-3.5 h-3.5" />
                    {p.member_count}
                  </span>
                </Tooltip>
              </div>
            </button>
          ))
        )}
      </div>

      {/* Desktop table view */}
      <div className="hidden sm:block bg-white dark:bg-gray-800 rounded-lg shadow overflow-hidden">
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
            maxLength={5}
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
