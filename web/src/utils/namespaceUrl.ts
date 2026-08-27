/**
 * Namespace slug ⇄ URL segment mapping.
 *
 * Lives in `utils/` rather than in `useNamespacePath` so that modules which
 * must stay free of `@/api/client` (it touches `localStorage` at import time,
 * and the Vitest suite runs without a DOM) can build namespace URLs too.
 * `useNamespacePath` re-exports both helpers, so existing imports still work.
 */

/** Map namespace slug to URL segment: 'default' → 'd', anything else unchanged */
export function toUrlSegment(slug: string): string {
  return slug === 'default' ? 'd' : slug
}

/** Map URL segment back to namespace slug: 'd' → 'default', anything else unchanged */
export function fromUrlSegment(segment: string): string {
  return segment === 'd' ? 'default' : segment
}
