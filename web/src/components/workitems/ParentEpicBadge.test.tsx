import { describe, it, expect, vi } from 'vitest'
import { renderToStaticMarkup } from 'react-dom/server'
import { ParentEpicBadge, shouldShowParentEpic } from './ParentEpicBadge'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

describe('ParentEpicBadge', () => {
  it('renders display id with tooltip metadata', () => {
    const html = renderToStaticMarkup(
      <ParentEpicBadge displayId="TASK-76" title="Split-pane list view" />,
    )
    expect(html).toContain('data-testid="parent-epic-badge"')
    expect(html).toContain('data-epic-id="TASK-76"')
    expect(html).toContain('TASK-76')
    expect(html).toContain('workitems.parentEpic')
  })
})

describe('shouldShowParentEpic', () => {
  it('is true only for non-epic items with a parent epic id', () => {
    expect(shouldShowParentEpic({ type: 'ticket', parent_epic_display_id: 'TASK-76' })).toBe(true)
    expect(shouldShowParentEpic({ type: 'task', parent_epic_display_id: 'TASK-76' })).toBe(true)
    expect(shouldShowParentEpic({ type: 'epic', parent_epic_display_id: 'TASK-76' })).toBe(false)
    expect(shouldShowParentEpic({ type: 'ticket', parent_epic_display_id: null })).toBe(false)
    expect(shouldShowParentEpic({ type: 'ticket' })).toBe(false)
  })
})
