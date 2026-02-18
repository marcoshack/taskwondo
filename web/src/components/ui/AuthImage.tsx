import { useState, useEffect } from 'react'
import { getToken } from '@/api/client'

interface AuthImageProps extends React.ImgHTMLAttributes<HTMLImageElement> {
  src?: string
}

/**
 * Image component that fetches images with the JWT Authorization header.
 * Used for attachment images rendered inside markdown, where browser <img> tags
 * cannot include auth headers on their own.
 */
export function AuthImage({ src, alt, ...props }: AuthImageProps) {
  const [blobUrl, setBlobUrl] = useState<string | null>(null)
  const [error, setError] = useState(false)

  const isAuthUrl = src?.startsWith('/api/v1/')

  useEffect(() => {
    if (!src || !isAuthUrl) return

    let revoked = false
    const token = getToken()

    fetch(src, {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    })
      .then((res) => {
        if (!res.ok) throw new Error(`HTTP ${res.status}`)
        return res.blob()
      })
      .then((blob) => {
        if (revoked) return
        setBlobUrl(URL.createObjectURL(blob))
      })
      .catch(() => {
        if (!revoked) setError(true)
      })

    return () => {
      revoked = true
      setBlobUrl((prev) => {
        if (prev) URL.revokeObjectURL(prev)
        return null
      })
    }
  }, [src, isAuthUrl])

  if (!isAuthUrl) {
    return <img src={src} alt={alt} {...props} />
  }

  if (error) {
    return <span className="text-xs text-red-500">[Image failed to load]</span>
  }

  if (!blobUrl) {
    return <span className="text-xs text-gray-400">Loading image...</span>
  }

  return <img src={blobUrl} alt={alt} {...props} />
}
