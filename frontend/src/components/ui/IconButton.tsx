import type { ButtonHTMLAttributes, ElementType } from 'react'
import { cn } from '../../lib/cn'

type Tone = 'edit' | 'indigo' | 'green' | 'red' | 'gray'

const TONES: Record<Tone, string> = {
  edit: 'p-1.5 text-indigo-700 bg-indigo-50 hover:bg-indigo-100 border border-indigo-200/70 rounded-md',
  indigo: 'p-1.5 text-indigo-600 hover:text-indigo-900 bg-indigo-50 hover:bg-indigo-100 rounded-lg',
  green: 'p-1.5 text-green-600 hover:text-green-900 bg-green-50 hover:bg-green-100 rounded-lg',
  red: 'p-1.5 text-red-600 hover:text-red-900 bg-red-50 hover:bg-red-100 rounded-lg',
  gray: 'p-1.5 text-gray-600 hover:text-indigo-900 bg-gray-50 hover:bg-indigo-50 rounded-lg',
}

interface IconButtonProps extends Omit<ButtonHTMLAttributes<HTMLButtonElement>, 'title'> {
  icon: ElementType
  label: string
  tone?: Tone
  size?: number
}

export function IconButton({ icon: Icon, label, tone = 'edit', size = 14, className, type = 'button', ...rest }: IconButtonProps) {
  return (
    <button
      type={type}
      title={label}
      aria-label={label}
      className={cn(
        'inline-flex items-center justify-center transition-colors cursor-pointer',
        'focus:outline-none focus-visible:ring-2 focus-visible:ring-indigo-500',
        'disabled:opacity-50 disabled:cursor-not-allowed',
        TONES[tone],
        className,
      )}
      {...rest}
    >
      <Icon size={size} />
    </button>
  )
}
