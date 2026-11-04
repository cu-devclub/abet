import { screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { makeMe, mockApi, renderApp } from '../../test/utils'
import { Sidebar } from '../../components/layout/Sidebar'
import { ConfigPage } from './ConfigPage'

const uploadMock = vi.hoisted(() => vi.fn())
vi.mock('../../lib/http', async (orig) => ({ ...(await orig<typeof import('../../lib/http')>()), uploadWithProgress: uploadMock }))

const custom = { source: 'custom', size_bytes: 2048, uploaded_at: '2026-11-05T10:00:00Z', uploaded_by: 'boss@chula.ac.th' }
const dflt = { source: 'default', size_bytes: null, uploaded_at: null, uploaded_by: null }

describe('ConfigPage', () => {
  it('shows the default state without Restore', async () => {
    mockApi({ 'GET /config/course-template': dflt })
    renderApp(<ConfigPage />)
    expect(await screen.findByText('Default')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /restore default/i })).toBeNull()
  })

  it('shows the custom state, then restores after confirming', async () => {
    const { calls } = mockApi({ 'GET /config/course-template': custom, 'DELETE /config/course-template': dflt })
    renderApp(<ConfigPage />)
    expect(await screen.findByText(/by boss@chula.ac.th/)).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /restore default/i }))
    expect(calls.some((c) => c.key === 'DELETE /config/course-template')).toBe(false)
    await userEvent.click(screen.getAllByRole('button', { name: /restore default/i }).at(-1)!)
    await waitFor(() => expect(calls.some((c) => c.key === 'DELETE /config/course-template')).toBe(true))
  })

  it('uploads a zip via PUT and rejects non-zip files client-side', async () => {
    mockApi({ 'GET /config/course-template': dflt })
    uploadMock.mockResolvedValue(custom)
    renderApp(<ConfigPage />)
    const input = await screen.findByLabelText('Template zip file')
    await userEvent.upload(input, new File(['x'], 'a.txt'), { applyAccept: false })
    expect(await screen.findByText('The template must be a .zip file.')).toBeInTheDocument()
    expect(uploadMock).not.toHaveBeenCalled()
    await userEvent.upload(input, new File(['x'], 'a.zip'))
    await waitFor(() => expect(uploadMock).toHaveBeenCalledWith('/config/course-template', expect.any(FormData), undefined, 'PUT'))
  })
})

describe('Sidebar System Config item', () => {
  it.each([['super_admin', true], ['admin', false]] as const)('%s -> %s', (role, visible) => {
    mockApi({})
    renderApp(<Sidebar me={makeMe(role)} />)
    expect(!!screen.queryByText('System Config')).toBe(visible)
  })
})
