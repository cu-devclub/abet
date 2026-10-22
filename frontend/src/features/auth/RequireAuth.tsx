import { Navigate, Outlet, useLocation } from 'react-router'
import { PageSpinner } from '../../components/ui/Spinner'
import { Button } from '../../components/ui/Button'
import { ApiError } from '../../lib/http'
import { useMe } from './api'

export function RequireAuth() {
  const me = useMe()
  const location = useLocation()

  if (me.isPending) {
    return (
      <div className="min-h-screen bg-gray-50 flex items-center justify-center">
        <PageSpinner label="Loading your account…" />
      </div>
    )
  }

  if (me.error instanceof ApiError && me.error.status === 401) {
    return <Navigate to="/login" replace state={{ from: location.pathname + location.search }} />
  }

  if (me.error) {
    return (
      <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center gap-3 text-center px-4">
        <h1 className="text-lg font-semibold text-gray-900">We couldn't load your account</h1>
        <p className="text-sm text-gray-500">{me.error.message}</p>
        <Button onClick={() => void me.refetch()}>Try again</Button>
      </div>
    )
  }

  return <Outlet />
}
