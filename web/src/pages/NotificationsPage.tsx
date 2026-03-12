import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { ChevronDown, ChevronRight, Check } from 'lucide-react'
import { useProjects } from '@/hooks/useProjects'
import { useUserSetting, useSetUserSetting, useGlobalPreference, useSetGlobalPreference } from '@/hooks/useUserSettings'
import { Spinner } from '@/components/ui/Spinner'

interface NotificationPreferences {
  assigned_to_me: boolean
  any_update_on_watched: boolean
  new_item_created: boolean
  comments_on_assigned: boolean
  comments_on_watched: boolean
  status_changes_intermediate: boolean
  status_changes_final: boolean
}

const defaultPreferences: NotificationPreferences = {
  assigned_to_me: true,
  any_update_on_watched: false,
  new_item_created: false,
  comments_on_assigned: false,
  comments_on_watched: false,
  status_changes_intermediate: false,
  status_changes_final: false,
}

interface GlobalNotificationPreferences {
  added_to_project: boolean
}

const defaultGlobalPreferences: GlobalNotificationPreferences = {
  added_to_project: false,
}

interface NotificationOption {
  key: keyof NotificationPreferences
  labelKey: string
  descKey: string
  enabled: boolean
}

const notificationOptions: NotificationOption[] = [
  { key: 'assigned_to_me', labelKey: 'preferences.notifications.assignedToMe', descKey: 'preferences.notifications.assignedToMeDesc', enabled: true },
  { key: 'any_update_on_watched', labelKey: 'preferences.notifications.anyUpdateOnWatched', descKey: 'preferences.notifications.anyUpdateOnWatchedDesc', enabled: true },
  { key: 'new_item_created', labelKey: 'preferences.notifications.newItemCreated', descKey: 'preferences.notifications.newItemCreatedDesc', enabled: true },
  { key: 'comments_on_assigned', labelKey: 'preferences.notifications.commentsOnAssigned', descKey: 'preferences.notifications.commentsOnAssignedDesc', enabled: true },
  { key: 'comments_on_watched', labelKey: 'preferences.notifications.commentsOnWatched', descKey: 'preferences.notifications.commentsOnWatchedDesc', enabled: true },
  { key: 'status_changes_intermediate', labelKey: 'preferences.notifications.statusChangesIntermediate', descKey: 'preferences.notifications.statusChangesIntermediateDesc', enabled: true },
  { key: 'status_changes_final', labelKey: 'preferences.notifications.statusChangesFinal', descKey: 'preferences.notifications.statusChangesFinalDesc', enabled: true },
]

interface GlobalNotificationOption {
  key: keyof GlobalNotificationPreferences
  labelKey: string
  descKey: string
  enabled: boolean
}

const globalNotificationOptions: GlobalNotificationOption[] = [
  { key: 'added_to_project', labelKey: 'preferences.notifications.addedToProject', descKey: 'preferences.notifications.addedToProjectDesc', enabled: true },
]

export function NotificationsPage() {
  const { t } = useTranslation()
  const { data: projects, isLoading } = useProjects()
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})

  const toggleExpanded = useCallback((key: string) => {
    setExpanded((prev) => ({ ...prev, [key]: !prev[key] }))
  }, [])

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  return (
    <div>
      <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
        {t('preferences.notifications.title')}
      </h2>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {t('preferences.notifications.description')}
      </p>

      {/* Global notifications section */}
      <div className="mt-6">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {t('preferences.notifications.globalSection')}
        </h3>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {t('preferences.notifications.globalDescription')}
        </p>
        <div className="mt-3">
          <GlobalNotificationCard />
        </div>
      </div>

      {/* Per-project notifications section */}
      <div className="mt-8">
        <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
          {t('preferences.notifications.perProjectSection')}
        </h3>
        <p className="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {t('preferences.notifications.perProjectDescription')}
        </p>
        <div className="mt-3 space-y-3">
          {!projects || projects.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">
              {t('preferences.notifications.noProjects')}
            </p>
          ) : (
            projects.map((project) => (
              <ProjectNotificationCard
                key={project.key}
                projectKey={project.key}
                projectName={project.name}
                isExpanded={expanded[project.key] ?? false}
                onToggle={() => toggleExpanded(project.key)}
              />
            ))
          )}
        </div>
      </div>
    </div>
  )
}

function GlobalNotificationCard() {
  const { t } = useTranslation()
  const { data: prefs, isLoading } = useGlobalPreference<GlobalNotificationPreferences>('global_notifications')
  const { mutate: setGlobal } = useSetGlobalPreference()
  const [savedId, setSavedId] = useState<string | null>(null)

  const currentPrefs: GlobalNotificationPreferences = prefs ?? defaultGlobalPreferences

  function handleToggle(optionKey: keyof GlobalNotificationPreferences) {
    const updated = { ...currentPrefs, [optionKey]: !currentPrefs[optionKey] }
    setGlobal(
      { key: 'global_notifications', value: updated },
      {
        onSuccess: () => {
          setSavedId(optionKey)
          setTimeout(() => setSavedId(null), 2000)
        },
      },
    )
  }

  if (isLoading) {
    return (
      <div className="flex justify-center py-4">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-3">
      <div className="space-y-3">
        {globalNotificationOptions.map((option) => (
          <label
            key={option.key}
            className={`flex items-start gap-3 ${
              option.enabled ? 'cursor-pointer' : 'cursor-not-allowed opacity-60'
            }`}
          >
            <input
              type="checkbox"
              checked={currentPrefs[option.key]}
              onChange={() => handleToggle(option.key)}
              disabled={!option.enabled}
              className="mt-0.5 h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-800"
            />
            <div className="flex-1 min-w-0">
              <div className="flex items-center gap-2">
                <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                  {t(option.labelKey)}
                </span>
                {savedId === option.key && (
                  <Check className="h-4 w-4 text-green-500" />
                )}
              </div>
              <p className="text-xs text-gray-500 dark:text-gray-400">
                {t(option.descKey)}
              </p>
            </div>
          </label>
        ))}
      </div>
    </div>
  )
}

function ProjectNotificationCard({
  projectKey,
  projectName,
  isExpanded,
  onToggle,
}: {
  projectKey: string
  projectName: string
  isExpanded: boolean
  onToggle: () => void
}) {
  const { t } = useTranslation()
  const { data: prefs, isLoading } = useUserSetting<NotificationPreferences>(projectKey, 'notifications')
  const { mutate: setSetting } = useSetUserSetting(projectKey)
  const [savedId, setSavedId] = useState<string | null>(null)

  const currentPrefs: NotificationPreferences = prefs ?? defaultPreferences

  function handleToggle(optionKey: keyof NotificationPreferences) {
    const updated = { ...currentPrefs, [optionKey]: !currentPrefs[optionKey] }
    setSetting(
      { key: 'notifications', value: updated },
      {
        onSuccess: () => {
          setSavedId(optionKey)
          setTimeout(() => setSavedId(null), 2000)
        },
      },
    )
  }

  return (
    <div className="rounded-lg border border-gray-200 dark:border-gray-700">
      <button
        onClick={onToggle}
        className="flex w-full items-center justify-between px-4 py-3 text-left hover:bg-gray-50 dark:hover:bg-gray-800/50 rounded-lg transition-colors"
      >
        <div className="flex items-center gap-2">
          {isExpanded ? (
            <ChevronDown className="h-4 w-4 text-gray-400" />
          ) : (
            <ChevronRight className="h-4 w-4 text-gray-400" />
          )}
          <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
            {projectName}
          </span>
          <span className="text-xs text-gray-400 dark:text-gray-500">{projectKey}</span>
        </div>
      </button>

      {isExpanded && (
        <div className="border-t border-gray-200 dark:border-gray-700 px-4 py-3">
          {isLoading ? (
            <div className="flex justify-center py-4">
              <Spinner />
            </div>
          ) : (
            <div className="space-y-3">
              {notificationOptions.map((option) => (
                <label
                  key={option.key}
                  className={`flex items-start gap-3 ${
                    option.enabled ? 'cursor-pointer' : 'cursor-not-allowed opacity-60'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={currentPrefs[option.key]}
                    onChange={() => handleToggle(option.key)}
                    disabled={!option.enabled}
                    className="mt-0.5 h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 disabled:opacity-50 dark:border-gray-600 dark:bg-gray-800"
                  />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                        {t(option.labelKey)}
                      </span>
                      {!option.enabled && (
                        <span className="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-500 dark:bg-gray-800 dark:text-gray-400">
                          {t('preferences.notifications.comingSoon')}
                        </span>
                      )}
                      {savedId === option.key && (
                        <Check className="h-4 w-4 text-green-500" />
                      )}
                    </div>
                    <p className="text-xs text-gray-500 dark:text-gray-400">
                      {t(option.descKey)}
                    </p>
                  </div>
                </label>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
