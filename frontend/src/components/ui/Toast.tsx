import { useCallback, useMemo, useRef, useState, type ReactNode } from 'react'
import { AlertCircle, CheckCircle2, Info, X } from 'lucide-react'
import { cn } from '../../lib/cn'
import { ToastContext, type ToastApi } from './useToast'

type ToastType = 'success' | 'error' | 'info'

interface ToastItem {
  id: number
  type: ToastType
  text: string
}

const STYLES: Record<ToastType, string> = {
  error: 'bg-red-50 text-red-800 border-red-200',
  info: 'bg-blue-50 text-blue-800 border-blue-200',
  success: 'bg-emerald-50 text-emerald-800 border-emerald-200',
}

const ICONS = {
  error: <AlertCircle size={18} className="text-red-600 shrink-0" />,
  info: <Info size={18} className="text-blue-600 shrink-0" />,
  success: <CheckCircle2 size={18} className="text-emerald-600 shrink-0" />,
}

export function ToastProvider({ children }: { children: ReactNode }) {
  const [items, setItems] = useState<ToastItem[]>([])
  const nextId = useRef(1)

  const dismiss = useCallback((id: number) => setItems((prev) => prev.filter((t) => t.id !== id)), [])

  const push = useCallback(
    (type: ToastType, text: string) => {
      const id = nextId.current++
      setItems((prev) => [...prev, { id, type, text }])
      setTimeout(() => dismiss(id), type === 'error' ? 6000 : 3500)
    },
    [dismiss],
  )

  const api = useMemo<ToastApi>(
    () => ({
      success: (text) => push('success', text),
      error: (text) => push('error', text),
      info: (text) => push('info', text),
    }),
    [push],
  )

  return (
    <ToastContext.Provider value={api}>
      {children}
      <div className="fixed bottom-6 right-6 z-[60] flex flex-col gap-2 items-end">
        {items.map((item) => (
          <div
            key={item.id}
            role={item.type === 'error' ? 'alert' : 'status'}
            className={cn('flex items-center gap-3 px-4 py-3 rounded-xl shadow-lg border text-sm font-medium max-w-md', STYLES[item.type])}
          >
            {ICONS[item.type]}
            <span>{item.text}</span>
            <button
              type="button"
              onClick={() => dismiss(item.id)}
              aria-label="Dismiss"
              className="ml-2 text-gray-400 hover:text-gray-600 p-0.5 rounded cursor-pointer"
            >
              <X size={14} />
            </button>
          </div>
        ))}
      </div>
    </ToastContext.Provider>
  )
}
