import { useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useQueues, useCreateQueue } from '@/hooks/useQueues'
import { useMembers } from '@/hooks/useProjects'
import { useAuth } from '@/contexts/AuthContext'
import { useNamespacePath } from '@/hooks/useNamespacePath'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Select } from '@/components/ui/Select'
import { Modal } from '@/components/ui/Modal'
import { Spinner } from '@/components/ui/Spinner'
import { Badge } from '@/components/ui/Badge'
import { Plus, Check, Settings } from 'lucide-react'
import type { CreateQueueInput } from '@/api/queues'
import { getLocalizedError } from '@/utils/apiError'

const QUEUE_TYPES = ['support', 'alerts', 'feedback', 'general'] as const

export function QueuesPage() {
  const { t } = useTranslation()
  const { projectKey } = useParams<{ projectKey: string }>()
  const { p } = useNamespacePath()
  const { user } = useAuth()
  const { data: members } = useMembers(projectKey ?? '')
  const { data: queues, isLoading } = useQueues(projectKey ?? '')

  const createMutation = useCreateQueue(projectKey ?? '')

  const [createOpen, setCreateOpen] = useState(false)
  const [error, setError] = useState('')
  const [savedId, setSavedId] = useState<string | null>(null)

  const currentUserMember = members?.find((m) => m.user_id === user?.id)
  const currentUserRole = currentUserMember?.role ?? (user?.global_role === 'admin' ? 'owner' : null)
  const canManage = currentUserRole === 'owner' || currentUserRole === 'admin' || user?.global_role === 'admin'

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <Spinner />
      </div>
    )
  }

  function flashSaved(id: string) {
    setSavedId(id)
    setTimeout(() => setSavedId(null), 2000)
  }

  function handleCreate(input: CreateQueueInput) {
    setError('')
    createMutation.mutate(input, {
      onSuccess: (data) => {
        flashSaved(data.id)
        setCreateOpen(false)
      },
      onError: (err) => {
        setError(getLocalizedError(err, t, 'queues.createError'))
      },
    })
  }

  return (
    <div className="max-w-3xl space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold text-[var(--foreground)]">{t('queues.title')}</h2>
          <p className="mt-1 text-sm text-[var(--foreground-secondary)]">{t('queues.description')}</p>
        </div>
        {canManage && (
          <Button onClick={() => setCreateOpen(true)} className="border border-transparent">
            <Plus className="h-4 w-4 mr-1" />
            {t('queues.create')}
          </Button>
        )}
      </div>

      {error && <p className="text-sm text-[var(--danger)]">{error}</p>}

      {(!queues || queues.length === 0) ? (
        <div className="border border-dashed border-[var(--border)] rounded-lg p-6 text-center">
          <p className="text-sm text-[var(--foreground-secondary)]">{t('queues.noQueues')}</p>
          {canManage && (
            <Button size="sm" variant="secondary" className="mt-3" onClick={() => setCreateOpen(true)}>
              <Plus className="h-4 w-4 mr-1" />
              {t('queues.createFirst')}
            </Button>
          )}
        </div>
      ) : (
        <div className="border border-[var(--border)] rounded-lg divide-y divide-[var(--border)]">
          {queues.map((queue) => (
            <Link
              key={queue.id}
              to={p(`/projects/${projectKey}/queues/${queue.id}/items`)}
              className="block p-4 hover:bg-[var(--surface-hover)] transition-colors first:rounded-t-lg last:rounded-b-lg"
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    <span className="text-base font-semibold text-[var(--foreground)]">
                      {queue.name}
                    </span>
                    <Badge color="gray">{t(`queues.types.${queue.queue_type}`)}</Badge>
                    {queue.is_public && (
                      <Badge color="indigo">{t('queues.public')}</Badge>
                    )}
                  </div>
                  {queue.description && (
                    <p className="text-xs text-[var(--foreground-secondary)] mt-1">{queue.description}</p>
                  )}
                </div>
                <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.preventDefault()}>
                  {savedId === queue.id && <Check className="h-5 w-5 text-green-500 animate-[pulse_0.6s_ease-in-out_2]" />}
                  <Link
                    to={p(`/projects/${projectKey}/queues/${queue.id}`)}
                    onClick={(e) => e.stopPropagation()}
                    title={t('queues.settings')}
                  >
                    <Button variant="ghost" size="sm">
                      <Settings className="h-4.5 w-4.5" />
                    </Button>
                  </Link>
                </div>
              </div>
            </Link>
          ))}
        </div>
      )}

      {/* Create modal */}
      <Modal
        open={createOpen}
        onClose={() => setCreateOpen(false)}
        title={t('queues.createQueue')}
      >
        <QueueCreateForm
          onSubmit={handleCreate}
          onCancel={() => setCreateOpen(false)}
          isPending={createMutation.isPending}
        />
      </Modal>

    </div>
  )
}

// --- Queue Create Form ---

function QueueCreateForm({
  onSubmit,
  onCancel,
  isPending,
}: {
  onSubmit: (input: CreateQueueInput) => void
  onCancel: () => void
  isPending: boolean
}) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [queueType, setQueueType] = useState<string>('general')
  const [validationError, setValidationError] = useState('')

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setValidationError('')

    if (!name.trim()) {
      setValidationError(t('queues.nameRequired'))
      return
    }

    onSubmit({
      name: name.trim(),
      description: description || undefined,
      queue_type: queueType,
    })
  }

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      {validationError && (
        <p className="text-sm text-[var(--danger)]">{validationError}</p>
      )}
      <Input
        label={t('queues.name')}
        value={name}
        onChange={(e) => setName(e.target.value)}
        placeholder={t('queues.namePlaceholder')}
        required
        autoFocus
      />
      <div>
        <label className="block text-sm font-medium text-[var(--foreground)] mb-1">
          {t('common.description')}
        </label>
        <textarea
          rows={3}
          className="block w-full rounded-md border border-[var(--border)] bg-[var(--surface)] text-[var(--foreground)] px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-[var(--focus-ring)] focus:border-[var(--primary)]"
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
      </div>
      <Select
        label={t('queues.type')}
        value={queueType}
        onChange={(e) => setQueueType(e.target.value)}
      >
        {QUEUE_TYPES.map((qt) => (
          <option key={qt} value={qt}>{t(`queues.types.${qt}`)}</option>
        ))}
      </Select>
      <div className="flex justify-end gap-3 pt-2">
        <Button type="button" variant="secondary" onClick={onCancel}>{t('common.cancel')}</Button>
        <Button type="submit" disabled={isPending}>
          {isPending ? t('common.creating') : t('common.create')}
        </Button>
      </div>
    </form>
  )
}
