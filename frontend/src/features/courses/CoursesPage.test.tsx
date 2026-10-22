import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { makeMe, mockApi, ref, renderApp } from '../../test/utils'
import { CoursesPage } from './CoursesPage'
import type { AccessEntry, Offering } from './types'

const offering = (over: Partial<Offering>): Offering => ({
  id: 'o1',
  course_id: 'c1',
  course_code: '2301170',
  course_name: 'Computer Programming',
  academic_year: 2026,
  semester: 3,
  curriculum: ref('cur1', 'CS', 'Computer Science'),
  department: ref('d1', 'MATH-CS', 'Math and CS'),
  instructor_count: 1,
  staff_count: 0,
  has_submissions: true,
  created_at: '',
  updated_at: '',
  ...over,
})

const entry = (over: Partial<AccessEntry>): AccessEntry => ({
  user_id: 'u2',
  name: 'Kanya Wattana',
  email: 'kanya.w@chula.ac.th',
  role: 'instructor',
  can_upload: true,
  can_download: true,
  is_default: false,
  default_reason: null,
  source: 'manual',
  ...over,
})

function setup() {
  return mockApi({
    'GET /me': makeMe('super_admin'),
    'GET /offerings/years': { items: [2026], current: 2026 },
    'GET /departments': { items: [] },
    'GET /curricula': { items: [] },
    'GET /offerings': { items: [offering({}), offering({ id: 'o2', course_code: '2301107', course_name: 'Calculus I', semester: 1, instructor_count: 0, has_submissions: false })] },
    'GET /offerings/o1/access': {
      items: [
        entry({}),
        entry({
          user_id: 'u3',
          name: 'Suda Kittisuk',
          email: 'suda.k@chula.ac.th',
          role: 'staff',
          can_upload: false,
          is_default: true,
          default_reason: 'Staff in department MATH-CS (default download)',
          source: 'default',
        }),
      ],
    },
  })
}

describe('CoursesPage', () => {
  it('labels Summer semesters, flags unassigned courses and requires year', async () => {
    const { calls } = setup()
    renderApp(<CoursesPage />, '/courses')
    await screen.findByText('Computer Programming')
    expect(calls.find((c) => c.key === 'GET /offerings')?.url.searchParams.get('academic_year')).toBe('2026')
    expect(within(screen.getByText('2301170').closest('tr') as HTMLElement).getByText('Summer')).toBeInTheDocument()
    const unassigned = screen.getByText('2301107').closest('tr') as HTMLElement
    expect(within(unassigned).getByText('0 assigned')).toBeInTheDocument()
    expect(within(unassigned).getByText('Sem 1')).toBeInTheDocument()
  })

  it('expands an offering to show access, with default entries read-only', async () => {
    const user = userEvent.setup()
    setup()
    renderApp(<CoursesPage />, '/courses')
    await screen.findByText('Computer Programming')

    await user.click(screen.getByRole('button', { name: /Expand assigned users of 2301170/ }))
    await screen.findByText('Kanya Wattana')

    const kanya = screen.getByText('kanya.w@chula.ac.th').closest('tr') as HTMLElement
    expect(within(kanya).getByText('Upload & Download')).toBeInTheDocument()
    expect(within(kanya).getByRole('button', { name: /Edit access of Kanya/ })).toBeInTheDocument()

    const suda = screen.getByText('suda.k@chula.ac.th').closest('tr') as HTMLElement
    expect(within(suda).getByText('(default)')).toBeInTheDocument()
    expect(within(suda).getByText('Download')).toBeInTheDocument()
    expect(within(suda).queryByRole('button')).not.toBeInTheDocument()
  })

  it('warns before deleting a course that has uploads', async () => {
    const user = userEvent.setup()
    setup()
    renderApp(<CoursesPage />, '/courses')
    await screen.findByText('Computer Programming')

    await user.click(screen.getByRole('checkbox', { name: 'Select 2301170' }))
    await user.click(screen.getByRole('button', { name: /Delete Selected \(1\)/ }))
    const dialog = await screen.findByRole('dialog', { name: 'Confirm Delete Course' })
    expect(within(dialog).getByText(/Warning: Uploaded Files Affected/)).toBeInTheDocument()
    expect(within(dialog).getByText('2301170 - Computer Programming')).toBeInTheDocument()
  })

  it('disables CSV import until a curriculum is selected', async () => {
    const user = userEvent.setup()
    setup()
    renderApp(<CoursesPage />, '/courses')
    await screen.findByText('Computer Programming')

    await user.click(screen.getByRole('button', { name: 'Add New Course' }))
    await user.click(await screen.findByRole('tab', { name: 'Upload CSV' }))
    const dialog = await screen.findByRole('dialog')
    expect(within(dialog).getByText(/Select a curriculum first/)).toBeInTheDocument()
    expect(within(dialog).getByRole('button', { name: 'Choose file' })).toBeDisabled()
    expect(within(dialog).getByRole('button', { name: 'Apply' })).toBeDisabled()
  })
})
