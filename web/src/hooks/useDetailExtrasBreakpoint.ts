import { useEffect, useState } from 'react'

/** Viewport ≥1600px — third "Relations & extras" column in work-item detail. */
export const DETAIL_EXTRAS_BREAKPOINT = '(min-width: 1600px)'

/**
 * True when the viewport is wide enough for the detail extras column (TASK-117).
 * Defaults to false for SSR / first paint; updates on resize.
 */
export function useDetailExtrasBreakpoint(): boolean {
  const [matches, setMatches] = useState(false)

  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return
    const mq = window.matchMedia(DETAIL_EXTRAS_BREAKPOINT)
    const update = () => setMatches(mq.matches)
    update()
    mq.addEventListener('change', update)
    return () => mq.removeEventListener('change', update)
  }, [])

  return matches
}
