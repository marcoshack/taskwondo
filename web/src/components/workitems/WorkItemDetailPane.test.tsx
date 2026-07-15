import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { MemoryRouter } from 'react-router-dom'
import { WorkItemDetailPane } from './WorkItemDetailPane'
import type { WorkItem } from '@/api/workitems'
import type { WorkflowStatus } from '@/api/workflows'

const listItem: WorkItem = {
  id: 'item-79',
  project_key: 'TASK',
  item_number: 79,
  display_id: 'TASK-79',
  type: 'task',
  title: 'Item detail panel component',
  description: '## Goal\n\nBuild the right-side detail panel.',
  status: 'open',
  priority: 'medium',
  assignee_id: null,
  reporter_id: 'r1',
  reporter_name: 'Admin',
  queue_id: null,
  milestone_id: null,
  visibility: 'internal',
  labels: ['ui'],
  complexity: null,
  custom_fields: {},
  due_date: null,
  estimated_seconds: null,
  sla: null,
  sla_target_at: null,
  resolved_at: null,
  created_at: '2026-06-16T00:00:00Z',
  updated_at: '2026-07-14T00:00:00Z',
}

const statuses: WorkflowStatus[] = [
  { id: 's1', name: 'open', display_name: 'Open', category: 'todo', color: null, position: 0 },
]

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: { defaultValue?: string }) => opts?.defaultValue ?? key,
  }),
}))

vi.mock('@/contexts/AuthContext', () => ({
  useAuth: () => ({ user: { id: 'u1', global_role: 'admin' } }),
}))

vi.mock('@/hooks/useWorkItems', () => ({
  useWorkItem: vi.fn(),
  useUpdateWorkItem: () => ({ mutate: vi.fn(), isError: false, isPending: false }),
  useAttachments: () => ({ data: [] }),
  useRelations: () => ({ data: [] }),
}))

vi.mock('@/hooks/useProjects', () => ({
  useProject: () => ({ data: { allowed_complexity_values: [] } }),
  useMembers: () => ({ data: [] }),
  useTypeWorkflows: () => ({ data: [] }),
}))

vi.mock('@/hooks/useWorkflows', () => ({
  useProjectWorkflow: () => ({ statuses, transitionsMap: { open: [{ to_status: 'in_progress' }] } }),
  useProjectWorkflows: () => ({ data: [] }),
}))

vi.mock('@/hooks/useMilestones', () => ({
  useMilestones: () => ({ data: [] }),
}))

vi.mock('@/hooks/useDetailExtrasBreakpoint', () => ({
  useDetailExtrasBreakpoint: vi.fn(() => false),
}))

vi.mock('@/hooks/useConfirmFeedback', () => ({
  useConfirmFeedback: () => ({ confirmed: false, showConfirm: vi.fn() }),
}))

vi.mock('@/hooks/usePasteUpload', () => ({
  usePasteUpload: () => ({
    handlePaste: vi.fn(),
    handleDrop: vi.fn(),
    handleDragOver: vi.fn(),
  }),
}))

vi.mock('@/hooks/useMentionAutocomplete', () => ({
  useMentionAutocomplete: () => ({
    onMentionKeyDown: vi.fn(),
    mentionModalOpen: false,
    dropdownPosition: null,
    onMentionClose: vi.fn(),
    onMentionSelect: vi.fn(),
  }),
}))

vi.mock('@/components/workitems/DetailSidebar', () => ({
  DetailSidebar: () => <div data-testid="detail-sidebar">sidebar</div>,
}))

vi.mock('@/components/workitems/CommentList', () => ({
  CommentList: () => <div data-testid="comment-list">comments</div>,
}))

vi.mock('@/components/workitems/DescriptionWithInlineComments', () => ({
  DescriptionWithInlineComments: ({ description }: { description: string }) => (
    <div data-testid="description">{description}</div>
  ),
}))

vi.mock('@/components/workitems/RelationList', () => ({
  RelationList: () => <div data-testid="relation-list" />,
}))

vi.mock('@/components/workitems/DetailExtrasColumn', () => ({
  DetailExtrasColumn: () => <div data-testid="detail-extras-column">extras</div>,
}))

vi.mock('@/components/workitems/AttachmentList', () => ({
  AttachmentList: () => <div data-testid="attachment-list" />,
}))

vi.mock('@/components/workitems/TimeEntryList', () => ({
  TimeEntryList: () => <div data-testid="time-list" />,
}))

vi.mock('@/components/workitems/FilePreviewModal', () => ({
  FilePreviewModal: () => null,
}))

vi.mock('@/components/ui/MentionSearchModal', () => ({
  MentionSearchModal: () => null,
}))

vi.mock('@/components/ui/Modal', () => ({
  Modal: ({ open, children }: { open: boolean; children: React.ReactNode }) =>
    open ? <div data-testid="modal">{children}</div> : null,
}))

vi.mock('@/components/ui/ConfirmCheck', () => ({
  ConfirmCheck: () => null,
}))

vi.mock('@/components/ui/CopyButton', () => ({
  CopyButton: () => null,
}))

vi.mock('@/components/ui/Tooltip', () => ({
  Tooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}))

vi.mock('@/components/ui/Button', () => ({
  Button: ({ children, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button {...props}>{children}</button>
  ),
}))

vi.mock('@/components/workitems/PriorityBadge', () => ({
  PriorityBadge: () => <span>priority</span>,
}))
vi.mock('@/components/workitems/TypeBadge', () => ({
  TypeBadge: () => <span>type</span>,
}))
vi.mock('@/components/workitems/StatusBadge', () => ({
  StatusBadge: () => <span>status</span>,
}))

import { useWorkItem } from '@/hooks/useWorkItems'
import { useDetailExtrasBreakpoint } from '@/hooks/useDetailExtrasBreakpoint'

function renderPane(itemNumber = 79) {
  return renderToStaticMarkup(
    <MemoryRouter>
      <WorkItemDetailPane
        projectKey="TASK"
        itemNumber={itemNumber}
        listItem={listItem}
        statuses={statuses}
        fullPageHref="/projects/TASK/items/79"
        onClose={() => {}}
      />
    </MemoryRouter>,
  )
}

describe('WorkItemDetailPane', () => {
  beforeEach(() => {
    vi.mocked(useWorkItem).mockReset()
    vi.mocked(useDetailExtrasBreakpoint).mockReturnValue(false)
  })

  it('shows a skeleton while the full item is loading and list data is unavailable', () => {
    vi.mocked(useWorkItem).mockReturnValue({
      data: undefined,
      isLoading: true,
      isFetching: true,
      isSuccess: false,
      dataUpdatedAt: 0,
    } as ReturnType<typeof useWorkItem>)

    const html = renderToStaticMarkup(
      <MemoryRouter>
        <WorkItemDetailPane
          projectKey="TASK"
          itemNumber={79}
          listItem={null}
          statuses={statuses}
          fullPageHref="/projects/TASK/items/79"
          onClose={() => {}}
        />
      </MemoryRouter>,
    )

    expect(html).toContain('data-testid="work-item-detail-pane-skeleton"')
    expect(html).toContain('data-testid="work-item-detail-pane"')
  })

  it('renders editable detail surfaces from cached/fetched item data', () => {
    vi.mocked(useWorkItem).mockReturnValue({
      data: listItem,
      isLoading: false,
      isFetching: false,
      isSuccess: true,
      dataUpdatedAt: Date.now(),
    } as ReturnType<typeof useWorkItem>)

    const html = renderPane()
    expect(html).toContain('TASK-79')
    expect(html).toContain('Item detail panel component')
    expect(html).toContain('data-testid="detail-sidebar"')
    expect(html).toContain('data-testid="comment-list"')
    expect(html).toContain('data-testid="description"')
    expect(html).toContain('workitems.splitPane.openFull')
    expect(html).toContain('data-item-number="79"')
    expect(html).toContain('data-extras-column="false"')
    expect(html).not.toContain('data-testid="detail-extras-column"')
  })

  it('shows the extras column and side metadata at wide breakpoint', () => {
    vi.mocked(useDetailExtrasBreakpoint).mockReturnValue(true)
    vi.mocked(useWorkItem).mockReturnValue({
      data: listItem,
      isLoading: false,
      isFetching: false,
      isSuccess: true,
      dataUpdatedAt: Date.now(),
    } as ReturnType<typeof useWorkItem>)

    const html = renderPane()
    expect(html).toContain('data-extras-column="true"')
    expect(html).toContain('data-testid="detail-extras-column"')
    expect(html).toContain('data-testid="detail-sidebar"')
  })

  it('requests the work item with retainInCache enabled', () => {
    vi.mocked(useWorkItem).mockReturnValue({
      data: listItem,
      isLoading: false,
      isFetching: false,
      isSuccess: true,
      dataUpdatedAt: Date.now(),
    } as ReturnType<typeof useWorkItem>)

    renderPane()
    expect(useWorkItem).toHaveBeenCalledWith('TASK', 79, expect.objectContaining({ retainInCache: true }))
  })
})
