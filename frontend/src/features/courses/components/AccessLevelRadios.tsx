import { cn } from '../../../lib/cn'
import { ACCESS_LEVEL_LABELS, type AccessLevel } from '../accessLevel'

const STYLES: Record<AccessLevel, { selected: string; radio: string }> = {
  both: { selected: 'border-emerald-500 bg-emerald-50/50 shadow-2xs', radio: 'text-emerald-600 focus:ring-emerald-500' },
  download: { selected: 'border-blue-500 bg-blue-50/50 shadow-2xs', radio: 'text-blue-600 focus:ring-blue-500' },
  upload: { selected: 'border-amber-500 bg-amber-50/50 shadow-2xs', radio: 'text-amber-600 focus:ring-amber-500' },
}

const ORDER: AccessLevel[] = ['both', 'download', 'upload']

interface AccessLevelRadiosProps {
  name: string
  value: AccessLevel
  onChange: (level: AccessLevel) => void
}

export function AccessLevelRadios({ name, value, onChange }: AccessLevelRadiosProps) {
  return (
    <fieldset>
      <legend className="block text-xs font-bold text-gray-700 uppercase tracking-wider mb-2">Select permission for this course portfolio</legend>
      <div className="space-y-2.5">
        {ORDER.map((level) => (
          <label
            key={level}
            className={cn(
              'flex items-center gap-3 p-3 rounded-xl border-2 cursor-pointer transition-all',
              value === level ? STYLES[level].selected : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50/50',
            )}
          >
            <input
              type="radio"
              name={name}
              value={level}
              checked={value === level}
              onChange={() => onChange(level)}
              className={cn('h-4 w-4 border-gray-300 cursor-pointer', STYLES[level].radio)}
            />
            <span className="text-sm font-bold text-gray-900">{ACCESS_LEVEL_LABELS[level]}</span>
          </label>
        ))}
      </div>
    </fieldset>
  )
}
