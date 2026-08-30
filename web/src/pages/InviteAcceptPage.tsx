import { useEffect, useRef, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation, Trans } from 'react-i18next'
import { isAxiosError } from 'axios'
import { useAuth } from '@/contexts/AuthContext'
import { useNotification } from '@/contexts/NotificationContext'
import { useInviteInfo, useAcceptInvite } from '@/hooks/useProjects'
import { Button } from '@/components/ui/Button'
import { Spinner } from '@/components/ui/Spinner'
import { toUrlSegment } from '@/hooks/useNamespacePath'
import type { AcceptInviteResult, InviteInfo } from '@/api/projects'

const PENDING_INVITE_KEY = 'taskwondo_pending_invite'

function getDisplayName(info: InviteInfo): string {
  return info.type === 'namespace' ? (info.namespace_display_name ?? '') : (info.project_name ?? '')
}

function getPostAcceptPath(result: AcceptInviteResult): string {
  if (result.type === 'namespace') {
    return `/${toUrlSegment(result.namespace_slug ?? 'd')}/projects`
  }
  if (result.project) {
    const nsSegment = toUrlSegment(result.project_namespace_slug ?? 'default')
    return `/${nsSegment}/projects/${result.project.key}`
  }
  return '/d/projects'
}

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
        const roleKey = result.type === 'namespace' ? 'namespaces.roles.' : 'projects.settings.roles.'
        if (result.role_not_applied) {
          const existingRole = t(`${roleKey}${result.existing_role}`)
          const inviteRole = t(`${roleKey}${result.invite_role}`)
          showNotification(t('invite.roleNotApplied', { existingRole, inviteRole }))
        } else {
          showNotification(t('invite.accepted', { projectName: getDisplayName(inviteInfo) }))
        }
        navigate(getPostAcceptPath(result), { replace: true })
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
      <div className="min-h-screen flex items-center justify-center bg-[var(--surface-secondary)] dark:bg-[var(--background)]">
        <Spinner size="lg" />
      </div>
    )
  }

  if (fetchError || !inviteInfo) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--surface-secondary)] dark:bg-[var(--background)] px-4">
        <div className="max-w-sm w-full text-center">
          <p className="text-[var(--foreground-secondary)] mb-4">{t('invite.notFound')}</p>
          <a href="/login" className="text-[var(--primary)] hover:underline text-sm">
            {t('login.oauth.backToLogin')}
          </a>
        </div>
      </div>
    )
  }

  // Show spinner during auto-accept
  if (autoAccepting) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--surface-secondary)] dark:bg-[var(--background)]">
        <div className="text-center">
          <Spinner size="lg" />
          <p className="mt-4 text-sm text-[var(--foreground-secondary)]">
            {t('invite.joining')}
          </p>
        </div>
      </div>
    )
  }

  if (inviteInfo.expired) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--surface-secondary)] dark:bg-[var(--background)] px-4">
        <div className="max-w-sm w-full text-center">
          <p className="text-[var(--foreground-secondary)] mb-4">{t('invite.expired')}</p>
          <a href="/login" className="text-[var(--primary)] hover:underline text-sm">
            {t('login.oauth.backToLogin')}
          </a>
        </div>
      </div>
    )
  }

  if (inviteInfo.full) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-[var(--surface-secondary)] dark:bg-[var(--background)] px-4">
        <div className="max-w-sm w-full text-center">
          <p className="text-[var(--foreground-secondary)] mb-4">{t('invite.full')}</p>
          <a href="/login" className="text-[var(--primary)] hover:underline text-sm">
            {t('login.oauth.backToLogin')}
          </a>
        </div>
      </div>
    )
  }

  const resourceDisplayName = getDisplayName(inviteInfo)
  const roleNamespace = inviteInfo.type === 'namespace' ? 'namespaces.roles.' : 'projects.settings.roles.'
  const roleLabel = t(`${roleNamespace}${inviteInfo.role}`)
  const joinKey = inviteInfo.type === 'namespace' ? 'invite.joinNamespaceAs' : 'invite.joinAs'

  const handleLoginToJoin = () => {
    localStorage.setItem(PENDING_INVITE_KEY, code ?? '')
    navigate('/login')
  }

  const handleAccept = async () => {
    try {
      const result = await acceptMutation.mutateAsync(code ?? '')
      localStorage.removeItem(PENDING_INVITE_KEY)
      if (result.role_not_applied) {
        const existingRole = t(`${roleNamespace}${result.existing_role}`)
        const inviteRole = t(`${roleNamespace}${result.invite_role}`)
        showNotification(t('invite.roleNotApplied', { existingRole, inviteRole }))
      } else {
        showNotification(t('invite.accepted', { projectName: resourceDisplayName }))
      }
      navigate(getPostAcceptPath(result), { replace: true })
    } catch (err) {
      if (isAxiosError(err) && err.response?.status === 409) {
        showNotification(t('invite.alreadyMember'), 'error')
      } else {
        showNotification(t('invite.error'), 'error')
      }
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-[var(--surface-secondary)] dark:bg-[var(--background)] px-4">
      <div className="max-w-sm w-full text-center space-y-6">
        <h1 className="text-2xl font-bold text-[var(--foreground)]">
          {t('invite.title')}
        </h1>

        <p className="text-[var(--foreground-secondary)]">
          <Trans
            i18nKey={joinKey}
            values={{ projectName: resourceDisplayName, namespaceName: resourceDisplayName, role: roleLabel }}
            components={{ bold: <strong className="text-[var(--foreground)]" /> }}
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
              : t('invite.join', { projectName: resourceDisplayName })}
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
