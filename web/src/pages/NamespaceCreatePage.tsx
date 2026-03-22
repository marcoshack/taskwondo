import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useCreateNamespace } from '@/hooks/useNamespaces'
import { useNamespaceContext } from '@/contexts/NamespaceContext'
import { useSidebar } from '@/contexts/SidebarContext'
import { useLayout } from '@/contexts/LayoutContext'
import { AppSidebar } from '@/components/AppSidebar'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { getLocalizedError } from '@/utils/apiError'

export function NamespaceCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { collapsed } = useSidebar('app')
  const { containerClass } = useLayout()
  const { setActiveNamespace } = useNamespaceContext()
  const createMutation = useCreateNamespace()

  const [slug, setSlug] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [formError, setFormError] = useState('')

  function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    if (!slug.trim() || !displayName.trim()) return

    createMutation.mutate(
      { slug: slug.trim(), display_name: displayName.trim() },
      {
        onSuccess: (ns) => {
          setActiveNamespace(ns.slug)
        },
        onError: (err) => {
          setFormError(getLocalizedError(err, t, 'namespaces.createError'))
        },
      },
    )
  }

  return (
    <div className={`${containerClass(true)} py-6`}>
      <div className={`flex transition-all duration-200 ${collapsed ? 'gap-4' : 'gap-8'}`}>
        <AppSidebar />
        <div className="flex-1 min-w-0 max-w-xl">
          <h1 className="text-lg font-semibold text-gray-900 dark:text-gray-100 mb-6">{t('namespaces.createTitle')}</h1>
          <form onSubmit={handleCreate} className="space-y-4">
            <Input
              label={t('namespaces.displayName')}
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder={t('namespaces.displayNamePlaceholder')}
              required
            />
            <Input
              label={t('namespaces.slug')}
              value={slug}
              onChange={(e) => setSlug(e.target.value.toLowerCase().replace(/[^a-z0-9-]/g, ''))}
              placeholder={t('namespaces.slugPlaceholder')}
              maxLength={30}
              required
            />
            <p className="text-xs text-gray-400 dark:text-gray-500 -mt-3">{t('namespaces.slugHint')}</p>
            {formError && <p className="text-sm text-red-600 dark:text-red-400">{formError}</p>}
            <div className="flex justify-end gap-3 pt-2">
              <Button type="button" variant="secondary" onClick={() => navigate(-1)}>{t('common.cancel')}</Button>
              <Button type="submit" disabled={createMutation.isPending || !slug.trim() || !displayName.trim()}>
                {createMutation.isPending ? t('common.creating') : t('common.create')}
              </Button>
            </div>
          </form>
        </div>
      </div>
    </div>
  )
}
