import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { useSystemSetting, useSetSystemSetting } from '@/hooks/useSystemSettings'
import { Input } from '@/components/ui/Input'
import { Button } from '@/components/ui/Button'
import { Spinner } from '@/components/ui/Spinner'

export function SystemGeneralPage() {
  const { t } = useTranslation()
  const { data: savedBrandName, isLoading } = useSystemSetting<string>('brand_name')
  const setSettingMutation = useSetSystemSetting()

  const [brandName, setBrandName] = useState('')
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (savedBrandName !== undefined) {
      setBrandName(savedBrandName ?? '')
    }
  }, [savedBrandName])

  const handleSave = () => {
    setSaved(false)
    setSettingMutation.mutate(
      { key: 'brand_name', value: brandName.trim() || null },
      {
        onSuccess: () => setSaved(true),
      },
    )
  }

  const isDirty = brandName !== (savedBrandName ?? '')

  if (isLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner />
      </div>
    )
  }

  return (
    <div className="max-w-3xl">
      <div className="mb-6">
        <h2 className="text-xl font-semibold text-gray-900 dark:text-gray-100">
          {t('admin.general.title')}
        </h2>
        <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {t('admin.general.description')}
        </p>
      </div>

      <div className="rounded-lg border border-gray-200 dark:border-gray-700 p-6">
        <h3 className="text-lg font-medium text-gray-900 dark:text-gray-100 mb-4">
          {t('admin.general.brand.title')}
        </h3>

        <div className="max-w-md space-y-4">
          <Input
            label={t('admin.general.brand.name')}
            value={brandName}
            onChange={(e) => {
              setBrandName(e.target.value)
              setSaved(false)
            }}
            placeholder="Taskwondo"
          />
          <p className="text-sm text-gray-500 dark:text-gray-400">
            {t('admin.general.brand.nameHelp')}
          </p>

          <div className="flex items-center gap-3">
            <Button
              onClick={handleSave}
              disabled={!isDirty || setSettingMutation.isPending}
            >
              {setSettingMutation.isPending ? t('common.saving') : t('common.save')}
            </Button>
            {saved && (
              <span className="text-sm text-green-600 dark:text-green-400">
                {t('admin.general.brand.saved')}
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}
