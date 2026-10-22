import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { Alert } from '../../../components/ui/Alert'
import { Button } from '../../../components/ui/Button'
import { ConfirmDialog } from '../../../components/ui/ConfirmDialog'
import { Field, NativeSelect, TextInput } from '../../../components/ui/Field'
import { Modal } from '../../../components/ui/Modal'
import { useToast } from '../../../components/ui/useToast'
import { errorText } from '../../../app/queryClient'
import type { Department } from '../../departments/types'
import { useCreateCurriculum, useDeleteCurriculum, useUpdateCurriculum } from '../api'
import type { Curriculum } from '../types'

interface CurriculumFormModalProps {
  /** `null` = create a new curriculum. */
  curriculum: Curriculum | null
  departments: Department[]
  onClose: () => void
}

export function CurriculumFormModal({ curriculum, departments, onClose }: CurriculumFormModalProps) {
  const isNew = curriculum === null
  const toast = useToast()
  const [code, setCode] = useState(curriculum?.code ?? '')
  const [name, setName] = useState(curriculum?.name ?? '')
  const [departmentId, setDepartmentId] = useState(curriculum?.department.id ?? departments[0]?.id ?? '')
  const [error, setError] = useState('')
  const [confirmDelete, setConfirmDelete] = useState(false)
  const create = useCreateCurriculum()
  const update = useUpdateCurriculum()
  const remove = useDeleteCurriculum()

  const save = async () => {
    if (!code.trim() || !name.trim() || !departmentId) {
      setError('Name, Code, and Department are required.')
      return
    }
    setError('')
    const input = { code: code.trim().toUpperCase(), name: name.trim(), department_id: departmentId }
    try {
      if (isNew) await create.mutateAsync(input)
      else await update.mutateAsync({ id: curriculum.id, ...input })
      toast.success(isNew ? `Curriculum ${input.code} created.` : 'Curriculum updated.')
      onClose()
    } catch (e) {
      setError(errorText(e))
    }
  }

  const doDelete = async () => {
    if (!curriculum) return
    setConfirmDelete(false)
    try {
      await remove.mutateAsync(curriculum.id)
      toast.info(`Curriculum ${curriculum.code} deleted.`)
      onClose()
    } catch (e) {
      setError(errorText(e))
    }
  }

  return (
    <>
      <Modal
        isOpen
        onClose={onClose}
        title={isNew ? 'Add New Curriculum' : 'Edit Curriculum'}
        footer={
          <>
            {!isNew && (
              <Button variant="dangerGhost" icon={Trash2} onClick={() => setConfirmDelete(true)} className="mr-auto">
                Delete Curriculum
              </Button>
            )}
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={() => void save()} loading={create.isPending || update.isPending}>
              {isNew ? 'Create Curriculum' : 'Save Changes'}
            </Button>
          </>
        }
      >
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            void save()
          }}
        >
          {error && <Alert>{error}</Alert>}
          <Field label="Curriculum Code">
            <TextInput data-autofocus value={code} onChange={(e) => setCode(e.target.value)} placeholder="e.g. CS" />
          </Field>
          <Field label="Curriculum Name">
            <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Computer Science" />
          </Field>
          <Field label="Department">
            <NativeSelect value={departmentId} onChange={(e) => setDepartmentId(e.target.value)}>
              <option value="" disabled>
                Select Department...
              </option>
              {departments.map((dept) => (
                <option key={dept.id} value={dept.id}>
                  {dept.name}
                </option>
              ))}
            </NativeSelect>
          </Field>
          <button type="submit" className="hidden" aria-hidden="true" tabIndex={-1} />
        </form>
      </Modal>
      <ConfirmDialog
        isOpen={confirmDelete}
        title="Delete Curriculum"
        confirmLabel="Delete"
        onClose={() => setConfirmDelete(false)}
        onConfirm={() => void doDelete()}
      >
        <p className="text-sm text-gray-600">
          Are you sure you want to delete the curriculum <strong>"{curriculum?.name}"</strong>? This action cannot be undone.
        </p>
      </ConfirmDialog>
    </>
  )
}
