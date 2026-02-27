import { useState, useCallback, useRef, useEffect } from 'react'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'

function loadFromLocalStorage(lsKey: string): Record<string, number> {
  try {
    const stored = localStorage.getItem(lsKey)
    return stored ? JSON.parse(stored) : {}
  } catch {
    return {}
  }
}

export function useColumnWidths(scope: string = 'workItems') {
  const prefKey = `columnWidths_${scope}`
  const lsKey = `taskwondo_${prefKey}`

  const [widths, setWidths] = useState<Record<string, number>>(() => loadFromLocalStorage(lsKey))

  const { data: apiWidths } = usePreference<Record<string, number>>(prefKey)
  const appliedApiRef = useRef(false)

  useEffect(() => {
    if (apiWidths && !appliedApiRef.current) {
      appliedApiRef.current = true
      setWidths((prev) => {
        const merged = { ...prev, ...apiWidths }
        localStorage.setItem(lsKey, JSON.stringify(merged))
        return merged
      })
    }
  }, [apiWidths, lsKey])

  const setPreference = useSetPreference()
  const saveTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined)

  const onColumnResize = useCallback((key: string, width: number) => {
    setWidths((prev) => {
      const next = { ...prev, [key]: width }
      localStorage.setItem(lsKey, JSON.stringify(next))
      clearTimeout(saveTimerRef.current)
      saveTimerRef.current = setTimeout(() => {
        setPreference.mutate({ key: prefKey, value: next })
      }, 500)
      return next
    })
  }, [setPreference, lsKey, prefKey])

  const resetColumnWidth = useCallback((key: string) => {
    setWidths((prev) => {
      const next = { ...prev }
      delete next[key]
      localStorage.setItem(lsKey, JSON.stringify(next))
      clearTimeout(saveTimerRef.current)
      saveTimerRef.current = setTimeout(() => {
        setPreference.mutate({ key: prefKey, value: next })
      }, 500)
      return next
    })
  }, [setPreference, lsKey, prefKey])

  const resetWidths = useCallback(() => {
    setWidths({})
    localStorage.removeItem(lsKey)
    setPreference.mutate({ key: prefKey, value: {} })
  }, [setPreference, lsKey, prefKey])

  return { columnWidths: widths, onColumnResize, resetColumnWidth, resetWidths }
}
