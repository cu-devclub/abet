import { Outlet } from 'react-router'
import { useMe } from '../../features/auth/api'
import { ActAsBanner } from './ActAsBanner'
import { Sidebar } from './Sidebar'

export function AppShell() {
  const { data: me } = useMe()
  if (!me) return null
  return (
    <div className="flex flex-col h-screen bg-gray-50 overflow-hidden font-sans">
      <ActAsBanner me={me} />
      <div className="flex flex-1 min-h-0">
        <Sidebar me={me} />
        <main className="flex-1 overflow-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
