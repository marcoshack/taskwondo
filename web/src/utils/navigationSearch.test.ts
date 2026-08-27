import { describe, it, expect } from 'vitest'
import { foldForSearch, matchNavigationItems } from './navigationSearch'

const items = [
  { label: 'Overview' },
  { label: 'Items' },
  { label: 'Milestones' },
  { label: 'Préférences' },
  { label: 'Señales' },
]

describe('foldForSearch', () => {
  it('lower-cases and strips diacritics', () => {
    expect(foldForSearch('Préférences')).toBe('preferences')
    expect(foldForSearch('SEÑALES')).toBe('senales')
    expect(foldForSearch('Überblick')).toBe('uberblick')
  })

  it('leaves non-latin scripts alone', () => {
    expect(foldForSearch('概览')).toBe('概览')
    expect(foldForSearch('受信トレイ')).toBe('受信トレイ')
  })
})

describe('matchNavigationItems', () => {
  it('returns everything for an empty query', () => {
    expect(matchNavigationItems(items, '')).toEqual(items)
    expect(matchNavigationItems(items, '   ')).toEqual(items)
  })

  it('matches from the first character — no two-character floor', () => {
    expect(matchNavigationItems(items, 'v')).toEqual([{ label: 'Overview' }])
    expect(matchNavigationItems(items, 'm')).toEqual([{ label: 'Items' }, { label: 'Milestones' }])
  })

  it('matches anywhere in the label, not just the prefix', () => {
    expect(matchNavigationItems(items, 'view')).toEqual([{ label: 'Overview' }])
  })

  it('is case-insensitive', () => {
    expect(matchNavigationItems(items, 'MILESTONES')).toEqual([{ label: 'Milestones' }])
    expect(matchNavigationItems(items, 'milestones')).toEqual([{ label: 'Milestones' }])
  })

  it('is accent-insensitive in both directions', () => {
    expect(matchNavigationItems(items, 'prefer')).toEqual([{ label: 'Préférences' }])
    expect(matchNavigationItems(items, 'Préf')).toEqual([{ label: 'Préférences' }])
    expect(matchNavigationItems(items, 'senales')).toEqual([{ label: 'Señales' }])
  })

  it('ignores surrounding whitespace', () => {
    expect(matchNavigationItems(items, '  items  ')).toEqual([{ label: 'Items' }])
  })

  it('returns an empty list when nothing matches', () => {
    expect(matchNavigationItems(items, 'zzz')).toEqual([])
  })

  it('preserves catalog order', () => {
    expect(matchNavigationItems(items, 'e').map((i) => i.label)).toEqual([
      'Overview',
      'Items',
      'Milestones',
      'Préférences',
      'Señales',
    ])
  })

  it('does not mutate or alias the input', () => {
    const result = matchNavigationItems(items, '')
    expect(result).not.toBe(items)
    expect(items).toHaveLength(5)
  })
})
