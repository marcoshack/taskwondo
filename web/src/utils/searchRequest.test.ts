import { describe, it, expect } from 'vitest'
import { buildSearchParams, meetsSearchFloor } from './searchRequest'

const params = (...args: Parameters<typeof buildSearchParams>) =>
  Object.fromEntries(buildSearchParams(...args))

describe('buildSearchParams', () => {
  it('always sends the query', () => {
    expect(params({ query: 'milestone' })).toEqual({ q: 'milestone' })
  })

  it('omits project entirely when none is active — the unscoped global search', () => {
    expect(params({ query: 'ab' })).not.toHaveProperty('project')
    expect(params({ query: 'ab', project: undefined })).not.toHaveProperty('project')
    expect(params({ query: 'ab', project: null })).not.toHaveProperty('project')
    expect(params({ query: 'ab', project: '' })).not.toHaveProperty('project')
    expect(params({ query: 'ab', project: '   ' })).not.toHaveProperty('project')
  })

  it('sends the project key — a key like TF, never a UUID', () => {
    expect(params({ query: 'ab', project: 'TF' })).toEqual({ q: 'ab', project: 'TF' })
  })

  it('trims the project key', () => {
    expect(params({ query: 'ab', project: '  TF  ' }).project).toBe('TF')
  })

  it('carries entity types and limit alongside the project scope', () => {
    expect(params({ query: 'ab', entityTypes: ['work_item', 'milestone'], limit: 20, project: 'TF' }))
      .toEqual({ q: 'ab', entity_type: 'work_item,milestone', limit: '20', project: 'TF' })
  })

  it('omits empty optional parameters', () => {
    expect(params({ query: 'ab', entityTypes: [], limit: 0 })).toEqual({ q: 'ab' })
  })

  it('url-encodes the query and the project key', () => {
    expect(buildSearchParams({ query: 'a b&c', project: 'T F' }).toString())
      .toBe('q=a+b%26c&project=T+F')
  })
})

describe('meetsSearchFloor', () => {
  it('rejects empty or whitespace-only queries', () => {
    expect(meetsSearchFloor('')).toBe(false)
    expect(meetsSearchFloor('   ')).toBe(false)
  })

  it('requires two characters for latin scripts', () => {
    expect(meetsSearchFloor('a')).toBe(false)
    expect(meetsSearchFloor('ab')).toBe(true)
    expect(meetsSearchFloor('login')).toBe(true)
  })

  it('accepts a single CJK character — one hanzi is already a word', () => {
    expect(meetsSearchFloor('猫')).toBe(true)
    expect(meetsSearchFloor('猫 崩溃')).toBe(true)
  })

  it('accepts single Hiragana/Katakana/Hangul characters', () => {
    expect(meetsSearchFloor('ね')).toBe(true)
    expect(meetsSearchFloor('ガ')).toBe(true)
    expect(meetsSearchFloor('한')).toBe(true)
  })

  it('ignores a single latin character padded with CJK punctuation only', () => {
    expect(meetsSearchFloor('！a！')).toBe(true) // 3 characters: over the length floor anyway
  })
})
