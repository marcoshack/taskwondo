import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Check } from 'lucide-react'
import { useTheme, type Theme, type ColorTheme, type FontSize } from '@/contexts/ThemeContext'
import { useLayout, type Layout } from '@/contexts/LayoutContext'
import { useLanguage } from '@/contexts/LanguageContext'
import { usePreference, useSetPreference } from '@/hooks/usePreferences'
export function AppearancePage({ hideCompletedItems = false }: { hideCompletedItems?: boolean } = {}) {
  const { t } = useTranslation()
  const { theme, setTheme, colorTheme, setColorTheme, fontSize, setFontSize } = useTheme()
  const { layout, setLayout } = useLayout()
  const { language, setLanguage, availableLanguages } = useLanguage()
  const { data: strikethroughPref } = usePreference<boolean>('strikethrough_completed')
  const setPreferenceMutation = useSetPreference()
  const [savedId, setSavedId] = useState<string | null>(null)
  const strikethroughEnabled = strikethroughPref ?? true

  const themes: { value: Theme; label: string; description: string }[] = [
    { value: 'light', label: t('preferences.themes.light'), description: t('preferences.themes.lightDesc') },
    { value: 'dark', label: t('preferences.themes.dark'), description: t('preferences.themes.darkDesc') },
    { value: 'system', label: t('preferences.themes.system'), description: t('preferences.themes.systemDesc') },
  ]

  const colorThemes: { value: ColorTheme; label: string; color: string }[] = [
    { value: 'default', label: t('preferences.colorThemes.default'), color: '#4f46e5' },
    { value: 'slate', label: t('preferences.colorThemes.slate'), color: '#475569' },
    { value: 'blue', label: t('preferences.colorThemes.blue'), color: '#2563eb' },
    { value: 'green', label: t('preferences.colorThemes.green'), color: '#059669' },
    { value: 'purple', label: t('preferences.colorThemes.purple'), color: '#7c3aed' },
  ]

  const fontSizes: { value: FontSize; label: string; description: string; previewSize: string }[] = [
    { value: 'small', label: t('preferences.fontSizes.smaller'), description: t('preferences.fontSizes.smallerDesc'), previewSize: '14px' },
    { value: 'normal', label: t('preferences.fontSizes.normal'), description: t('preferences.fontSizes.normalDesc'), previewSize: '15.4px' },
    { value: 'large', label: t('preferences.fontSizes.larger'), description: t('preferences.fontSizes.largerDesc'), previewSize: '17px' },
  ]

  return (
    <div>
      <h1 className="text-xl font-semibold text-[var(--foreground)] mb-6">{t('preferences.appearance')}</h1>

      <div className="space-y-8">
        <div>
          <h2 className="text-sm font-medium text-[var(--foreground-secondary)] uppercase tracking-wide mb-3">
            {t('preferences.theme')}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {themes.map((th) => (
              <button
                key={th.value}
                onClick={() => setTheme(th.value)}
                className={`rounded-lg border-2 p-4 text-left transition-colors ${
                  theme === th.value
                    ? 'border-[var(--primary)] bg-[var(--primary-muted)]'
                    : 'border-[var(--border)] bg-[var(--surface)] hover:border-[var(--border)] dark:hover:border-[var(--border)]'
                }`}
              >
                {th.value === 'system' ? (
                  <div className="mb-3 rounded-md border border-[var(--border)] overflow-hidden flex">
                    <div className="w-1/2">
                      <div className="h-3 bg-white border-b border-[var(--border)]" />
                      <div className="h-12 flex gap-1 p-1.5 bg-[var(--surface-secondary)]">
                        <div className="w-6 rounded bg-[var(--surface-tertiary)]" />
                        <div className="flex-1 space-y-1 pt-0.5">
                          <div className="h-1.5 rounded bg-[var(--border)] w-3/4" />
                          <div className="h-1.5 rounded bg-[var(--surface-tertiary)] w-1/2" />
                        </div>
                      </div>
                    </div>
                    <div className="w-1/2">
                      <div className="h-3 bg-[var(--background)] border-b border-[var(--border)]" />
                      <div className="h-12 flex gap-1 p-1.5 bg-[var(--background)]">
                        <div className="w-6 rounded bg-[var(--surface-secondary)]" />
                        <div className="flex-1 space-y-1 pt-0.5">
                          <div className="h-1.5 rounded bg-[var(--foreground-secondary)] w-3/4" />
                          <div className="h-1.5 rounded bg-[var(--surface-secondary)] w-1/2" />
                        </div>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className={`mb-3 rounded-md border overflow-hidden ${
                    th.value === 'light' ? 'border-[var(--border)]' : 'border-[var(--border)]'
                  }`}>
                    <div className={`h-3 ${th.value === 'light' ? 'bg-white border-b border-[var(--border)]' : 'bg-[var(--background)] border-b border-[var(--border)]'}`} />
                    <div className={`h-12 flex gap-1 p-1.5 ${th.value === 'light' ? 'bg-[var(--surface-secondary)]' : 'bg-[var(--background)]'}`}>
                      <div className={`w-8 rounded ${th.value === 'light' ? 'bg-[var(--surface-tertiary)]' : 'bg-[var(--surface-secondary)]'}`} />
                      <div className="flex-1 space-y-1 pt-0.5">
                        <div className={`h-1.5 rounded ${th.value === 'light' ? 'bg-[var(--border)]' : 'bg-[var(--foreground-secondary)]'} w-3/4`} />
                        <div className={`h-1.5 rounded ${th.value === 'light' ? 'bg-[var(--surface-tertiary)]' : 'bg-[var(--surface-secondary)]'} w-1/2`} />
                      </div>
                    </div>
                  </div>
                )}
                <p className="text-sm font-medium text-[var(--foreground)]">{th.label}</p>
                <p className="text-xs text-[var(--foreground-secondary)]">{th.description}</p>
              </button>
            ))}
          </div>
        </div>

        <div>
          <h2 className="text-sm font-medium text-[var(--foreground-secondary)] uppercase tracking-wide mb-3">
            {t('preferences.colorTheme')}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {colorThemes.map((ct) => (
              <button
                key={ct.value}
                onClick={() => setColorTheme(ct.value)}
                className={`rounded-lg border-2 p-4 text-left transition-colors ${
                  colorTheme === ct.value
                    ? 'border-[var(--primary)] bg-[var(--primary-muted)]'
                    : 'border-[var(--border)] bg-[var(--surface)] hover:border-[var(--border)] dark:hover:border-[var(--border)]'
                }`}
              >
                <div className="mb-3 flex items-center gap-2">
                  <div
                    className="w-8 h-8 rounded-full border-2 border-white dark:border-[var(--border)] shadow-sm"
                    style={{ backgroundColor: ct.color }}
                  />
                  {colorTheme === ct.value && (
                    <Check className="h-5 w-5 text-[var(--primary)]" />
                  )}
                </div>
                <p className="text-sm font-medium text-[var(--foreground)]">{ct.label}</p>
              </button>
            ))}
          </div>
        </div>

        <div>
          <h2 className="text-sm font-medium text-[var(--foreground-secondary)] uppercase tracking-wide mb-3">
            {t('preferences.layout')}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {([
              { value: 'centered' as Layout, label: t('preferences.layouts.centered'), description: t('preferences.layouts.centeredDesc') },
              { value: 'expanded' as Layout, label: t('preferences.layouts.expanded'), description: t('preferences.layouts.expandedDesc') },
            ]).map((lo) => (
              <button
                key={lo.value}
                onClick={() => setLayout(lo.value)}
                className={`rounded-lg border-2 p-4 text-left transition-colors ${
                  layout === lo.value
                    ? 'border-[var(--primary)] bg-[var(--primary-muted)]'
                    : 'border-[var(--border)] bg-[var(--surface)] hover:border-[var(--border)] dark:hover:border-[var(--border)]'
                }`}
              >
                <div className="mb-3 rounded-md border border-[var(--border)] overflow-hidden">
                  <div className="h-3 bg-[var(--surface-secondary)] border-b border-[var(--border)]" />
                  <div className={`h-12 bg-[var(--surface)] flex items-start pt-1.5 ${
                    lo.value === 'centered' ? 'px-4' : 'px-1.5'
                  }`}>
                    <div className="flex-1 space-y-1">
                      <div className="h-1.5 rounded bg-[var(--border)] w-3/4" />
                      <div className="h-1.5 rounded bg-[var(--surface-tertiary)] w-1/2" />
                    </div>
                  </div>
                </div>
                <p className="text-sm font-medium text-[var(--foreground)]">{lo.label}</p>
                <p className="text-xs text-[var(--foreground-secondary)]">{lo.description}</p>
              </button>
            ))}
          </div>
        </div>

        <div>
          <h2 className="text-sm font-medium text-[var(--foreground-secondary)] uppercase tracking-wide mb-3">
            {t('preferences.fontSize')}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {fontSizes.map((fs) => (
              <button
                key={fs.value}
                onClick={() => setFontSize(fs.value)}
                className={`rounded-lg border-2 p-4 text-left transition-colors ${
                  fontSize === fs.value
                    ? 'border-[var(--primary)] bg-[var(--primary-muted)]'
                    : 'border-[var(--border)] bg-[var(--surface)] hover:border-[var(--border)] dark:hover:border-[var(--border)]'
                }`}
              >
                <p className="text-sm font-medium text-[var(--foreground)]">{fs.label}</p>
                <p className="text-[var(--foreground-secondary)] mt-1" style={{ fontSize: fs.previewSize }}>{fs.description}</p>
              </button>
            ))}
          </div>
        </div>

        <div>
          <h2 className="text-sm font-medium text-[var(--foreground-secondary)] uppercase tracking-wide mb-3">
            {t('preferences.language')}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {availableLanguages.map((lang) => (
              <button
                key={lang.value}
                onClick={() => setLanguage(lang.value)}
                className={`rounded-lg border-2 p-4 text-left transition-colors ${
                  language === lang.value
                    ? 'border-[var(--primary)] bg-[var(--primary-muted)]'
                    : 'border-[var(--border)] bg-[var(--surface)] hover:border-[var(--border)] dark:hover:border-[var(--border)]'
                }`}
              >
                <p className="text-sm font-medium text-[var(--foreground)]">{lang.nativeLabel}</p>
                <p className="text-xs text-[var(--foreground-secondary)]">{lang.label}</p>
              </button>
            ))}
          </div>
        </div>

        {!hideCompletedItems && <div>
          <h2 className="text-sm font-medium text-[var(--foreground-secondary)] uppercase tracking-wide mb-3">
            {t('preferences.completedItems')}
          </h2>
          <div className="rounded-lg border border-[var(--border)] px-4 py-3">
            <label className="flex items-start gap-3 cursor-pointer">
              <input
                type="checkbox"
                checked={strikethroughEnabled}
                onChange={() => {
                  setPreferenceMutation.mutate(
                    { key: 'strikethrough_completed', value: !strikethroughEnabled },
                    {
                      onSuccess: () => {
                        setSavedId('strikethrough')
                        setTimeout(() => setSavedId(null), 2000)
                      },
                    },
                  )
                }}
                className="mt-0.5 h-4 w-4 rounded border-[var(--border)] text-[var(--primary)] focus:ring-[var(--focus-ring)] dark:border-[var(--border)] bg-[var(--surface)]"
              />
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-medium text-[var(--foreground)]">
                    {t('preferences.strikethroughCompleted')}
                  </span>
                  {savedId === 'strikethrough' && (
                    <Check className="h-4 w-4 text-green-500" />
                  )}
                </div>
                <p className="text-xs text-[var(--foreground-secondary)]">
                  {t('preferences.strikethroughCompletedDesc')}
                </p>
              </div>
            </label>
          </div>
        </div>}

      </div>
    </div>
  )
}
