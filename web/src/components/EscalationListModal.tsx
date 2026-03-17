import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { X, Plus, Trash2 } from 'lucide-react'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { Input } from '@/components/ui/Input'
import { Avatar } from '@/components/ui/Avatar'
import { UserSearchInput } from '@/components/UserSearchInput'
import { useCreateEscalationList, useUpdateEscalationList, useEscalationListDetail } from '@/hooks/useEscalation'
import type { UserSearchResult } from '@/api/users'

interface LevelDraft {
  threshold_pct: string
  users: { id: string; display_name: string; email: string }[]
}

interface Props {
  open: boolean
  onClose: () => void
  onSave?: () => void
  projectKey: string
  editingId?: string | null
}

export function EscalationListModal({ open, onClose, onSave, projectKey, editingId }: Props) {
  const { t } = useTranslation()
  const createMutation = useCreateEscalationList(projectKey)
  const updateMutation = useUpdateEscalationList(projectKey)
  const { data: existing } = useEscalationListDetail(projectKey, editingId ?? '')

  const [name, setName] = useState('')
  const [levels, setLevels] = useState<LevelDraft[]>([])
  const [error, setError] = useState('')
  const [initialized, setInitialized] = useState(false)

  const isEdit = !!editingId

  // Initialize from existing when editing
  useEffect(() => {
    if (isEdit && existing && !initialized) {
      setName(existing.name)
      setLevels(
        existing.levels.map((lv) => ({
          threshold_pct: String(lv.threshold_pct),
          users: lv.users.map((u) => ({
            id: u.id,
            display_name: u.display_name,
            email: u.email,
          })),
        }))
      )
      setInitialized(true)
    }
  }, [isEdit, existing, initialized])

  // Reset when modal opens/closes
  useEffect(() => {
    if (!open) {
      setName('')
      setLevels([])
      setError('')
      setInitialized(false)
    }
  }, [open])

  function addLevel() {
    setLevels([...levels, { threshold_pct: '', users: [] }])
  }

  function removeLevel(index: number) {
    setLevels(levels.filter((_, i) => i !== index))
  }

  function updateLevelThreshold(index: number, value: string) {
    setLevels(levels.map((lv, i) => (i === index ? { ...lv, threshold_pct: value } : lv)))
  }

  function addUserToLevel(index: number, user: UserSearchResult) {
    setLevels(
      levels.map((lv, i) => {
        if (i !== index) return lv
        if (lv.users.some((u) => u.id === user.id)) return lv
        return {
          ...lv,
          users: [...lv.users, { id: user.id, display_name: user.display_name, email: user.email }],
        }
      })
    )
  }

  function removeUserFromLevel(levelIndex: number, userId: string) {
    setLevels(
      levels.map((lv, i) => {
        if (i !== levelIndex) return lv
        return { ...lv, users: lv.users.filter((u) => u.id !== userId) }
      })
    )
  }

  function validate(): boolean {
    setError('')

    if (!name.trim()) {
      setError(t('escalation.nameRequired'))
      return false
    }

    if (levels.length === 0) {
      setError(t('escalation.levelsRequired'))
      return false
    }

    // Check thresholds
    const thresholds = new Set<number>()
    for (const lv of levels) {
      const pct = Number(lv.threshold_pct)
      if (!pct || pct <= 0) {
        setError(t('escalation.thresholdRequired'))
        return false
      }
      if (thresholds.has(pct)) {
        setError(t('escalation.duplicateThreshold'))
        return false
      }
      thresholds.add(pct)
    }

    // Check users
    for (const lv of levels) {
      if (lv.users.length === 0) {
        setError(t('escalation.usersRequired'))
        return false
      }
    }

    return true
  }

  function handleSave() {
    if (!validate()) return

    const input = {
      name: name.trim(),
      levels: levels.map((lv) => ({
        threshold_pct: Number(lv.threshold_pct),
        user_ids: lv.users.map((u) => u.id),
      })),
    }

    if (isEdit && editingId) {
      updateMutation.mutate(
        { escalationListId: editingId, input },
        {
          onSuccess: () => {
            onSave?.()
            onClose()
          },
          onError: () => {
            setError(t('escalation.saveError'))
          },
        }
      )
    } else {
      createMutation.mutate(input, {
        onSuccess: () => {
          onSave?.()
          onClose()
        },
        onError: () => {
          setError(t('escalation.saveError'))
        },
      })
    }
  }

  const isPending = createMutation.isPending || updateMutation.isPending

  return (
    <Modal
      open={open}
      onClose={onClose}
      title={isEdit ? t('escalation.edit') : t('escalation.create')}
      className="!max-w-2xl"
    >
      <div className="space-y-5">
        {error && <p className="text-sm text-red-600 dark:text-red-400">{error}</p>}

        {/* Name */}
        <Input
          label={t('escalation.name')}
          value={name}
          onChange={(e) => setName(e.target.value)}
          placeholder={t('escalation.namePlaceholder')}
        />

        {/* Levels */}
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="block text-sm font-medium text-gray-700 dark:text-gray-300">
              {t('escalation.levels')}
            </label>
            <Button variant="ghost" size="sm" onClick={addLevel}>
              <Plus className="h-3.5 w-3.5 mr-1" />
              {t('escalation.addLevel')}
            </Button>
          </div>

          {levels.length === 0 ? (
            <p className="text-sm text-gray-500 dark:text-gray-400">{t('escalation.noLevels')}</p>
          ) : (
            <div className="space-y-4">
              {levels.map((level, index) => (
                <div
                  key={index}
                  className="border border-gray-200 dark:border-gray-700 rounded-lg p-4 space-y-3"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex items-center gap-3 flex-1">
                      <div className="w-32">
                        <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                          {t('escalation.threshold')}
                        </label>
                        <div className="flex items-center gap-1">
                          <input
                            type="number"
                            min="1"
                            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-1.5 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                            value={level.threshold_pct}
                            onChange={(e) => updateLevelThreshold(index, e.target.value)}
                            placeholder={t('escalation.thresholdPlaceholder')}
                          />
                          <span className="text-xs text-gray-500 dark:text-gray-400 shrink-0">
                            {t('escalation.thresholdHelp')}
                          </span>
                        </div>
                      </div>
                    </div>
                    <button
                      type="button"
                      className="text-gray-400 hover:text-red-500 p-1"
                      onClick={() => removeLevel(index)}
                      title={t('escalation.removeLevel')}
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>

                  {/* Users */}
                  <div>
                    <label className="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">
                      {t('escalation.users')}
                    </label>

                    {/* Selected user chips */}
                    {level.users.length > 0 && (
                      <div className="flex flex-wrap gap-1.5 mb-2">
                        {level.users.map((user) => (
                          <span
                            key={user.id}
                            className="inline-flex items-center gap-1 rounded-full bg-indigo-50 dark:bg-indigo-900/30 px-2.5 py-1 text-xs font-medium text-indigo-700 dark:text-indigo-300 border border-indigo-200 dark:border-indigo-700"
                          >
                            <Avatar name={user.display_name} size="xs" />
                            <span>{user.display_name}</span>
                            <button
                              type="button"
                              className="ml-0.5 hover:text-red-500"
                              onClick={() => removeUserFromLevel(index, user.id)}
                            >
                              <X className="h-3 w-3" />
                            </button>
                          </span>
                        ))}
                      </div>
                    )}

                    {/* User search */}
                    <UserSearchInput
                      excludeUserIds={level.users.map((u) => u.id)}
                      onSelect={(user) => addUserToLevel(index, user)}
                    />
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Actions */}
        <div className="flex justify-end gap-2 pt-2 border-t border-gray-200 dark:border-gray-700">
          <Button variant="secondary" onClick={onClose}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSave} disabled={isPending}>
            {isPending ? t('common.saving') : isEdit ? t('common.save') : t('common.create')}
          </Button>
        </div>
      </div>
    </Modal>
  )
}
