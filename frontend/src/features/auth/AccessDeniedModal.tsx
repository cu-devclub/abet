import { AlertCircle } from 'lucide-react'
import { Button } from '../../components/ui/Button'
import { Modal } from '../../components/ui/Modal'

interface AccessDeniedModalProps {
  isOpen: boolean
  onClose: () => void
  variant: 'access_denied' | 'login_failed'
}

export function AccessDeniedModal({ isOpen, onClose, variant }: AccessDeniedModalProps) {
  const denied = variant === 'access_denied'
  return (
    <Modal isOpen={isOpen} onClose={onClose} title={denied ? 'Access Denied' : 'Sign-in Failed'}>
      <div className="flex flex-col items-center py-6 text-center">
        <div className="bg-red-100 p-3 rounded-full mb-4">
          <AlertCircle className="h-8 w-8 text-red-600" />
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-1">{denied ? 'Insufficient Permissions' : 'Something went wrong'}</h3>
        <p className="text-sm text-gray-500">
          {denied
            ? 'Your Google account was verified, but you do not have permission to access the ABET system. Please contact the system administrator.'
            : 'We could not complete your sign-in. Please try again, and contact the system administrator if the problem continues.'}
        </p>
        <Button onClick={onClose} className="mt-6 px-6">
          Back to Login
        </Button>
      </div>
    </Modal>
  )
}
