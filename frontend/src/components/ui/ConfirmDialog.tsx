import type { ReactNode } from 'react'
import { Button } from './Button'
import { Modal } from './Modal'

interface ConfirmDialogProps {
  isOpen: boolean
  title: string
  confirmLabel: string
  onConfirm: () => void
  onClose: () => void
  loading?: boolean
  children: ReactNode
}

export function ConfirmDialog({ isOpen, title, confirmLabel, onConfirm, onClose, loading, children }: ConfirmDialogProps) {
  return (
    <Modal
      isOpen={isOpen}
      onClose={onClose}
      title={title}
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="danger" onClick={onConfirm} loading={loading}>
            {confirmLabel}
          </Button>
        </>
      }
    >
      {children}
    </Modal>
  )
}
