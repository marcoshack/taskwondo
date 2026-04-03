import { useState, useRef, useEffect } from 'react'
import { Outlet, useNavigate, useParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '@/contexts/AuthContext'
import { Avatar } from '@/components/ui/Avatar'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { LogOut, ChevronDown, UserCog } from 'lucide-react'
import { PoweredByFooter } from '@/components/PoweredByFooter'

export function PortalShell() {
  const { t } = useTranslation()
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const { namespace, projectKey } = useParams<{ namespace: string; projectKey: string }>()

  const portalProjects = user?.portal_projects ?? []
  const showProjectSwitcher = portalProjects.length > 1
  const activeProject = portalProjects.find((p) => p.project_key === projectKey) ?? portalProjects[0]

  const [dropdownOpen, setDropdownOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const [userMenuOpen, setUserMenuOpen] = useState(false)
  const userMenuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!dropdownOpen && !userMenuOpen) return
    const handler = (e: MouseEvent) => {
      if (dropdownOpen && dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setDropdownOpen(false)
      }
      if (userMenuOpen && userMenuRef.current && !userMenuRef.current.contains(e.target as Node)) {
        setUserMenuOpen(false)
      }
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [dropdownOpen, userMenuOpen])

  const handleLogout = () => {
    logout()
    navigate('/login')
  }

  const handleProjectChange = (newKey: string) => {
    setDropdownOpen(false)
    const ns = namespace || 'd'
    navigate(`/portal/${ns}/projects/${newKey}/tickets`)
  }

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      <nav className="bg-white dark:bg-gray-900 border-b border-gray-200 dark:border-gray-700">
        <div className="max-w-4xl mx-auto px-4 sm:px-6">
          <div className="flex justify-between h-14">
            <div className="flex items-center gap-4">
              {showProjectSwitcher && activeProject ? (
                <div className="relative" ref={dropdownRef}>
                  <button
                    onClick={() => setDropdownOpen(!dropdownOpen)}
                    className="flex items-center gap-2 rounded-md py-1 text-base transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
                  >
                    <ProjectKeyBadge>{activeProject.project_key}</ProjectKeyBadge>
                    <span className="hidden sm:inline font-semibold text-gray-700 dark:text-gray-200">{activeProject.project_name}</span>
                    <ChevronDown className="h-4 w-4 text-gray-400" />
                  </button>
                  {dropdownOpen && (
                    <div className="absolute left-0 top-full mt-1 z-50 w-64 rounded-md border border-gray-200 dark:border-gray-600 bg-white dark:bg-gray-800 shadow-lg py-1">
                      {portalProjects.map((p) => (
                        <button
                          key={p.project_key}
                          onClick={() => handleProjectChange(p.project_key)}
                          className={`w-full text-left px-3 py-2 text-sm flex items-center gap-2.5 ${
                            p.project_key === projectKey
                              ? 'bg-indigo-50 dark:bg-indigo-900/30 text-indigo-700 dark:text-indigo-300'
                              : 'text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700'
                          }`}
                        >
                          <ProjectKeyBadge size="icon">{p.project_key}</ProjectKeyBadge>
                          <span className="truncate">{p.project_name}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              ) : activeProject ? (
                <div className="flex items-center gap-2">
                  <ProjectKeyBadge>{activeProject.project_key}</ProjectKeyBadge>
                  <span className="hidden sm:inline text-base font-semibold text-gray-700 dark:text-gray-200">{activeProject.project_name}</span>
                </div>
              ) : null}
              <span className="text-base font-medium text-gray-400 dark:text-gray-500">
                {t('portal.title')}
              </span>
            </div>
            <div className="flex items-center relative" ref={userMenuRef}>
              <button
                onClick={() => setUserMenuOpen(!userMenuOpen)}
                className="flex items-center gap-2 rounded-md px-2 py-1 text-sm transition-colors hover:bg-gray-100 dark:hover:bg-gray-800"
              >
                <Avatar name={user?.display_name ?? ''} avatarUrl={user?.avatar_url} size="sm" />
                <span className="hidden sm:block text-gray-700 dark:text-gray-300">{user?.display_name}</span>
                <ChevronDown className="h-3.5 w-3.5 text-gray-400" />
              </button>
              {userMenuOpen && (
                <div className="absolute right-0 top-full mt-1 z-50 w-48 bg-white/40 dark:bg-gray-800/40 backdrop-blur-sm rounded-md shadow-lg border border-gray-200 dark:border-gray-600 py-1">
                  <div className="px-4 py-2 text-xs text-gray-500 dark:text-gray-400 border-b border-gray-100 dark:border-gray-700">
                    {user?.email}
                  </div>
                  <button
                    onClick={() => {
                      setUserMenuOpen(false)
                      const ns = namespace || 'd'
                      navigate(`/portal/${ns}/projects/${projectKey}/preferences`)
                    }}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                  >
                    <UserCog className="h-4 w-4 text-gray-400 dark:text-gray-500" />
                    {t('nav.preferences')}
                  </button>
                  <div className="border-t border-gray-100 dark:border-gray-700" />
                  <button
                    onClick={() => {
                      setUserMenuOpen(false)
                      handleLogout()
                    }}
                    className="w-full text-left px-4 py-2 text-sm text-gray-700 dark:text-gray-300 hover:bg-gray-100 dark:hover:bg-gray-700 flex items-center gap-2"
                  >
                    <LogOut className="h-4 w-4 text-gray-400 dark:text-gray-500" />
                    {t('nav.signOut')}
                  </button>
                </div>
              )}
            </div>
          </div>
        </div>
      </nav>
      <main className="flex-1">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 py-6">
          <Outlet context={{ namespace, projectKey }} />
        </div>
      </main>
      <PoweredByFooter />
    </div>
  )
}
