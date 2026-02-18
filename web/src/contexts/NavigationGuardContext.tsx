import { createContext, useContext, useCallback, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import type { ReactNode, MutableRefObject } from 'react'

interface NavigationGuardContextValue {
  /** Ref to a guard function — return true to block navigation */
  guardRef: MutableRefObject<(() => boolean) | null>
  /** Ref to a callback invoked when the user cancels (keeps editing) */
  cancelCallbackRef: MutableRefObject<(() => void) | null>
  /** Navigate only if no guard blocks; otherwise store pending path */
  guardedNavigate: (path: string) => void
  /** The path that was blocked (null if none) */
  pendingPath: string | null
  /** Confirm the pending navigation — navigates and clears */
  confirmNavigation: () => void
  /** Cancel the pending navigation — clears and runs cancel callback */
  cancelNavigation: () => void
}

const NavigationGuardContext = createContext<NavigationGuardContextValue | null>(null)

export function NavigationGuardProvider({ children }: { children: ReactNode }) {
  const navigate = useNavigate()
  const guardRef = useRef<(() => boolean) | null>(null)
  const cancelCallbackRef = useRef<(() => void) | null>(null)
  const [pendingPath, setPendingPath] = useState<string | null>(null)

  const guardedNavigate = useCallback((path: string) => {
    if (guardRef.current?.()) {
      setPendingPath(path)
    } else {
      navigate(path)
    }
  }, [navigate])

  const confirmNavigation = useCallback(() => {
    const path = pendingPath
    setPendingPath(null)
    if (path) navigate(path)
  }, [pendingPath, navigate])

  const cancelNavigation = useCallback(() => {
    setPendingPath(null)
    cancelCallbackRef.current?.()
  }, [])

  return (
    <NavigationGuardContext.Provider value={{ guardRef, cancelCallbackRef, guardedNavigate, pendingPath, confirmNavigation, cancelNavigation }}>
      {children}
    </NavigationGuardContext.Provider>
  )
}

export function useNavigationGuard() {
  const ctx = useContext(NavigationGuardContext)
  if (!ctx) throw new Error('useNavigationGuard must be used within NavigationGuardProvider')
  return ctx
}
