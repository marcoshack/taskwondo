import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Modal } from '@/components/ui/Modal'
import { Button } from '@/components/ui/Button'
import { ProjectKeyBadge } from '@/components/ui/ProjectKeyBadge'
import { WorkItemForm } from '@/components/workitems/WorkItemForm'
import { useAllProjects, useMembers } from '@/hooks/useProjects'
import { useCreateWorkItem } from '@/hooks/useWorkItems'
import { useMilestones } from '@/hooks/useMilestones'
import { useProjectWorkflow } from '@/hooks/useWorkflows'
import { useLastProjectKey } from '@/hooks/useLastProjectKey'
import { usePublicSettings } from '@/hooks/useSystemSettings'
import { useNotification } from '@/contexts/NotificationContext'
import { useStagedAttachments, uploadStagedAttachments } from '@/hooks/useStagedAttachments'
import type { StagedUploadProgress } from '@/hooks/useStagedAttachments'
import { resolveStagedDescription } from '@/utils/stagedAttachments'
import { updateWorkItem } from '@/api/workitems'
import type { WorkItem } from '@/api/workitems'
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
  const { showNotification } = useNotification()
  const lastProjectKey = useLastProjectKey()
  const [selectedProjectKey, setSelectedProjectKey] = useState(
    lockedProjectKey ?? lastProjectKey ?? '',
  )
  const [type, setType] = useState('')
  // Attachments upload only once the item exists, so the Create button stays in
  // its pending state until they finish.
  const [uploadingAttachments, setUploadingAttachments] = useState(false)
  const [uploadProgress, setUploadProgress] = useState<StagedUploadProgress | null>(null)
  const [formDirty, setFormDirty] = useState(false)
  const [confirmDiscard, setConfirmDiscard] = useState(false)

  const { data: projects } = useAllProjects()
  const { data: publicSettings } = usePublicSettings()
  const maxUploadSize = typeof publicSettings?.max_upload_size === 'number' ? publicSettings.max_upload_size : undefined
  const staged = useStagedAttachments(maxUploadSize)

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
  // Statuses come from the workflow the project maps this type to.
  const { statuses } = useProjectWorkflow(activeProjectKey, type || undefined, projectNamespaceSlug)

  const createMutation = useCreateWorkItem(activeProjectKey, projectNamespaceSlug)

  function handleClose() {
    createMutation.reset()
    setSelectedProjectKey(lockedProjectKey ?? lastProjectKey ?? '')
    setType('')
    staged.clear()
    setUploadingAttachments(false)
    setUploadProgress(null)
    setFormDirty(false)
    setConfirmDiscard(false)
    onClose()
  }

  // The type and the staged files live here rather than in the form, so they
  // are part of "has the user put anything in yet?" alongside the form's own
  // fields.
  const hasContent = formDirty || type !== '' || staged.staged.length > 0

  /** Closes straight away when nothing would be lost, otherwise asks first. */
  function requestClose() {
    if (uploadingAttachments) return
    if (hasContent) {
      setConfirmDiscard(true)
      return
    }
    handleClose()
  }

  /**
   * Uploads the files staged while the form was open and rewrites any
   * `staged:` description links to their real attachment URLs. The item already
   * exists by this point, so failures are reported rather than blocking.
   */
  async function uploadStagedFiles(item: WorkItem, description: string) {
    const files = staged.staged
    if (files.length === 0) return

    const { urlById, failed } = await uploadStagedAttachments(
      files,
      activeProjectKey,
      item.item_number,
      projectNamespaceSlug,
      setUploadProgress,
    )

    if (files.some((f) => f.inline)) {
      const resolved = resolveStagedDescription(description, files, urlById, (filename) =>
        t('attachments.uploadFailedInline', { filename }),
      )
      if (resolved !== description) {
        try {
          await updateWorkItem(activeProjectKey, item.item_number, { description: resolved }, projectNamespaceSlug)
        } catch {
          showNotification(t('workitems.form.attachmentsLinkFailed'), 'error')
        }
      }
    }

    if (failed.length > 0) {
      showNotification(t('workitems.form.attachmentsUploadFailed', { files: failed.join(', ') }), 'error')
    }
  }

  // Styled to match the project badge in the app's top bar.
  const headerRight = project && (
    <span className="flex min-w-0 shrink items-center gap-2.5">
      <ProjectKeyBadge size="nav">{project.key}</ProjectKeyBadge>
      <span className="truncate text-base font-semibold text-gray-900 dark:text-gray-100">{project.name}</span>
    </span>
  )

  return (
    <Modal
      open={open}
      onClose={requestClose}
      title={t('workitems.newTitle')}
      headerRight={headerRight}
      size="wide"
      dismissable={false}
      // While the discard prompt is up it owns Escape; otherwise Escape asks
      // to close the form.
      onEscape={() => { if (!confirmDiscard) requestClose() }}
    >
      <WorkItemForm
        projectKey={activeProjectKey}
        mode="create"
        members={members ?? []}
        milestones={milestones}
        allowedComplexityValues={project?.allowed_complexity_values}
        projects={lockedProjectKey ? undefined : creatableProjects}
        projectLocked={!!lockedProjectKey}
        onProjectChange={setSelectedProjectKey}
        type={type}
        onTypeChange={setType}
        statuses={statuses}
        staged={staged}
        maxUploadSize={maxUploadSize}
        onDirtyChange={setFormDirty}
        uploadProgress={uploadProgress}
        onSubmit={(values) => {
          createMutation.mutate(values as { type: string; title: string }, {
            onSuccess: async (item) => {
              setUploadingAttachments(true)
              try {
                await uploadStagedFiles(item, (values.description as string) ?? '')
              } finally {
                setUploadingAttachments(false)
              }
              onCreated?.(item.id)
              handleClose()
            },
          })
        }}
        onCancel={requestClose}
        isSubmitting={createMutation.isPending || uploadingAttachments}
        submitError={createMutation.error ? t('workitems.form.submitError', { message: getLocalizedError(createMutation.error, t, 'common.unknown') }) : null}
      />

      <Modal open={confirmDiscard} onClose={() => setConfirmDiscard(false)} title={t('workitems.form.discardTitle')}>
        <p className="text-sm text-gray-600 dark:text-gray-300 mb-4">{t('workitems.form.discardBody')}</p>
        <div className="flex justify-end gap-3">
          <Button variant="secondary" onClick={() => setConfirmDiscard(false)}>{t('workitems.form.discardKeepEditing')}</Button>
          <Button variant="danger" autoFocus onClick={handleClose}>{t('workitems.form.discardConfirm')}</Button>
        </div>
      </Modal>
    </Modal>
  )
}
