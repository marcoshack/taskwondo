/**
 * A file picked in the New Work Item modal. Attachments upload to
 * /items/{itemNumber}/attachments, which needs the item to exist, so files are
 * held until the user clicks Create and uploaded afterwards.
 */
export interface StagedAttachment {
  id: string
  file: File
  /**
   * True when the file was dropped or pasted into the description. The
   * description then carries a `staged:<id>` markdown link that is rewritten to
   * the real attachment URL once the upload completes.
   */
  inline: boolean
}

/** Scheme used by description links that point at a not-yet-uploaded file. */
export const STAGED_URL_SCHEME = 'staged:'

/** Markdown link for a staged file — an image embed for images, a link otherwise. */
export function stagedMarkdownLink(file: File, id: string): string {
  const label = file.name || id
  const prefix = file.type.startsWith('image/') ? '!' : ''
  return `${prefix}[${label}](${STAGED_URL_SCHEME}${id})`
}

function escapeForRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/**
 * Rewrites `staged:<id>` description links to their uploaded attachment URLs.
 * Links whose upload failed are replaced with failedLabel so the description
 * never ships a dangling staged: reference.
 */
export function resolveStagedDescription(
  description: string,
  staged: StagedAttachment[],
  urlById: Record<string, string>,
  failedLabel: (filename: string) => string,
): string {
  let result = description

  for (const entry of staged) {
    if (!entry.inline) continue
    const token = `${STAGED_URL_SCHEME}${entry.id}`
    const url = urlById[entry.id]
    if (url) {
      result = result.split(`(${token})`).join(`(${url})`)
      continue
    }
    // Drop the whole markdown link, label included — a half-rewritten link
    // would render as a broken image.
    result = result.replace(new RegExp(`!?\\[[^\\]]*\\]\\(${escapeForRegExp(token)}\\)`, 'g'), failedLabel(entry.file.name))
  }

  return result
}
