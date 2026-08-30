import { useTranslation } from 'react-i18next'
import { FlaskConical } from 'lucide-react'
import { usePublicSettings, useSetSystemSetting } from '@/hooks/useSystemSettings'
import { Toggle } from '@/components/ui/Toggle'
import { LoadingState } from '@/components/ui/LoadingState'

export function SystemFeaturesPage() {
  const { t } = useTranslation()
  const { data: publicSettings, isLoading } = usePublicSettings()
  const setSetting = useSetSystemSetting()

  if (isLoading) {
    return <LoadingState />
  }

  const settings = publicSettings ?? {}

  const statsTimelineEnabled = settings.feature_stats_timeline !== undefined
    ? settings.feature_stats_timeline === true
    : true // default: enabled

  const semanticSearchEnabled = settings.feature_semantic_search === true
  const ollamaAvailable = settings.ollama_available as boolean | undefined

  const namespacesEnabled = settings.namespaces_enabled === true
  const hasCustomNamespaces = (settings.custom_namespace_count as number | undefined) ?? 0

  const handleToggle = (key: string, value: boolean) => {
    setSetting.mutate({ key, value })
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-[var(--foreground)]">
          {t('admin.features.title')}
        </h2>
        <p className="mt-1 text-sm text-[var(--foreground-secondary)]">
          {t('admin.features.description')}
        </p>
      </div>

      {/* Activity Graph */}
      <div className="rounded-lg border border-[var(--border)] p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-[var(--foreground)]">
              {t('admin.features.statsTimeline.title')}
            </h3>
            <p className="mt-1 text-sm text-[var(--foreground-secondary)]">
              {t('admin.features.statsTimeline.description')}
            </p>
          </div>
          <Toggle
            enabled={statsTimelineEnabled}
            onChange={(val) => handleToggle('feature_stats_timeline', val)}
          />
        </div>
      </div>

      {/* Semantic Search */}
      <div className="rounded-lg border border-[var(--border)] p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-[var(--foreground)] flex items-center gap-2">
              {t('admin.features.semanticSearch.title')}
              <span className="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-medium bg-[var(--surface-tertiary)] text-[var(--foreground-secondary)] bg-[var(--surface-secondary)]/50 text-[var(--foreground-muted)]">
                <FlaskConical className="h-3 w-3" />
                {t('common.experimental')}
              </span>
            </h3>
            <p className="mt-1 text-sm text-[var(--foreground-secondary)]">
              {t('admin.features.semanticSearch.description')}
            </p>
            {ollamaAvailable !== undefined && (
              <p className={`mt-2 text-xs ${ollamaAvailable ? 'text-green-600 dark:text-green-400' : 'text-amber-600 dark:text-amber-400'}`}>
                {ollamaAvailable
                  ? t('admin.features.semanticSearch.ollamaAvailable')
                  : t('admin.features.semanticSearch.ollamaUnavailable')}
              </p>
            )}
          </div>
          <Toggle
            enabled={semanticSearchEnabled}
            onChange={(val) => handleToggle('feature_semantic_search', val)}
            disabled={!ollamaAvailable}
          />
        </div>
      </div>
      {/* Namespaces */}
      <div className="rounded-lg border border-[var(--border)] p-6">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-lg font-medium text-[var(--foreground)]">
              {t('admin.features.namespaces.title')}
            </h3>
            <p className="mt-1 text-sm text-[var(--foreground-secondary)]">
              {t('admin.features.namespaces.description')}
            </p>
            {hasCustomNamespaces > 0 && namespacesEnabled && (
              <p className="mt-2 text-xs text-amber-600 dark:text-amber-400">
                {t('admin.features.namespaces.cannotDisable', { count: hasCustomNamespaces })}
              </p>
            )}
          </div>
          <Toggle
            enabled={namespacesEnabled}
            onChange={(val) => handleToggle('namespaces_enabled', val)}
            disabled={!namespacesEnabled ? false : hasCustomNamespaces > 0}
          />
        </div>
      </div>
    </div>
  )
}
