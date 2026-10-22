import { useEffect, useRef, useState, type CSSProperties } from 'react'

const HIDDEN: CSSProperties = { position: 'fixed', top: 0, left: 0, zIndex: 9999, visibility: 'hidden' }

/** Positions a body-level popover next to its trigger, flipping to stay inside the viewport. */
export function useSmartPopover(isOpen: boolean, setIsOpen: (open: boolean) => void) {
  const buttonRef = useRef<HTMLButtonElement>(null)
  const popoverRef = useRef<HTMLDivElement>(null)
  const [popoverStyle, setPopoverStyle] = useState<CSSProperties>(HIDDEN)

  useEffect(() => {
    if (!isOpen) return

    const updatePosition = () => {
      const button = buttonRef.current
      if (!button) return
      const rect = button.getBoundingClientRect()
      const popover = popoverRef.current
      const width = popover ? popover.offsetWidth || popover.getBoundingClientRect().width : 224
      const height = popover ? popover.offsetHeight || popover.getBoundingClientRect().height : 260
      const pad = 12

      let left = rect.left
      if (left + width > window.innerWidth - pad) left = rect.right - width
      if (left < pad) left = pad
      if (left + width > window.innerWidth - pad) left = Math.max(pad, window.innerWidth - width - pad)

      let top = rect.bottom + 6
      if (top + height > window.innerHeight - pad && rect.top - height > pad) {
        top = rect.top - height - 6
      } else if (top + height > window.innerHeight - pad) {
        top = Math.max(pad, window.innerHeight - height - pad)
      }

      setPopoverStyle({
        position: 'fixed',
        top: `${Math.round(top)}px`,
        left: `${Math.round(left)}px`,
        zIndex: 9999,
        visibility: 'visible',
      })
    }

    const onMouseDown = (event: MouseEvent) => {
      const target = event.target as Node
      if (buttonRef.current?.contains(target) || popoverRef.current?.contains(target)) return
      setIsOpen(false)
    }
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setIsOpen(false)
    }

    updatePosition()
    const frame = requestAnimationFrame(updatePosition)
    document.addEventListener('mousedown', onMouseDown)
    document.addEventListener('keydown', onKeyDown)
    window.addEventListener('scroll', updatePosition, true)
    window.addEventListener('resize', updatePosition)
    return () => {
      cancelAnimationFrame(frame)
      document.removeEventListener('mousedown', onMouseDown)
      document.removeEventListener('keydown', onKeyDown)
      window.removeEventListener('scroll', updatePosition, true)
      window.removeEventListener('resize', updatePosition)
    }
  }, [isOpen, setIsOpen])

  return { buttonRef, popoverRef, popoverStyle }
}
