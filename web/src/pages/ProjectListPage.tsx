import { useState, useMemo, useRef, useEffect } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Search, Plus, Check, FolderKanban } from 'lucide-react'
import { getLocalizedError } from '@/utils/apiError'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { useProjects, useCreateProject, useOwnedProjectCount, useMaxProjects } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLayout } from '@/contexts/LayoutContext'
import { AppSidebar } from '@/components/AppSidebar'
import { CreateNamespaceModal } from '@/components/CreateNamespaceModal'
import { NamespaceIcon } from '@/components/NamespaceIcon'
import { DataTable } from '@/components/ui/DataTable'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { Tooltip } from '@/components/ui/Tooltip'
import { PageHeader } from '@/components/ui/PageHeader'
import { EmptyState } from '@/components/ui/EmptyState'
import { LoadingState } from '@/components/ui/LoadingState'
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
  const { collapsed } = useSidebar('app')
  const { containerClass } = useLayout()
  const { data: projects, isLoading, error } = useProjects()
  const { data: ownedCount } = useOwnedProjectCount()
  const { data: maxProjectsValue } = useMaxProjects()
  const { user } = useAuth()
  const navigate = useNavigate()
  const { p } = useNamespacePath()
  const createMutation = useCreateProject()

  const isAdmin = user?.global_role === 'admin'
  const maxProjects = maxProjectsValue ?? 5
  const projectCount = ownedCount ?? 0
  const atLimit = !isAdmin && maxProjects > 0 && projectCount >= maxProjects

  // Customer-only users in this namespace cannot create projects
  const isCustomerOnly = !isAdmin
    && (projects ?? []).length > 0
    && (projects ?? []).every((proj) => proj.member_role === 'customer')

  const location = useLocation()

  // Auto-redirect: when arriving via login or namespace switch, if the user is
  // customer-only with exactly 1 project, go straight to that project's support page.
  useEffect(() => {
    if (
      location.state?.autoRedirect
      && projects
      && projects.length === 1
      && projects[0].member_role === 'customer'
    ) {
      navigate(p(`/projects/${projects[0].key}/support`), { replace: true })
    }
  }, [projects, location.state, navigate, p])

  const { namespaces, activeNamespace, showSwitcher } = useNamespaceContext()

  const [showCreate, setShowCreate] = useState(false)
  const [activeRow, setActiveRow] = useState(-1)
  const [searchInput, setSearchInput] = useState('')
  const searchRef = useRef<HTMLInputElement>(null)
  const [name, setName] = useState('')
  const [key, setKey] = useState('')
  const [description, setDescription] = useState('')
  const [selectedNamespace, setSelectedNamespace] = useState(activeNamespace?.slug ?? '')
  const [formError, setFormError] = useState('')
  const [showNsPicker, setShowNsPicker] = useState(false)
  const [showNsCreate, setShowNsCreate] = useState(false)

  const selectedNs = namespaces.find((ns) => ns.slug === selectedNamespace) ?? activeNamespace

  const openCreateModal = () => {
    setSelectedNamespace(activeNamespace?.slug ?? '')
    setShowCreate(true)
  }

  useKeyboardShortcut({ key: 'n' }, () => openCreateModal(), !isCustomerOnly)
  useKeyboardShortcut({ key: '/' }, () => searchRef.current?.focus())

  const projectList = useMemo(() => {
    const all = projects ?? []
    if (!searchInput.trim()) return all
    const q = searchInput.trim().toLowerCase()
    return all.filter((p) => p.name.toLowerCase().includes(q) || p.key.toLowerCase().includes(q))
  }, [projects, searchInput])
  useKeyboardShortcut([{ key: 'ArrowDown' }, { key: 'j' }], () => setActiveRow((prev) => Math.min(prev + 1, projectList.length - 1)))
  useKeyboardShortcut([{ key: 'ArrowUp' }, { key: 'k' }], () => setActiveRow((prev) => Math.max(prev - 1, 0)))
  useKeyboardShortcut([{ key: 'Enter' }, { key: 'o' }], () => {
    if (activeRow >= 0 && activeRow < projectList.length) {
      const proj = projectList[activeRow]
      navigate(p(`/projects/${proj.key}${proj.member_role === 'customer' ? '/support' : ''}`))
    }
  }, activeRow >= 0)
  useKeyboardShortcut({ key: 'Escape' }, () => setActiveRow(-1), activeRow >= 0)

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
      render: (p) => <span className="font-medium text-[var(--foreground)] truncate block">{p.name}</span>,
    },
    {
      key: 'role',
      header: t('projects.table.role'),
      width: '10%',
      className: 'text-right',
      render: (p) => (
        <span className="text-[var(--foreground-secondary)] capitalize">
          {p.member_role ? t(`projects.settings.roles.${p.member_role}`) : ''}
        </span>
      ),
    },
    {
      key: 'open',
      header: t('projects.table.open'),
      width: '12%',
      className: 'text-right',
      render: (p) => p.member_role === 'customer'
        ? <span className="text-[var(--foreground-secondary)]">&mdash;</span>
        : <span className="text-[var(--foreground-secondary)]">{p.open_count}</span>,
    },
    {
      key: 'in_progress',
      header: t('projects.table.inProgress'),
      width: '12%',
      className: 'text-right',
      render: (p) => p.member_role === 'customer'
        ? <span className="text-[var(--foreground-secondary)]">&mdash;</span>
        : <span className="text-[var(--foreground-secondary)]">{p.in_progress_count}</span>,
    },
    {
      key: 'total',
      header: t('projects.table.total'),
      width: '12%',
      className: 'text-right',
      render: (p) => p.member_role === 'customer'
        ? <span className="text-[var(--foreground-secondary)]">&mdash;</span>
        : <span className="text-[var(--foreground-secondary)]">{p.item_counter}</span>,
    },
    {
      key: 'members',
      header: t('projects.table.members'),
      width: '12%',
      className: 'text-right',
      render: (p) => p.member_role === 'customer'
        ? <span className="text-[var(--foreground-secondary)]">&mdash;</span>
        : <span className="text-[var(--foreground-secondary)]">{p.member_count}</span>,
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
          navigate(p(`/projects/${project.key}`))
        },
        onError: (err) => {
          setFormError(getLocalizedError(err, t, 'projects.createError'))
        },
      },
    )
  }

  if (isLoading) {
    return (
      <div className={`${containerClass(true)} py-6`}>
        <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
          <AppSidebar />
          <div className="flex-1 min-w-0">
            <LoadingState />
          </div>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={`${containerClass(true)} py-6`}>
        <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
          <AppSidebar />
          <div className="flex-1 min-w-0">
            <p className="text-[var(--danger)]">{t('projects.loadError')}</p>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div className={`${containerClass(true)} py-6`}>
      <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <AppSidebar />
        <div className="flex-1 min-w-0">
      <PageHeader
        title={t('projects.title')}
        actions={!isCustomerOnly ? <Button onClick={openCreateModal} className="border border-transparent">{t('projects.new')}</Button> : undefined}
      />

      {/* Search */}
      <div className="relative mb-6">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-[var(--foreground-muted)]" />
        <Input
          ref={searchRef}
          value={searchInput}
          onChange={(e) => setSearchInput(e.target.value)}
          placeholder={t('projects.searchPlaceholder')}
          className="pl-10"
          onKeyDown={(e) => { if (e.key === 'Escape') searchRef.current?.blur() }}
        />
      </div>

      {/* Mobile card view */}
      <div className="sm:hidden space-y-3">
        {projectList.length === 0 ? (
          <EmptyState
            icon={FolderKanban}
            title={t('projects.empty')}
            description={!isCustomerOnly ? t('projects.emptyDescription') : undefined}
            action={!isCustomerOnly ? <Button onClick={openCreateModal}>{t('projects.new')}</Button> : undefined}
          />
        ) : (
          projectList.map((proj) => {
            const isCustomer = proj.member_role === 'customer'
            return (
              <button
                key={proj.key}
                onClick={() => navigate(p(`/projects/${proj.key}${isCustomer ? '/support' : ''}`))}
                className="w-full text-left bg-[var(--surface)] rounded-lg border border-[var(--border)] p-4 active:bg-[var(--surface-secondary)] dark:active:bg-[var(--surface-secondary)] transition-colors"
              >
                <div className="flex items-center gap-3">
                  <ProjectKeyBadge>{proj.key}</ProjectKeyBadge>
                  <span className="font-medium text-[var(--foreground)] truncate flex-1">{proj.name}</span>
                  {proj.member_role && (
                    <span className="text-xs text-[var(--foreground-muted)] capitalize shrink-0">
                      {t(`projects.settings.roles.${proj.member_role}`)}
                    </span>
                  )}
                </div>
                {!isCustomer && (
                  <div className="flex items-center gap-4 mt-2.5 ml-1 text-xs text-[var(--foreground-secondary)]">
                    <Tooltip content={t('projects.table.open')}>
                      <span className="inline-flex items-center gap-1">
                        <IconOpen className="w-3.5 h-3.5" />
                        {proj.open_count}
                      </span>
                    </Tooltip>
                    <Tooltip content={t('projects.table.inProgress')}>
                      <span className="inline-flex items-center gap-1">
                        <IconInProgress className="w-3.5 h-3.5" />
                        {proj.in_progress_count}
                      </span>
                    </Tooltip>
                    <Tooltip content={t('projects.table.total')}>
                      <span className="inline-flex items-center gap-1">
                        <IconTotal className="w-3.5 h-3.5" />
                        {proj.item_counter}
                      </span>
                    </Tooltip>
                    <Tooltip content={t('projects.table.members')}>
                      <span className="inline-flex items-center gap-1">
                        <IconMembers className="w-3.5 h-3.5" />
                        {proj.member_count}
                      </span>
                    </Tooltip>
                  </div>
                )}
              </button>
            )
          })
        )}
      </div>

      {/* Desktop table view */}
      <div className="hidden sm:block bg-[var(--surface)] rounded-lg border border-[var(--border)] overflow-hidden">
        <DataTable
          columns={columns}
          data={projectList}
          onRowClick={(proj) => navigate(p(`/projects/${proj.key}${proj.member_role === 'customer' ? '/support' : ''}`))}
          emptyMessage={t('projects.empty')}
          activeRowIndex={activeRow}
        />
      </div>

      <Modal
        open={showCreate}
        onClose={() => setShowCreate(false)}
        title={
          <span className="flex items-center gap-3">
            {t('projects.create.title')}
            {!isAdmin && maxProjects > 0 && (
              <span className={`text-sm font-normal ${atLimit ? 'text-amber-500 dark:text-amber-400' : 'text-[var(--foreground-secondary)]'}`}>
                {t('projects.limitCounter', { count: projectCount, limit: maxProjects })}
              </span>
            )}
          </span>
        }
      >
        {atLimit && (
          <p className="text-sm text-amber-600 dark:text-amber-400 mb-4">{t('projects.limitReached')}</p>
        )}
        <form onSubmit={handleCreate} className="space-y-4">
          <Input
            label={t('projects.create.name')}
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder={t('projects.create.namePlaceholder')}
            required
            disabled={atLimit}
          />
          <Input
            label={t('projects.create.key')}
            value={key}
            onChange={(e) => setKey(e.target.value.toUpperCase().replace(/[^A-Z0-9]/g, ''))}
            placeholder={t('projects.create.keyPlaceholder')}
            maxLength={5}
            required
            disabled={atLimit}
          />
          <p className="text-xs text-[var(--foreground-muted)] -mt-3">{t('projects.create.keyHint')}</p>
          <Input
            label={t('projects.create.description')}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            placeholder={t('projects.create.descriptionPlaceholder')}
            disabled={atLimit}
          />
          {showSwitcher && (
            <div>
              <label className="block text-sm font-medium text-[var(--foreground)] mb-1">
                {t('namespaces.namespace')}
              </label>
              <button
                type="button"
                onClick={() => setShowNsPicker(true)}
                disabled={atLimit}
                className="w-full flex items-center gap-2.5 rounded-md border border-[var(--border)] bg-[var(--surface)] px-3 py-2 text-sm text-left hover:bg-[var(--surface-hover)] transition-colors"
              >
                <NamespaceIcon icon={selectedNs?.icon ?? 'globe'} color={selectedNs?.color} className="h-4 w-4 shrink-0" />
                <span className="flex-1 truncate text-[var(--foreground)]">
                  {selectedNs?.display_name ?? t('namespaces.selectNamespace')}
                </span>
              </button>
            </div>
          )}
          {formError && <p className="text-sm text-[var(--danger)]">{formError}</p>}
          <div className="flex justify-end gap-3 pt-2">
            <Button type="button" variant="secondary" onClick={() => setShowCreate(false)}>{t('common.cancel')}</Button>
            <Button type="submit" disabled={atLimit || createMutation.isPending || !name.trim() || !key.trim()}>
              {createMutation.isPending ? t('common.creating') : t('common.create')}
            </Button>
          </div>
        </form>
      </Modal>

      {/* Namespace picker modal (overlays New Project modal) */}
      <Modal
        open={showNsPicker}
        onClose={() => setShowNsPicker(false)}
        title={t('namespaces.selectNamespace')}
      >
        <div className="space-y-1">
          {namespaces.map((ns) => (
            <button
              key={ns.slug}
              onClick={() => {
                setSelectedNamespace(ns.slug)
                setShowNsPicker(false)
              }}
              className={`w-full text-left px-3 py-3 rounded-md text-sm flex items-center gap-3 ${
                ns.slug === selectedNamespace
                  ? 'bg-[var(--primary-muted)] text-[var(--primary)]'
                  : 'text-[var(--foreground)] hover:bg-[var(--surface-hover)]'
              }`}
            >
              <NamespaceIcon icon={ns.icon ?? 'globe'} color={ns.color} className="h-5 w-5 shrink-0" />
              <div className="min-w-0 flex-1">
                <div className="font-medium">{ns.display_name}</div>
                <p className="text-xs text-[var(--foreground-muted)] mt-0.5">
                  {ns.is_default ? t('namespaces.publicHint') : t('namespaces.customHint')}
                </p>
              </div>
              {ns.slug === selectedNamespace && (
                <Check className="h-4 w-4 shrink-0 text-[var(--primary)]" />
              )}
            </button>
          ))}
        </div>
        <div className="border-t border-[var(--border)] mt-3 pt-3">
          <button
            onClick={() => setShowNsCreate(true)}
            className="w-full text-left px-3 py-2.5 rounded-md text-sm text-[var(--foreground-secondary)] hover:bg-[var(--surface-hover)] flex items-center gap-3"
          >
            <Plus className="h-5 w-5" />
            {t('namespaces.createNew')}
          </button>
        </div>
      </Modal>

      <CreateNamespaceModal
        open={showNsCreate}
        onClose={() => setShowNsCreate(false)}
        onCreated={(created) => {
          setSelectedNamespace(created.slug)
          setShowNsCreate(false)
          setShowNsPicker(false)
        }}
      />
        </div>
      </div>
    </div>
  )
}
