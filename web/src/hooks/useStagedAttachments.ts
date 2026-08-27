import { useCallback, useState } from 'react'
import { uploadAttachment, getAttachmentDownloadURL } from '@/api/workitems'
import type { StagedAttachment } from '@/utils/stagedAttachments'

let nextStagedId = 0

export interface StagedAttachmentsState {
  staged: StagedAttachment[]
  /**
   * Stages files, skipping any larger than maxUploadSize. Returns the accepted
   * entries (in order) and the names of the ones that were too large.
   */
  add: (files: Iterable<File>, options?: { inline?: boolean }) => {
    accepted: StagedAttachment[]
    tooLarge: string[]
  }
  remove: (id: string) => void
  clear: () => void
}

export function useStagedAttachments(maxUploadSize?: number): StagedAttachmentsState {
  const [staged, setStaged] = useState<StagedAttachment[]>([])

  const add = useCallback(
    (files: Iterable<File>, options?: { inline?: boolean }) => {
      const accepted: StagedAttachment[] = []
      const tooLarge: string[] = []

      for (const file of files) {
        if (maxUploadSize && file.size > maxUploadSize) {
          tooLarge.push(file.name)
          continue
        }
        accepted.push({ id: `staged-${nextStagedId++}`, file, inline: options?.inline ?? false })
      }

      if (accepted.length > 0) setStaged((prev) => [...prev, ...accepted])
      return { accepted, tooLarge }
    },
    [maxUploadSize],
  )

  const remove = useCallback((id: string) => {
    setStaged((prev) => prev.filter((s) => s.id !== id))
  }, [])

  const clear = useCallback(() => setStaged([]), [])

  return { staged, add, remove, clear }
}

/** Progress of the post-create attachment uploads, for the modal's progress bar. */
export interface StagedUploadProgress {
  /** Share of the total bytes sent so far, 0 to 1. */
  ratio: number
  /** 1-based position of the file currently uploading. */
  fileIndex: number
  totalFiles: number
  filename: string
}

export interface StagedUploadResult {
  /** Download URL per staged id, for the files that uploaded successfully. */
  urlById: Record<string, string>
  /** Names of the files that failed to upload. */
  failed: string[]
}

/**
 * Uploads staged files to a just-created work item. Uploads run one at a time,
 * as they do in the detail-page editor, and a failure is reported rather than
 * thrown so the remaining files still get their chance.
 */
export async function uploadStagedAttachments(
  staged: StagedAttachment[],
  projectKey: string,
  itemNumber: number,
  namespaceSlug?: string,
  onProgress?: (progress: StagedUploadProgress) => void,
): Promise<StagedUploadResult> {
  const urlById: Record<string, string> = {}
  const failed: string[] = []

  // Progress is measured in bytes across the whole batch, so a large file does
  // not advance the bar at the same rate as a small one.
  const totalBytes = staged.reduce((sum, entry) => sum + entry.file.size, 0)
  let sentBytes = 0
  const report = (loaded: number, index: number, filename: string) =>
    onProgress?.({
      ratio: totalBytes > 0 ? Math.min(1, (sentBytes + loaded) / totalBytes) : 0,
      fileIndex: index + 1,
      totalFiles: staged.length,
      filename,
    })

  for (const [index, entry] of staged.entries()) {
    report(0, index, entry.file.name)
    try {
      const attachment = await uploadAttachment(
        projectKey,
        itemNumber,
        entry.file,
        undefined,
        namespaceSlug,
        (loaded) => report(loaded, index, entry.file.name),
      )
      urlById[entry.id] = getAttachmentDownloadURL(projectKey, itemNumber, attachment.id, namespaceSlug)
    } catch {
      failed.push(entry.file.name)
    }
    // Counted whether or not it succeeded, so a failure doesn't stall the bar.
    sentBytes += entry.file.size
    report(0, index, entry.file.name)
  }

  return { urlById, failed }
}
