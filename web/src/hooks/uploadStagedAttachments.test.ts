import { describe, it, expect, vi, beforeEach } from 'vitest'
import type { StagedAttachment } from '@/utils/stagedAttachments'

// Mocked so the test never pulls in the axios client (it reads localStorage at
// import time) and so upload progress can be driven deliberately.
vi.mock('@/api/workitems', () => ({
  uploadAttachment: vi.fn(),
  getAttachmentDownloadURL: (_p: string, _n: number, id: string) => `/url/${id}`,
}))

import { uploadAttachment } from '@/api/workitems'
import { uploadStagedAttachments } from './useStagedAttachments'
import type { StagedUploadProgress } from './useStagedAttachments'

const mockUpload = vi.mocked(uploadAttachment)

function staged(id: string, name: string, size: number): StagedAttachment {
  const file = new File(['x'], name, { type: 'text/plain' })
  Object.defineProperty(file, 'size', { value: size })
  return { id, file, inline: false }
}

beforeEach(() => mockUpload.mockReset())

describe('uploadStagedAttachments', () => {
  it('reports progress by bytes across the batch, ending at 100%', async () => {
    // 1 KB then 3 KB: finishing the first file is a quarter of the way.
    mockUpload
      .mockImplementationOnce(async (_p, _n, _f, _c, _ns, onProgress) => {
        onProgress?.(1024)
        return { id: 'a1' } as never
      })
      .mockImplementationOnce(async (_p, _n, _f, _c, _ns, onProgress) => {
        onProgress?.(1536)
        return { id: 'a2' } as never
      })

    const seen: StagedUploadProgress[] = []
    const result = await uploadStagedAttachments(
      [staged('s1', 'small.txt', 1024), staged('s2', 'big.txt', 3072)],
      'PROJ',
      7,
      undefined,
      (p) => seen.push(p),
    )

    expect(result.failed).toEqual([])
    expect(result.urlById).toEqual({ s1: '/url/a1', s2: '/url/a2' })

    const ratios = seen.map((p) => Number(p.ratio.toFixed(3)))
    expect(ratios[0]).toBe(0)
    expect(ratios).toContain(0.25) // first file done
    expect(ratios).toContain(0.625) // 1024 + 1536 of 4096
    expect(ratios[ratios.length - 1]).toBe(1)
    // Ratios never go backwards.
    expect([...ratios].sort((a, b) => a - b)).toEqual(ratios)
  })

  it('names the file being uploaded and its position in the batch', async () => {
    mockUpload.mockResolvedValue({ id: 'x' } as never)

    const seen: StagedUploadProgress[] = []
    await uploadStagedAttachments(
      [staged('s1', 'first.txt', 10), staged('s2', 'second.txt', 10)],
      'PROJ',
      7,
      undefined,
      (p) => seen.push(p),
    )

    expect(seen[0]).toMatchObject({ filename: 'first.txt', fileIndex: 1, totalFiles: 2 })
    expect(seen[seen.length - 1]).toMatchObject({ filename: 'second.txt', fileIndex: 2, totalFiles: 2 })
  })

  it('keeps the bar moving and the batch going when one upload fails', async () => {
    mockUpload
      .mockRejectedValueOnce(new Error('boom'))
      .mockResolvedValueOnce({ id: 'a2' } as never)

    const seen: StagedUploadProgress[] = []
    const result = await uploadStagedAttachments(
      [staged('s1', 'bad.txt', 1024), staged('s2', 'good.txt', 1024)],
      'PROJ',
      7,
      undefined,
      (p) => seen.push(p),
    )

    expect(result.failed).toEqual(['bad.txt'])
    expect(result.urlById).toEqual({ s2: '/url/a2' })
    // The failed file's bytes still count, so the bar reaches the end.
    expect(seen[seen.length - 1].ratio).toBe(1)
  })

  it('does not divide by zero when every staged file is empty', async () => {
    mockUpload.mockResolvedValue({ id: 'x' } as never)

    const seen: StagedUploadProgress[] = []
    await uploadStagedAttachments([staged('s1', 'empty.txt', 0)], 'PROJ', 7, undefined, (p) => seen.push(p))

    expect(seen.every((p) => Number.isFinite(p.ratio))).toBe(true)
  })
})
