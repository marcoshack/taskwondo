import { createContext, useContext, useState, useCallback } from 'react'
import type { ReactNode } from 'react'
import { Check, X } from 'lucide-react'

interface Notification {
  id: number
  message: string
  type: 'success' | 'error'
}

interface NotificationContextValue {
  showNotification: (message: string, type?: 'success' | 'error') => void
}

const NotificationContext = createContext<NotificationContextValue | null>(null)

let nextId = 0

export function NotificationProvider({ children }: { children: ReactNode }) {
  const [notifications, setNotifications] = useState<Notification[]>([])

  const showNotification = useCallback((message: string, type: 'success' | 'error' = 'success') => {
    const id = nextId++
    setNotifications((prev) => [...prev, { id, message, type }])
    setTimeout(() => {
      setNotifications((prev) => prev.filter((n) => n.id !== id))
    }, 4000)
  }, [])

  const dismiss = useCallback((id: number) => {
    setNotifications((prev) => prev.filter((n) => n.id !== id))
  }, [])

  return (
    <NotificationContext.Provider value={{ showNotification }}>
      {children}
      {notifications.length > 0 && (
        <div className="fixed bottom-4 right-4 z-50 flex flex-col gap-2">
          {notifications.map((n) => (
            <div
              key={n.id}
              className={`flex items-center gap-2 px-4 py-3 rounded-lg shadow-lg text-sm font-medium animate-[slideIn_0.2s_ease-out] ${
                n.type === 'success'
                  ? 'bg-green-50 text-green-800 dark:bg-green-900/80 dark:text-green-200 border border-green-200 dark:border-green-700'
                  : 'bg-red-50 text-red-800 dark:bg-red-900/80 dark:text-red-200 border border-red-200 dark:border-red-700'
              }`}
            >
              {n.type === 'success' ? (
                <Check className="h-4 w-4 flex-shrink-0" />
              ) : (
                <X className="h-4 w-4 flex-shrink-0" />
              )}
              <span>{n.message}</span>
              <button
                onClick={() => dismiss(n.id)}
                className="ml-2 opacity-60 hover:opacity-100"
                aria-label="Dismiss"
              >
                <X className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
        </div>
      )}
    </NotificationContext.Provider>
  )
}

export function useNotification() {
  const ctx = useContext(NotificationContext)
  if (!ctx) throw new Error('useNotification must be used within NotificationProvider')
  return ctx
}
