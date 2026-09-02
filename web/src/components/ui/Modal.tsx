import { useEffect } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { useKeyboardShortcutContext } from '@/contexts/KeyboardShortcutContext'

interface ModalProps {
  open: boolean
  onClose: () => void
  title?: ReactNode
  position?: 'center' | 'top'
  size?: 'default' | 'full' | 'wide'
  /** Rendered in the header between the title and the close button. */
  headerRight?: ReactNode
  dismissable?: boolean
  /**
   * Handles Escape instead of onClose. Use for modals that must not close
   * outright — a form that asks to confirm discarding what was typed.
   */
  onEscape?: () => void
  className?: string
  containerClassName?: string
  children: ReactNode
}

const sizeClasses: Record<NonNullable<ModalProps['size']>, string> = {
  default: 'max-w-lg w-full mx-4 p-6 max-h-[90vh] overflow-y-auto overscroll-contain',
  full: 'w-[96vw] h-[96vh] p-0 flex flex-col overflow-hidden',
  // A definite height (not just a cap) so the body can divide the space between
  // a scrolling fields column and a description that fills what is left.
  wide: 'max-w-[67.2rem] w-full mx-4 p-0 max-h-[90vh] md:h-[min(90vh,56rem)] flex flex-col overflow-hidden',
}

export function Modal({ open, onClose, title, position = 'center', size = 'default', headerRight, dismissable = true, onEscape, className, containerClassName, children }: ModalProps) {
  const { incrementModalOpen, decrementModalOpen } = useKeyboardShortcutContext()

  useEffect(() => {
    if (!open) return
    incrementModalOpen()
    const prev = document.body.style.overflow
    document.body.style.overflow = 'hidden'
    return () => {
      decrementModalOpen()
      document.body.style.overflow = prev
    }
  }, [open, incrementModalOpen, decrementModalOpen])

  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return
      if (onEscape) onEscape()
      else if (dismissable) onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose, onEscape, dismissable])

  if (!open) return null

  return createPortal(
    <div className={`fixed inset-0 z-50 flex justify-center ${position === 'top' ? 'items-start pt-4' : 'items-center'} ${containerClassName ?? ''}`}>
      <div className="fixed inset-0 bg-black/50" onClick={dismissable ? onClose : undefined} />
      <div role="dialog" aria-modal="true" className={`relative bg-[var(--surface)] rounded-[var(--radius-lg)] shadow-lg ${sizeClasses[size]} ${className ?? ''}`}>
        {title && (
          // 'wide' owns its own padding and separates the header with a rule so
          // the body can scroll under a pinned header and footer.
          <div className={`flex items-center gap-3 ${size === 'wide' ? 'shrink-0 border-b border-[var(--border)] px-6 py-4' : 'mb-4'}`}>
            <h2 className="flex-1 min-w-0 truncate text-lg font-semibold text-[var(--foreground)]">{title}</h2>
            {headerRight}
            {/* Matching flex-1 on both sides centres headerRight in the row. */}
            <div className={headerRight ? 'flex flex-1 justify-end' : ''}>
              <button
                onClick={onClose}
                className="p-1 rounded-[var(--radius)] text-[var(--foreground-muted)] hover:text-[var(--foreground)] hover:bg-[var(--hover)] transition-colors"
              >
                <X className="h-5 w-5" />
              </button>
            </div>
          </div>
        )}
        {children}
      </div>
    </div>,
    document.body,
  )
}
