import type { ReactNode } from 'react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { act, renderHook, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { makeMe, mockApi, renderApp } from '../../test/utils'
import { ME_KEY, useLogout } from '../../features/auth/api'
import type { Me } from '../../features/auth/types'
import { ActAsBanner } from './ActAsBanner'
import { ActAsSwitcher } from './ActAsSwitcher'
import { Sidebar } from './Sidebar'

const realUser = { id: 'real', email: 'root@chula.ac.th', name: 'Root', role: 'super_admin' as const }

const withActAs = (me: Me, act_as: Me['act_as']): Me => ({ ...me, act_as })

const adminUser = { id: 'a1', email: 'admin1@chula.ac.th', name: 'Admin One', role: 'admin', department_id: null, curriculum_id: null }

beforeEach(() => {
  localStorage.clear()
})

describe('ActAsSwitcher', () => {
  it('is hidden when act-as is unavailable or missing', () => {
    mockApi({})
    const { unmount } = renderApp(<ActAsSwitcher me={makeMe('super_admin')} />)
    expect(screen.queryByRole('button', { name: 'Switch role' })).not.toBeInTheDocument()
    unmount()
    renderApp(<ActAsSwitcher me={withActAs(makeMe('super_admin'), { available: false, real_user: null })} />)
    expect(screen.queryByRole('button', { name: 'Switch role' })).not.toBeInTheDocument()
  })

  it('is shown in the sidebar when available', () => {
    mockApi({})
    renderApp(<Sidebar me={withActAs(makeMe('super_admin'), { available: true, real_user: null })} />)
    expect(screen.getByRole('button', { name: 'Switch role' })).toBeInTheDocument()
  })

  it('lists users of the chosen role, acts as the clicked user and closes on Escape', async () => {
    const user = userEvent.setup()
    const { calls } = mockApi({
      'GET /dev/act-as/users': { users: [adminUser] },
      'POST /dev/act-as': null,
    })
    renderApp(<ActAsSwitcher me={withActAs(makeMe('super_admin'), { available: true, real_user: null })} />)

    await user.click(screen.getByRole('button', { name: 'Switch role' }))
    const list = calls.find((c) => c.key === 'GET /dev/act-as/users')
    await waitFor(() => expect(list?.url.searchParams.get('role')).toBe('admin'))
    await user.click(await screen.findByRole('option', { name: /Admin One/ }))

    await waitFor(() => expect(calls.some((c) => c.key === 'POST /dev/act-as')).toBe(true))
    expect(JSON.parse(String(calls.find((c) => c.key === 'POST /dev/act-as')?.init?.body))).toEqual({ user_id: 'a1' })
    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument())

    await user.click(screen.getByRole('button', { name: 'Switch role' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    await user.keyboard('{Escape}')
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  })
})

describe('ActAsBanner', () => {
  it('renders nothing when not acting', () => {
    mockApi({})
    renderApp(<ActAsBanner me={withActAs(makeMe('admin'), { available: true, real_user: null })} />)
    expect(screen.queryByRole('status')).not.toBeInTheDocument()
  })

  it('shows who is being acted as and calls DELETE on "Back to Super Admin"', async () => {
    const user = userEvent.setup()
    const { calls } = mockApi({ 'DELETE /dev/act-as': null })
    renderApp(<ActAsBanner me={withActAs(makeMe('admin'), { available: true, real_user: realUser })} />)

    expect(screen.getByRole('status')).toHaveTextContent('Acting as Test User (admin) — dev testing as root@chula.ac.th')
    await user.click(screen.getByRole('button', { name: /Back to Super Admin/ }))
    await waitFor(() => expect(calls.map((c) => c.key)).toContain('DELETE /dev/act-as'))
  })
})

describe('useLogout while acting', () => {
  const run = async (me: Me) => {
    const { calls } = mockApi({ 'DELETE /dev/act-as': null, 'POST /public/api/v1/auth/logout': null })
    const queryClient = new QueryClient()
    queryClient.setQueryData(ME_KEY, me)
    const wrapper = ({ children }: { children: ReactNode }) => <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
    const { result } = renderHook(() => useLogout(), { wrapper })
    await act(() => result.current.mutateAsync())
    return calls.map((c) => c.key)
  }

  it('drops the act-as cookie first, then logs out', async () => {
    expect(await run(withActAs(makeMe('admin'), { available: true, real_user: realUser }))).toEqual(['DELETE /dev/act-as', 'POST /public/api/v1/auth/logout'])
  })

  it('only logs out when not acting', async () => {
    expect(await run(withActAs(makeMe('super_admin'), { available: true, real_user: null }))).toEqual(['POST /public/api/v1/auth/logout'])
  })
})
