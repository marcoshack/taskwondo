import { useEffect } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import type { ReactNode } from 'react'
import { useKeyboardShortcutContext } from '@/contexts/KeyboardShortcutContext'

interface ModalProps {
  open: boolean
  onClose: () => void
  title?: string
  position?: 'center' | 'top'
  size?: 'default' | 'full'
  className?: string
  children: ReactNode
}

export function Modal({ open, onClose, title, position = 'center', size = 'default', className, children }: ModalProps) {
  const { incrementModalOpen, decrementModalOpen } = useKeyboardShortcutContext()

  useEffect(() => {
    if (!open) return
    incrementModalOpen()
    return () => decrementModalOpen()
  }, [open, incrementModalOpen, decrementModalOpen])

  useEffect(() => {
    if (!open) return
    const handler = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handler)
    return () => document.removeEventListener('keydown', handler)
  }, [open, onClose])

  if (!open) return null

  return createPortal(
    <div className={`fixed inset-0 z-50 flex justify-center ${position === 'top' ? 'items-start pt-4' : 'items-center'}`}>
      <div className="fixed inset-0 bg-black/50" onClick={onClose} />
      <div className={`relative bg-white/40 dark:bg-gray-800/40 backdrop-blur-sm rounded-lg shadow-xl ${size === 'full' ? 'w-[96vw] h-[96vh] p-0 flex flex-col overflow-hidden' : 'max-w-lg w-full mx-4 p-6'} ${className ?? ''}`}>
        {title && (
          <div className="flex items-center justify-between mb-4">
            <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100">{title}</h2>
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
