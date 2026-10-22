import { useMemo, useState } from 'react'
import { Alert } from '../../../components/ui/Alert'
import { Button } from '../../../components/ui/Button'
import { NativeSelect } from '../../../components/ui/Field'
import { Modal } from '../../../components/ui/Modal'
import { useToast } from '../../../components/ui/useToast'
import { errorText } from '../../../app/queryClient'
import { displayName, roleLabel } from '../../../lib/format'
import { useUsers } from '../../users/api'
import { accessInputOf, ACCESS_LEVEL_LABELS, type AccessLevel } from '../accessLevel'
import { useSetAccess } from '../api'
import type { AccessEntry, Offering } from '../types'
import { AccessLevelRadios } from './AccessLevelRadios'
import { OfferingBanner } from './OfferingBanner'

interface AssignUserModalProps {
  offering: Offering
  /** Current access list; users with an explicit (non-default) row are disabled. */
  entries: AccessEntry[]
  onClose: () => void
}

export function AssignUserModal({ offering, entries, onClose }: AssignUserModalProps) {
  const toast = useToast()
  const users = useUsers()
  const setAccess = useSetAccess()
  const [userId, setUserId] = useState('')
  const [level, setLevel] = useState<AccessLevel>('both')
  const [error, setError] = useState('')

  const assigned = useMemo(() => new Set(entries.filter((e) => !e.is_default).map((e) => e.user_id)), [entries])
  const assignable = (users.data ?? []).filter((u) => u.role === 'instructor' || u.role === 'staff')
  const selectedUser = assignable.find((u) => u.id === userId)

  const save = async () => {
    if (!selectedUser) return
    setError('')
    try {
      await setAccess.mutateAsync({ offeringId: offering.id, userId: selectedUser.id, ...accessInputOf(level) })
      toast.success(`Assigned ${displayName(selectedUser)} to ${offering.course_code} (${ACCESS_LEVEL_LABELS[level]}).`)
      onClose()
    } catch (e) {
      setError(errorText(e))
    }
  }

  return (
    <Modal
      isOpen
      onClose={onClose}
      title="Assign New User"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button onClick={() => void save()} disabled={!selectedUser} loading={setAccess.isPending}>
            Assign
          </Button>
        </>
      }
    >
      <div className="space-y-4">
        {error && <Alert>{error}</Alert>}
        {users.error && <Alert>{users.error.message}</Alert>}
        <OfferingBanner offering={offering} />
        <div>
          <label htmlFor="assign-user" className="block text-xs font-bold text-gray-700 uppercase tracking-wider mb-1.5">
            Select Instructor or Staff Member
          </label>
          <NativeSelect id="assign-user" value={userId} onChange={(e) => setUserId(e.target.value)} disabled={users.isPending}>
            <option value="">{users.isPending ? 'Loading users…' : 'Select a user to assign...'}</option>
            {assignable.map((u) => (
              <option key={u.id} value={u.id} disabled={assigned.has(u.id)}>
                {displayName(u)} ({u.email}) - {roleLabel(u.role)} {assigned.has(u.id) ? '— [Already Assigned]' : ''}
              </option>
            ))}
          </NativeSelect>
        </div>
        <AccessLevelRadios name="assignAccessLevel" value={level} onChange={setLevel} />
      </div>
    </Modal>
  )
}
