import type { Components } from 'react-markdown'
import { AuthImage } from './AuthImage'

/**
 * Creates custom react-markdown component overrides.
 * Replaces <img> with AuthImage to handle authenticated attachment URLs.
 * When onImageClick is provided, images become clickable to open a preview modal.
 */
export function getMarkdownComponents(onImageClick?: (src: string) => void): Components {
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
  }
}

export const markdownComponents = getMarkdownComponents()
