import { useId, type InputHTMLAttributes, type ReactElement, type ReactNode, type SelectHTMLAttributes } from 'react'
import { cloneElement } from 'react'
import { cn } from '../../lib/cn'

const CONTROL =
  'w-full border border-gray-300 rounded-lg px-3 py-2 text-sm bg-white focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 focus:outline-none'
const DISABLED = 'disabled:bg-gray-100 disabled:text-gray-600 disabled:cursor-not-allowed'

export function TextInput({ className, ...rest }: InputHTMLAttributes<HTMLInputElement>) {
  return <input className={cn(CONTROL, DISABLED, className)} {...rest} />
}

export function NativeSelect({ className, children, ...rest }: SelectHTMLAttributes<HTMLSelectElement>) {
  return (
    <select className={cn(CONTROL, DISABLED, 'cursor-pointer', className)} {...rest}>
      {children}
    </select>
  )
}

interface FieldProps {
  label: string
  hint?: ReactNode
  /** The single control to label (gets `id` injected). */
  children: ReactElement<{ id?: string }>
}

export function Field({ label, hint, children }: FieldProps) {
  const id = useId()
  return (
    <div>
      <label htmlFor={id} className="block text-sm font-medium text-gray-700 mb-1">
        {label}
      </label>
      {cloneElement(children, { id })}
      {hint && <p className="text-xs text-gray-400 mt-1">{hint}</p>}
    </div>
  )
}

export function SmallLabel({ children }: { children: ReactNode }) {
  return <span className="block text-xs font-bold text-gray-700 uppercase tracking-wider mb-2">{children}</span>
}
