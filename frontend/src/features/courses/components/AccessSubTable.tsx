import { Edit2, UserPlus, Users } from 'lucide-react'
import { Alert } from '../../../components/ui/Alert'
import { IconButton } from '../../../components/ui/IconButton'
import { Tooltip } from '../../../components/ui/Tooltip'
import { PageSpinner } from '../../../components/ui/Spinner'
import { cn } from '../../../lib/cn'
import { displayName, plural, roleLabel } from '../../../lib/format'
import { accessLevelOf, ACCESS_LEVEL_LABELS } from '../accessLevel'
import { useOfferingAccess } from '../api'
import type { AccessEntry, Offering } from '../types'

interface AccessSubTableProps {
  offering: Offering
  onAssign: (entries: AccessEntry[]) => void
  onEdit: (entry: AccessEntry) => void
}

export function AccessSubTable({ offering, onAssign, onEdit }: AccessSubTableProps) {
  const access = useOfferingAccess(offering.id, true)
  const entries = access.data ?? []
  const instructors = entries.filter((e) => e.role === 'instructor').length
  const staff = entries.filter((e) => e.role === 'staff').length

  return (
    <div className="py-3.5 px-8 bg-gradient-to-b from-slate-50 via-slate-100/50 to-slate-50 border-y border-slate-200/80 shadow-inner">
      <div className="flex items-center justify-between gap-3 mb-3">
        <div className="flex items-center gap-2.5">
          <div className="p-1.5 bg-indigo-100 text-indigo-700 rounded-lg">
            <Users size={16} />
          </div>
          <div>
            <h4 className="text-xs font-bold text-gray-900 uppercase tracking-wider flex flex-wrap items-center gap-2">
              <span>Assigned users with course portfolio access permissions</span>
              <span className="text-[11px] font-medium bg-emerald-50 text-emerald-800 border border-emerald-200/80 px-2 py-0.5 rounded-full normal-case">
                {instructors} {plural(instructors, 'Instructor')}
              </span>
              <span className="text-[11px] font-medium bg-blue-50 text-blue-800 border border-blue-200/80 px-2 py-0.5 rounded-full normal-case">{staff} Staff</span>
            </h4>
            <p className="text-[11px] text-gray-500">
              For course <strong className="text-gray-700">{offering.course_code}</strong> ({offering.course_name})
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => onAssign(entries)}
          disabled={access.isPending}
          className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-indigo-200 hover:border-indigo-300 text-indigo-700 hover:bg-indigo-50 rounded-lg text-xs font-semibold shadow-2xs transition-colors cursor-pointer disabled:opacity-50"
        >
          <UserPlus size={13} />
          <span>Assign New User</span>
        </button>
      </div>

      {access.error && <Alert>{access.error.message}</Alert>}
      {access.isPending ? (
        <PageSpinner />
      ) : (
        <div className="bg-white rounded-lg border border-slate-200 shadow-2xs overflow-hidden">
          <table className="min-w-full divide-y divide-gray-200 text-xs">
            <thead className="bg-slate-50/90 text-gray-600 uppercase font-semibold text-[10px] tracking-wider">
              <tr>
                <th className="px-4 py-2.5 text-left">Name</th>
                <th className="px-4 py-2.5 text-left">Email</th>
                <th className="px-4 py-2.5 text-left">Access</th>
                <th className="px-4 py-2.5 text-right">Action</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100 bg-white">
              {entries.map((entry) => {
                const level = accessLevelOf(entry)
                return (
                  <tr key={entry.user_id} className="hover:bg-slate-50/70 transition-colors">
                    <td className="px-4 py-3 whitespace-nowrap">
                      <div className="flex items-center gap-2.5">
                        <div className="h-7 w-7 rounded-full bg-indigo-50 text-indigo-700 border border-indigo-200 flex items-center justify-center font-bold text-xs shrink-0">
                          {displayName(entry).charAt(0).toUpperCase()}
                        </div>
                        <div>
                          <div className="font-semibold text-gray-900">{displayName(entry)}</div>
                          <div className="text-[10px] text-gray-400 capitalize">{roleLabel(entry.role)}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-gray-600 font-mono text-[11px]">{entry.email}</td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <div className="inline-flex items-center gap-1.5">
                        <span
                          className={cn(
                            'inline-flex items-center px-2.5 py-0.5 rounded-full text-[11px] font-semibold border',
                            level === 'both' && 'bg-emerald-50 text-emerald-700 border-emerald-200',
                            level === 'download' && 'bg-blue-50 text-blue-700 border-blue-200',
                            level === 'upload' && 'bg-amber-50 text-amber-700 border-amber-200',
                          )}
                        >
                          {ACCESS_LEVEL_LABELS[level]}
                        </span>
                        {entry.is_default && (
                          <Tooltip text={entry.default_reason ?? 'Default role permission'}>
                            <span className="text-[10px] font-medium text-gray-500 bg-gray-100 border border-gray-200/80 px-1.5 py-0.5 rounded cursor-help">(default)</span>
                          </Tooltip>
                        )}
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-right">
                      {!entry.is_default && <IconButton icon={Edit2} label={`Edit access of ${displayName(entry)}`} size={13} onClick={() => onEdit(entry)} />}
                    </td>
                  </tr>
                )
              })}
              {entries.length === 0 && (
                <tr>
                  <td colSpan={4} className="text-center py-6 text-gray-400 italic">
                    No instructors or staff currently assigned to this course. Click "Assign New User" to add access.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}
