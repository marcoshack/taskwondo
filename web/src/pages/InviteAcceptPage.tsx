import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation, Trans } from 'react-i18next'
import { isAxiosError } from 'axios'
import { useAuth } from '@/contexts/AuthContext'
import { useNotification } from '@/contexts/NotificationContext'
import { useInviteInfo, useAcceptInvite } from '@/hooks/useProjects'
import { Button } from '@/components/ui/Button'
import { Spinner } from '@/components/ui/Spinner'

const PENDING_INVITE_KEY = 'taskwondo_pending_invite'

export function InviteAcceptPage() {
  const { t } = useTranslation()
  const { code } = useParams<{ code: string }>()
  const navigate = useNavigate()
  const { user } = useAuth()
  const { showNotification } = useNotification()
  const { data: inviteInfo, isLoading, error: fetchError } = useInviteInfo(code ?? '')
  const acceptMutation = useAcceptInvite()
  const [autoAccepting, setAutoAccepting] = useState(false)
  const autoAcceptStarted = useRef(false)

  // Auto-accept when arriving from login redirect (pending invite matches current code)
  useEffect(() => {
    if (autoAcceptStarted.current) return
    if (!user || !inviteInfo || !code) return
    if (inviteInfo.expired || inviteInfo.full) return

    const pending = localStorage.getItem(PENDING_INVITE_KEY)
    if (pending !== code) return

    autoAcceptStarted.current = true
    localStorage.removeItem(PENDING_INVITE_KEY)
    setAutoAccepting(true)

    acceptMutation.mutateAsync(code)
      .then((result) => {
        if (result.role_not_applied) {
          const existingRole = t(`projects.settings.roles.${result.existing_role}`)
          const inviteRole = t(`projects.settings.roles.${result.invite_role}`)
          showNotification(t('invite.roleNotApplied', { existingRole, inviteRole }))
        } else {
          showNotification(t('invite.accepted', { projectName: inviteInfo.project_name }))
        }
        navigate(`/d/projects/${result.key}`, { replace: true })
      })
      .catch((err) => {
        setAutoAccepting(false)
        if (isAxiosError(err) && err.response?.status === 409) {
          showNotification(t('invite.alreadyMember'), 'error')
        } else {
          showNotification(t('invite.error'), 'error')
        }
      })
  }, [user, inviteInfo, code, acceptMutation, navigate, t, showNotification])

  if (isLoading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <Spinner size="lg" />
      </div>
    )
  }

  if (fetchError || !inviteInfo) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="max-w-sm w-full text-center">
          <p className="text-gray-600 dark:text-gray-400 mb-4">{t('invite.notFound')}</p>
          <a href="/login" className="text-indigo-600 dark:text-indigo-400 hover:underline text-sm">
            {t('login.oauth.backToLogin')}
          </a>
        </div>
      </div>
    )
  }

  // Show spinner during auto-accept
  if (autoAccepting) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900">
        <div className="text-center">
          <Spinner size="lg" />
          <p className="mt-4 text-sm text-gray-500 dark:text-gray-400">
            {t('invite.joining')}
          </p>
        </div>
      </div>
    )
  }

  if (inviteInfo.expired) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="max-w-sm w-full text-center">
          <p className="text-gray-600 dark:text-gray-400 mb-4">{t('invite.expired')}</p>
          <a href="/login" className="text-indigo-600 dark:text-indigo-400 hover:underline text-sm">
            {t('login.oauth.backToLogin')}
          </a>
        </div>
      </div>
    )
  }

  if (inviteInfo.full) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
        <div className="max-w-sm w-full text-center">
          <p className="text-gray-600 dark:text-gray-400 mb-4">{t('invite.full')}</p>
          <a href="/login" className="text-indigo-600 dark:text-indigo-400 hover:underline text-sm">
            {t('login.oauth.backToLogin')}
          </a>
        </div>
      </div>
    )
  }

  const roleLabel = t(`projects.settings.roles.${inviteInfo.role}`)

  const handleLoginToJoin = () => {
    localStorage.setItem(PENDING_INVITE_KEY, code ?? '')
    navigate('/login')
  }

  const handleAccept = async () => {
    try {
      const result = await acceptMutation.mutateAsync(code ?? '')
      localStorage.removeItem(PENDING_INVITE_KEY)
      if (result.role_not_applied) {
        const existingRole = t(`projects.settings.roles.${result.existing_role}`)
        const inviteRole = t(`projects.settings.roles.${result.invite_role}`)
        showNotification(t('invite.roleNotApplied', { existingRole, inviteRole }))
      } else {
        showNotification(t('invite.accepted', { projectName: inviteInfo.project_name }))
      }
      navigate(`/d/projects/${result.key}`, { replace: true })
    } catch (err) {
      if (isAxiosError(err) && err.response?.status === 409) {
        showNotification(t('invite.alreadyMember'), 'error')
      } else {
        showNotification(t('invite.error'), 'error')
      }
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 px-4">
      <div className="max-w-sm w-full text-center space-y-6">
        <h1 className="text-2xl font-bold text-gray-900 dark:text-gray-100">
          {t('invite.title')}
        </h1>

        <p className="text-gray-600 dark:text-gray-400">
          <Trans
            i18nKey="invite.joinAs"
            values={{ projectName: inviteInfo.project_name, role: roleLabel }}
            components={{ bold: <strong className="text-gray-900 dark:text-gray-100" /> }}
          />
        </p>

        {user ? (
          <Button
            onClick={handleAccept}
            disabled={acceptMutation.isPending}
            className="w-full"
          >
            {acceptMutation.isPending
              ? t('invite.joining')
              : t('invite.join', { projectName: inviteInfo.project_name })}
          </Button>
        ) : (
          <Button onClick={handleLoginToJoin} className="w-full">
            {t('invite.loginToJoin')}
          </Button>
        )}
      </div>
    </div>
  )
}
