import { useRef, useState } from 'react'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { useTranslation } from 'react-i18next'
import { SlidersHorizontal } from 'lucide-react'
import { Input } from '@/components/ui/Input'
import { Modal } from '@/components/ui/Modal'
import { MultiSelect, type MultiSelectOption } from '@/components/ui/MultiSelect'
import type { WorkItemFilter } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

interface WorkItemFiltersProps {
  filter: WorkItemFilter
  onFilterChange: (filter: WorkItemFilter) => void
  statuses: WorkflowStatus[]
  search: string
  onSearchChange: (value: string) => void
}

const closedCategories = new Set(['done', 'cancelled'])

export function WorkItemFilters({ filter, onFilterChange, statuses, search, onSearchChange }: WorkItemFiltersProps) {
  const { t } = useTranslation()
  const searchRef = useRef<HTMLInputElement>(null)
  const [filtersOpen, setFiltersOpen] = useState(false)

  const typeOptions: MultiSelectOption[] = [
    { value: 'task', label: t('workitems.types.task') },
    { value: 'ticket', label: t('workitems.types.ticket') },
    { value: 'bug', label: t('workitems.types.bug') },
    { value: 'feedback', label: t('workitems.types.feedback') },
    { value: 'epic', label: t('workitems.types.epic') },
  ]

  const priorityOptions: MultiSelectOption[] = [
    { value: 'critical', label: t('workitems.priorities.critical') },
    { value: 'high', label: t('workitems.priorities.high') },
    { value: 'medium', label: t('workitems.priorities.medium') },
    { value: 'low', label: t('workitems.priorities.low') },
  ]

  const assigneeOptions: MultiSelectOption[] = [
    { value: 'me', label: t('workitems.filters.assignedToMe') },
    { value: 'unassigned', label: t('workitems.filters.unassigned') },
  ]

  function buildStatusOptions(ss: WorkflowStatus[]): MultiSelectOption[] {
    return ss.map((s) => ({
      value: s.name,
      label: t(`workitems.statuses.${s.name}`, { defaultValue: s.display_name }),
      group: closedCategories.has(s.category) ? t('workitems.filters.statusGroupClosed') : t('workitems.filters.statusGroupOpen'),
    }))
  }

  useKeyboardShortcut({ key: '/' }, () => searchRef.current?.focus())

  function setArray(key: 'type' | 'status' | 'priority' | 'assignee', values: string[]) {
    onFilterChange({ ...filter, [key]: values.length > 0 ? values : undefined, cursor: undefined })
  }

  const statusOptions = buildStatusOptions(statuses)

  const activeFilterCount =
    (filter.type?.length ? 1 : 0) +
    (filter.priority?.length ? 1 : 0) +
    (filter.status?.length ? 1 : 0) +
    (filter.assignee?.length ? 1 : 0)

  return (
    <>
      {/* Desktop: inline layout */}
      <div className="hidden sm:flex flex-wrap items-end gap-3">
        <div className="flex-1 min-w-[200px]">
          <Input
            ref={searchRef}
            placeholder={t('workitems.filters.search')}
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
            onKeyDown={(e) => { if (e.key === 'Escape') searchRef.current?.blur() }}
          />
        </div>
        <div className="w-36">
          <MultiSelect options={typeOptions} selected={filter.type ?? []} onChange={(v) => setArray('type', v)} placeholder={t('workitems.filters.allTypes')} />
        </div>
        <div className="w-36">
          <MultiSelect options={priorityOptions} selected={filter.priority ?? []} onChange={(v) => setArray('priority', v)} placeholder={t('workitems.filters.allPriorities')} />
        </div>
        <div className="w-40">
          <MultiSelect options={statusOptions} selected={filter.status ?? []} onChange={(v) => setArray('status', v)} placeholder={t('workitems.filters.allStatuses')} />
        </div>
        <div className="w-40">
          <MultiSelect options={assigneeOptions} selected={filter.assignee ?? []} onChange={(v) => setArray('assignee', v)} placeholder={t('workitems.filters.allAssignees')} />
        </div>
      </div>

      {/* Mobile: search + filter icon */}
      <div className="flex sm:hidden items-center gap-2">
        <div className="flex-1">
          <Input
            placeholder={t('workitems.filters.search')}
            value={search}
            onChange={(e) => onSearchChange(e.target.value)}
          />
        </div>
        <button
          onClick={() => setFiltersOpen(true)}
          className="relative shrink-0 p-2.5 rounded-md border border-gray-300 dark:border-gray-600 text-gray-500 dark:text-gray-400 hover:bg-gray-100 dark:hover:bg-gray-800"
          aria-label={t('workitems.filters.title')}
        >
          <SlidersHorizontal className="h-5 w-5" />
          {activeFilterCount > 0 && (
            <span className="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-indigo-600 text-[10px] font-bold text-white">
              {activeFilterCount}
            </span>
          )}
        </button>
      </div>

      {/* Mobile filter modal */}
      <Modal open={filtersOpen} onClose={() => setFiltersOpen(false)} title={t('workitems.filters.title')}>
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.filters.allTypes')}</label>
            <MultiSelect options={typeOptions} selected={filter.type ?? []} onChange={(v) => setArray('type', v)} placeholder={t('workitems.filters.allTypes')} />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.filters.allPriorities')}</label>
            <MultiSelect options={priorityOptions} selected={filter.priority ?? []} onChange={(v) => setArray('priority', v)} placeholder={t('workitems.filters.allPriorities')} />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.filters.allStatuses')}</label>
            <MultiSelect options={statusOptions} selected={filter.status ?? []} onChange={(v) => setArray('status', v)} placeholder={t('workitems.filters.allStatuses')} />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">{t('workitems.filters.allAssignees')}</label>
            <MultiSelect options={assigneeOptions} selected={filter.assignee ?? []} onChange={(v) => setArray('assignee', v)} placeholder={t('workitems.filters.allAssignees')} />
          </div>
        </div>
      </Modal>
    </>
  )
}
