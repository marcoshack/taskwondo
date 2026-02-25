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
  size?: 'default' | 'full'
  dismissable?: boolean
  className?: string
  children: ReactNode
}

export function Modal({ open, onClose, title, position = 'center', size = 'default', dismissable = true, className, children }: ModalProps) {
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
      if (e.key === 'Escape' && dismissable) onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  if (!open) return null

  return createPortal(
    <div className={`fixed inset-0 z-50 flex justify-center ${position === 'top' ? 'items-start pt-4' : 'items-center'}`}>
      <div className="fixed inset-0 bg-black/50" onClick={dismissable ? onClose : undefined} />
      <div className={`relative bg-white dark:bg-gray-800/40 dark:backdrop-blur-sm rounded-lg shadow-xl ${size === 'full' ? 'w-[96vw] h-[96vh] p-0 flex flex-col overflow-hidden' : 'max-w-lg w-full mx-4 p-6 max-h-[90vh] overflow-y-auto overscroll-contain'} ${className ?? ''}`}>
        {title && (
          <div className="flex items-center justify-between mb-4">
            <h2 className="flex-1 text-lg font-semibold text-gray-900 dark:text-gray-100">{title}</h2>
            <button
              onClick={onClose}
              className="p-1 rounded-md text-gray-400 hover:text-gray-600 hover:bg-gray-100 dark:hover:text-gray-300 dark:hover:bg-gray-700"
            >
              <X className="h-5 w-5" />
            </button>
          </div>
        )}
        {children}
      </div>
    </div>,
    document.body,
  )
}
