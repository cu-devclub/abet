import type { ReactNode } from 'react'

export function EmptyState({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="text-center py-8 text-gray-500 italic text-sm">
      {children}
      {action && <div className="mt-2 not-italic">{action}</div>}
    </div>
  )
}

/** Full-width table row for empty / loading / error states. */
export function TableMessageRow({ colSpan, children }: { colSpan: number; children: ReactNode }) {
  return (
    <tr>
      <td colSpan={colSpan}>{children}</td>
    </tr>
  )
}
