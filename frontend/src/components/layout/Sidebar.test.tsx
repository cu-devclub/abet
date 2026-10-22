import { screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { makeMe, mockApi, renderApp } from '../../test/utils'
import { Sidebar } from './Sidebar'

beforeEach(() => {
  localStorage.clear()
  mockApi({})
})

const navLabels = () => screen.getAllByRole('link').map((a) => a.textContent)

describe('Sidebar', () => {
  it('shows only the portfolio page to instructors', () => {
    renderApp(<Sidebar me={makeMe('instructor')} />)
    expect(navLabels()).toEqual(['Course Portfolios'])
    expect(screen.getByText('instructor')).toBeInTheDocument()
  })

  it('adds management pages for admins but not departments/curriculums', () => {
    renderApp(<Sidebar me={makeMe('admin')} />)
    expect(navLabels()).toEqual(['Course Portfolios', 'Courses', 'Cover Page', 'Users'])
  })

  it('shows everything to super admins', () => {
    renderApp(<Sidebar me={makeMe('super_admin')} />)
    expect(navLabels()).toEqual(['Course Portfolios', 'Courses', 'Cover Page', 'Users', 'Departments', 'Curriculums'])
  })

  it('persists the collapsed state', async () => {
    const user = userEvent.setup()
    renderApp(<Sidebar me={makeMe('staff')} />)
    await user.click(screen.getByRole('button', { name: 'Collapse sidebar' }))
    expect(localStorage.getItem('abet.sidebar.collapsed')).toBe('1')
    expect(screen.getByRole('button', { name: 'Expand sidebar' })).toBeInTheDocument()
  })

  it('shows the profile picture when available', () => {
    const { container } = renderApp(<Sidebar me={makeMe('staff', { picture_url: 'https://example.com/a.png' })} />)
    expect(container.querySelector('img')).toHaveAttribute('src', 'https://example.com/a.png')
  })
})
