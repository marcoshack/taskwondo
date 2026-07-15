import { describe, it, expect, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { WorkItemCompactRow } from './WorkItemCompactRow'
import type { WorkItem } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: { defaultValue?: string }) => opts?.defaultValue ?? key,
  }),
}))

vi.mock('@/components/ui/Avatar', () => ({
  Avatar: ({ name }: { name: string }) => <span data-testid="avatar">{name}</span>,
}))

vi.mock('@/components/ui/Tooltip', () => ({
  Tooltip: ({ children, content }: { children: React.ReactNode; content: string }) => (
    <span title={content}>{children}</span>
  ),
}))

const statuses: WorkflowStatus[] = [
  { id: 's1', name: 'open', display_name: 'Open', category: 'todo', color: null, position: 0 },
  { id: 's2', name: 'in_progress', display_name: 'In Progress', category: 'in_progress', color: null, position: 1 },
  { id: 's3', name: 'done', display_name: 'Done', category: 'done', color: null, position: 2 },
]

function makeItem(overrides: Partial<WorkItem> = {}): WorkItem {
  return {
    id: 'item-1',
    project_key: 'TASK',
    item_number: 78,
    display_id: 'TASK-78',
    type: 'task',
    title: 'Compact list row redesign for split-pane mode',
    description: null,
    status: 'open',
    priority: 'medium',
    assignee_id: null,
    reporter_id: 'r1',
    reporter_name: 'Admin',
    queue_id: null,
    milestone_id: null,
    visibility: 'internal',
    labels: ['agent:pipeline', 'ui'],
    complexity: null,
    custom_fields: {},
    due_date: '2026-07-20',
    estimated_seconds: null,
    sla: null,
    sla_target_at: null,
    resolved_at: null,
    created_at: '2026-06-16T00:00:00Z',
    updated_at: '2026-07-14T00:00:00Z',
    ...overrides,
  }
}

describe('WorkItemCompactRow', () => {
  it('renders nav-rail anatomy: id, title, status bar, dual selection bar', () => {
    const html = renderToStaticMarkup(
      <WorkItemCompactRow
        item={makeItem()}
        statuses={statuses}
        selected
        assignee={{ name: 'Ada Lovelace' }}
        onSelect={() => {}}
        onStatusChange={() => {}}
      />,
    )

    expect(html).toContain('data-testid="work-item-compact-row"')
    expect(html).toContain('TASK-78')
    expect(html).toContain('Compact list row redesign for split-pane mode')
    expect(html).toContain('data-selected="true"')
    expect(html).toContain('data-status-category="todo"')
    expect(html).toContain('data-testid="status-category-bar"')
    expect(html).toContain('data-testid="selection-bar"')
    expect(html).toContain('aria-label="TASK-78, Open: Compact list row redesign for split-pane mode"')
    // Rail omits chips / dots / avatar / due / hover status select
    expect(html).not.toContain('data-testid="label-dot"')
    expect(html).not.toContain('data-testid="avatar"')
    expect(html).not.toContain('Ada Lovelace')
    expect(html).not.toContain('Jul')
    expect(html).not.toContain('<select')
    expect(html).not.toContain('workitems.compactRow.changeStatus')
    // Medium priority has no flag
    expect(html).not.toContain('data-testid="priority-flag"')
  })

  it('shows lucide priority flag only for high and critical', () => {
    const high = renderToStaticMarkup(
      <WorkItemCompactRow
        item={makeItem({ priority: 'high', status: 'in_progress' })}
        statuses={statuses}
        onSelect={() => {}}
      />,
    )
    expect(high).toContain('data-testid="priority-flag"')
    expect(high).toContain('data-priority="high"')
    expect(high).toContain(
      'aria-label="TASK-78, In Progress, high priority: Compact list row redesign for split-pane mode"',
    )

    const critical = renderToStaticMarkup(
      <WorkItemCompactRow
        item={makeItem({ priority: 'critical' })}
        statuses={statuses}
        onSelect={() => {}}
      />,
    )
    expect(critical).toContain('data-priority="critical"')

    const low = renderToStaticMarkup(
      <WorkItemCompactRow
        item={makeItem({ priority: 'low' })}
        statuses={statuses}
        onSelect={() => {}}
      />,
    )
    expect(low).not.toContain('data-testid="priority-flag"')
  })

  it('does not render assignee chrome in rail mode', () => {
    const html = renderToStaticMarkup(
      <WorkItemCompactRow
        item={makeItem()}
        statuses={statuses}
        assignee={{ name: 'Grace Hopper' }}
        onSelect={() => {}}
      />,
    )
    expect(html).not.toContain('userPicker.unassigned')
    expect(html).not.toContain('Grace Hopper')
    expect(html).not.toContain('data-testid="avatar"')
  })

  it('shows parent epic badge on non-epic rows when enriched', () => {
    const html = renderToStaticMarkup(
      <WorkItemCompactRow
        item={makeItem({
          parent_epic_display_id: 'TASK-76',
          parent_epic_title: 'Split-pane list view',
        })}
        statuses={statuses}
        onSelect={() => {}}
      />,
    )
    expect(html).toContain('data-testid="parent-epic-badge"')
    expect(html).toContain('data-epic-id="TASK-76"')
    expect(html).toContain('TASK-76')
  })

  it('hides parent epic badge for epic items even if field is set', () => {
    const html = renderToStaticMarkup(
      <WorkItemCompactRow
        item={makeItem({
          type: 'epic',
          parent_epic_display_id: 'TASK-1',
          parent_epic_title: 'Should not show',
        })}
        statuses={statuses}
        onSelect={() => {}}
      />,
    )
    expect(html).not.toContain('data-testid="parent-epic-badge"')
  })

  it('applies completed strikethrough on the title', () => {
    const html = renderToStaticMarkup(
      <WorkItemCompactRow
        item={makeItem({ status: 'done' })}
        statuses={statuses}
        isCompleted
        onSelect={() => {}}
      />,
    )
    expect(html).toContain('line-through')
    expect(html).toContain('data-status-category="done"')
  })
})
