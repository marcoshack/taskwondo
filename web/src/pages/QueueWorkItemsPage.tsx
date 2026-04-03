import { useState, useMemo } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useWorkItems } from '@/hooks/useWorkItems'
import { useQueue } from '@/hooks/useQueues'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { Spinner } from '@/components/ui/Spinner'
import { PriorityBadge } from '@/components/workitems/PriorityBadge'
import { TypeBadge } from '@/components/workitems/TypeBadge'
import { StatusBadge } from '@/components/workitems/StatusBadge'
import { Badge } from '@/components/ui/Badge'
import { Input } from '@/components/ui/Input'
import { useDebounce } from '@/hooks/useDebounce'
import { ArrowLeft, Search } from 'lucide-react'
import type { WorkItem } from '@/api/workitems'

export function QueueWorkItemsPage() {
  const { t } = useTranslation()
  const { projectKey = '', queueId = '' } = useParams<{ projectKey: string; queueId: string }>()
  const { p } = useNamespacePath()
  const { data: queue, isLoading: queueLoading } = useQueue(projectKey, queueId)

  const [search, setSearch] = useState('')
  const debouncedSearch = useDebounce(search, 300)

  const filter = useMemo(() => ({
    queue: queueId,
    q: debouncedSearch || undefined,
    limit: 50,
  }), [queueId, debouncedSearch])

  const { data, isLoading } = useWorkItems(projectKey, filter)
  const items = data?.data ?? []

  if (queueLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    )
  }

  return (
    <div>
      <Link
        to={p(`/projects/${projectKey}/queues`)}
        className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        {t('queues.backToQueues')}
      </Link>

      <div className="flex items-center gap-3 mb-6">
        <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
          {queue?.name ?? t('queues.workItems')}
        </h1>
        {queue && (
          <Badge color="gray">{queue.queue_type}</Badge>
        )}
      </div>

      <div className="relative mb-4">
        <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-gray-400" />
        <Input
          placeholder={t('common.search')}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          className="pl-9"
        />
      </div>

      {isLoading ? (
        <div className="flex justify-center py-12">
          <Spinner size="lg" />
        </div>
      ) : items.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400 py-8 text-center">
          {t('queues.noItems')}
        </p>
      ) : (
        <div className="space-y-2">
          {items.map((item: WorkItem) => (
            <Link
              key={item.id}
              to={p(`/projects/${projectKey}/items/${item.item_number}`)}
              className="block rounded-lg border border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4 hover:border-indigo-300 dark:hover:border-indigo-600 transition-colors"
            >
              <div className="flex items-center gap-3">
                <span className="text-xs font-mono text-gray-400 dark:text-gray-500 shrink-0">
                  {item.display_id}
                </span>
                <span className="text-sm font-medium text-gray-900 dark:text-gray-100 truncate flex-1">
                  {item.title}
                </span>
                <div className="flex items-center gap-2 shrink-0">
                  <TypeBadge type={item.type} />
                  <StatusBadge status={item.status} />
                  <PriorityBadge priority={item.priority} />
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
