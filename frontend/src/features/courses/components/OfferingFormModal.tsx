import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { Alert } from '../../../components/ui/Alert'
import { Button } from '../../../components/ui/Button'
import { Field, NativeSelect, TextInput } from '../../../components/ui/Field'
import { Modal } from '../../../components/ui/Modal'
import { Tabs } from '../../../components/ui/Tabs'
import { useToast } from '../../../components/ui/useToast'
import { errorText } from '../../../app/queryClient'
import { SEMESTER_OPTIONS } from '../../../lib/format'
import type { Curriculum } from '../../curricula/types'
import type { Department } from '../../departments/types'
import { ADD_COURSE_TABS } from '../addCourseTabs'
import { useCreateOffering, useUpdateOffering } from '../api'
import type { Offering } from '../types'

interface OfferingFormModalProps {
  /** `null` = add a new course. */
  offering: Offering | null
  departments: Department[]
  curricula: Curriculum[]
  /** Years offered when editing; creating always uses `defaultYear`. */
  years: number[]
  defaults: { departmentId: string; curriculumId: string; year: number; semester: number | null }
  onClose: () => void
  onDelete: (offering: Offering) => void
  onSwitchToImport: () => void
}

export function OfferingFormModal({ offering, departments, curricula, years, defaults, onClose, onDelete, onSwitchToImport }: OfferingFormModalProps) {
  const isNew = offering === null
  const toast = useToast()
  const create = useCreateOffering()
  const update = useUpdateOffering()

  const initialCurriculum = curricula.find((c) => c.id === (offering?.curriculum.id ?? defaults.curriculumId))
  const [code, setCode] = useState(offering?.course_code ?? '')
  const [name, setName] = useState(offering?.course_name ?? '')
  const [year, setYear] = useState(offering?.academic_year ?? defaults.year)
  const [semester, setSemester] = useState(offering?.semester ?? defaults.semester ?? 1)
  const [departmentId, setDepartmentId] = useState(
    offering?.department.id ?? (defaults.departmentId || initialCurriculum?.department.id || departments[0]?.id || ''),
  )
  const [curriculumId, setCurriculumId] = useState(initialCurriculum?.id ?? '')
  const [error, setError] = useState('')

  const visibleCurricula = departmentId ? curricula.filter((c) => c.department.id === departmentId) : curricula
  const yearOptions = years.includes(year) ? years : [year, ...years]

  const save = async () => {
    if (!code.trim() || !name.trim()) {
      setError('Course ID and Course Name are required.')
      return
    }
    if (!curriculumId) {
      setError('Please select a curriculum.')
      return
    }
    setError('')
    const input = { course_code: code.trim().toUpperCase(), course_name: name.trim(), curriculum_id: curriculumId, academic_year: year, semester }
    try {
      if (isNew) await create.mutateAsync(input)
      else await update.mutateAsync({ id: offering.id, ...input })
      toast.success(isNew ? `Added course ${input.course_code}.` : `Updated course ${input.course_code}.`)
      onClose()
    } catch (e) {
      setError(errorText(e))
    }
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      title={isNew ? 'Add New Course' : 'Edit Course'}
      footer={
        <div className="flex items-center justify-between w-full">
          {isNew ? <span /> : (
            <Button variant="dangerSoft" icon={Trash2} onClick={() => onDelete(offering)} title="Delete this course">
              Delete Course
            </Button>
          )}
          <div className="flex items-center gap-2">
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={() => void save()} loading={create.isPending || update.isPending}>
              {isNew ? 'Add Course' : 'Save Changes'}
            </Button>
          </div>
        </div>
      }
    >
      <div className="space-y-4">
        {isNew && (
          <Tabs
            label="Creation method"
            tabs={ADD_COURSE_TABS}
            active="single"
            onChange={(tab) => {
              if (tab === 'csv') onSwitchToImport()
            }}
          />
        )}
        {error && <Alert>{error}</Alert>}
        <div className="grid grid-cols-2 gap-4">
          <Field label="Course ID">
            <TextInput data-autofocus placeholder="e.g. 2301170" value={code} onChange={(e) => setCode(e.target.value)} />
          </Field>
          <Field label="Course Name">
            <TextInput placeholder="e.g. Computer Programming" value={name} onChange={(e) => setName(e.target.value)} />
          </Field>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Academic Year">
            {isNew ? (
              <TextInput readOnly disabled value={year} />
            ) : (
              <NativeSelect value={year} onChange={(e) => setYear(Number(e.target.value))}>
                {yearOptions.map((y) => (
                  <option key={y} value={y}>
                    {y}
                  </option>
                ))}
              </NativeSelect>
            )}
          </Field>
          <Field label="Semester">
            <NativeSelect value={semester} onChange={(e) => setSemester(Number(e.target.value))}>
              {SEMESTER_OPTIONS.map((s) => (
                <option key={s.value} value={s.value}>
                  {s.label}
                </option>
              ))}
            </NativeSelect>
          </Field>
        </div>
        <div className="grid grid-cols-2 gap-4">
          <Field label="Department">
            <NativeSelect
              value={departmentId}
              onChange={(e) => {
                setDepartmentId(e.target.value)
                setCurriculumId('')
              }}
            >
              <option value="">Select Department...</option>
              {departments.map((dept) => (
                <option key={dept.id} value={dept.id}>
                  {dept.name}
                </option>
              ))}
            </NativeSelect>
          </Field>
          <Field label="Curriculum">
            <NativeSelect value={curriculumId} onChange={(e) => setCurriculumId(e.target.value)}>
              <option value="">Select Curriculum...</option>
              {visibleCurricula.map((curr) => (
                <option key={curr.id} value={curr.id}>
                  {curr.name}
                </option>
              ))}
            </NativeSelect>
          </Field>
        </div>
      </div>
    </Modal>
  )
}
