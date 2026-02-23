import { useState, useRef, useEffect, type ReactNode } from 'react'

interface TooltipProps {
  content: string | undefined
  children: ReactNode
  position?: 'top' | 'bottom' | 'left' | 'right'
  className?: string
}

const positionClasses = {
  top: 'bottom-full left-1/2 -translate-x-1/2 mb-1.5',
  bottom: 'top-full left-1/2 -translate-x-1/2 mt-1.5',
  left: 'right-full top-1/2 -translate-y-1/2 mr-1.5',
  right: 'left-full top-1/2 -translate-y-1/2 ml-1.5',
}

const arrowClasses = {
  top: 'top-full left-1/2 -translate-x-1/2 border-t-gray-600 dark:border-t-gray-600 border-x-transparent border-b-transparent',
  bottom: 'bottom-full left-1/2 -translate-x-1/2 border-b-gray-600 dark:border-b-gray-600 border-x-transparent border-t-transparent',
  left: 'left-full top-1/2 -translate-y-1/2 border-l-gray-600 dark:border-l-gray-600 border-y-transparent border-r-transparent',
  right: 'right-full top-1/2 -translate-y-1/2 border-r-gray-600 dark:border-r-gray-600 border-y-transparent border-l-transparent',
}

export function Tooltip({ content, children, position = 'top', className }: TooltipProps) {
  const [visible, setVisible] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current)
    }
  }, [])

  if (!content) return <>{children}</>

  function show() {
    timerRef.current = setTimeout(() => setVisible(true), 300)
  }

  function hide() {
    if (timerRef.current) clearTimeout(timerRef.current)
    timerRef.current = null
    setVisible(false)
  }

  return (
    <span
      className={className ?? 'relative inline-flex'}
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
    >
      {children}
      {visible && (
        <span
          className={`absolute ${positionClasses[position]} z-50 pointer-events-none whitespace-nowrap rounded bg-gray-600 dark:bg-gray-600 px-2 py-1 text-xs text-white shadow-lg animate-in fade-in duration-100`}
          role="tooltip"
        >
          {content}
          <span className={`absolute border-4 ${arrowClasses[position]}`} />
        </span>
      )}
    </span>
  )
}
