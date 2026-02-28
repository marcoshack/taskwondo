import { useRef, useState, useEffect, useCallback } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'

interface ScrollableDateProps {
  date: string
  className?: string
}

export function ScrollableDate({ date, className = '' }: ScrollableDateProps) {
  const ref = useRef<HTMLSpanElement>(null)
  const [canScrollLeft, setCanScrollLeft] = useState(false)
  const [canScrollRight, setCanScrollRight] = useState(false)

  const checkScroll = useCallback(() => {
    const el = ref.current
    if (!el) return
    setCanScrollLeft(el.scrollLeft > 0)
    setCanScrollRight(el.scrollLeft + el.clientWidth < el.scrollWidth - 1)
  }, [])

  useEffect(() => {
    checkScroll()
    const el = ref.current
    if (!el) return
    const observer = new ResizeObserver(checkScroll)
    observer.observe(el)
    return () => observer.disconnect()
  }, [checkScroll, date])

  return (
    <span className={`relative inline-flex items-center min-w-0 ${className}`}>
      {canScrollLeft && (
        <>
          <span className="absolute left-0 top-0 bottom-0 w-4 bg-gradient-to-r from-white dark:from-gray-900 to-transparent z-10 pointer-events-none" />
          <ChevronLeft
            className="absolute left-0 w-3 h-3 text-gray-400 dark:text-gray-500 z-20 pointer-events-none"
          />
        </>
      )}
      <span
        ref={ref}
        className="overflow-x-auto scrollbar-none text-xs text-gray-400 dark:text-gray-500 whitespace-nowrap min-w-0"
        onScroll={checkScroll}
      >
        {new Date(date).toLocaleString()}
      </span>
      {canScrollRight && (
        <>
          <span className="absolute right-0 top-0 bottom-0 w-4 bg-gradient-to-l from-white dark:from-gray-900 to-transparent z-10 pointer-events-none" />
          <ChevronRight
            className="absolute right-0 w-3 h-3 text-gray-400 dark:text-gray-500 z-20 pointer-events-none"
          />
        </>
      )}
    </span>
  )
}
