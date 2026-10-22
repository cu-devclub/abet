import type { ReactElement } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render } from '@testing-library/react'
import { MemoryRouter } from 'react-router'
import { ToastProvider } from '../components/ui/Toast'
import type { Me } from '../features/auth/types'

type Handler = unknown | ((request: { url: URL; init?: RequestInit }) => unknown)

/** Route table keyed by `METHOD /path` (path relative to /auth/api/v1, no query string). */
export function mockApi(routes: Record<string, Handler>) {
  const calls: { key: string; url: URL; init?: RequestInit }[] = []
  const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
    const url = new URL(String(input), 'http://localhost')
    const method = (init?.method ?? 'GET').toUpperCase()
    const key = `${method} ${url.pathname.replace('/auth/api/v1', '')}`
    calls.push({ key, url, init })
    if (!(key in routes)) return new Response(JSON.stringify({ error: `unmocked ${key}` }), { status: 404 })
    const handler = routes[key]
    const body = typeof handler === 'function' ? handler({ url, init }) : handler
    return new Response(JSON.stringify(body), { status: 200, headers: { 'Content-Type': 'application/json' } })
  })
  vi.stubGlobal('fetch', fetchMock)
  return { calls, fetchMock }
}

export function renderApp(ui: ReactElement, route = '/') {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(
    <ToastProvider>
      <QueryClientProvider client={queryClient}>
        <MemoryRouter initialEntries={[route]}>{ui}</MemoryRouter>
      </QueryClientProvider>
    </ToastProvider>,
  )
}

export const ref = (id: string, code = id.toUpperCase(), name = `${id} name`) => ({ id, code, name })

export function makeMe(role: 'super_admin' | 'admin' | 'instructor' | 'staff', extra: Partial<Me['user']> = {}): Me {
  const admin = role === 'super_admin' || role === 'admin'
  return {
    user: {
      id: 'u1',
      email: 'me@chula.ac.th',
      name: 'Test User',
      picture_url: null,
      role,
      department: null,
      curriculum: null,
      admin_scope: null,
      last_login_at: null,
      editable: true,
      created_at: '',
      updated_at: '',
      ...extra,
    },
    capabilities: { manage_org: role === 'super_admin', manage_users: admin, manage_courses: admin, manage_covers: admin },
  }
}
