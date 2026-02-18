import type { Components } from 'react-markdown'
import { AuthImage } from './AuthImage'

/**
 * Custom react-markdown component overrides.
 * Replaces <img> with AuthImage to handle authenticated attachment URLs.
 */
export const markdownComponents: Components = {
  img: ({ src, alt, ...props }) => <AuthImage src={src} alt={alt} {...props} />,
}
