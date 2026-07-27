/**
 * Top-level project sections that exist in every project, so they can be
 * carried over when the user switches projects (TF-412).
 *
 * The project overview lives at the project root and needs no entry here.
 * `teams/:teamId` and `support` are deliberately absent: team IDs are
 * project-scoped, and the support view is only reachable for projects where
 * the user is a customer.
 */
const SWITCHABLE_SECTIONS = ['items', 'queues', 'milestones', 'workflows', 'settings']

/**
 * Given the splat portion of `/:namespace/projects/:projectKey/*`, return the
 * path suffix (with leading slash, or empty for the overview) to open in the
 * target project when switching projects.
 *
 * Detail pages collapse to their list — item numbers, queue IDs and milestone
 * IDs are project-scoped, so they mean nothing in the target project.
 */
export function projectSwitchSuffix(rest: string | undefined | null): string {
  const section = (rest ?? '').replace(/^\//, '').split('/')[0]
  return SWITCHABLE_SECTIONS.includes(section) ? `/${section}` : ''
}
