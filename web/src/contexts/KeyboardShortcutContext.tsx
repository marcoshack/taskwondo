import { createContext, useContext, useRef, useCallback, useLayoutEffect } from 'react'
import type { ReactNode } from 'react'

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

      const key = e.key.toLowerCase()

      // Check if this key completes a pending combo
      if (pendingKeyRef.current) {
        const sequence = [pendingKeyRef.current, key]
        clearTimeout(timeoutRef.current)
        pendingKeyRef.current = null

        for (const combo of combosRef.current) {
          if (combo.keys.length === sequence.length &&
              combo.keys.every((k, i) => k === sequence[i])) {
            e.preventDefault()
            combo.callback()
            return
          }
        }
        // No match — fall through
        return
      }

      // Check if this key starts a combo
      const startsCombo = combosRef.current.some((c) => c.keys[0] === key)
      if (startsCombo) {
        pendingKeyRef.current = key
        timeoutRef.current = setTimeout(() => {
          pendingKeyRef.current = null
        }, COMBO_TIMEOUT_MS)
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
