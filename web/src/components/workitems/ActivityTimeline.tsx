import { useEvents } from '@/hooks/useWorkItems'
import { Spinner } from '@/components/ui/Spinner'

interface ActivityTimelineProps {
  projectKey: string
  itemNumber: number
}

export function ActivityTimeline({ projectKey, itemNumber }: ActivityTimelineProps) {
  const { data: events, isLoading } = useEvents(projectKey, itemNumber)

  if (isLoading) return <Spinner size="sm" />

  if (!events?.length) {
    return <p className="text-sm text-gray-400">No activity yet.</p>
  }

  return (
    <div className="border-l-2 border-gray-200 pl-4 space-y-4">
      {events.map((event) => (
        <div key={event.id} className="relative">
          <div className="absolute -left-[21px] top-1.5 w-2.5 h-2.5 rounded-full bg-gray-300 border-2 border-white" />
          <div className="text-sm">
            <span className="font-medium text-gray-700">{event.actor?.display_name ?? 'System'}</span>
            {' '}
            <span className="text-gray-500">{formatEvent(event)}</span>
          </div>
          <span className="text-xs text-gray-400">{new Date(event.created_at).toLocaleString()}</span>
        </div>
      ))}
    </div>
  )
}

function formatEvent(event: { event_type: string; field_name?: string; old_value?: string; new_value?: string }) {
  if (event.event_type === 'created') return 'created this item'
  if (event.event_type === 'field_change' && event.field_name) {
    if (event.old_value && event.new_value) {
      return `changed ${event.field_name} from "${event.old_value}" to "${event.new_value}"`
    }
    if (event.new_value) return `set ${event.field_name} to "${event.new_value}"`
    if (event.old_value) return `cleared ${event.field_name}`
  }
  if (event.event_type === 'comment_added') return 'added a comment'
  return event.event_type
}
