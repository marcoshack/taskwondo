import { useTranslation } from 'react-i18next'
import { usePublicSettings, useSetSystemSetting } from '@/hooks/useSystemSettings'
import { Toggle } from '@/components/ui/Toggle'
import { Spinner } from '@/components/ui/Spinner'

export function SystemFeaturesPage() {
  const { t } = useTranslation()
  const { data: publicSettings, isLoading } = usePublicSettings()
  const setSetting = useSetSystemSetting()

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  const settings = publicSettings ?? {}

  const statsTimelineEnabled = settings.feature_stats_timeline !== undefined
    ? settings.feature_stats_timeline === true
    : true // default: enabled

  const handleToggle = (key: string, value: boolean) => {
    setSetting.mutate({ key, value })
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
          {t('admin.features.title')}
        </h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t('admin.features.description')}
        </p>
      </div>

      {/* Activity Graph */}
      <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100">
              {t('admin.features.statsTimeline.title')}
            </h3>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {t('admin.features.statsTimeline.description')}
            </p>
          </div>
          <Toggle
            enabled={statsTimelineEnabled}
            onChange={(val) => handleToggle('feature_stats_timeline', val)}
          />
        </div>
      </div>
    </div>
  )
}
