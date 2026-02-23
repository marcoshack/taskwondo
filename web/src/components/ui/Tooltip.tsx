import { useState, useRef, useEffect, useCallback, type ReactNode } from 'react'
import { createPortal } from 'react-dom'

interface TooltipProps {
  content: string | undefined
  children: ReactNode
  position?: 'top' | 'bottom' | 'left' | 'right'
  className?: string
}

const GAP = 6

function getTooltipStyle(
  triggerRect: DOMRect,
  tooltipEl: HTMLElement,
  position: 'top' | 'bottom' | 'left' | 'right',
): React.CSSProperties {
  const tw = tooltipEl.offsetWidth
  const th = tooltipEl.offsetHeight

  let top = 0
  let left = 0

  switch (position) {
    case 'top':
      top = triggerRect.top - th - GAP
      left = triggerRect.left + triggerRect.width / 2 - tw / 2
      break
    case 'bottom':
      top = triggerRect.bottom + GAP
      left = triggerRect.left + triggerRect.width / 2 - tw / 2
      break
    case 'left':
      top = triggerRect.top + triggerRect.height / 2 - th / 2
      left = triggerRect.left - tw - GAP
      break
    case 'right':
      top = triggerRect.top + triggerRect.height / 2 - th / 2
      left = triggerRect.right + GAP
      break
  }

  // Clamp to viewport
  left = Math.max(4, Math.min(left, window.innerWidth - tw - 4))
  top = Math.max(4, Math.min(top, window.innerHeight - th - 4))

  return { position: 'fixed', top, left }
}

const arrowStyle: Record<string, React.CSSProperties> = {
  top: { position: 'absolute', top: '100%', left: '50%', transform: 'translateX(-50%)' },
  bottom: { position: 'absolute', bottom: '100%', left: '50%', transform: 'translateX(-50%)' },
  left: { position: 'absolute', left: '100%', top: '50%', transform: 'translateY(-50%)' },
  right: { position: 'absolute', right: '100%', top: '50%', transform: 'translateY(-50%)' },
}

const arrowClasses = {
  top: 'border-t-gray-600 dark:border-t-gray-600 border-x-transparent border-b-transparent',
  bottom: 'border-b-gray-600 dark:border-b-gray-600 border-x-transparent border-t-transparent',
  left: 'border-l-gray-600 dark:border-l-gray-600 border-y-transparent border-r-transparent',
  right: 'border-r-gray-600 dark:border-r-gray-600 border-y-transparent border-l-transparent',
}

export function Tooltip({ content, children, position = 'top', className }: TooltipProps) {
  const [visible, setVisible] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)
  const triggerRef = useRef<HTMLSpanElement>(null)
  const tooltipRef = useRef<HTMLSpanElement>(null)
  const [style, setStyle] = useState<React.CSSProperties>({})

  const updatePosition = useCallback(() => {
    if (!triggerRef.current || !tooltipRef.current) return
    const rect = triggerRef.current.getBoundingClientRect()
    setStyle(getTooltipStyle(rect, tooltipRef.current, position))
  }, [position])

  useEffect(() => {
    if (visible) {
      // Use rAF to measure after the portal renders
      requestAnimationFrame(updatePosition)
    }
  }, [visible, updatePosition])

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
      ref={triggerRef}
      className={className ?? 'relative inline-flex'}
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
    >
      {children}
      {visible && createPortal(
        <span
          ref={tooltipRef}
          className="z-50 pointer-events-none whitespace-nowrap rounded bg-gray-600 dark:bg-gray-600 px-2 py-1 text-xs text-white shadow-lg animate-in fade-in duration-100"
          role="tooltip"
          style={style}
        >
          {content}
          <span className={`absolute border-4 ${arrowClasses[position]}`} style={arrowStyle[position]} />
        </span>,
        document.body,
      )}
    </span>
  )
}
