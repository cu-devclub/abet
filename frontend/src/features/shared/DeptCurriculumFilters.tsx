import { Building2, GraduationCap } from 'lucide-react'
import { SearchableSelect } from '../../components/ui/SearchableSelect'
import type { Curriculum } from '../curricula/types'
import type { Department } from '../departments/types'

interface DeptCurriculumFiltersProps {
  departments: Department[]
  curricula: Curriculum[]
  departmentId: string
  curriculumId: string
  onDepartmentChange: (id: string, keepCurriculum: boolean) => void
  onCurriculumChange: (id: string) => void
}

export function DeptCurriculumFilters({
  departments,
  curricula,
  departmentId,
  curriculumId,
  onDepartmentChange,
  onCurriculumChange,
}: DeptCurriculumFiltersProps) {
  const visibleCurricula = departmentId ? curricula.filter((c) => c.department.id === departmentId) : curricula

  const changeDepartment = (id: string) => {
    const current = curricula.find((c) => c.id === curriculumId)
    onDepartmentChange(id, !id || current?.department.id === id)
  }

  return (
    <>
      <SearchableSelect
        label="Department"
        value={departmentId}
        onChange={changeDepartment}
        options={departments.map((d) => ({ value: d.id, label: d.name, subtext: d.code }))}
        allLabel="All Departments"
        icon={Building2}
      />
      <SearchableSelect
        label="Curriculum"
        value={curriculumId}
        onChange={onCurriculumChange}
        options={visibleCurricula.map((c) => ({ value: c.id, label: c.name, subtext: c.code }))}
        allLabel="All Curriculums"
        icon={GraduationCap}
      />
    </>
  )
}
