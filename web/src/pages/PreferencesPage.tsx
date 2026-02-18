import { useTheme, type Theme, type FontSize } from '@/contexts/ThemeContext'

const themes: { value: Theme; label: string; description: string }[] = [
  { value: 'light', label: 'Light', description: 'Always light' },
  { value: 'dark', label: 'Dark', description: 'Always dark' },
  { value: 'system', label: 'System', description: 'Match your OS setting' },
]

const fontSizes: { value: FontSize; label: string; description: string; previewSize: string }[] = [
  { value: 'small', label: 'Smaller', description: 'Compact text', previewSize: '14px' },
  { value: 'normal', label: 'Normal', description: 'Default size', previewSize: '15.4px' },
  { value: 'large', label: 'Larger', description: 'Easier to read', previewSize: '17px' },
]

export function PreferencesPage() {
  const { theme, setTheme, fontSize, setFontSize } = useTheme()

  return (
    <div className="max-w-2xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
      <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">Preferences</h1>

      <div className="space-y-8">
        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            Theme
          </h2>
          <div className="grid grid-cols-3 gap-4">
            {themes.map((t) => (
              <button
                key={t.value}
                onClick={() => setTheme(t.value)}
                className={`rounded-lg border-2 p-4 text-left transition-colors ${
                  theme === t.value
                    ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-900/20'
                    : 'border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 hover:border-gray-300 dark:hover:border-gray-600'
                }`}
              >
                {t.value === 'system' ? (
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
                    t.value === 'light' ? 'border-gray-200' : 'border-gray-700'
                  }`}>
                    <div className={`h-3 ${t.value === 'light' ? 'bg-white border-b border-gray-200' : 'bg-gray-900 border-b border-gray-700'}`} />
                    <div className={`h-12 flex gap-1 p-1.5 ${t.value === 'light' ? 'bg-gray-50' : 'bg-gray-900'}`}>
                      <div className={`w-8 rounded ${t.value === 'light' ? 'bg-gray-200' : 'bg-gray-700'}`} />
                      <div className="flex-1 space-y-1 pt-0.5">
                        <div className={`h-1.5 rounded ${t.value === 'light' ? 'bg-gray-300' : 'bg-gray-600'} w-3/4`} />
                        <div className={`h-1.5 rounded ${t.value === 'light' ? 'bg-gray-200' : 'bg-gray-700'} w-1/2`} />
                      </div>
                    </div>
                  </div>
                )}
                <p className="text-sm font-medium text-gray-900 dark:text-gray-100">{t.label}</p>
                <p className="text-xs text-gray-500 dark:text-gray-400">{t.description}</p>
              </button>
            ))}
          </div>
        </div>

        <div>
          <h2 className="text-sm font-medium text-gray-500 dark:text-gray-400 uppercase tracking-wide mb-3">
            Font Size
          </h2>
          <div className="grid grid-cols-3 gap-4">
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
      </div>
    </div>
  )
}
