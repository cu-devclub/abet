import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { Alert } from '../../../components/ui/Alert'
import { Button } from '../../../components/ui/Button'
import { ConfirmDialog } from '../../../components/ui/ConfirmDialog'
import { Field, TextInput } from '../../../components/ui/Field'
import { Modal } from '../../../components/ui/Modal'
import { useToast } from '../../../components/ui/useToast'
import { errorText } from '../../../app/queryClient'
import { useCreateDepartment, useDeleteDepartment, useUpdateDepartment } from '../api'
import type { Department } from '../types'

interface DepartmentFormModalProps {
  /** `null` = create a new department. */
  department: Department | null
  onClose: () => void
}

export function DepartmentFormModal({ department, onClose }: DepartmentFormModalProps) {
  const isNew = department === null
  const toast = useToast()
  const [code, setCode] = useState(department?.code ?? '')
  const [name, setName] = useState(department?.name ?? '')
  const [error, setError] = useState('')
  const [confirmDelete, setConfirmDelete] = useState(false)
  const create = useCreateDepartment()
  const update = useUpdateDepartment()
  const remove = useDeleteDepartment()

  const save = async () => {
    if (!code.trim() || !name.trim()) {
      setError('Name and Code are required.')
      return
    }
    setError('')
    const input = { code: code.trim().toUpperCase(), name: name.trim() }
    try {
      if (isNew) await create.mutateAsync(input)
      else await update.mutateAsync({ id: department.id, ...input })
      toast.success(isNew ? `Department ${input.code} created.` : 'Department updated.')
      onClose()
    } catch (e) {
      setError(errorText(e))
    }
  }

  const doDelete = async () => {
    if (!department) return
    setConfirmDelete(false)
    try {
      await remove.mutateAsync(department.id)
      toast.info(`Department ${department.code} deleted.`)
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
        title={isNew ? 'Add New Department' : 'Edit Department'}
        footer={
          <>
            {!isNew && (
              <Button variant="dangerGhost" icon={Trash2} onClick={() => setConfirmDelete(true)} className="mr-auto">
                Delete Department
              </Button>
            )}
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={() => void save()} loading={create.isPending || update.isPending}>
              {isNew ? 'Create Department' : 'Save Changes'}
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
          <Field label="Department Code">
            <TextInput data-autofocus value={code} onChange={(e) => setCode(e.target.value)} placeholder="e.g. MCS" />
          </Field>
          <Field label="Department Name">
            <TextInput value={name} onChange={(e) => setName(e.target.value)} placeholder="e.g. Mathematics and Computer Science" />
          </Field>
          <button type="submit" className="hidden" aria-hidden="true" tabIndex={-1} />
        </form>
      </Modal>
      <ConfirmDialog
        isOpen={confirmDelete}
        title="Delete Department"
        confirmLabel="Delete"
        onClose={() => setConfirmDelete(false)}
        onConfirm={() => void doDelete()}
      >
        <p className="text-sm text-gray-600">
          Are you sure you want to delete the department <strong>"{department?.name}"</strong>? This action cannot be undone.
        </p>
      </ConfirmDialog>
    </>
  )
}
