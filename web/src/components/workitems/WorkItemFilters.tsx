import { useRef, useEffect } from 'react'
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

const typeOptions: MultiSelectOption[] = [
  { value: 'task', label: 'Task' },
  { value: 'ticket', label: 'Ticket' },
  { value: 'bug', label: 'Bug' },
  { value: 'feedback', label: 'Feedback' },
  { value: 'epic', label: 'Epic' },
]

const priorityOptions: MultiSelectOption[] = [
  { value: 'critical', label: 'Critical' },
  { value: 'high', label: 'High' },
  { value: 'medium', label: 'Medium' },
  { value: 'low', label: 'Low' },
]

const assigneeOptions: MultiSelectOption[] = [
  { value: 'me', label: 'Assigned to me' },
  { value: 'unassigned', label: 'Unassigned' },
]

const closedCategories = new Set(['done', 'cancelled'])

function buildStatusOptions(statuses: WorkflowStatus[]): MultiSelectOption[] {
  return statuses.map((s) => ({
    value: s.name,
    label: s.display_name,
    group: closedCategories.has(s.category) ? 'Closed' : 'Open',
  }))
}

export function WorkItemFilters({ filter, onFilterChange, statuses, search, onSearchChange }: WorkItemFiltersProps) {
  const searchRef = useRef<HTMLInputElement>(null)

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
          placeholder="Search items... ( / )"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>
      <div className="w-36">
        <MultiSelect
          options={typeOptions}
          selected={filter.type ?? []}
          onChange={(v) => setArray('type', v)}
          placeholder="All types"
        />
      </div>
      <div className="w-36">
        <MultiSelect
          options={priorityOptions}
          selected={filter.priority ?? []}
          onChange={(v) => setArray('priority', v)}
          placeholder="All priorities"
        />
      </div>
      <div className="w-40">
        <MultiSelect
          options={statusOptions}
          selected={filter.status ?? []}
          onChange={(v) => setArray('status', v)}
          placeholder="All statuses"
        />
      </div>
      <div className="w-40">
        <MultiSelect
          options={assigneeOptions}
          selected={filter.assignee ?? []}
          onChange={(v) => setArray('assignee', v)}
          placeholder="All assignees"
        />
      </div>
    </div>
  )
}
