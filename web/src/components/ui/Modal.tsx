import { useEffect } from 'react'
import { createPortal } from 'react-dom'
import type { ReactNode } from 'react'

interface ModalProps {
  open: boolean
  onClose: () => void
  title?: string
  position?: 'center' | 'top'
  children: ReactNode
}

export function Modal({ open, onClose, title, position = 'center', children }: ModalProps) {
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
      <div className="relative bg-white dark:bg-gray-800 rounded-lg shadow-xl max-w-lg w-full mx-4 p-6">
        {title && <h2 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-4">{title}</h2>}
        {children}
      </div>
    </div>,
    document.body,
  )
}
