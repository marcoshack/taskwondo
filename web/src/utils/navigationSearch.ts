/**
 * Matching for the command palette's static navigation catalog.
 *
 * Entity hits come from the API with its own two-character floor and a debounce;
 * navigation is a small known list held in memory, so it matches from the very
 * first character and answers instantly. An empty query returns everything.
 */

/**
 * Lower-case and strip diacritics, so "Prefer" matches "Préférences" and
 * "senales" matches "Señales".
 */
export function foldForSearch(value: string): string {
  // U+0300–U+036F is the Combining Diacritical Marks block that NFD splits off.
  return value.normalize('NFD').replace(/[\u0300-\u036f]/g, '').toLocaleLowerCase()
}

/**
 * Case- and accent-insensitive substring match on the translated label, from
 * the first character. An empty (or whitespace-only) query returns every item.
 */
export function matchNavigationItems<T extends { label: string }>(
  items: readonly T[],
  query: string,
): T[] {
  const q = foldForSearch(query.trim())
  if (!q) return [...items]
  return items.filter((item) => foldForSearch(item.label).includes(q))
}
