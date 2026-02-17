import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import type { WorkItemFilter } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

interface WorkItemFiltersProps {
  filter: WorkItemFilter
  onFilterChange: (filter: WorkItemFilter) => void
  statuses: WorkflowStatus[]
  search: string
  onSearchChange: (value: string) => void
}

export function WorkItemFilters({ filter, onFilterChange, statuses, search, onSearchChange }: WorkItemFiltersProps) {
  function set(key: keyof WorkItemFilter, value: string) {
    onFilterChange({ ...filter, [key]: value || undefined, cursor: undefined })
  }

  return (
    <div className="flex flex-wrap items-end gap-3">
      <div className="flex-1 min-w-[200px]">
        <Input
          placeholder="Search items..."
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
        />
      </div>
      <div className="w-32">
        <Select value={filter.type ?? ''} onChange={(e) => set('type', e.target.value)}>
          <option value="">All types</option>
          <option value="task">Task</option>
          <option value="ticket">Ticket</option>
          <option value="bug">Bug</option>
          <option value="feedback">Feedback</option>
          <option value="epic">Epic</option>
        </Select>
      </div>
      <div className="w-36">
        <Select value={filter.priority ?? ''} onChange={(e) => set('priority', e.target.value)}>
          <option value="">All priorities</option>
          <option value="critical">Critical</option>
          <option value="high">High</option>
          <option value="medium">Medium</option>
          <option value="low">Low</option>
        </Select>
      </div>
      <div className="w-36">
        <Select value={filter.status ?? ''} onChange={(e) => set('status', e.target.value)}>
          <option value="">All statuses</option>
          {statuses.map((s) => (
            <option key={s.name} value={s.name}>{s.display_name}</option>
          ))}
        </Select>
      </div>
      <div className="w-36">
        <Select value={filter.assignee ?? ''} onChange={(e) => set('assignee', e.target.value)}>
          <option value="">All assignees</option>
          <option value="me">Assigned to me</option>
          <option value="unassigned">Unassigned</option>
        </Select>
      </div>
    </div>
  )
}
