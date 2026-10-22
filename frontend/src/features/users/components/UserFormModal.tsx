import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { Alert } from '../../../components/ui/Alert'
import { Button } from '../../../components/ui/Button'
import { ConfirmDialog } from '../../../components/ui/ConfirmDialog'
import { Field, NativeSelect, SmallLabel, TextInput } from '../../../components/ui/Field'
import { Modal } from '../../../components/ui/Modal'
import { useToast } from '../../../components/ui/useToast'
import { errorText } from '../../../app/queryClient'
import { displayName } from '../../../lib/format'
import type { Curriculum } from '../../curricula/types'
import type { Department } from '../../departments/types'
import type { Me, Role, User } from '../../auth/types'
import { useCreateUser, useDeleteUser, useUpdateUser } from '../api'
import type { AdminScopeInput } from '../types'

const ALL_ROLES: { value: Role; label: string }[] = [
  { value: 'super_admin', label: 'Super Admin' },
  { value: 'admin', label: 'Admin' },
  { value: 'instructor', label: 'Instructor' },
  { value: 'staff', label: 'Staff' },
]

interface UserFormModalProps {
  /** `null` = create a new user. */
  user: User | null
  me: Me
  departments: Department[]
  curricula: Curriculum[]
  onClose: () => void
}

function CheckList({
  legend,
  items,
  selected,
  onToggle,
}: {
  legend: string
  items: { id: string; label: string }[]
  selected: string[]
  onToggle: (id: string) => void
}) {
  return (
    <fieldset className="border border-gray-200 rounded-lg p-2.5">
      <legend className="px-1 text-xs font-semibold text-gray-600">{legend}</legend>
      <div className="max-h-32 overflow-y-auto space-y-1">
        {items.map((item) => (
          <label key={item.id} className="flex items-center gap-2 text-xs text-gray-700 cursor-pointer">
            <input
              type="checkbox"
              checked={selected.includes(item.id)}
              onChange={() => onToggle(item.id)}
              className="h-3.5 w-3.5 text-indigo-600 rounded border-gray-300 focus:ring-indigo-500 cursor-pointer"
            />
            {item.label}
          </label>
        ))}
        {items.length === 0 && <p className="text-xs text-gray-400 italic">Nothing to select.</p>}
      </div>
    </fieldset>
  )
}

function toggled(list: string[], id: string): string[] {
  return list.includes(id) ? list.filter((x) => x !== id) : [...list, id]
}

export function UserFormModal({ user, me, departments, curricula, onClose }: UserFormModalProps) {
  const isNew = user === null
  const viewerIsSuper = me.user.role === 'super_admin'
  const isSelf = user?.id === me.user.id
  const toast = useToast()
  const create = useCreateUser()
  const update = useUpdateUser()
  const remove = useDeleteUser()

  const [email, setEmail] = useState(user?.email ?? '')
  const [role, setRole] = useState<Role>(user?.role ?? (viewerIsSuper ? 'staff' : 'instructor'))
  const [departmentId, setDepartmentId] = useState(user?.department?.id ?? (isNew ? (departments[0]?.id ?? '') : ''))
  const [curriculumId, setCurriculumId] = useState(user?.curriculum?.id ?? '')
  const [scopeDepartments, setScopeDepartments] = useState<string[]>(user?.admin_scope?.departments.map((d) => d.id) ?? [])
  const [scopeCurricula, setScopeCurricula] = useState<string[]>(user?.admin_scope?.curricula.map((c) => c.id) ?? [])
  const [error, setError] = useState('')
  const [confirmDelete, setConfirmDelete] = useState(false)

  const roleOptions = viewerIsSuper ? ALL_ROLES : ALL_ROLES.filter((r) => r.value === 'instructor' || r.value === 'staff')
  const visibleCurricula = departmentId ? curricula.filter((c) => c.department.id === departmentId) : curricula
  const showScope = viewerIsSuper && role === 'admin'

  const changeCurriculum = (id: string) => {
    setCurriculumId(id)
    const picked = curricula.find((c) => c.id === id)
    if (picked && !departmentId) setDepartmentId(picked.department.id)
  }

  const save = async () => {
    if (isNew && !email.trim()) {
      setError('Email is required.')
      return
    }
    setError('')
    const adminScope: AdminScopeInput | null = showScope
      ? { department_ids: scopeDepartments, curriculum_ids: scopeCurricula }
      : null
    const common = {
      role,
      department_id: departmentId || null,
      curriculum_id: curriculumId || null,
    }
    try {
      if (isNew) {
        await create.mutateAsync({ email: email.trim(), ...common, admin_scope: adminScope })
        toast.success(`User ${email.trim().toLowerCase()} created.`)
      } else {
        await update.mutateAsync({ id: user.id, ...common, ...(viewerIsSuper ? { admin_scope: adminScope } : {}) })
        toast.success(`User ${displayName(user)} updated.`)
      }
      onClose()
    } catch (e) {
      setError(errorText(e))
    }
  }

  const doDelete = async () => {
    if (!user) return
    setConfirmDelete(false)
    try {
      await remove.mutateAsync(user.id)
      toast.info(`User ${displayName(user)} deleted.`)
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
        title={isNew ? 'Add New User' : `Edit User: ${displayName(user)}`}
        footer={
          <>
            {!isNew && !isSelf && (
              <Button variant="dangerGhost" icon={Trash2} onClick={() => setConfirmDelete(true)} className="mr-auto">
                Delete User
              </Button>
            )}
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={() => void save()} loading={create.isPending || update.isPending}>
              {isNew ? 'Create User' : 'Save Changes'}
            </Button>
          </>
        }
      >
        <div className="space-y-5">
          {error && <Alert>{error}</Alert>}
          <div className="grid grid-cols-2 gap-4">
            <Field label="Name" hint={isNew ? 'Name is fetched from Google at first sign-in.' : undefined}>
              <TextInput disabled value={user?.name ?? ''} placeholder={isNew ? 'Auto-synced from Google' : ''} />
            </Field>
            <Field label="Email">
              <TextInput
                type="email"
                data-autofocus={isNew || undefined}
                disabled={!isNew}
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                placeholder="e.g. user@chula.ac.th"
              />
            </Field>
          </div>
          {!isNew && <p className="text-xs text-gray-400 -mt-3">Both Name and Email are automatically synced with the Google account.</p>}

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
              <NativeSelect value={curriculumId} onChange={(e) => changeCurriculum(e.target.value)}>
                <option value="">Select Curriculum...</option>
                {visibleCurricula.map((curr) => (
                  <option key={curr.id} value={curr.id}>
                    {curr.name}
                  </option>
                ))}
              </NativeSelect>
            </Field>
          </div>

          <Field
            label="Role"
            hint={
              !viewerIsSuper
                ? 'As an Admin, you can only set or edit roles below Admin (Instructor or Staff).'
                : isSelf
                  ? 'You cannot change your own role.'
                  : undefined
            }
          >
            <NativeSelect value={role} onChange={(e) => setRole(e.target.value as Role)} disabled={isSelf}>
              {roleOptions.map((r) => (
                <option key={r.value} value={r.value}>
                  {r.label}
                </option>
              ))}
            </NativeSelect>
          </Field>

          {showScope && (
            <div className="space-y-2">
              <SmallLabel>Admin scope</SmallLabel>
              <div className="grid grid-cols-2 gap-3">
                <CheckList
                  legend="Departments"
                  items={departments.map((d) => ({ id: d.id, label: `${d.code} — ${d.name}` }))}
                  selected={scopeDepartments}
                  onToggle={(id) => setScopeDepartments((prev) => toggled(prev, id))}
                />
                <CheckList
                  legend="Curriculums"
                  items={curricula.map((c) => ({ id: c.id, label: `${c.code} — ${c.name}` }))}
                  selected={scopeCurricula}
                  onToggle={(id) => setScopeCurricula((prev) => toggled(prev, id))}
                />
              </div>
              <p className="text-xs text-gray-400">An admin manages every curriculum of a selected department plus any individually selected curriculum.</p>
            </div>
          )}
        </div>
      </Modal>
      <ConfirmDialog
        isOpen={confirmDelete}
        title="Delete User"
        confirmLabel="Delete"
        onClose={() => setConfirmDelete(false)}
        onConfirm={() => void doDelete()}
      >
        <p className="text-sm text-gray-600">
          Are you sure you want to delete <strong>{user ? displayName(user) : ''}</strong> ({user?.email})? This action cannot be undone.
        </p>
      </ConfirmDialog>
    </>
  )
}
