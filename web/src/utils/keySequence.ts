/**
 * Pure decision logic for the sequential-combo layer (`g` then `i`, `g` then `o`).
 *
 * Kept free of React and of the DOM so the rules can be unit-tested exhaustively.
 * `KeyboardShortcutContext` owns the pending-key ref and the timeout; this module
 * only answers "given what is pending and what was just pressed, what happens?".
 */

/** The subset of `KeyboardEvent` the sequence rules look at. */
export interface SequenceKeyEvent {
  key: string
  ctrlKey?: boolean
  metaKey?: boolean
  altKey?: boolean
}

/** The subset of a registered combo the sequence rules look at. */
export interface SequenceComboSpec {
  id: string
  keys: readonly string[]
}

export type SequenceDecision =
  /** Nothing pending and the key starts nothing — let other handlers have it. */
  | { action: 'passThrough' }
  /** The key is the first half of at least one combo — arm the sequence. */
  | { action: 'start'; pendingKey: string }
  /** The key completed a combo — fire it and swallow the key. */
  | { action: 'complete'; comboId: string }
  /** The key was the second half of a sequence but matched nothing — swallow it anyway. */
  | { action: 'consume' }
  /** A modifier chord arrived mid-sequence — drop the sequence, let the chord through. */
  | { action: 'cancel' }

/** True when the press carries a modifier that makes it a chord rather than a plain key. */
export function hasChordModifier(e: SequenceKeyEvent): boolean {
  return e.ctrlKey === true || e.metaKey === true || e.altKey === true
}

/**
 * Decide what the sequence layer does with a keypress.
 *
 * The two rules that matter beyond the happy path:
 *  - The second key of a sequence belongs to the sequence whether or not it
 *    matched, so an unmatched one is consumed rather than falling through to a
 *    single-key shortcut.
 *  - A modifier chord is never a sequence's second key. It cancels the pending
 *    sequence and passes through, so `g` then Cmd+K still opens the palette.
 */
export function resolveSequenceKey(
  pendingKey: string | null,
  e: SequenceKeyEvent,
  combos: readonly SequenceComboSpec[],
): SequenceDecision {
  const key = e.key.toLowerCase()

  if (pendingKey !== null) {
    if (hasChordModifier(e)) return { action: 'cancel' }
    const match = combos.find(
      (c) => c.keys.length === 2 && c.keys[0] === pendingKey && c.keys[1] === key,
    )
    return match ? { action: 'complete', comboId: match.id } : { action: 'consume' }
  }

  if (hasChordModifier(e)) return { action: 'passThrough' }
  const starts = combos.some((c) => c.keys.length > 1 && c.keys[0] === key)
  return starts ? { action: 'start', pendingKey: key } : { action: 'passThrough' }
}
