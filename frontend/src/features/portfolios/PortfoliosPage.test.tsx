import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { makeMe, mockApi, ref, renderApp } from '../../test/utils'
import { PortfoliosPage } from './PortfoliosPage'
import type { Issue, PortfolioRow } from './types'

const baseSubmission = {
  offering_id: 'o1',
  original_filename: 'x.zip',
  size_bytes: 10,
  uploaded_by: null,
  warning_count: 0,
  created_at: '2026-10-01T10:00:00Z',
  completed_at: '2026-10-01T10:05:00Z',
}

const row = (over: Partial<PortfolioRow>): PortfolioRow => ({
  offering_id: 'o1',
  course_code: '2301170',
  course_name: 'Computer Programming',
  academic_year: 2026,
  semester: 1,
  curriculum: ref('c1'),
  department: ref('d1'),
  can_upload: true,
  can_download: true,
  latest_submission: null,
  pdf_available: false,
  ...over,
})

const issues: Issue[] = [
  { severity: 'warning', stage: 'naming', code: 'W_NAME', title: 'Odd file name', description: 'd', location: 'a/b', suggested_fix: '' },
  { severity: 'error', stage: 'missing', code: 'E_MISSING', title: 'Missing syllabus', description: 'Syllabus not found', location: 'root/', suggested_fix: 'Add syllabus.pdf' },
]

function setup(rows: PortfolioRow[]) {
  return mockApi({
    'GET /me': makeMe('instructor'),
    'GET /offerings/years': { items: [2026, 2025], current: 2026 },
    'GET /departments': { items: [] },
    'GET /curricula': { items: [] },
    'GET /portfolios': { items: rows },
    'GET /submissions/s1': {
      ...baseSubmission,
      id: 's1',
      status: 'invalid',
      error_count: 1,
      warning_count: 1,
      issues,
    },
  })
}

const tableRow = (code: string) => screen.getByText(code).closest('tr') as HTMLElement

describe('PortfoliosPage', () => {
  it('queries the default (current) year and renders statuses and permitted actions', async () => {
    const { calls } = setup([
      row({}),
      row({
        offering_id: 'o2',
        course_code: '2301107',
        course_name: 'Calculus I',
        can_upload: false,
        pdf_available: true,
        latest_submission: { ...baseSubmission, id: 's2', offering_id: 'o2', status: 'generated', error_count: 0 },
      }),
      row({
        offering_id: 'o3',
        course_code: '2301263',
        course_name: 'Data Structures',
        can_download: false,
        pdf_available: true,
        latest_submission: { ...baseSubmission, id: 's1', offering_id: 'o3', status: 'invalid', error_count: 1 },
      }),
    ])
    renderApp(<PortfoliosPage />, '/portfolios')

    await screen.findByText('Computer Programming')
    expect(calls.find((c) => c.key === 'GET /portfolios')?.url.searchParams.get('academic_year')).toBe('2026')

    const none = tableRow('2301170')
    expect(within(none).getByText('No Files')).toBeInTheDocument()
    expect(within(none).getByRole('button', { name: 'Upload Files' })).toBeInTheDocument()
    expect(within(none).queryByRole('button', { name: /Download/ })).not.toBeInTheDocument()

    const generated = tableRow('2301107')
    expect(within(generated).getByText('Generated')).toBeInTheDocument()
    expect(within(generated).getByRole('button', { name: /Download 2301107/ })).toBeInTheDocument()
    expect(within(generated).queryByRole('button', { name: /Upload/ })).not.toBeInTheDocument()

    const failed = tableRow('2301263')
    expect(within(failed).getByText('Failed')).toBeInTheDocument()
    expect(within(failed).queryByRole('button', { name: /Download 2301263/ })).not.toBeInTheDocument()
    expect(within(failed).getByRole('button', { name: 'Re-upload Files' })).toBeInTheDocument()
  })

  it('opens the issues modal from a failed badge with errors listed before warnings', async () => {
    const user = userEvent.setup()
    setup([
      row({
        latest_submission: { ...baseSubmission, id: 's1', status: 'invalid', error_count: 1, warning_count: 1 },
      }),
    ])
    renderApp(<PortfoliosPage />, '/portfolios')
    await screen.findByText('Computer Programming')

    await user.click(screen.getByTitle('Click to view processing errors'))
    const dialog = await screen.findByRole('dialog', { name: 'Processing & Validation Errors' })
    await within(dialog).findByText('Missing syllabus')

    const titles = within(dialog).getAllByText(/Missing syllabus|Odd file name/).map((n) => n.textContent)
    expect(titles).toEqual(['Missing syllabus', 'Odd file name'])
    expect(within(dialog).getByRole('button', { name: 'Re-upload Files' })).toBeInTheDocument()

    await user.click(within(dialog).getByRole('button', { name: 'Expand All' }))
    expect(within(dialog).getByText('Add syllabus.pdf')).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Collapse All' })).toBeInTheDocument()
  })

  it('polls while something is queued or processing', async () => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'], shouldAdvanceTime: true })
    try {
      const { calls } = setup([
        row({ latest_submission: { ...baseSubmission, id: 's9', status: 'processing', error_count: 0 } }),
      ])
      renderApp(<PortfoliosPage />, '/portfolios')
      await screen.findByText(/^Processing/)
      const before = calls.filter((c) => c.key === 'GET /portfolios').length
      await vi.advanceTimersByTimeAsync(5200)
      expect(calls.filter((c) => c.key === 'GET /portfolios').length).toBeGreaterThan(before)
    } finally {
      vi.useRealTimers()
    }
  })
})
