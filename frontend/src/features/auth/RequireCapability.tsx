import { Navigate, Outlet } from 'react-router'
import { useMe } from './api'
import type { Capability } from './types'

/** UI guard only — the server enforces authorization on every request. */
export function RequireCapability({ capability }: { capability: Capability }) {
  const { data: me } = useMe()
  if (!me) return null
  return me.capabilities[capability] ? <Outlet /> : <Navigate to="/portfolios" replace />
}
