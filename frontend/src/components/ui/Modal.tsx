import { useEffect, useId, useRef, type ReactNode } from 'react'
import { createPortal } from 'react-dom'
import { X } from 'lucide-react'
import { cn } from '../../lib/cn'

const FOCUSABLE =
  'a[href], button:not([disabled]), input:not([disabled]):not([type="hidden"]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

/** Open modals, topmost last — only the topmost reacts to Escape / Tab. */
const modalStack: symbol[] = []

interface ModalProps {
  isOpen: boolean
  onClose: () => void
  title: string
  children: ReactNode
  footer?: ReactNode
  maxWidth?: string
}

export function Modal({ isOpen, onClose, title, children, footer, maxWidth = 'max-w-lg' }: ModalProps) {
  const titleId = useId()
  const panelRef = useRef<HTMLDivElement>(null)
  const onCloseRef = useRef(onClose)

  useEffect(() => {
    onCloseRef.current = onClose
  })

  useEffect(() => {
    if (!isOpen) return
    const id = Symbol('modal')
    modalStack.push(id)
    const previouslyFocused = document.activeElement as HTMLElement | null
    const panel = panelRef.current
    const first =
      panel?.querySelector<HTMLElement>('[data-autofocus]') ??
      panel?.querySelector<HTMLElement>(`[data-modal-body] :is(${FOCUSABLE})`) ??
      panel?.querySelector<HTMLElement>(FOCUSABLE)
    ;(first ?? panel)?.focus()

    const onKeyDown = (event: KeyboardEvent) => {
      if (modalStack[modalStack.length - 1] !== id) return
      if (event.key === 'Escape') {
        event.stopPropagation()
        onCloseRef.current()
        return
      }
      if (event.key !== 'Tab' || !panel) return
      const items = Array.from(panel.querySelectorAll<HTMLElement>(FOCUSABLE))
      if (items.length === 0) {
        event.preventDefault()
        return
      }
      const firstItem = items[0]
      const lastItem = items[items.length - 1]
      if (event.shiftKey && document.activeElement === firstItem) {
        event.preventDefault()
        lastItem.focus()
      } else if (!event.shiftKey && document.activeElement === lastItem) {
        event.preventDefault()
        firstItem.focus()
      } else if (!panel.contains(document.activeElement)) {
        event.preventDefault()
        firstItem.focus()
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      modalStack.splice(modalStack.indexOf(id), 1)
      previouslyFocused?.focus?.()
    }
  }, [isOpen])

  if (!isOpen) return null

  return createPortal(
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        tabIndex={-1}
        className={cn('bg-white rounded-xl shadow-xl w-full overflow-hidden flex flex-col max-h-[90vh] focus:outline-none', maxWidth)}
      >
        <div className="flex justify-between items-center p-4 border-b">
          <h3 id={titleId} className="text-lg font-semibold text-gray-900">
            {title}
          </h3>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="text-gray-400 hover:text-gray-600 rounded-lg p-1 hover:bg-gray-100 transition-colors cursor-pointer"
          >
            <X size={20} />
          </button>
        </div>
        <div data-modal-body className="p-4 overflow-y-auto flex-1">
          {children}
        </div>
        {footer && <div className="p-4 border-t bg-gray-50 flex justify-end items-center gap-3">{footer}</div>}
      </div>
    </div>,
    document.body,
  )
}
