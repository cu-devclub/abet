import type { Issue, IssueSeverity } from './types'

/** Errors first, then warnings; original order is kept within each group. */
export function groupIssues(issues: readonly Issue[]): { severity: IssueSeverity; items: { issue: Issue; key: string }[] }[] {
  return (['error', 'warning'] as const)
    .map((severity) => ({
      severity,
      items: issues.flatMap((issue, index) => (issue.severity === severity ? [{ issue, key: `${index}-${issue.code}` }] : [])),
    }))
    .filter((group) => group.items.length > 0)
}
