import { createContext, useContext, useRef, useCallback, useLayoutEffect } from 'react'
import type { ReactNode } from 'react'
import { resolveSequenceKey } from '@/utils/keySequence'

interface SequentialCombo {
  keys: string[]
  callback: () => void
  id: string
}

interface KeyboardShortcutContextValue {
  incrementModalOpen: () => void
  decrementModalOpen: () => void
  registerSequentialCombo: (combo: SequentialCombo) => () => void
  isModalOpen: () => boolean
}

const KeyboardShortcutContext = createContext<KeyboardShortcutContextValue | null>(null)

const COMBO_TIMEOUT_MS = 800

export function KeyboardShortcutProvider({ children }: { children: ReactNode }) {
  const modalOpenCount = useRef(0)
  const combosRef = useRef<SequentialCombo[]>([])
  const pendingKeyRef = useRef<string | null>(null)
  const timeoutRef = useRef<ReturnType<typeof setTimeout>>(undefined)

  const incrementModalOpen = useCallback(() => {
    modalOpenCount.current++
  }, [])

  const decrementModalOpen = useCallback(() => {
    modalOpenCount.current = Math.max(0, modalOpenCount.current - 1)
  }, [])

  const isModalOpen = useCallback(() => modalOpenCount.current > 0, [])

  const registerSequentialCombo = useCallback((combo: SequentialCombo) => {
    combosRef.current.push(combo)
    return () => {
      combosRef.current = combosRef.current.filter((c) => c.id !== combo.id)
    }
  }, [])

  useLayoutEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (modalOpenCount.current > 0) return

      const tag = (e.target as HTMLElement).tagName
      if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
      if ((e.target as HTMLElement).isContentEditable) return

      const decision = resolveSequenceKey(pendingKeyRef.current, e, combosRef.current)

      if (decision.action !== 'start') {
        clearTimeout(timeoutRef.current)
        pendingKeyRef.current = null
      }

      switch (decision.action) {
        case 'start':
          pendingKeyRef.current = decision.pendingKey
          clearTimeout(timeoutRef.current)
          timeoutRef.current = setTimeout(() => {
            pendingKeyRef.current = null
          }, COMBO_TIMEOUT_MS)
          return
        case 'complete': {
          // The second key of a sequence belongs to the sequence. This provider's
          // listener is registered in a layout effect, ahead of every
          // `useKeyboardShortcut` listener (passive effects), so stopping
          // immediate propagation reliably keeps the key from also firing a bare
          // single-key shortcut such as `o` on a list page.
          e.preventDefault()
          e.stopImmediatePropagation()
          combosRef.current.find((c) => c.id === decision.comboId)?.callback()
          return
        }
        case 'consume':
          // Matched nothing, but it was still the sequence's second key.
          e.preventDefault()
          e.stopImmediatePropagation()
          return
        case 'cancel':
        case 'passThrough':
          // A chord is never a sequence key: any pending sequence was dropped
          // above and the event continues on to the chord's own handler.
          return
      }
    }

    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [])

  return (
    <KeyboardShortcutContext.Provider value={{ incrementModalOpen, decrementModalOpen, registerSequentialCombo, isModalOpen }}>
      {children}
    </KeyboardShortcutContext.Provider>
  )
}

export function useKeyboardShortcutContext() {
  const ctx = useContext(KeyboardShortcutContext)
  if (!ctx) throw new Error('useKeyboardShortcutContext must be used within KeyboardShortcutProvider')
  return ctx
}
