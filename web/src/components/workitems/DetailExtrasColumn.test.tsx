import { describe, it, expect, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { DetailExtrasColumn } from './DetailExtrasColumn'
import type { WorkItem } from '@/api/workitems'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/hooks/useNamespacePath', () => ({
  useNamespacePath: () => ({ p: (path: string) => path }),
}))

vi.mock('@/components/workitems/RelationList', () => ({
  RelationList: () => <div data-testid="relation-list">relations</div>,
}))

vi.mock('@/components/workitems/ParentEpicBadge', () => ({
  ParentEpicBadge: ({ displayId }: { displayId: string }) => (
    <span data-testid="parent-epic-badge">{displayId}</span>
  ),
  shouldShowParentEpic: (item: { type: string; parent_epic_display_id?: string | null }) =>
    item.type !== 'epic' && !!item.parent_epic_display_id,
}))

const item: WorkItem = {
  id: 'item-117',
  project_key: 'TASK',
  item_number: 117,
  display_id: 'TASK-117',
  type: 'task',
  title: 'Responsive layout third column',
  description: null,
  status: 'open',
  priority: 'medium',
  assignee_id: null,
  reporter_id: 'r1',
  reporter_name: 'Admin',
  queue_id: null,
  milestone_id: 'ms-1',
  visibility: 'internal',
  labels: [],
  complexity: null,
  custom_fields: {},
  due_date: null,
  estimated_seconds: null,
  sla: null,
  sla_target_at: null,
  resolved_at: null,
  parent_epic_display_id: 'TASK-76',
  parent_epic_title: 'Split-pane list view',
  created_at: '2026-07-15T00:00:00Z',
  updated_at: '2026-07-15T00:00:00Z',
}

describe('DetailExtrasColumn', () => {
  it('renders relations plus epic and milestone links', () => {
    const html = renderToStaticMarkup(
      <MemoryRouter>
        <DetailExtrasColumn
          projectKey="TASK"
          itemNumber={117}
          item={item}
          milestones={[
            {
              id: 'ms-1',
              project_id: 'p1',
              name: 'Sprint 1',
              status: 'open',
              description: null,
              due_date: null,
              open_count: 0,
              closed_count: 0,
              total_count: 0,
              total_estimated_seconds: 0,
              total_spent_seconds: 0,
              created_at: '2026-07-01T00:00:00Z',
              updated_at: '2026-07-01T00:00:00Z',
            },
          ]}
          readOnly
        />
      </MemoryRouter>,
    )

    expect(html).toContain('data-testid="detail-extras-column"')
    expect(html).toContain('data-testid="relation-list"')
    expect(html).toContain('workitems.detail.extrasColumn')
    expect(html).toContain('TASK-76')
    expect(html).toContain('Sprint 1')
    expect(html).toContain('/projects/TASK/items/76')
    expect(html).toContain('/projects/TASK/milestones/ms-1')
  })
})
