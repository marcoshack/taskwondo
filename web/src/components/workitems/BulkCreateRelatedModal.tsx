import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { Select } from '@/components/ui/Select'
import { ProjectPicker } from '@/components/ui/ProjectPicker'
import { useAllProjects } from '@/hooks/useProjects'
import { createWorkItem, createRelation } from '@/api/workitems'
import { useQueryClient } from '@tanstack/react-query'

const RELATION_TYPES = ['blocks', 'blocked_by', 'relates_to', 'duplicates', 'caused_by', 'parent_of', 'child_of']
const TYPES = ['task', 'ticket', 'bug', 'feedback', 'epic', 'story']

interface BulkCreateRelatedModalProps {
  open: boolean
  onClose: () => void
  projectKey: string
  itemNumber: number
}

export function BulkCreateRelatedModal({ open, onClose, projectKey: sourceProjectKey, itemNumber }: BulkCreateRelatedModalProps) {
  const { t } = useTranslation()
  const qc = useQueryClient()
  const { data: projects } = useAllProjects()
  
  const [selectedProjectKey, setSelectedProjectKey] = useState(sourceProjectKey)
  const [type, setType] = useState('task')
  const [relationType, setRelationType] = useState('child_of')
  const [titles, setTitles] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [progress, setProgress] = useState({ current: 0, total: 0 })
  const [error, setError] = useState<string | null>(null)

  const creatableProjects = useMemo(
    () => projects?.filter((p) => !p.member_role || ['owner', 'admin', 'member'].includes(p.member_role)) ?? [],
    [projects],
  )

  const handleClose = () => {
    if (isSubmitting) return
    setTitles('')
    setError(null)
    setProgress({ current: 0, total: 0 })
    onClose()
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    const titleList = titles.split('\n').map(t => t.trim()).filter(Boolean)
    if (titleList.length === 0) return

    setIsSubmitting(true)
    setError(null)
    setProgress({ current: 0, total: titleList.length })

    const targetProject = projects?.find(p => p.key === selectedProjectKey)
    const namespaceSlug = targetProject?.namespace_slug

    let succeeded = 0
    for (const title of titleList) {
      try {
        const newItem = await createWorkItem(selectedProjectKey, {
          type,
          title,
        }, namespaceSlug)
        
        await createRelation(sourceProjectKey, itemNumber, newItem.display_id, relationType)
        succeeded++
        setProgress(prev => ({ ...prev, current: succeeded }))
      } catch (err) {
        console.error('Failed to create related item:', err)
        setError(t('workitems.bulkCreateRelated.error'))
        // We continue with others
      }
    }

    setIsSubmitting(false)
    if (succeeded > 0) {
      qc.invalidateQueries({ queryKey: ['projects', sourceProjectKey, 'items', itemNumber, 'relations'] })
      qc.invalidateQueries({ queryKey: ['projects', selectedProjectKey, 'items'] })
      if (succeeded === titleList.length) {
        handleClose()
      }
    }
  }

  return (
    <Modal open={open} onClose={handleClose} title={t('workitems.bulkCreateRelated.title')} dismissable={!isSubmitting}>
      <form onSubmit={handleSubmit} className="space-y-4">
        <ProjectPicker
          projects={creatableProjects}
          value={selectedProjectKey}
          onChange={setSelectedProjectKey}
          disabled={isSubmitting}
        />
        
        <div className="grid grid-cols-2 gap-4">
          <Select
            label={t('workitems.bulkCreateRelated.type')}
            value={type}
            onChange={(e) => setType(e.target.value)}
            disabled={isSubmitting}
          >
            {TYPES.map((tp) => (
              <option key={tp} value={tp}>{t(`workitems.types.${tp}`)}</option>
            ))}
          </Select>

          <Select
            label={t('workitems.bulkCreateRelated.relationType')}
            value={relationType}
            onChange={(e) => setRelationType(e.target.value)}
            disabled={isSubmitting}
          >
            {RELATION_TYPES.map((tp) => (
              <option key={tp} value={tp}>{t(`relations.types.${tp}`)}</option>
            ))}
          </Select>
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-1">
            {t('workitems.bulkCreateRelated.titles')}
          </label>
          <textarea
            className="block w-full rounded-md border border-gray-300 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 px-3 py-2 text-sm shadow-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 disabled:opacity-50"
            rows={6}
            placeholder={t('workitems.bulkCreateRelated.titlesPlaceholder')}
            value={titles}
            onChange={(e) => setTitles(e.target.value)}
            disabled={isSubmitting}
            required
          />
        </div>

        {isSubmitting && (
          <div className="text-sm text-gray-500">
            {t('workitems.bulkCreateRelated.creating', { current: progress.current, total: progress.total })}
          </div>
        )}

        {error && (
          <div className="text-sm text-red-600 dark:text-red-400">
            {error}
          </div>
        )}

        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={handleClose} disabled={isSubmitting}>
            {t('common.cancel')}
          </Button>
          <Button type="submit" disabled={isSubmitting || !titles.trim()}>
            {isSubmitting ? t('common.creating') : t('workitems.bulkCreateRelated.create')}
          </Button>
        </div>
      </form>
    </Modal>
  )
}
