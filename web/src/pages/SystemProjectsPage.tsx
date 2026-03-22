import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { FolderKanban, Building2, Users, HardDrive, Settings, X, Plus } from 'lucide-react'
import { ScrollableRow } from '@/components/ui/ScrollableRow'
import { useAdminProjects, useAdminNamespaces, useAdminStats } from '@/hooks/useAdmin'
import { useSystemSetting, useSetSystemSetting } from '@/hooks/useSystemSettings'
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

function formatStorageBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`
  return `${(bytes / 1073741824).toFixed(1)} GB`
}

export function SystemProjectsPage() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('projects')
  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)
  const statsQuery = useAdminStats()

  const tabs = [
    { key: 'projects', label: t('admin.projects.tab.projects') },
    { key: 'namespaces', label: t('admin.projects.tab.namespaces') },
    { key: 'settings', label: t('admin.projects.tab.settings'), icon: Settings },
  ]

  const projectsQuery = useAdminProjects(activeTab === 'projects' ? debouncedSearch : '')
  const namespacesQuery = useAdminNamespaces(activeTab === 'namespaces' ? debouncedSearch : '')

  const projects = useMemo(
    () => projectsQuery.data?.pages.flatMap((p) => p.data) ?? [],
    [projectsQuery.data],
  )

  const namespaces = useMemo(
    () => namespacesQuery.data?.pages.flatMap((p) => p.data) ?? [],
    [namespacesQuery.data],
  )

  const projectColumns: Column<AdminProject>[] = [
    {
      key: 'key',
      header: t('admin.projects.col.key'),
      className: 'w-28',
      render: (row) => <ProjectKeyBadge>{row.key}</ProjectKeyBadge>,
    },
    {
      key: 'name',
      header: t('admin.projects.col.name'),
      hiddenOnMobile: true,
      render: (row) => <span className="text-gray-900 dark:text-gray-100 font-medium">{row.name}</span>,
    },
    {
      key: 'namespace',
      header: t('admin.projects.col.namespace'),
      hiddenOnMobile: true,
      render: (row) => (
        <span className="text-gray-600 dark:text-gray-400">{row.namespace_display_name}</span>
      ),
    },
    {
      key: 'owner',
      header: t('admin.projects.col.owner'),
      hiddenOnMobile: true,
      render: (row) => (
        <div>
          <div className="text-gray-900 dark:text-gray-100">{row.owner_display_name}</div>
          <div className="text-xs text-gray-500 dark:text-gray-400">{row.owner_email}</div>
        </div>
      ),
    },
    {
      key: 'members',
      header: t('admin.projects.col.members'),
      className: 'w-24',
      render: (row) => <span className="text-gray-600 dark:text-gray-400">{row.member_count}</span>,
    },
    {
      key: 'items',
      header: t('admin.projects.col.items'),
      className: 'w-20',
      render: (row) => <span className="text-gray-600 dark:text-gray-400">{row.item_count}</span>,
    },
    {
      key: 'storage',
      header: t('admin.projects.col.storage'),
      className: 'w-24',
      render: (row) => <span className="text-gray-600 dark:text-gray-400">{formatStorageBytes(row.storage_bytes)}</span>,
    },
    {
      key: 'created',
      header: t('admin.projects.col.created'),
      className: 'w-28',
      hiddenOnMobile: true,
      render: (row) => <span className="text-gray-500 dark:text-gray-400">{formatRelativeTime(row.created_at)}</span>,
    },
  ]

  const namespaceColumns: Column<AdminNamespace>[] = [
    {
      key: 'slug',
      header: t('admin.projects.col.slug'),
      className: 'w-32',
      render: (row) => <ProjectKeyBadge>{row.slug}</ProjectKeyBadge>,
    },
    {
      key: 'displayName',
      header: t('admin.projects.col.displayName'),
      hiddenOnMobile: true,
      render: (row) => <span className="text-gray-900 dark:text-gray-100 font-medium">{row.display_name}</span>,
    },
    {
      key: 'default',
      header: t('admin.projects.col.default'),
      className: 'w-24',
      render: (row) => row.is_default ? <Badge color="green">{t('admin.projects.col.defaultBadge')}</Badge> : null,
    },
    {
      key: 'projects',
      header: t('admin.projects.col.projects'),
      className: 'w-24',
      render: (row) => <span className="text-gray-600 dark:text-gray-400">{row.project_count}</span>,
    },
    {
      key: 'members',
      header: t('admin.projects.col.members'),
      className: 'w-24',
      render: (row) => <span className="text-gray-600 dark:text-gray-400">{row.member_count}</span>,
    },
    {
      key: 'storage',
      header: t('admin.projects.col.storage'),
      className: 'w-24',
      hiddenOnMobile: true,
      render: (row) => <span className="text-gray-600 dark:text-gray-400">{formatStorageBytes(row.storage_bytes)}</span>,
    },
    {
      key: 'created',
      header: t('admin.projects.col.created'),
      className: 'w-28',
      hiddenOnMobile: true,
      render: (row) => <span className="text-gray-500 dark:text-gray-400">{formatRelativeTime(row.created_at)}</span>,
    },
  ]

  const isLoading = activeTab === 'projects' ? projectsQuery.isLoading : namespacesQuery.isLoading
  const hasNextPage = activeTab === 'projects' ? projectsQuery.hasNextPage : namespacesQuery.hasNextPage
  const isFetchingNextPage = activeTab === 'projects' ? projectsQuery.isFetchingNextPage : namespacesQuery.isFetchingNextPage
  const fetchNextPage = activeTab === 'projects' ? projectsQuery.fetchNextPage : namespacesQuery.fetchNextPage

  function handleTabChange(key: string) {
    setActiveTab(key)
    setSearch('')
  }

  const statCards = [
    { icon: FolderKanban, value: statsQuery.data?.projects ?? '-', label: t('admin.projects.stats.projects') },
    { icon: Building2, value: statsQuery.data?.namespaces ?? '-', label: t('admin.projects.stats.namespaces') },
    { icon: Users, value: statsQuery.data?.users ?? '-', label: t('admin.projects.stats.users') },
    { icon: HardDrive, value: statsQuery.data ? formatStorageBytes(statsQuery.data.storage_bytes) : '-', label: t('admin.projects.stats.storage') },
  ]

  return (
    <div>
      <div className="flex flex-col sm:flex-row sm:items-start sm:justify-between gap-4">
        <div>
          <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{t('admin.projects.title')}</h2>
          <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('admin.projects.description')}</p>
        </div>
        {/* Mobile: single bordered row with horizontal scroll */}
        <ScrollableRow className="sm:hidden rounded-lg border border-gray-200 dark:border-gray-700 px-3 py-2" gradientFrom="from-white dark:from-gray-900">
          {statCards.map((card, i) => (
            <div key={card.label} className={`flex items-center gap-1.5 shrink-0 ${i < statCards.length - 1 ? 'pr-3 border-r border-gray-200 dark:border-gray-700' : ''}`}>
              <card.icon className="h-3.5 w-3.5 text-indigo-400" />
              <span className="text-sm font-bold text-gray-900 dark:text-gray-100">{card.value}</span>
              <span className="text-xs text-gray-500 dark:text-gray-400">{card.label}</span>
            </div>
          ))}
        </ScrollableRow>
        {/* Desktop: individual cards */}
        <div className="hidden sm:grid grid-cols-4 gap-3">
          {statCards.map((card) => (
            <div key={card.label} className="rounded-lg border border-gray-200 dark:border-gray-700 p-3 text-center min-w-[100px]">
              <div className="flex items-center justify-center gap-2">
                <card.icon className="h-4 w-4 text-indigo-400" />
                <span className="text-xl font-bold text-gray-900 dark:text-gray-100">{card.value}</span>
              </div>
              <div className="text-xs text-gray-500 dark:text-gray-400 mt-0.5">{card.label}</div>
            </div>
          ))}
        </div>
      </div>

      <div className="mt-6">
        <Tabs tabs={tabs} activeTab={activeTab} onTabChange={handleTabChange} />
      </div>

      {activeTab !== 'settings' && (
        <div className="mt-4 max-w-sm">
          <Input
            placeholder={activeTab === 'projects' ? t('admin.projects.search.projects') : t('admin.projects.search.namespaces')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </div>
      )}

      <div className="mt-4">
        {activeTab === 'settings' ? (
          <SettingsTab />
        ) : isLoading ? (
          <div className="flex justify-center py-12">
            <Spinner />
          </div>
        ) : activeTab === 'projects' ? (
          <DataTable
            columns={projectColumns}
            data={projects}
            emptyMessage={t('admin.projects.empty.projects')}
            alwaysShowHeader
          />
        ) : (
          <DataTable
            columns={namespaceColumns}
            data={namespaces}
            emptyMessage={t('admin.projects.empty.namespaces')}
            alwaysShowHeader
          />
        )}

        {activeTab !== 'settings' && hasNextPage && (
          <div className="flex justify-center pt-4">
            <Button
              variant="secondary"
              onClick={() => fetchNextPage()}
              disabled={isFetchingNextPage}
            >
              {isFetchingNextPage ? t('admin.projects.loading') : t('admin.projects.loadMore')}
            </Button>
          </div>
        )}
      </div>
    </div>
  )
}

function SettingsTab() {
  const { t } = useTranslation()

  return (
    <div className="space-y-6">
      <ReservedListSection
        settingKey="reserved_namespace_slugs"
        title={t('admin.projects.settings.reservedSlugs.title')}
        description={t('admin.projects.settings.reservedSlugs.description')}
        placeholder={t('admin.projects.settings.reservedSlugs.placeholder')}
        emptyMessage={t('admin.projects.settings.reservedSlugs.empty')}
        normalize={(v) => v.toLowerCase().replace(/[^a-z0-9-]/g, '')}
      />
      <ReservedListSection
        settingKey="reserved_project_keys"
        title={t('admin.projects.settings.reservedKeys.title')}
        description={t('admin.projects.settings.reservedKeys.description')}
        placeholder={t('admin.projects.settings.reservedKeys.placeholder')}
        emptyMessage={t('admin.projects.settings.reservedKeys.empty')}
        normalize={(v) => v.toUpperCase().replace(/[^A-Z0-9]/g, '')}
      />
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
      setError(t('admin.projects.settings.duplicate'))
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
    <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-4 sm:p-6">
      <h3 className="text-sm font-medium text-gray-900 dark:text-gray-100">{title}</h3>
      <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">{description}</p>

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
          {t('admin.projects.settings.add')}
        </Button>
      </div>
      {error && <p className="mt-1 text-xs text-red-500">{error}</p>}

      <div className="mt-3">
        {isLoading ? (
          <Spinner />
        ) : list.length === 0 ? (
          <p className="text-sm text-gray-500 dark:text-gray-400">{emptyMessage}</p>
        ) : (
          <div className="flex flex-wrap gap-2">
            {list.map((item) => (
              <span
                key={item}
                className="inline-flex items-center gap-1 rounded-md bg-gray-100 dark:bg-gray-800 px-2.5 py-1 text-sm font-medium text-gray-700 dark:text-gray-300"
              >
                {item}
                <button
                  onClick={() => handleRemove(item)}
                  className="ml-0.5 rounded p-0.5 hover:bg-gray-200 dark:hover:bg-gray-700 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200 transition-colors"
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
