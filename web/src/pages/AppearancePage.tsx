import { useTranslation } from 'react-i18next'
import { useTheme, type Theme, type FontSize } from '@/contexts/ThemeContext'
import { useLayout, type Layout } from '@/contexts/LayoutContext'
import { useLanguage } from '@/contexts/LanguageContext'
export function AppearancePage() {
  const { t } = useTranslation()
  const { theme, setTheme, fontSize, setFontSize } = useTheme()
  const { layout, setLayout } = useLayout()
  const { language, setLanguage, availableLanguages } = useLanguage()

  const themes: { value: Theme; label: string; description: string }[] = [
    { value: 'light', label: t('preferences.themes.light'), description: t('preferences.themes.lightDesc') },
    { value: 'dark', label: t('preferences.themes.dark'), description: t('preferences.themes.darkDesc') },
    { value: 'system', label: t('preferences.themes.system'), description: t('preferences.themes.systemDesc') },
  ]

  const fontSizes: { value: FontSize; label: string; description: string; previewSize: string }[] = [
    { value: 'small', label: t('preferences.fontSizes.smaller'), description: t('preferences.fontSizes.smallerDesc'), previewSize: '14px' },
    { value: 'normal', label: t('preferences.fontSizes.normal'), description: t('preferences.fontSizes.normalDesc'), previewSize: '15.4px' },
    { value: 'large', label: t('preferences.fontSizes.larger'), description: t('preferences.fontSizes.largerDesc'), previewSize: '17px' },
  ]

  return (
    <div>
      <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">{t('preferences.appearance')}</h1>

      <div className="space-y-8">
        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            {t('preferences.theme')}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {themes.map((th) => (
              <button
                key={th.value}
                onClick={() => setTheme(th.value)}
                className={`rounded-lg border-2 p-4 text-left transition-colors ${
                  theme === th.value
                    ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20'
                    : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-600'
                }`}
              >
                {th.value === 'system' ? (
                  <div className="mb-3 rounded-md border border-gray-300 overflow-hidden flex">
                    <div className="w-1/2">
                      <div className="h-3 bg-white border-b border-gray-200" />
                      <div className="h-12 flex gap-1 p-1.5 bg-gray-50">
                        <div className="w-6 rounded bg-gray-200" />
                        <div className="flex-1 space-y-1 pt-0.5">
                          <div className="h-1.5 rounded bg-gray-300 w-3/4" />
                          <div className="h-1.5 rounded bg-gray-200 w-1/2" />
                        </div>
                      </div>
                    </div>
                    <div className="w-1/2">
                      <div className="h-3 bg-gray-900 border-b border-gray-700" />
                      <div className="h-12 flex gap-1 p-1.5 bg-gray-900">
                        <div className="w-6 rounded bg-gray-700" />
                        <div className="flex-1 space-y-1 pt-0.5">
                          <div className="h-1.5 rounded bg-gray-600 w-3/4" />
                          <div className="h-1.5 rounded bg-gray-700 w-1/2" />
                        </div>
                      </div>
                    </div>
                  </div>
                ) : (
                  <div className={`mb-3 rounded-md border overflow-hidden ${
                    th.value === 'light' ? 'border-gray-200' : 'border-gray-700'
                  }`}>
                    <div className={`h-3 ${th.value === 'light' ? 'bg-white border-b border-gray-200' : 'bg-gray-900 border-b border-gray-700'}`} />
                    <div className={`h-12 flex gap-1 p-1.5 ${th.value === 'light' ? 'bg-gray-50' : 'bg-gray-900'}`}>
                      <div className={`w-8 rounded ${th.value === 'light' ? 'bg-gray-200' : 'bg-gray-700'}`} />
                      <div className="flex-1 space-y-1 pt-0.5">
                        <div className={`h-1.5 rounded ${th.value === 'light' ? 'bg-gray-300' : 'bg-gray-600'} w-3/4`} />
                        <div className={`h-1.5 rounded ${th.value === 'light' ? 'bg-gray-200' : 'bg-gray-700'} w-1/2`} />
                      </div>
                    </div>
                  </div>
                )}
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{th.label}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400">{th.description}</p>
              </button>
            ))}
          </div>
        </div>

        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
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
                    ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20'
                    : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-600'
                }`}
              >
                <div className="mb-3 rounded-md border border-gray-300 dark:border-gray-600 overflow-hidden">
                  <div className="h-3 bg-gray-100 dark:bg-gray-700 border-b border-gray-200 dark:border-gray-600" />
                  <div className={`h-12 bg-white dark:bg-gray-800 flex items-start pt-1.5 ${
                    lo.value === 'centered' ? 'px-4' : 'px-1.5'
                  }`}>
                    <div className="flex-1 space-y-1">
                      <div className="h-1.5 rounded bg-gray-300 dark:bg-gray-600 w-3/4" />
                      <div className="h-1.5 rounded bg-gray-200 dark:bg-gray-700 w-1/2" />
                    </div>
                  </div>
                </div>
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{lo.label}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400">{lo.description}</p>
              </button>
            ))}
          </div>
        </div>

        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            {t('preferences.fontSize')}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {fontSizes.map((fs) => (
              <button
                key={fs.value}
                onClick={() => setFontSize(fs.value)}
                className={`rounded-lg border-2 p-4 text-left transition-colors ${
                  fontSize === fs.value
                    ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20'
                    : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-600'
                }`}
              >
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{fs.label}</p>
                <p className="text-gray-500 dark:text-gray-400 mt-1" style={{ fontSize: fs.previewSize }}>{fs.description}</p>
              </button>
            ))}
          </div>
        </div>

        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            {t('preferences.language')}
          </h2>
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
            {availableLanguages.map((lang) => (
              <button
                key={lang.value}
                onClick={() => setLanguage(lang.value)}
                className={`rounded-lg border-2 p-4 text-left transition-colors ${
                  language === lang.value
                    ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20'
                    : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-600'
                }`}
              >
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{lang.nativeLabel}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400">{lang.label}</p>
              </button>
            ))}
          </div>
        </div>

      </div>
    </div>
  )
}
