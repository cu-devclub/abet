import type { AdminScope } from '../auth/types'
import type { Curriculum } from '../curricula/types'
import type { Department } from '../departments/types'

/**
 * Limit department/curriculum choices to an admin's scope (authz.md). `scope === null`
 * means "no restriction" (super admin, instructor, staff — the server filters rows).
 */
export function scopeOrg(
  scope: AdminScope | null,
  departments: readonly Department[],
  curricula: readonly Curriculum[],
): { departments: Department[]; curricula: Curriculum[] } {
  if (!scope) return { departments: [...departments], curricula: [...curricula] }
  const scopeDepartmentIds = new Set(scope.departments.map((d) => d.id))
  const scopeCurriculumIds = new Set(scope.curricula.map((c) => c.id))
  const allowedCurricula = curricula.filter((c) => scopeCurriculumIds.has(c.id) || scopeDepartmentIds.has(c.department.id))
  const allowedDepartmentIds = new Set([...scopeDepartmentIds, ...allowedCurricula.map((c) => c.department.id)])
  return {
    departments: departments.filter((d) => allowedDepartmentIds.has(d.id)),
    curricula: allowedCurricula,
  }
}
