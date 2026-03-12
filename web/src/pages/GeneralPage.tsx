import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useQueryClient } from '@tanstack/react-query'
import { Check } from 'lucide-react'
import { useAuth } from '@/contexts/AuthContext'
import { useGlobalPreference, useSetGlobalPreference } from '@/hooks/useUserSettings'
import { Spinner } from '@/components/ui/Spinner'

export function GeneralPage() {
  const { t } = useTranslation()
  const { user } = useAuth()
  const isAdmin = user?.global_role === 'admin'

  const { data: hideNonMember, isLoading } = useGlobalPreference<boolean>('hide_non_member_projects')
  const { mutate: setGlobal } = useSetGlobalPreference()
  const queryClient = useQueryClient()
  const [savedId, setSavedId] = useState<string | null>(null)

  const currentValue = hideNonMember ?? false

  function handleToggle() {
    setGlobal(
      { key: 'hide_non_member_projects', value: !currentValue },
      {
        onSuccess: () => {
          queryClient.invalidateQueries({ queryKey: ['projects'] })
          setSavedId('hide')
          setTimeout(() => setSavedId(null), 2000)
        },
      },
    )
  }

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
        {t('preferences.general.title')}
      </h2>
      <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {t('preferences.general.description')}
      </p>

      {isAdmin && (
        <div className="mt-6">
          <h3 className="text-sm font-semibold text-gray-900 dark:text-gray-100">
            {t('preferences.general.adminSection')}
          </h3>
          <div className="mt-3 rounded-lg border border-gray-200 dark:border-gray-700 px-4 py-3">
            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={currentValue}
                onChange={handleToggle}
                className="mt-0.5 h-4 w-4 rounded border-gray-300 text-indigo-600 focus:ring-indigo-500 dark:border-gray-600 dark:bg-gray-800"
              />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-gray-900 dark:text-gray-100">
                    {t('preferences.general.hideNonMember')}
                  </span>
                  {savedId === 'hide' && (
                    <Check className="h-4 w-4 text-green-500" />
                  )}
                </div>
                <p className="text-xs text-gray-500 dark:text-gray-400">
                  {t('preferences.general.hideNonMemberDesc')}
                </p>
              </div>
            </label>
          </div>
        </div>
      )}

      {!isAdmin && (
        <p className="mt-6 text-sm text-gray-500 dark:text-gray-400">
          {t('preferences.general.noSettings')}
        </p>
      )}
    </div>
  )
}
