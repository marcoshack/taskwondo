import { useEffect, useMemo, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { FolderKanban, Building2, Users, HardDrive, Settings, X, Plus } from 'lucide-react'
import { ScrollableRow } from '@/components/ui/ScrollableRow'
import { useAdminProjects, useAdminNamespaces, useAdminStats } from '@/hooks/useAdmin'
import { useSystemSetting, useSetSystemSetting, usePublicSettings } from '@/hooks/useSystemSettings'
import { useDebounce } from '@/hooks/useDebounce'
import { DataTable } from '@/components/ui/DataTable'
import { Badge } from '@/components/ui/Badge'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { Spinner } from '@/components/ui/Spinner'
import { Tabs } from '@/components/ui/Tabs'
import { formatRelativeTime } from '@/utils/duration'
import type { Column } from '@/components/ui/DataTable'
import type { AdminProject, AdminNamespace } from '@/api/admin'
import { DirectoryUsersTab } from './DirectoryUsersTab'

const VALID_TABS = ['users', 'projects', 'namespaces', 'settings'] as const
type DirectoryTab = (typeof VALID_TABS)[number]

function formatStorageBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`
  return `${(bytes / 1073741824).toFixed(1)} GB`
}

const PROJECT_SORT_ACCESSORS: Record<string, (p: AdminProject) => string | number> = {
  key: (p) => p.key,
  name: (p) => p.name.toLowerCase(),
  namespace: (p) => p.namespace_display_name.toLowerCase(),
  owner: (p) => p.owner_display_name.toLowerCase(),
  members: (p) => p.member_count,
  items: (p) => p.item_count,
  storage: (p) => p.storage_bytes,
  created: (p) => p.created_at,
}

const NAMESPACE_SORT_ACCESSORS: Record<string, (n: AdminNamespace) => string | number> = {
  slug: (n) => n.slug,
  displayName: (n) => n.display_name.toLowerCase(),
  default: (n) => (n.is_default ? 0 : 1),
  projects: (n) => n.project_count,
  members: (n) => n.member_count,
  storage: (n) => n.storage_bytes,
  created: (n) => n.created_at,
}

function compareValues(a: string | number, b: string | number, order: 'asc' | 'desc'): number {
  let cmp: number
  if (typeof a === 'number' && typeof b === 'number') cmp = a - b
  else cmp = String(a).localeCompare(String(b))
  return order === 'asc' ? cmp : -cmp
}

export function SystemDirectoryPage() {
  const { t } = useTranslation()
  const [searchParams, setSearchParams] = useSearchParams()
  const tabParam = searchParams.get('tab')
  const activeTab: DirectoryTab = (VALID_TABS as readonly string[]).includes(tabParam ?? '')
    ? (tabParam as DirectoryTab)
    : 'users'

  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)
  const statsQuery = useAdminStats()
  const [projectsSortBy, setProjectsSortBy] = useState<string | null>(null)
  const [projectsSortOrder, setProjectsSortOrder] = useState<'asc' | 'desc'>('asc')
  const [namespacesSortBy, setNamespacesSortBy] = useState<string | null>(null)
  const [namespacesSortOrder, setNamespacesSortOrder] = useState<'asc' | 'desc'>('asc')

  const tabs = [
    { key: 'users', label: t('admin.directory.tab.users') },
    { key: 'projects', label: t('admin.directory.tab.projects') },
    { key: 'namespaces', label: t('admin.directory.tab.namespaces') },
    { key: 'settings', label: t('admin.directory.tab.settings'), icon: Settings },
  ]

  const projectsQuery = useAdminProjects(activeTab === 'projects' ? debouncedSearch : '')
  const namespacesQuery = useAdminNamespaces(activeTab === 'namespaces' ? debouncedSearch : '')

  const projects = useMemo(() => {
    const rows = projectsQuery.data?.pages.flatMap((p) => p.data) ?? []
    if (!projectsSortBy) return rows
    const accessor = PROJECT_SORT_ACCESSORS[projectsSortBy]
    if (!accessor) return rows
    return [...rows].sort((a, b) => compareValues(accessor(a), accessor(b), projectsSortOrder))
  }, [projectsQuery.data, projectsSortBy, projectsSortOrder])

  const namespaces = useMemo(() => {
    const rows = namespacesQuery.data?.pages.flatMap((p) => p.data) ?? []
    if (!namespacesSortBy) return rows
    const accessor = NAMESPACE_SORT_ACCESSORS[namespacesSortBy]
    if (!accessor) return rows
    return [...rows].sort((a, b) => compareValues(accessor(a), accessor(b), namespacesSortOrder))
  }, [namespacesQuery.data, namespacesSortBy, namespacesSortOrder])

  function handleProjectSort(key: string) {
    if (projectsSortBy === key) {
      setProjectsSortOrder(projectsSortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setProjectsSortBy(key)
      setProjectsSortOrder('asc')
    }
  }

  function handleNamespaceSort(key: string) {
    if (namespacesSortBy === key) {
      setNamespacesSortOrder(namespacesSortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setNamespacesSortBy(key)
      setNamespacesSortOrder('asc')
    }
  }

  // Reset the shared search input when switching tabs.
  useEffect(() => {
    setSearch('')
  }, [activeTab])

  const projectColumns: Column<AdminProject>[] = [
    {
      key: 'key',
      sortKey: 'key',
      header: t('admin.directory.col.key'),
      className: 'w-28',
      render: (row) => <ProjectKeyBadge>{row.key}</ProjectKeyBadge>,
    },
    {
      key: 'name',
      sortKey: 'name',
      header: t('admin.directory.col.name'),
      hiddenOnMobile: true,
      render: (row) => <span className="text-[var(--foreground)] font-medium">{row.name}</span>,
    },
    {
      key: 'namespace',
      sortKey: 'namespace',
      header: t('admin.directory.col.namespace'),
      hiddenOnMobile: true,
      render: (row) => (
        <span className="text-[var(--foreground-secondary)]">{row.namespace_display_name}</span>
      ),
    },
    {
      key: 'owner',
      sortKey: 'owner',
      header: t('admin.directory.col.owner'),
      hiddenOnMobile: true,
      render: (row) => (
        <div>
          <div className="text-[var(--foreground)]">{row.owner_display_name}</div>
          <div className="text-xs text-[var(--foreground-secondary)]">{row.owner_email}</div>
        </div>
      ),
    },
    {
      key: 'members',
      sortKey: 'members',
      header: t('admin.directory.col.members'),
      className: 'w-24',
      render: (row) => <span className="text-[var(--foreground-secondary)]">{row.member_count}</span>,
    },
    {
      key: 'items',
      sortKey: 'items',
      header: t('admin.directory.col.items'),
      className: 'w-20',
      render: (row) => <span className="text-[var(--foreground-secondary)]">{row.item_count}</span>,
    },
    {
      key: 'storage',
      sortKey: 'storage',
      header: t('admin.directory.col.storage'),
      className: 'w-24',
      render: (row) => <span className="text-[var(--foreground-secondary)]">{formatStorageBytes(row.storage_bytes)}</span>,
    },
    {
      key: 'created',
      sortKey: 'created',
      header: t('admin.directory.col.created'),
      className: 'w-28',
      hiddenOnMobile: true,
      render: (row) => <span className="text-[var(--foreground-secondary)]">{formatRelativeTime(row.created_at)}</span>,
    },
  ]

  const namespaceColumns: Column<AdminNamespace>[] = [
    {
      key: 'slug',
      sortKey: 'slug',
      header: t('admin.directory.col.slug'),
      className: 'w-32',
      render: (row) => <ProjectKeyBadge>{row.slug}</ProjectKeyBadge>,
    },
    {
      key: 'displayName',
      sortKey: 'displayName',
      header: t('admin.directory.col.displayName'),
      hiddenOnMobile: true,
      render: (row) => <span className="text-[var(--foreground)] font-medium">{row.display_name}</span>,
    },
    {
      key: 'default',
      sortKey: 'default',
      header: t('admin.directory.col.default'),
      className: 'w-24',
      render: (row) => row.is_default ? <Badge color="green">{t('admin.directory.col.defaultBadge')}</Badge> : null,
    },
    {
      key: 'projects',
      sortKey: 'projects',
      header: t('admin.directory.col.projects'),
      className: 'w-24',
      render: (row) => <span className="text-[var(--foreground-secondary)]">{row.project_count}</span>,
    },
    {
      key: 'members',
      sortKey: 'members',
      header: t('admin.directory.col.members'),
      className: 'w-24',
      render: (row) => <span className="text-[var(--foreground-secondary)]">{row.member_count}</span>,
    },
    {
      key: 'storage',
      sortKey: 'storage',
      header: t('admin.directory.col.storage'),
      className: 'w-24',
      hiddenOnMobile: true,
      render: (row) => <span className="text-[var(--foreground-secondary)]">{formatStorageBytes(row.storage_bytes)}</span>,
    },
    {
      key: 'created',
      sortKey: 'created',
      header: t('admin.directory.col.created'),
      className: 'w-28',
      hiddenOnMobile: true,
      render: (row) => <span className="text-[var(--foreground-secondary)]">{formatRelativeTime(row.created_at)}</span>,
    },
  ]

  const isLoading = activeTab === 'projects' ? projectsQuery.isLoading : namespacesQuery.isLoading
  const hasNextPage = activeTab === 'projects' ? projectsQuery.hasNextPage : namespacesQuery.hasNextPage
  const isFetchingNextPage = activeTab === 'projects' ? projectsQuery.isFetchingNextPage : namespacesQuery.isFetchingNextPage
  const fetchNextPage = activeTab === 'projects' ? projectsQuery.fetchNextPage : namespacesQuery.fetchNextPage

  function handleTabChange(key: string) {
    const params = new URLSearchParams(searchParams)
    if (key === 'users') {
      params.delete('tab')
    } else {
      params.set('tab', key)
    }
    setSearchParams(params, { replace: true })
  }

  const statCards = [
    { icon: FolderKanban, value: statsQuery.data?.projects ?? '-', label: t('admin.directory.stats.projects') },
    { icon: Building2, value: statsQuery.data?.namespaces ?? '-', label: t('admin.directory.stats.namespaces') },
    { icon: Users, value: statsQuery.data?.users ?? '-', label: t('admin.directory.stats.users') },
    { icon: HardDrive, value: statsQuery.data ? formatStorageBytes(statsQuery.data.storage_bytes) : '-', label: t('admin.directory.stats.storage') },
  ]

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold text-[var(--foreground)]">{t('admin.directory.title')}</h2>
          <p className="mt-1 text-sm text-[var(--foreground-secondary)]">{t('admin.directory.description')}</p>
        </div>
        {/* Mobile: single bordered row with horizontal scroll */}
        <ScrollableRow className="sm:hidden rounded-lg border border-[var(--border)] px-3 py-2" gradientFrom="from-white dark:from-gray-900">
          {statCards.map((card, i) => (
            <div key={card.label} className={`flex items-center gap-1.5 shrink-0 ${i < statCards.length - 1 ? 'pr-3 border-r border-[var(--border)]' : ''}`}>
              <card.icon className="h-3.5 w-3.5 text-[var(--primary)]" />
              <span className="text-sm font-bold text-[var(--foreground)]">{card.value}</span>
              <span className="text-xs text-[var(--foreground-secondary)]">{card.label}</span>
            </div>
          ))}
        </ScrollableRow>
        {/* Desktop: individual cards */}
        <div className="hidden sm:grid grid-cols-4 gap-3">
          {statCards.map((card) => (
            <div key={card.label} className="rounded-lg border border-[var(--border)] p-3 text-center min-w-[100px]">
              <div className="flex items-center justify-center gap-2">
                <card.icon className="h-4 w-4 text-[var(--primary)]" />
                <span className="text-xl font-bold text-[var(--foreground)]">{card.value}</span>
              </div>
              <div className="text-xs text-[var(--foreground-secondary)] mt-0.5">{card.label}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="mt-6">
        <Tabs tabs={tabs} activeTab={activeTab} onTabChange={handleTabChange} />
      </div>

      <div className="mt-4">
        {activeTab === 'users' ? (
          <DirectoryUsersTab />
        ) : activeTab === 'settings' ? (
          <SettingsTab />
        ) : (
          <>
            <div className="max-w-sm mb-4">
              <Input
                placeholder={activeTab === 'projects' ? t('admin.directory.search.projects') : t('admin.directory.search.namespaces')}
                value={search}
                onChange={(e) => setSearch(e.target.value)}
              />
            </div>
            {isLoading ? (
              <div className="flex justify-center py-12">
                <Spinner />
              </div>
            ) : activeTab === 'projects' ? (
              <DataTable
                columns={projectColumns}
                data={projects}
                emptyMessage={t('admin.directory.empty.projects')}
                alwaysShowHeader
                sortBy={projectsSortBy ?? undefined}
                sortOrder={projectsSortOrder}
                onSort={handleProjectSort}
              />
            ) : (
              <DataTable
                columns={namespaceColumns}
                data={namespaces}
                emptyMessage={t('admin.directory.empty.namespaces')}
                alwaysShowHeader
                sortBy={namespacesSortBy ?? undefined}
                sortOrder={namespacesSortOrder}
                onSort={handleNamespaceSort}
              />
            )}

            {hasNextPage && (
              <div className="flex justify-center pt-4">
                <Button
                  variant="secondary"
                  onClick={() => fetchNextPage()}
                  disabled={isFetchingNextPage}
                >
                  {isFetchingNextPage ? t('admin.directory.loading') : t('admin.directory.loadMore')}
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </div>
  )
}

function SettingsTab() {
  const { t } = useTranslation()
  const { data: publicSettings } = usePublicSettings()
  const namespacesEnabled = publicSettings?.namespaces_enabled === true

  return (
    <div className="space-y-6">
      <ProjectLimitSection />
      {namespacesEnabled && <NamespaceLimitSection />}
      <ReservedListSection
        settingKey="reserved_namespace_slugs"
        title={t('admin.directory.settings.reservedSlugs.title')}
        description={t('admin.directory.settings.reservedSlugs.description')}
        placeholder={t('admin.directory.settings.reservedSlugs.placeholder')}
        emptyMessage={t('admin.directory.settings.reservedSlugs.empty')}
        normalize={(v) => v.toLowerCase().replace(/[^a-z0-9-]/g, '')}
      />
      <ReservedListSection
        settingKey="reserved_project_keys"
        title={t('admin.directory.settings.reservedKeys.title')}
        description={t('admin.directory.settings.reservedKeys.description')}
        placeholder={t('admin.directory.settings.reservedKeys.placeholder')}
        emptyMessage={t('admin.directory.settings.reservedKeys.empty')}
        normalize={(v) => v.toUpperCase().replace(/[^A-Z0-9]/g, '')}
      />
    </div>
  )
}

function ProjectLimitSection() {
  const { t } = useTranslation()
  const { data: savedValue, isLoading } = useSystemSetting<number>('max_projects_per_user')
  const setSetting = useSetSystemSetting()
  const [value, setValue] = useState('')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (savedValue !== undefined) {
      setValue(savedValue != null ? String(savedValue) : '5')
    }
  }, [savedValue])

  const dirty = value !== (savedValue != null ? String(savedValue) : '5')

  function handleSave() {
    setSaved(false)
    const v = parseInt(value, 10)
    if (isNaN(v) || v < 0) return
    setSetting.mutate(
      { key: 'max_projects_per_user', value: v },
      { onSuccess: () => setSaved(true) },
    )
  }

  return (
    <div className="rounded-lg border border-[var(--border)] p-4 sm:p-6">
      <h3 className="text-sm font-medium text-[var(--foreground)]">
        {t('admin.general.projectLimit.title')}
      </h3>
      <p className="mt-1 text-xs text-[var(--foreground-secondary)]">
        {t('admin.general.projectLimit.help')}
      </p>
      <div className="mt-4 flex items-end gap-3">
        <Input
          label={t('admin.general.projectLimit.label')}
          type="number"
          min={0}
          value={value}
          onChange={(e) => { setValue(e.target.value); setSaved(false) }}
          className="w-24"
          disabled={isLoading}
        />
        <Button
          onClick={handleSave}
          disabled={!dirty || setSetting.isPending}
        >
          {setSetting.isPending ? t('common.saving') : t('common.save')}
        </Button>
        {saved && (
          <span className="text-sm text-green-600 dark:text-green-400 pb-2">
            {t('admin.general.projectLimit.saved')}
          </span>
        )}
      </div>
    </div>
  )
}

function NamespaceLimitSection() {
  const { t } = useTranslation()
  const { data: savedValue, isLoading } = useSystemSetting<number>('max_namespaces_per_user')
  const setSetting = useSetSystemSetting()
  const [value, setValue] = useState('')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (savedValue !== undefined) {
      setValue(savedValue != null ? String(savedValue) : '1')
    }
  }, [savedValue])

  const dirty = value !== (savedValue != null ? String(savedValue) : '1')

  function handleSave() {
    setSaved(false)
    const v = parseInt(value, 10)
    if (isNaN(v) || v < 0) return
    setSetting.mutate(
      { key: 'max_namespaces_per_user', value: v },
      { onSuccess: () => setSaved(true) },
    )
  }

  return (
    <div className="rounded-lg border border-[var(--border)] p-4 sm:p-6">
      <h3 className="text-sm font-medium text-[var(--foreground)]">
        {t('admin.general.namespaceLimit.title')}
      </h3>
      <p className="mt-1 text-xs text-[var(--foreground-secondary)]">
        {t('admin.general.namespaceLimit.help')}
      </p>
      <div className="mt-4 flex items-end gap-3">
        <Input
          label={t('admin.general.namespaceLimit.label')}
          type="number"
          min={0}
          value={value}
          onChange={(e) => { setValue(e.target.value); setSaved(false) }}
          className="w-24"
          disabled={isLoading}
        />
        <Button
          onClick={handleSave}
          disabled={!dirty || setSetting.isPending}
        >
          {setSetting.isPending ? t('common.saving') : t('common.save')}
        </Button>
        {saved && (
          <span className="text-sm text-green-600 dark:text-green-400 pb-2">
            {t('admin.general.namespaceLimit.saved')}
          </span>
        )}
      </div>
    </div>
  )
}

function ReservedListSection({
  settingKey,
  title,
  description,
  placeholder,
  emptyMessage,
  normalize,
}: {
  settingKey: string
  title: string
  description: string
  placeholder: string
  emptyMessage: string
  normalize: (value: string) => string
}) {
  const { t } = useTranslation()
  const { data: items, isLoading } = useSystemSetting<string[]>(settingKey)
  const setSetting = useSetSystemSetting()
  const [input, setInput] = useState('')
  const [error, setError] = useState('')

  const list = items ?? []

  function handleAdd() {
    const value = normalize(input.trim())
    if (!value) return
    if (list.includes(value)) {
      setError(t('admin.directory.settings.duplicate'))
      return
    }
    setSetting.mutate({ key: settingKey, value: [...list, value] })
    setInput('')
    setError('')
  }

  function handleRemove(value: string) {
    setSetting.mutate({ key: settingKey, value: list.filter((v) => v !== value) })
  }

  return (
    <div className="rounded-lg border border-[var(--border)] p-4 sm:p-6">
      <h3 className="text-sm font-medium text-[var(--foreground)]">{title}</h3>
      <p className="mt-1 text-xs text-[var(--foreground-secondary)]">{description}</p>

      <div className="mt-4 flex gap-2">
        <Input
          value={input}
          onChange={(e) => { setInput(e.target.value); setError('') }}
          placeholder={placeholder}
          onKeyDown={(e) => e.key === 'Enter' && handleAdd()}
          className="max-w-xs"
        />
        <Button onClick={handleAdd} disabled={!input.trim() || setSetting.isPending} variant="secondary" size="sm">
          <Plus className="h-4 w-4 mr-1" />
          {t('admin.directory.settings.add')}
        </Button>
      </div>
      {error && <p className="mt-1 text-xs text-[var(--danger)]">{error}</p>}

      <div className="mt-3">
        {isLoading ? (
          <Spinner />
        ) : list.length === 0 ? (
          <p className="text-sm text-[var(--foreground-secondary)]">{emptyMessage}</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {list.map((item) => (
              <span
                key={item}
                className="inline-flex items-center gap-1 rounded-md bg-[var(--surface-secondary)] px-2.5 py-1 text-sm font-medium text-[var(--foreground)]"
              >
                {item}
                <button
                  onClick={() => handleRemove(item)}
                  className="ml-0.5 rounded p-0.5 hover:bg-[var(--surface-tertiary)] hover:bg-[var(--surface-hover)] text-[var(--foreground-muted)] hover:text-[var(--foreground-secondary)] hover:text-[var(--foreground)] transition-colors"
                >
                  <X className="h-3 w-3" />
                </button>
              </span>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
