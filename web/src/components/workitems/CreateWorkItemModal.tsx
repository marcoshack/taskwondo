import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { WorkItemForm } from '@/components/workitems/WorkItemForm'
import { useAllProjects, useMembers } from '@/hooks/useProjects'
import { useCreateWorkItem } from '@/hooks/useWorkItems'
import { useMilestones } from '@/hooks/useMilestones'
import { useLastProjectKey } from '@/hooks/useLastProjectKey'
import { getLocalizedError } from '@/utils/apiError'

// Roles that are allowed to create work items via the regular POST
// /projects/{key}/items endpoint. Customers and viewers are rejected by the
// API, so we filter them out of the project picker as well — customers must
// open tickets through the Support page instead.
const CREATABLE_ROLES = new Set(['owner', 'admin', 'member'])

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
  const lastProjectKey = useLastProjectKey()
  const [selectedProjectKey, setSelectedProjectKey] = useState(
    lockedProjectKey ?? lastProjectKey ?? '',
  )

  const { data: projects } = useAllProjects()

  const creatableProjects = useMemo(
    () => projects?.filter((p) => !p.member_role || CREATABLE_ROLES.has(p.member_role)),
    [projects],
  )

  // Resolve the active project key against the creatable list. If the
  // remembered selection belongs to a project the user can only access as
  // customer/viewer (or no longer exists), clear it so we don't fire
  // members/milestones requests that will 404. We intentionally do NOT
  // auto-pick a fallback project — the picker should remain in its
  // "Select project" placeholder state until the user makes a choice.
  const activeProjectKey = useMemo(() => {
    if (lockedProjectKey) return lockedProjectKey
    if (!creatableProjects) return ''
    if (!selectedProjectKey) return ''
    if (creatableProjects.some((p) => p.key === selectedProjectKey)) return selectedProjectKey
    return ''
  }, [lockedProjectKey, creatableProjects, selectedProjectKey])

  const project = projects?.find((p) => p.key === activeProjectKey)

  // The picker can select a project from any namespace, so every project-scoped
  // request must target the project's own namespace rather than whatever the
  // user currently has selected in the UI — otherwise we'd hit 404s like
  // /api/v1/default/projects/TEST1/members when TEST1 lives in "test1".
  const projectNamespaceSlug = project?.namespace_slug

  const { data: members } = useMembers(activeProjectKey, projectNamespaceSlug)
  const { data: milestones } = useMilestones(activeProjectKey, projectNamespaceSlug)

  const createMutation = useCreateWorkItem(activeProjectKey, projectNamespaceSlug)

  function handleClose() {
    createMutation.reset()
    setSelectedProjectKey(lockedProjectKey ?? lastProjectKey ?? '')
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
        projects={lockedProjectKey ? undefined : creatableProjects}
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
