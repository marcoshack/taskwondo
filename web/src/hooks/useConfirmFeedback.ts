import { useState, useRef, useCallback } from 'react'

export function useConfirmFeedback(duration: number = 1500) {
  const [confirmed, setConfirmed] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout>>(undefined)

  const showConfirm = useCallback(() => {
    clearTimeout(timerRef.current)
    setConfirmed(true)
    timerRef.current = setTimeout(() => setConfirmed(false), duration)
  }, [duration])

  return { confirmed, showConfirm }
}
