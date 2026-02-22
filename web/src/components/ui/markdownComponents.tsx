import type { Components } from 'react-markdown'
import { AuthImage } from './AuthImage'
import { getToken } from '@/api/client'

interface MarkdownComponentOptions {
  onImageClick?: (src: string) => void
  onAttachmentLinkClick?: (href: string, attachmentId: string) => void
}

/** Extract attachment ID from URLs like /api/v1/projects/X/items/N/attachments/{id} */
function extractAttachmentId(href: string): string | null {
  const parts = href.split('/')
  const idx = parts.indexOf('attachments')
  return idx >= 0 ? parts[idx + 1] ?? null : null
}

function authDownload(href: string, filename: string) {
  const token = getToken()
  fetch(href, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
    .then((res) => {
      if (!res.ok) throw new Error(`HTTP ${res.status}`)
      return res.blob()
    })
    .then((blob) => {
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
    })
    .catch(() => {
      window.open(href, '_blank')
    })
}

/**
 * Creates custom react-markdown component overrides.
 * Replaces <img> with AuthImage to handle authenticated attachment URLs.
 * Replaces <a> pointing at /api/v1/ with an auth-aware download/preview handler.
 * When onImageClick is provided, images become clickable to open a preview modal.
 * When onAttachmentLinkClick is provided, attachment links open preview for supported types.
 */
export function getMarkdownComponents(opts?: MarkdownComponentOptions | ((src: string) => void)): Components {
  // Support legacy signature: getMarkdownComponents(onImageClick?)
  const options: MarkdownComponentOptions = typeof opts === 'function' ? { onImageClick: opts } : (opts ?? {})
  const { onImageClick, onAttachmentLinkClick } = options

  return {
    img: ({ src, alt, ...props }) => {
      const img = <AuthImage src={src} alt={alt} {...props} />
      if (onImageClick && src) {
        return (
          <button
            type="button"
            onClick={(e) => { e.stopPropagation(); onImageClick(src) }}
            className="cursor-zoom-in inline"
          >
            {img}
          </button>
        )
      }
      return img
    },
    a: ({ href, children, ...props }) => {
      if (href?.startsWith('/api/v1/')) {
        return (
          <a
            {...props}
            href={href}
            onClick={(e) => {
              e.preventDefault()
              const attachmentId = extractAttachmentId(href)
              if (attachmentId && onAttachmentLinkClick) {
                onAttachmentLinkClick(href, attachmentId)
              } else {
                const text = typeof children === 'string' ? children : ''
                authDownload(href, text || href.split('/').pop() || 'download')
              }
            }}
          >
            {children}
          </a>
        )
      }
      return <a href={href} {...props}>{children}</a>
    },
  }
}

export const markdownComponents = getMarkdownComponents()
