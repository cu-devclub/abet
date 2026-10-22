import { useMemo } from 'react'
import { useMe } from '../auth/api'
import type { Ref } from '../auth/types'
import { useCurricula } from '../curricula/api'
import type { Curriculum } from '../curricula/types'
import { useDepartments } from '../departments/api'
import type { Department } from '../departments/types'
import { scopeOrg } from './scope'

/** Department/curriculum objects found on list rows (e.g. offerings assigned outside the caller's own department). */
export interface RowOrg {
  department: Ref
  curriculum: Ref
}

/**
 * Union of the server lists and the org objects present in `rows`. Instructors/staff only receive their
 * own department/curriculum from the API, but may be assigned to offerings elsewhere; those rows must stay filterable.
 */
export function mergeRowOrg(
  departments: readonly Department[],
  curricula: readonly Curriculum[],
  rows: readonly RowOrg[],
): { departments: Department[]; curricula: Curriculum[] } {
  const depts = new Map(departments.map((d) => [d.id, d]))
  const curs = new Map(curricula.map((c) => [c.id, c]))
  for (const row of rows) {
    if (!depts.has(row.department.id)) {
      depts.set(row.department.id, { ...row.department, curriculum_count: 0, created_at: '', updated_at: '' })
    }
    if (!curs.has(row.curriculum.id)) {
      curs.set(row.curriculum.id, { ...row.curriculum, department: row.department, cover: null, created_at: '', updated_at: '' })
    }
  }
  return { departments: [...depts.values()], curricula: [...curs.values()] }
}

/**
 * Department/curriculum choices for filters and forms, already limited to the caller's scope.
 * Pass `rows` (instructor/staff list pages) to also include the org objects present in those rows.
 */
export function useOrgOptions(rows?: readonly RowOrg[]) {
  const me = useMe()
  const departments = useDepartments()
  const curricula = useCurricula()
  const role = me.data?.user.role
  const scope = role === 'admin' ? me.data?.user.admin_scope ?? null : null
  const mergeRows = role === 'instructor' || role === 'staff'

  const scoped = useMemo(() => {
    const base = scopeOrg(scope, departments.data ?? [], curricula.data ?? [])
    return mergeRows && rows ? mergeRowOrg(base.departments, base.curricula, rows) : base
  }, [scope, departments.data, curricula.data, mergeRows, rows])

  return {
    departments: scoped.departments,
    curricula: scoped.curricula,
    isLoading: departments.isLoading || curricula.isLoading,
  }
}
