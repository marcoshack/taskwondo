import { useSyncExternalStore } from 'react'

const LAST_PROJECT_KEY = 'taskwondo_last_project_key'

const listeners = new Set<() => void>()

function emit() {
  listeners.forEach((l) => l())
}

function subscribe(callback: () => void) {
  listeners.add(callback)
  return () => {
    listeners.delete(callback)
  }
}

function getSnapshot(): string | null {
  return localStorage.getItem(LAST_PROJECT_KEY)
}

function getServerSnapshot(): string | null {
  return null
}

/**
 * React hook returning the last-active project key from localStorage.
 * Components using this hook re-render whenever the stored key is changed
 * via `setLastProjectKey` or `clearLastProjectKey`, keeping the sidebar and
 * top bar in sync even across sibling subtrees.
 */
export function useLastProjectKey(): string | null {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot)
}

/** Persist the given project key and notify subscribers. */
export function setLastProjectKey(key: string): void {
  localStorage.setItem(LAST_PROJECT_KEY, key)
  emit()
}

/** Remove the stored project key and notify subscribers. */
export function clearLastProjectKey(): void {
  localStorage.removeItem(LAST_PROJECT_KEY)
  emit()
}
