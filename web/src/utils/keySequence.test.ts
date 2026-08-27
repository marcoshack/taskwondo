import { describe, it, expect } from 'vitest'
import { hasChordModifier, resolveSequenceKey } from './keySequence'
import type { SequenceComboSpec } from './keySequence'

const COMBOS: SequenceComboSpec[] = [
  { id: 'go-to-inbox', keys: ['g', 'i'] },
  { id: 'go-to-items', keys: ['g', 'o'] },
]

describe('hasChordModifier', () => {
  it('is false for a bare key', () => {
    expect(hasChordModifier({ key: 'k' })).toBe(false)
  })

  it.each(['ctrlKey', 'metaKey', 'altKey'] as const)('is true for %s', (mod) => {
    expect(hasChordModifier({ key: 'k', [mod]: true })).toBe(true)
  })

  it('ignores explicitly false modifiers', () => {
    expect(hasChordModifier({ key: 'k', ctrlKey: false, metaKey: false, altKey: false })).toBe(false)
  })
})

describe('resolveSequenceKey with nothing pending', () => {
  it('starts a sequence on a key that opens a combo', () => {
    expect(resolveSequenceKey(null, { key: 'g' }, COMBOS)).toEqual({
      action: 'start',
      pendingKey: 'g',
    })
  })

  it('lowercases the pending key so Shift+G still arms the sequence', () => {
    expect(resolveSequenceKey(null, { key: 'G' }, COMBOS)).toEqual({
      action: 'start',
      pendingKey: 'g',
    })
  })

  it('passes through a key that opens no combo', () => {
    expect(resolveSequenceKey(null, { key: 'x' }, COMBOS)).toEqual({ action: 'passThrough' })
  })

  it('passes through the second key of a combo pressed on its own', () => {
    expect(resolveSequenceKey(null, { key: 'i' }, COMBOS)).toEqual({ action: 'passThrough' })
  })

  it('passes through a chord rather than arming a sequence', () => {
    expect(resolveSequenceKey(null, { key: 'g', metaKey: true }, COMBOS)).toEqual({
      action: 'passThrough',
    })
  })

  it('passes through Cmd+K so the palette shortcut sees it', () => {
    expect(resolveSequenceKey(null, { key: 'k', metaKey: true }, COMBOS)).toEqual({
      action: 'passThrough',
    })
  })

  it('passes through everything when no combos are registered', () => {
    expect(resolveSequenceKey(null, { key: 'g' }, [])).toEqual({ action: 'passThrough' })
  })
})

describe('resolveSequenceKey with a pending key', () => {
  it('completes the combo the second key matches', () => {
    expect(resolveSequenceKey('g', { key: 'i' }, COMBOS)).toEqual({
      action: 'complete',
      comboId: 'go-to-inbox',
    })
    expect(resolveSequenceKey('g', { key: 'o' }, COMBOS)).toEqual({
      action: 'complete',
      comboId: 'go-to-items',
    })
  })

  it('completes on an uppercase second key', () => {
    expect(resolveSequenceKey('g', { key: 'O' }, COMBOS)).toEqual({
      action: 'complete',
      comboId: 'go-to-items',
    })
  })

  // Defect 1: the unmatched second key used to fall through to a single-key shortcut.
  it('consumes an unmatched second key instead of letting it fall through', () => {
    expect(resolveSequenceKey('g', { key: 'p' }, COMBOS)).toEqual({ action: 'consume' })
    expect(resolveSequenceKey('g', { key: 'k' }, COMBOS)).toEqual({ action: 'consume' })
    expect(resolveSequenceKey('g', { key: 'x' }, COMBOS)).toEqual({ action: 'consume' })
  })

  it('consumes the second key even when no combos are registered at all', () => {
    expect(resolveSequenceKey('g', { key: 'i' }, [])).toEqual({ action: 'consume' })
  })

  it('does not complete a combo whose first key differs from the pending one', () => {
    expect(resolveSequenceKey('z', { key: 'i' }, COMBOS)).toEqual({ action: 'consume' })
  })

  // Defect 2: a chord used to be swallowed as the sequence's second key.
  it('cancels and passes through on Cmd+K so g then Cmd+K opens the palette', () => {
    expect(resolveSequenceKey('g', { key: 'k', metaKey: true }, COMBOS)).toEqual({
      action: 'cancel',
    })
  })

  it.each(['ctrlKey', 'metaKey', 'altKey'] as const)(
    'cancels and passes through on a %s chord',
    (mod) => {
      expect(resolveSequenceKey('g', { key: 'i', [mod]: true }, COMBOS)).toEqual({
        action: 'cancel',
      })
    },
  )
})

describe('resolveSequenceKey combo shape guards', () => {
  it('never starts a sequence for a single-key combo registration', () => {
    expect(resolveSequenceKey(null, { key: 'g' }, [{ id: 'lone', keys: ['g'] }])).toEqual({
      action: 'passThrough',
    })
  })

  it('never completes against a combo longer than two keys', () => {
    const long: SequenceComboSpec[] = [{ id: 'triple', keys: ['g', 'i', 'x'] }]
    expect(resolveSequenceKey('g', { key: 'i' }, long)).toEqual({ action: 'consume' })
  })
})
