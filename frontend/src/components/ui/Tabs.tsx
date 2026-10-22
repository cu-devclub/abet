import { cn } from '../../lib/cn'

interface TabsProps<T extends string> {
  tabs: readonly { id: T; label: string }[]
  active: T
  onChange: (id: T) => void
  label: string
}

export function Tabs<T extends string>({ tabs, active, onChange, label }: TabsProps<T>) {
  return (
    <div role="tablist" aria-label={label} className="flex border-b border-gray-200 mb-4">
      {tabs.map((tab) => (
        <button
          key={tab.id}
          type="button"
          role="tab"
          aria-selected={active === tab.id}
          onClick={() => onChange(tab.id)}
          className={cn(
            'flex-1 pb-2.5 text-sm font-medium border-b-2 text-center transition-colors cursor-pointer',
            active === tab.id
              ? 'border-indigo-600 text-indigo-600 font-semibold'
              : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
          )}
        >
          {tab.label}
        </button>
      ))}
    </div>
  )
}
