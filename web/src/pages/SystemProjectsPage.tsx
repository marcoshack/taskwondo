import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAdminProjects, useAdminNamespaces } from '@/hooks/useAdmin'
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

  const tabs = [
    { key: 'projects', label: t('admin.projects.tab.projects') },
    { key: 'namespaces', label: t('admin.projects.tab.namespaces') },
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
      render: (row) => <span className="text-gray-900 dark:text-gray-100 font-medium">{row.name}</span>,
    },
    {
      key: 'namespace',
      header: t('admin.projects.col.namespace'),
      render: (row) => (
        <span className="text-gray-600 dark:text-gray-400">{row.namespace_display_name}</span>
      ),
    },
    {
      key: 'owner',
      header: t('admin.projects.col.owner'),
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
      render: (row) => <span className="text-gray-600 dark:text-gray-400">{formatStorageBytes(row.storage_bytes)}</span>,
    },
    {
      key: 'created',
      header: t('admin.projects.col.created'),
      className: 'w-28',
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

  return (
    <div>
      <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">{t('admin.projects.title')}</h2>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{t('admin.projects.description')}</p>

      <div className="mt-6">
        <Tabs tabs={tabs} activeTab={activeTab} onTabChange={handleTabChange} />
      </div>

      <div className="mt-4 max-w-sm">
        <Input
          placeholder={activeTab === 'projects' ? t('admin.projects.search.projects') : t('admin.projects.search.namespaces')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </div>

      <div className="mt-4">
        {isLoading ? (
          <div className="flex justify-center py-12">
            <Spinner />
          </div>
        ) : activeTab === 'projects' ? (
          <DataTable
            columns={projectColumns}
            data={projects}
            emptyMessage={t('admin.projects.empty.projects')}
          />
        ) : (
          <DataTable
            columns={namespaceColumns}
            data={namespaces}
            emptyMessage={t('admin.projects.empty.namespaces')}
          />
        )}

        {hasNextPage && (
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
