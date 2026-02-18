import { Select } from '@/components/ui/Select'
import { Input } from '@/components/ui/Input'
import { UserPicker } from '@/components/ui/UserPicker'
import type { WorkItem, UpdateWorkItemInput } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'
import type { ProjectMember } from '@/api/projects'

interface DetailSidebarProps {
  item: WorkItem
  statuses: WorkflowStatus[]
  allowedTransitions: string[]
  members: ProjectMember[]
  onUpdate: (input: UpdateWorkItemInput) => void
}

const PRIORITIES = ['low', 'medium', 'high', 'critical']
const TYPES = ['task', 'ticket', 'bug', 'feedback', 'epic']
const VISIBILITIES = ['internal', 'portal', 'public']

export function DetailSidebar({ item, statuses, allowedTransitions, members, onUpdate }: DetailSidebarProps) {
  const currentStatusDisplay = statuses.find((s) => s.name === item.status)?.display_name ?? item.status

  return (
    <div className="space-y-4">
      <Field label="Status">
        <Select
          value={item.status}
          onChange={(e) => onUpdate({ status: e.target.value })}
        >
          <option value={item.status}>{currentStatusDisplay}</option>
          {allowedTransitions
            .filter((t) => t !== item.status)
            .map((t) => {
              const ws = statuses.find((s) => s.name === t)
              return <option key={t} value={t}>{ws?.display_name ?? t}</option>
            })}
        </Select>
      </Field>

      <Field label="Priority">
        <Select value={item.priority} onChange={(e) => onUpdate({ priority: e.target.value })}>
          {PRIORITIES.map((p) => <option key={p} value={p}>{p}</option>)}
        </Select>
      </Field>

      <Field label="Type">
        <Select value={item.type} onChange={(e) => onUpdate({ type: e.target.value })}>
          {TYPES.map((t) => <option key={t} value={t}>{t}</option>)}
        </Select>
      </Field>

      <Field label="Assignee">
        <UserPicker
          members={members}
          value={item.assignee_id}
          onChange={(userId) => onUpdate({ assignee_id: userId })}
        />
      </Field>

      <Field label="Visibility">
        <Select value={item.visibility} onChange={(e) => onUpdate({ visibility: e.target.value })}>
          {VISIBILITIES.map((v) => <option key={v} value={v}>{v}</option>)}
        </Select>
      </Field>

      <Field label="Due Date">
        <Input
          type="date"
          value={item.due_date ?? ''}
          onChange={(e) => onUpdate({ due_date: e.target.value || null })}
        />
      </Field>

      <Field label="Labels">
        <Input
          defaultValue={item.labels.join(', ')}
          placeholder="Comma-separated"
          onBlur={(e) => {
            const newLabels = e.target.value ? e.target.value.split(',').map((l) => l.trim()).filter(Boolean) : []
            if (JSON.stringify(newLabels) !== JSON.stringify(item.labels)) {
              onUpdate({ labels: newLabels })
            }
          }}
        />
      </Field>

      <div className="border-t border-gray-100 dark:border-gray-700 pt-4 space-y-2 text-xs text-gray-400 dark:text-gray-500">
        <div>Reporter: {members.find((m) => m.user_id === item.reporter_id)?.display_name ?? item.reporter_id}</div>
        <div>Created: {new Date(item.created_at).toLocaleString()}</div>
        <div>Updated: {new Date(item.updated_at).toLocaleString()}</div>
        {item.resolved_at && <div>Resolved: {new Date(item.resolved_at).toLocaleString()}</div>}
      </div>
    </div>
  )
}

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{label}</label>
      {children}
    </div>
  )
}
