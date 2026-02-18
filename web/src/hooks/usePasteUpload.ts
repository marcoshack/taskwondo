import { useCallback } from 'react'
import { uploadAttachment, getAttachmentDownloadURL } from '@/api/workitems'
import { useQueryClient } from '@tanstack/react-query'

interface UsePasteUploadOptions {
  projectKey: string
  itemNumber: number
  onTextChange: (updater: (prev: string) => string) => void
}

export function usePasteUpload({ projectKey, itemNumber, onTextChange }: UsePasteUploadOptions) {
  const qc = useQueryClient()

  const handleFile = useCallback(
    async (file: File) => {
      const filename = file.name || `pasted-${Date.now()}.${file.type.split('/')[1] || 'png'}`
      const placeholder = `![Uploading ${filename}...]()`

      onTextChange((prev) => prev + placeholder)

      try {
        const attachment = await uploadAttachment(projectKey, itemNumber, file)
        const downloadURL = getAttachmentDownloadURL(projectKey, itemNumber, attachment.id)
        const isImage = file.type.startsWith('image/')
        const markdownLink = isImage
          ? `![${filename}](${downloadURL})`
          : `[${filename}](${downloadURL})`

        onTextChange((prev) => prev.replace(placeholder, markdownLink))
        qc.invalidateQueries({ queryKey: ['projects', projectKey, 'items', itemNumber, 'attachments'] })
      } catch {
        onTextChange((prev) => prev.replace(placeholder, `[Upload failed: ${filename}]`))
      }
    },
    [projectKey, itemNumber, onTextChange, qc],
  )

  const handlePaste = useCallback(
    async (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
      const items = e.clipboardData?.items
      if (!items) return

      for (const item of items) {
        if (item.kind === 'file') {
          e.preventDefault()
          const file = item.getAsFile()
          if (!file) continue
          await handleFile(file)
          break
        }
      }
    },
    [handleFile],
  )

  const handleDrop = useCallback(
    async (e: React.DragEvent<HTMLTextAreaElement>) => {
      e.preventDefault()
      e.stopPropagation()
      const files = e.dataTransfer?.files
      if (!files?.length) return

      for (const file of files) {
        await handleFile(file)
      }
    },
    [handleFile],
  )

  const handleDragOver = useCallback(
    (e: React.DragEvent<HTMLTextAreaElement>) => {
      e.preventDefault()
      e.dataTransfer.dropEffect = 'copy'
    },
    [],
  )

  return { handlePaste, handleDrop, handleDragOver }
}
