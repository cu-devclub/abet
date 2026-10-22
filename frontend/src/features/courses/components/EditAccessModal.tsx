import { useState } from 'react'
import { Trash2 } from 'lucide-react'
import { Alert } from '../../../components/ui/Alert'
import { Button } from '../../../components/ui/Button'
import { Modal } from '../../../components/ui/Modal'
import { useToast } from '../../../components/ui/useToast'
import { errorText } from '../../../app/queryClient'
import { displayName, roleLabel } from '../../../lib/format'
import { accessInputOf, accessLevelOf, ACCESS_LEVEL_LABELS, type AccessLevel } from '../accessLevel'
import { useRemoveAccess, useSetAccess } from '../api'
import type { AccessEntry, Offering } from '../types'
import { AccessLevelRadios } from './AccessLevelRadios'
import { OfferingBanner } from './OfferingBanner'

interface EditAccessModalProps {
  offering: Offering
  entry: AccessEntry
  onClose: () => void
}

export function EditAccessModal({ offering, entry, onClose }: EditAccessModalProps) {
  const toast = useToast()
  const setAccess = useSetAccess()
  const removeAccess = useRemoveAccess()
  const [level, setLevel] = useState<AccessLevel>(accessLevelOf(entry))
  const [error, setError] = useState('')

  const save = async () => {
    setError('')
    try {
      await setAccess.mutateAsync({ offeringId: offering.id, userId: entry.user_id, ...accessInputOf(level) })
      toast.success(`Updated access level for ${displayName(entry)} to "${ACCESS_LEVEL_LABELS[level]}".`)
      onClose()
    } catch (e) {
      setError(errorText(e))
    }
  }

  const remove = () =>
    removeAccess.mutate(
      { offeringId: offering.id, userId: entry.user_id },
      {
        onSuccess: () => {
          toast.info(`Removed ${displayName(entry)}'s access from ${offering.course_code}.`)
          onClose()
        },
      },
    )

  return (
    <Modal
      isOpen
      onClose={onClose}
      title="Edit Course Access"
      footer={
        <div className="flex items-center justify-between w-full">
          <Button variant="dangerSoft" icon={Trash2} loading={removeAccess.isPending} onClick={remove} title="Remove user access from this course">
            Remove Access
          </Button>
          <div className="flex items-center gap-2">
            <Button variant="secondary" onClick={onClose}>
              Cancel
            </Button>
            <Button onClick={() => void save()} loading={setAccess.isPending}>
              Save Changes
            </Button>
          </div>
        </div>
      }
    >
      <div className="space-y-4">
        {error && <Alert>{error}</Alert>}
        <div className="bg-slate-50 border border-slate-200 rounded-xl p-3.5 flex items-center gap-3">
          <div className="h-9 w-9 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center font-bold text-sm shrink-0">
            {displayName(entry).charAt(0).toUpperCase()}
          </div>
          <div className="min-w-0 flex-1">
            <h4 className="text-sm font-bold text-gray-900 truncate">{displayName(entry)}</h4>
            <p className="text-xs text-gray-500 font-mono truncate">{entry.email}</p>
          </div>
          <span className="text-xs font-semibold px-2 py-0.5 rounded-md bg-white border border-gray-200 text-gray-700 shadow-2xs capitalize">
            {roleLabel(entry.role)}
          </span>
        </div>
        <OfferingBanner offering={offering} />
        <AccessLevelRadios name="editAccessLevel" value={level} onChange={setLevel} />
      </div>
    </Modal>
  )
}
