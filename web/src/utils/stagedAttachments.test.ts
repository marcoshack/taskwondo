import { describe, it, expect } from 'vitest'
import { resolveStagedDescription, stagedMarkdownLink, STAGED_URL_SCHEME } from './stagedAttachments'
import type { StagedAttachment } from './stagedAttachments'

function staged(id: string, name: string, type: string, inline = true): StagedAttachment {
  return { id, inline, file: new File(['x'], name, { type }) }
}

describe('stagedMarkdownLink', () => {
  it('embeds images and links everything else', () => {
    expect(stagedMarkdownLink(new File([''], 'shot.png', { type: 'image/png' }), 'a')).toBe(
      `![shot.png](${STAGED_URL_SCHEME}a)`,
    )
    expect(stagedMarkdownLink(new File([''], 'trace.har', { type: 'application/json' }), 'b')).toBe(
      `[trace.har](${STAGED_URL_SCHEME}b)`,
    )
  })
})

describe('resolveStagedDescription', () => {
  const failed = (filename: string) => `[Upload failed: ${filename}]`

  it('rewrites placeholders to the uploaded attachment URLs', () => {
    const files = [staged('a', 'shot.png', 'image/png'), staged('b', 'trace.har', 'application/json')]
    const description = `See ![shot.png](${STAGED_URL_SCHEME}a) and [trace.har](${STAGED_URL_SCHEME}b).`

    const result = resolveStagedDescription(description, files, { a: '/api/v1/x/1', b: '/api/v1/x/2' }, failed)

    expect(result).toBe('See ![shot.png](/api/v1/x/1) and [trace.har](/api/v1/x/2).')
  })

  it('rewrites every occurrence of the same placeholder', () => {
    const files = [staged('a', 'shot.png', 'image/png')]
    const description = `![shot.png](${STAGED_URL_SCHEME}a) ![shot.png](${STAGED_URL_SCHEME}a)`

    expect(resolveStagedDescription(description, files, { a: '/u' }, failed)).toBe('![shot.png](/u) ![shot.png](/u)')
  })

  it('replaces the whole link when the upload failed, leaving no staged reference', () => {
    const files = [staged('a', 'shot.png', 'image/png')]
    const description = `before ![shot.png](${STAGED_URL_SCHEME}a) after`

    const result = resolveStagedDescription(description, files, {}, failed)

    expect(result).toBe('before [Upload failed: shot.png] after')
    expect(result).not.toContain(STAGED_URL_SCHEME)
  })

  it('ignores files that were never referenced from the description', () => {
    const files = [staged('a', 'notes.txt', 'text/plain', false)]

    expect(resolveStagedDescription('untouched', files, { a: '/u' }, failed)).toBe('untouched')
  })

  it('does not treat filename regex characters as patterns', () => {
    const files = [staged('a+b', 'weird(name).png', 'image/png')]
    const description = `![weird(name).png](${STAGED_URL_SCHEME}a+b)`

    expect(resolveStagedDescription(description, files, {}, failed)).toBe('[Upload failed: weird(name).png]')
  })
})
