import type { AccessAction, CourseAction, ImportPlan, ImportSummary, UserAction } from './types'

/** The 422 response of `/offerings/import/*` carries the ImportPlan in the error body. */
export function isImportPlan(value: unknown): value is ImportPlan {
  if (!value || typeof value !== 'object') return false
  const plan = value as Partial<ImportPlan>
  return typeof plan.summary === 'object' && plan.summary !== null && Array.isArray(plan.errors) && Array.isArray(plan.courses)
}

export const COURSE_ACTION_LABELS: Record<CourseAction, string> = {
  create: 'Create',
  update: 'Update name',
  keep: 'Keep',
  delete: 'Delete',
  keep_has_uploads: 'Keep (has uploads)',
}

export const ACCESS_ACTION_LABELS: Record<AccessAction, string> = {
  add: 'Add',
  update: 'Update',
  remove: 'Remove',
}

export const USER_ACTION_LABELS: Record<UserAction, string> = {
  create: 'Create user',
  raise_role: 'Raise role to instructor',
}

export interface SummaryItem {
  key: keyof ImportSummary
  label: string
  value: number
  tone: 'green' | 'blue' | 'gray' | 'red' | 'amber'
}

/** Non-zero summary counters in a stable display order. */
export function summaryItems(summary: ImportSummary): SummaryItem[] {
  const items: SummaryItem[] = [
    { key: 'rows', label: 'CSV rows', value: summary.rows, tone: 'gray' },
    { key: 'courses_create', label: 'Courses to create', value: summary.courses_create, tone: 'green' },
    { key: 'courses_update', label: 'Courses renamed', value: summary.courses_update, tone: 'blue' },
    { key: 'courses_keep', label: 'Courses kept', value: summary.courses_keep, tone: 'gray' },
    { key: 'courses_delete', label: 'Courses to delete', value: summary.courses_delete, tone: 'red' },
    { key: 'courses_kept_with_uploads', label: 'Kept (has uploads)', value: summary.courses_kept_with_uploads, tone: 'amber' },
    { key: 'access_add', label: 'Access to add', value: summary.access_add, tone: 'green' },
    { key: 'access_update', label: 'Access to update', value: summary.access_update, tone: 'blue' },
    { key: 'access_remove', label: 'Access to remove', value: summary.access_remove, tone: 'red' },
    { key: 'users_create', label: 'Users to create', value: summary.users_create, tone: 'green' },
    { key: 'users_role_raised', label: 'Roles raised', value: summary.users_role_raised, tone: 'blue' },
  ]
  return items.filter((item) => item.key === 'rows' || item.value > 0)
}
