import { useRef, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Input } from '@/components/ui/Input'
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
      label: s.display_name,
      group: closedCategories.has(s.category) ? t('workitems.filters.statusGroupClosed') : t('workitems.filters.statusGroupOpen'),
    }))
  }

  useEffect(() => {
    function handleKeyDown(e: KeyboardEvent) {
      if (e.key !== '/') return
      const tag = (e.target as HTMLElement).tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
      if ((e.target as HTMLElement).isContentEditable) return
      e.preventDefault()
      searchRef.current?.focus()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [])

  function setArray(key: 'type' | 'status' | 'priority' | 'assignee', values: string[]) {
    onFilterChange({ ...filter, [key]: values.length > 0 ? values : undefined, cursor: undefined })
  }

  const statusOptions = buildStatusOptions(statuses)

  return (
    <div className="flex flex-wrap items-end gap-3">
      <div className="flex-1 min-w-[200px]">
        <Input
          ref={searchRef}
          placeholder={t('workitems.filters.search')}
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>
      <div className="w-36">
        <MultiSelect
          options={typeOptions}
          selected={filter.type ?? []}
          onChange={(v) => setArray('type', v)}
          placeholder={t('workitems.filters.allTypes')}
        />
      </div>
      <div className="w-36">
        <MultiSelect
          options={priorityOptions}
          selected={filter.priority ?? []}
          onChange={(v) => setArray('priority', v)}
          placeholder={t('workitems.filters.allPriorities')}
        />
      </div>
      <div className="w-40">
        <MultiSelect
          options={statusOptions}
          selected={filter.status ?? []}
          onChange={(v) => setArray('status', v)}
          placeholder={t('workitems.filters.allStatuses')}
        />
      </div>
      <div className="w-40">
        <MultiSelect
          options={assigneeOptions}
          selected={filter.assignee ?? []}
          onChange={(v) => setArray('assignee', v)}
          placeholder={t('workitems.filters.allAssignees')}
        />
      </div>
    </div>
  )
}
