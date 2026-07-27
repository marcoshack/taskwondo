import { describe, it, expect } from 'vitest'
import { projectSwitchSuffix } from './projectSection'

describe('projectSwitchSuffix', () => {
  it('keeps top-level sections', () => {
    expect(projectSwitchSuffix('items')).toBe('/items')
    expect(projectSwitchSuffix('queues')).toBe('/queues')
    expect(projectSwitchSuffix('milestones')).toBe('/milestones')
    expect(projectSwitchSuffix('workflows')).toBe('/workflows')
    expect(projectSwitchSuffix('settings')).toBe('/settings')
  })

  it('collapses detail pages to their list', () => {
    expect(projectSwitchSuffix('items/42')).toBe('/items')
    expect(projectSwitchSuffix('queues/f9c1-uuid')).toBe('/queues')
    expect(projectSwitchSuffix('queues/f9c1-uuid/items')).toBe('/queues')
    expect(projectSwitchSuffix('milestones/f9c1-uuid')).toBe('/milestones')
  })

  it('falls back to the project overview', () => {
    expect(projectSwitchSuffix('')).toBe('')
    expect(projectSwitchSuffix(undefined)).toBe('')
    expect(projectSwitchSuffix(null)).toBe('')
    expect(projectSwitchSuffix('teams/f9c1-uuid')).toBe('')
    expect(projectSwitchSuffix('support')).toBe('')
    expect(projectSwitchSuffix('support/7')).toBe('')
    expect(projectSwitchSuffix('nonsense')).toBe('')
  })

  it('tolerates a leading slash', () => {
    expect(projectSwitchSuffix('/items/42')).toBe('/items')
  })
})
