import type { ButtonHTMLAttributes, ElementType, ReactNode } from 'react'
import { cn } from '../../lib/cn'
import { Spinner } from './Spinner'

type Variant = 'primary' | 'secondary' | 'danger' | 'dangerSoft' | 'dangerGhost' | 'link'

const VARIANTS: Record<Variant, string> = {
  primary: 'px-4 py-2 text-white bg-indigo-600 hover:bg-indigo-700 shadow-sm',
  secondary: 'px-4 py-2 text-gray-700 bg-white border border-gray-300 hover:bg-gray-50',
  danger: 'px-4 py-2 text-white bg-red-600 hover:bg-red-700 shadow-2xs',
  dangerSoft: 'px-3 py-2 text-red-600 hover:text-red-800 bg-red-50 hover:bg-red-100 border border-red-200',
  dangerGhost: 'px-4 py-2 text-red-600 hover:bg-red-50',
  link: 'text-indigo-600 hover:text-indigo-800 font-semibold underline',
}

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant
  icon?: ElementType
  loading?: boolean
  children?: ReactNode
}

export function Button({ variant = 'primary', icon: Icon, loading, disabled, className, children, type = 'button', ...rest }: ButtonProps) {
  return (
    <button
      type={type}
      disabled={disabled || loading}
      className={cn(
        'inline-flex items-center justify-center text-sm font-medium rounded-lg transition-colors whitespace-nowrap cursor-pointer',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:ring-offset-2',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        VARIANTS[variant],
        className,
      )}
      {...rest}
    >
      {loading ? <Spinner size={15} className="mr-2" /> : Icon && <Icon size={16} className="mr-2 shrink-0" />}
      {children}
    </button>
  )
}
