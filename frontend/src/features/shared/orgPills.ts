import type { ActiveFilter } from '../../components/ui/FilterToolbar'
import { semesterText } from '../../lib/format'
import type { Curriculum } from '../curricula/types'
import type { Department } from '../departments/types'
import type { OrgFilters } from './useOrgFilters'

/** Active-filter pills for the shared department/curriculum/semester/search filters. */
export function orgPills(
  filters: OrgFilters,
  departments: readonly Department[],
  curricula: readonly Curriculum[],
): ActiveFilter[] {
  const pills: ActiveFilter[] = []
  const department = departments.find((d) => d.id === filters.departmentId)
  const curriculum = curricula.find((c) => c.id === filters.curriculumId)
  if (filters.departmentId) {
    pills.push({
      key: 'dept',
      label: `Dept: ${department?.name ?? filters.departmentId}`,
      onClear: () => filters.setDepartment('', false),
    })
  }
  if (filters.curriculumId) {
    pills.push({
      key: 'cur',
      label: `Curriculum: ${curriculum?.name ?? filters.curriculumId}`,
      onClear: () => filters.setCurriculum(''),
    })
  }
  if (filters.semesters.length > 0) {
    pills.push({
      key: 'sem',
      label: `Semesters: ${filters.semesters.map(semesterText).join(', ')}`,
      onClear: () => filters.setSemesters([]),
    })
  }
  if (filters.search) {
    pills.push({
      key: 'q',
      label: `Search: "${filters.search}"`,
      neutral: true,
      onClear: () => filters.setSearch(''),
    })
  }
  return pills
}
