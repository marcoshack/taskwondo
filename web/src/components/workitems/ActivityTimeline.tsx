import { useEvents } from '@/hooks/useWorkItems'
import { Spinner } from '@/components/ui/Spinner'
import type { WorkItemEvent } from '@/api/workitems'

interface ActivityTimelineProps {
  projectKey: string
  itemNumber: number
  sortOrder?: 'asc' | 'desc'
}

export function ActivityTimeline({ projectKey, itemNumber, sortOrder = 'desc' }: ActivityTimelineProps) {
  const { data: events, isLoading } = useEvents(projectKey, itemNumber)

  if (isLoading) return <Spinner size="sm" />

  if (!events?.length) {
    return <p className="text-sm text-gray-400">No activity yet.</p>
  }

  const sorted = sortOrder === 'desc' ? [...events].reverse() : events

  return (
    <div className="border-l-2 border-gray-200 pl-4 space-y-4">
      {sorted.map((event) => (
        <div key={event.id} className="relative">
          <div className="absolute -left-[21px] top-1.5 w-2.5 h-2.5 rounded-full bg-gray-300 border-2 border-white" />
          <div className="text-sm">
            <span className="font-medium text-gray-700">{event.actor?.display_name ?? 'System'}</span>
            {' '}
            <span className="text-gray-500">{formatEventLabel(event)}</span>
          </div>
          <FieldChangeDiff event={event} />
          <span className="text-xs text-gray-400">{new Date(event.created_at).toLocaleString()}</span>
        </div>
      ))}
    </div>
  )
}

const fieldLabels: Record<string, string> = {
  status: 'Status',
  priority: 'Priority',
  type: 'Type',
  title: 'Title',
  description: 'Description',
  assignee_id: 'Assignee',
  labels: 'Labels',
  visibility: 'Visibility',
  due_date: 'Due date',
  milestone_id: 'Milestone',
  queue_id: 'Queue',
}

function fieldLabel(name: string): string {
  return fieldLabels[name] ?? name.replace(/_/g, ' ')
}

function formatEventLabel(event: WorkItemEvent): string {
  if (event.event_type === 'created') return 'created this item'
  if (event.event_type === 'comment_added') return 'added a comment'
  if (event.event_type === 'comment_deleted') return 'deleted a comment'
  if (event.field_name) {
    if (event.old_value && event.new_value) {
      return `changed ${fieldLabel(event.field_name)}`
    }
    if (event.new_value) return `set ${fieldLabel(event.field_name)}`
    if (event.old_value) return `cleared ${fieldLabel(event.field_name)}`
  }
  return event.event_type.replace(/_/g, ' ')
}

function truncate(value: string, max: number = 120): string {
  return value.length > max ? value.slice(0, max) + '\u2026' : value
}

function FieldChangeDiff({ event }: { event: WorkItemEvent }) {
  if (!event.field_name) return null
  if (!event.old_value && !event.new_value) return null

  return (
    <div className="mt-1 mb-1 rounded-md border border-gray-200 bg-gray-50 text-xs font-mono overflow-hidden">
      {event.old_value && (
        <div className="px-3 py-1.5 bg-red-50 text-red-800 border-b border-gray-200">
          <span className="select-none text-red-400 mr-2">&minus;</span>
          {truncate(event.old_value)}
        </div>
      )}
      {event.new_value && (
        <div className="px-3 py-1.5 bg-green-50 text-green-800">
          <span className="select-none text-green-400 mr-2">+</span>
          {truncate(event.new_value)}
        </div>
      )}
    </div>
  )
}
