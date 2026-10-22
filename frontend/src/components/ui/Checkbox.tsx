import type { InputHTMLAttributes } from 'react'
import { cn } from '../../lib/cn'

interface CheckboxProps extends Omit<InputHTMLAttributes<HTMLInputElement>, 'type'> {
  'aria-label': string
}

export function Checkbox({ className, ...rest }: CheckboxProps) {
  return (
    <input
      type="checkbox"
      className={cn('h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer', className)}
      {...rest}
    />
  )
}
