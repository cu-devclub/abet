import { semesterText } from '../../../lib/format'
import type { Offering } from '../types'

/** Read-only course context shown at the top of access modals. */
export function OfferingBanner({ offering }: { offering: Offering }) {
  return (
    <div className="bg-indigo-50/70 border border-indigo-100 rounded-xl p-3.5 space-y-1.5 text-xs">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1">
          <span className="text-gray-500">Course: </span>
          <span className="font-bold text-indigo-900 text-sm">
            {offering.course_code}: {offering.course_name}
          </span>
        </div>
        <div className="shrink-0 text-right whitespace-nowrap">
          <span className="text-gray-500">Year: </span>
          <span className="font-bold text-indigo-900 text-sm">
            {offering.academic_year} · {semesterText(offering.semester)}
          </span>
        </div>
      </div>
      <div>
        <span className="text-gray-500">Department: </span>
        <span className="font-semibold text-gray-800">{offering.department.name}</span>
      </div>
      <div>
        <span className="text-gray-500">Curriculum: </span>
        <span className="font-semibold text-gray-800">{offering.curriculum.name}</span>
      </div>
    </div>
  )
}
