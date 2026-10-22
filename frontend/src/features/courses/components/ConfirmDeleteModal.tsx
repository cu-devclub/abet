import { AlertCircle } from 'lucide-react'
import { ConfirmDialog } from '../../../components/ui/ConfirmDialog'
import { useToast } from '../../../components/ui/useToast'
import { useBulkDeleteOfferings, useDeleteOffering } from '../api'
import type { Offering } from '../types'

interface ConfirmDeleteModalProps {
  offerings: Offering[]
  onClose: () => void
  onDeleted: (ids: string[]) => void
}

export function ConfirmDeleteModal({ offerings, onClose, onDeleted }: ConfirmDeleteModalProps) {
  const toast = useToast()
  const removeOne = useDeleteOffering()
  const removeMany = useBulkDeleteOfferings()
  const withSubmissions = offerings.filter((o) => o.has_submissions)
  const single = offerings.length === 1 ? offerings[0] : null

  const confirm = () => {
    const ids = offerings.map((o) => o.id)
    const done = (message: string) => {
      toast.info(message)
      onDeleted(ids)
      onClose()
    }
    if (single) removeOne.mutate(single.id, { onSuccess: () => done(`Deleted course ${single.course_code} - ${single.course_name}.`) })
    else removeMany.mutate(ids, { onSuccess: (res) => done(`Deleted ${res.deleted} course(s).`) })
  }

  return (
    <ConfirmDialog
      isOpen
      title="Confirm Delete Course"
      confirmLabel="Delete"
      loading={removeOne.isPending || removeMany.isPending}
      onClose={onClose}
      onConfirm={confirm}
    >
      <div className="space-y-4">
        {withSubmissions.length > 0 && (
          <div className="p-4 bg-red-50 text-red-800 border border-red-200 rounded-xl space-y-3">
            <div className="flex items-start">
              <AlertCircle className="w-5 h-5 text-red-600 mr-2 shrink-0 mt-0.5" />
              <div>
                <p className="font-semibold text-sm">Warning: Uploaded Files Affected</p>
                <p className="text-xs text-red-700 mt-1">
                  The following course(s) have uploaded portfolios. Deleting them will permanently delete their submissions and generated files:
                </p>
              </div>
            </div>
            <ul className="list-disc list-inside text-xs font-semibold pl-1 space-y-1">
              {withSubmissions.map((o) => (
                <li key={o.id}>
                  {o.course_code} - {o.course_name}
                </li>
              ))}
            </ul>
          </div>
        )}
        <p className="text-sm text-gray-600">
          {single
            ? `Are you sure you want to delete course "${single.course_code} - ${single.course_name}"? This action cannot be undone.`
            : `Are you sure you want to delete the ${offerings.length} selected course(s)? This action cannot be undone.`}
        </p>
      </div>
    </ConfirmDialog>
  )
}
