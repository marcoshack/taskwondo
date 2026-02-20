import { Outlet, useNavigate, useMatch } from 'react-router-dom'

import { useTranslation } from 'react-i18next'
import { Settings, Menu } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useSidebar } from '@/contexts/SidebarContext'
import { useNavigationGuard } from '@/contexts/NavigationGuardContext'
import { useProject, useProjects } from '@/hooks/useProjects'
import { Avatar } from '@/components/ui/Avatar'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { useState, useRef, useEffect, useCallback } from 'react'
import { useKeyboardShortcutContext } from '@/contexts/KeyboardShortcutContext'
import { useKeyboardShortcut } from '@/hooks/useKeyboardShortcut'
import { KeyboardShortcutsModal } from '@/components/KeyboardShortcutsModal'

export function AppShell() {
  const { t } = useTranslation()
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { guardedNavigate } = useNavigationGuard()
  const { toggleMobileOpen } = useSidebar()
  const [menuOpen, setMenuOpen] = useState(false)
  const [switcherOpen, setSwitcherOpen] = useState(false)
  const [shortcutsOpen, setShortcutsOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  const projectMatch = useMatch('/projects/:projectKey/*')
  const adminMatch = useMatch('/admin/*')
  const activeProjectKey = projectMatch?.params.projectKey
  const { data: activeProject } = useProject(activeProjectKey ?? '')

  useEffect(() => {
    if (!menuOpen) return
    const handler = (e: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [menuOpen])

  // Sequential combos: g-p (project switcher), g-i (go to items)
  const { registerSequentialCombo } = useKeyboardShortcutContext()
  useEffect(() => {
    return registerSequentialCombo({
      id: 'go-to-projects',
      keys: ['g', 'p'],
      callback: () => setSwitcherOpen(true),
    })
  }, [registerSequentialCombo])
  useEffect(() => {
    if (!activeProjectKey) return
    return registerSequentialCombo({
      id: 'go-to-items',
      keys: ['g', 'i'],
      callback: () => guardedNavigate(`/projects/${activeProjectKey}/items`),
    })
  }, [activeProjectKey, navigate, registerSequentialCombo])

  useKeyboardShortcut({ key: '?' }, () => setShortcutsOpen(true))

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  return (
    <div className="min-h-screen bg-gray-50 dark:bg-gray-900">
      <nav className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex justify-between h-14 relative">
            <div className="flex items-center gap-6 min-w-0">
              <button onClick={() => guardedNavigate('/projects')} className="text-lg font-bold text-indigo-600 dark:text-indigo-400 shrink-0">
                {t('brand.name')}
              </button>
              {adminMatch ? (
                <div className="flex items-center gap-2.5 min-w-0">
                  <Settings className="h-5 w-5 text-gray-500 dark:text-gray-400 shrink-0" />
                  <span className="text-base font-semibold text-gray-900 dark:text-gray-100 sm:hidden">{t('admin.titleShort')}</span>
                  <span className="text-base font-semibold text-gray-900 dark:text-gray-100 hidden sm:inline">{t('admin.title')}</span>
                </div>
              ) : activeProject ? (
                <button
                  onClick={() => setSwitcherOpen(true)}
                  className="hidden sm:flex items-center gap-2.5 hover:opacity-80 transition-opacity min-w-0"
                >
                  <span className="inline-flex items-center rounded-md bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 px-2.5 py-1 text-sm font-bold shrink-0">
                    {activeProject.key}
                  </span>
                  <span className="text-base font-semibold text-gray-900 dark:text-gray-100 truncate">
                    {activeProject.name}
                  </span>
                </button>
              ) : null}
            </div>
            {activeProject && (
              <button
                onClick={() => setSwitcherOpen(true)}
                className="sm:hidden absolute left-1/2 -translate-x-1/2 top-0 h-full flex items-center hover:opacity-80 transition-opacity"
              >
                <span className="inline-flex items-center rounded-md bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 px-2.5 py-1 text-base font-bold">
                  {activeProject.key}
                </span>
              </button>
            )}
            <div className="relative flex items-center gap-2" ref={menuRef}>
              {(activeProjectKey || adminMatch) && (
                <button
                  onClick={toggleMobileOpen}
                  className="sm:hidden p-2 rounded-md text-gray-500 hover:bg-gray-100 dark:text-gray-400 dark:hover:bg-gray-800"
                  aria-label={t('sidebar.menu')}
                >
                  <Menu className="h-5 w-5" />
                </button>
              )}
              <button
                onClick={() => setMenuOpen(!menuOpen)}
                className="flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 hover:text-gray-900 dark:hover:text-gray-100"
              >
                <Avatar name={user?.display_name ?? ''} size="sm" />
                <span className="hidden sm:block">{user?.display_name}</span>
              </button>
              {menuOpen && (
                <div className="absolute right-0 top-full mt-1 w-48 bg-white/40 dark:bg-gray-800/40 backdrop-blur-sm rounded-md shadow-lg border border-gray-200 dark:border-gray-600 py-1 z-50">
                  <div className="px-4 py-2 text-xs text-gray-500 dark:text-gray-400 border-b border-gray-100 dark:border-gray-700">
                    {user?.email}
                  </div>
                  <button
                    onClick={() => { setMenuOpen(false); guardedNavigate('/preferences') }}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                  >
                    {t('nav.preferences')}
                  </button>
                  {user?.global_role === 'admin' && (
                    <button
                      onClick={() => { setMenuOpen(false); guardedNavigate('/admin') }}
                      className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                    >
                      {t('nav.systemSettings')}
                    </button>
                  )}
                  <div className="border-t border-gray-100 dark:border-gray-700" />
                  <button
                    onClick={handleLogout}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700"
                  >
                    {t('nav.signOut')}
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </nav>
      <main>
        <Outlet />
      </main>
      <ProjectSwitcherModal
        open={switcherOpen}
        onClose={() => setSwitcherOpen(false)}
        activeProjectKey={activeProjectKey}
        onSelect={(key) => {
          setSwitcherOpen(false)
          guardedNavigate(`/projects/${key}`)
        }}
      />
      <KeyboardShortcutsModal open={shortcutsOpen} onClose={() => setShortcutsOpen(false)} />
    </div>
  )
}

function ProjectSwitcherModal({
  open,
  onClose,
  activeProjectKey,
  onSelect,
}: {
  open: boolean
  onClose: () => void
  activeProjectKey?: string
  onSelect: (key: string) => void
}) {
  const { t } = useTranslation()
  const { data: projects, isLoading } = useProjects()
  const [search, setSearch] = useState('')
  const [selectedIndex, setSelectedIndex] = useState(0)
  const inputRef = useRef<HTMLInputElement>(null)
  const listRef = useRef<HTMLUListElement>(null)

  useEffect(() => {
    if (open) {
      setSearch('')
      setSelectedIndex(0)
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open])

  const filtered = (projects ?? []).filter((p) => {
    if (!search) return true
    const q = search.toLowerCase()
    return p.key.toLowerCase().includes(q) || p.name.toLowerCase().includes(q)
  })

  // Reset selection when search changes
  useEffect(() => {
    setSelectedIndex(0)
  }, [search])

  // Scroll selected item into view
  const scrollSelectedIntoView = useCallback((index: number) => {
    const list = listRef.current
    if (!list) return
    const item = list.children[index] as HTMLElement | undefined
    item?.scrollIntoView({ block: 'nearest' })
  }, [])

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      const next = Math.min(selectedIndex + 1, filtered.length - 1)
      setSelectedIndex(next)
      scrollSelectedIntoView(next)
    } else if (e.key === 'ArrowUp') {
      e.preventDefault()
      const prev = Math.max(selectedIndex - 1, 0)
      setSelectedIndex(prev)
      scrollSelectedIntoView(prev)
    } else if (e.key === 'Enter' && filtered.length > 0) {
      e.preventDefault()
      onSelect(filtered[selectedIndex].key)
    }
  }, [selectedIndex, filtered, onSelect, scrollSelectedIntoView])

  return (
    <Modal open={open} onClose={onClose} title={t('projects.switcher.title')} position="top">
      <input
        ref={inputRef}
        type="text"
        placeholder={t('projects.switcher.search')}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        onKeyDown={handleKeyDown}
        className="block w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400 mb-3"
      />
      {isLoading ? (
        <div className="flex justify-center py-6"><Spinner /></div>
      ) : filtered.length === 0 ? (
        <p className="text-sm text-gray-500 dark:text-gray-400 py-4 text-center">{t('projects.noProjectsFound')}</p>
      ) : (
        <ul ref={listRef} className="max-h-64 overflow-y-auto -mx-2">
          {filtered.map((p, i) => (
            <li key={p.key}>
              <button
                onClick={() => onSelect(p.key)}
                onMouseEnter={() => setSelectedIndex(i)}
                className={`w-full text-left flex items-center gap-3 px-3 py-2.5 rounded-md text-sm ${
                  i === selectedIndex
                    ? 'bg-indigo-50 dark:bg-indigo-900/30'
                    : 'hover:bg-gray-100 dark:hover:bg-gray-700'
                }`}
              >
                <span className="inline-flex items-center justify-center rounded-md bg-indigo-100 dark:bg-indigo-900/40 text-indigo-700 dark:text-indigo-300 px-2 py-0.5 text-xs font-bold shrink-0 min-w-[4rem]">
                  {p.key}
                </span>
                <span className="text-gray-900 dark:text-gray-100 font-medium truncate">{p.name}</span>
                {p.key === activeProjectKey && (
                  <span className="ml-auto text-xs text-indigo-600 dark:text-indigo-400 shrink-0">{t('common.current')}</span>
                )}
              </button>
            </li>
          ))}
        </ul>
      )}
    </Modal>
  )
}
