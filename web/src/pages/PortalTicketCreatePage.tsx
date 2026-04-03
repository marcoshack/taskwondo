import { useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { usePortalQueues, useCreatePortalTicket } from '@/hooks/usePortal'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Spinner } from '@/components/ui/Spinner'
import { ArrowLeft } from 'lucide-react'

const PRIORITIES = ['low', 'medium', 'high', 'critical'] as const

export function PortalTicketCreatePage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { namespace = 'default', projectKey = '' } = useParams<{ namespace: string; projectKey: string }>()

  const { data: queues, isLoading: queuesLoading } = usePortalQueues(namespace, projectKey)
  const createMutation = useCreatePortalTicket(namespace, projectKey)

  const [categoryId, setCategoryId] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState('medium')
  const [error, setError] = useState('')

  // The public queue is the only one returned by the portal API
  const publicQueue = queues?.[0]

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError('')

    if (!title.trim()) {
      setError(t('portal.errorTitleRequired'))
      return
    }

    try {
      const ticket = await createMutation.mutateAsync({
        title: title.trim(),
        description: description.trim() || undefined,
        priority,
        category_id: categoryId || undefined,
      })
      navigate(`../${ticket.item_number}`, { replace: true })
    } catch {
      setError(t('portal.errorCreateFailed'))
    }
  }

  if (queuesLoading) {
    return (
      <div className="flex justify-center py-12">
        <Spinner size="lg" />
      </div>
    )
  }

  if (!publicQueue) {
    navigate('..', { replace: true })
    return null
  }

  return (
    <div>
      <button
        onClick={() => navigate(-1)}
        className="flex items-center gap-1.5 text-sm text-gray-500 dark:text-gray-400 hover:text-gray-700 dark:hover:text-gray-200 mb-4"
      >
        <ArrowLeft className="h-4 w-4" />
        {t('portal.backToTickets')}
      </button>

      <h1 className="text-xl font-semibold text-gray-900 dark:text-gray-100 mb-6">
        {t('portal.createTicket')}
      </h1>

      <form onSubmit={handleSubmit} className="space-y-4 max-w-lg">
        {publicQueue && publicQueue.categories.length > 0 && (
          <div>
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
              {t('portal.category')}
            </label>
            <Select value={categoryId} onChange={(e) => setCategoryId(e.target.value)}>
              <option value="">{t('portal.selectCategory')}</option>
              {publicQueue.categories.map((c) => (
                <option key={c.id} value={c.id}>{c.name}</option>
              ))}
            </Select>
          </div>
        )}

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('portal.ticketTitle')} *
          </label>
          <Input
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder={t('portal.ticketTitle')}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('portal.ticketDescription')}
          </label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            rows={5}
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-400"
            placeholder={t('portal.ticketDescription')}
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('portal.ticketPriority')}
          </label>
          <Select value={priority} onChange={(e) => setPriority(e.target.value)}>
            {PRIORITIES.map((p) => (
              <option key={p} value={p}>{t(`workitems.priorities.${p}`)}</option>
            ))}
          </Select>
        </div>

        {error && (
          <p className="text-sm text-red-600 dark:text-red-400">{error}</p>
        )}

        <div className="flex gap-3 pt-2">
          <Button type="submit" disabled={createMutation.isPending}>
            {createMutation.isPending ? t('common.creating') : t('portal.submit')}
          </Button>
          <Button type="button" variant="secondary" onClick={() => navigate(-1)}>
            {t('portal.cancel')}
          </Button>
        </div>
      </form>
    </div>
  )
}
