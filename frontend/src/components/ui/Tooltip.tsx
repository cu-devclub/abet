import type { ReactNode } from 'react'

/** Lightweight CSS tooltip; also keeps a native `title` fallback for assistive tech. */
export function Tooltip({ text, children }: { text: string; children: ReactNode }) {
  return (
    <span className="relative inline-flex group" title={text}>
      {children}
      <span
        role="tooltip"
        className="pointer-events-none absolute bottom-full left-1/2 -translate-x-1/2 mb-1 hidden group-hover:block group-focus-within:block whitespace-nowrap rounded-md bg-gray-900 px-2 py-1 text-[11px] font-medium text-white shadow-lg z-40"
      >
        {text}
      </span>
    </span>
  )
}
