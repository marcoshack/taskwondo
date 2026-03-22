import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { WorkItemForm } from '@/components/workitems/WorkItemForm'
import { useProjects, useMembers } from '@/hooks/useProjects'
import { useCreateWorkItem } from '@/hooks/useWorkItems'
import { useMilestones } from '@/hooks/useMilestones'
import { getLocalizedError } from '@/utils/apiError'

interface CreateWorkItemModalProps {
  open: boolean
  onClose: () => void
  /** When set, the project selector is locked to this value. */
  lockedProjectKey?: string
  /** Called after a work item is successfully created. Receives the new work item's id. */
  onCreated?: (workItemId: string) => void
}

export function CreateWorkItemModal({ open, onClose, lockedProjectKey, onCreated }: CreateWorkItemModalProps) {
  const { t } = useTranslation()
  const [selectedProjectKey, setSelectedProjectKey] = useState(
    lockedProjectKey ?? localStorage.getItem('taskwondo_last_project_key') ?? '',
  )
  const activeProjectKey = lockedProjectKey ?? selectedProjectKey

  const { data: projects } = useProjects()
  const { data: members } = useMembers(activeProjectKey)
  const { data: milestones } = useMilestones(activeProjectKey)

  const project = projects?.find((p) => p.key === activeProjectKey)

  const createMutation = useCreateWorkItem(activeProjectKey)

  function handleClose() {
    createMutation.reset()
    setSelectedProjectKey(lockedProjectKey ?? localStorage.getItem('taskwondo_last_project_key') ?? '')
    onClose()
  }

  return (
    <Modal open={open} onClose={handleClose} title={t('workitems.newTitle')} dismissable={false}>
      <WorkItemForm
        projectKey={activeProjectKey}
        mode="create"
        members={members ?? []}
        milestones={milestones}
        allowedComplexityValues={project?.allowed_complexity_values}
        projects={lockedProjectKey ? undefined : projects}
        projectLocked={!!lockedProjectKey}
        onProjectChange={setSelectedProjectKey}
        onSubmit={(values) => {
          createMutation.mutate(values as { type: string; title: string }, {
            onSuccess: (item) => {
              onCreated?.(item.id)
              handleClose()
            },
          })
        }}
        onCancel={handleClose}
        isSubmitting={createMutation.isPending}
        submitError={createMutation.error ? t('workitems.form.submitError', { message: getLocalizedError(createMutation.error, t, 'common.unknown') }) : null}
      />
    </Modal>
  )
}
