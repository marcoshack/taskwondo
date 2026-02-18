import { useEffect, useRef } from 'react'
import { useKeyboardShortcutContext } from '@/contexts/KeyboardShortcutContext'

interface KeyboardShortcutOptions {
  key: string
  ctrlKey?: boolean
  metaKey?: boolean
  /** If true, fires even when focus is in INPUT/TEXTAREA/SELECT */
  allowInInput?: boolean
}

export function useKeyboardShortcut(
  options: KeyboardShortcutOptions | KeyboardShortcutOptions[],
  callback: (e: KeyboardEvent) => void,
  enabled: boolean = true,
) {
  const { isModalOpen } = useKeyboardShortcutContext()
  const callbackRef = useRef(callback)
  callbackRef.current = callback

  const optsRef = useRef(options)
  optsRef.current = options

  useEffect(() => {
    if (!enabled) return

    const handler = (e: KeyboardEvent) => {
      if (isModalOpen()) return

      const opts = Array.isArray(optsRef.current) ? optsRef.current : [optsRef.current]

      for (const opt of opts) {
        if (e.key.toLowerCase() !== opt.key.toLowerCase()) continue
        if (opt.ctrlKey && !(e.ctrlKey || e.metaKey)) continue
        if (opt.metaKey && !e.metaKey) continue
        if (!opt.ctrlKey && !opt.metaKey && (e.ctrlKey || e.metaKey || e.altKey)) continue

        if (!opt.allowInInput) {
          const tag = (e.target as HTMLElement).tagName
          if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return
          if ((e.target as HTMLElement).isContentEditable) return
        }

        e.preventDefault()
        callbackRef.current(e)
        return
      }
    }

    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [enabled, isModalOpen])
}
