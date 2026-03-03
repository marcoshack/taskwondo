import { useRef, useState, useEffect, useCallback, forwardRef, type ReactNode } from 'react'
import { ChevronLeft, ChevronRight } from 'lucide-react'

interface ScrollableRowProps {
  children: ReactNode
  className?: string
  /** Extra classes on the inner scrollable container, e.g. 'pr-6' for right padding */
  contentClassName?: string
  /** Tailwind gradient from-* class for the background, e.g. 'from-white dark:from-gray-800' */
  gradientFrom?: string
}

export const ScrollableRow = forwardRef<HTMLDivElement, ScrollableRowProps>(function ScrollableRow({ children, className = '', contentClassName = '', gradientFrom = 'from-white dark:from-gray-800' }, forwardedRef) {
  const ref = useRef<HTMLDivElement>(null)
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
  }, [checkScroll])

  return (
    <div ref={forwardedRef} className={`relative ${className}`}>
      {canScrollLeft && (
        <>
          <span className={`absolute left-0 top-0 bottom-0 w-5 bg-gradient-to-r ${gradientFrom} to-transparent z-10 pointer-events-none`} />
          <ChevronLeft className="absolute left-0 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400 dark:text-gray-500 z-20 pointer-events-none" />
        </>
      )}
      <div
        ref={ref}
        className={`flex items-center gap-2 overflow-x-auto scrollbar-none ${contentClassName}`}
        onScroll={checkScroll}
      >
        {children}
      </div>
      {canScrollRight && (
        <>
          <span className={`absolute right-0 top-0 bottom-0 w-5 bg-gradient-to-l ${gradientFrom} to-transparent z-10 pointer-events-none`} />
          <ChevronRight className="absolute right-0 top-1/2 -translate-y-1/2 w-3.5 h-3.5 text-gray-400 dark:text-gray-500 z-20 pointer-events-none" />
        </>
      )}
    </div>
  )
})
