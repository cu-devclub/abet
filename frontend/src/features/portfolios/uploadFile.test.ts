import { validatePortfolioZip } from './uploadFile'

const MAX = 300 * 1024 * 1024

describe('validatePortfolioZip', () => {
  it('accepts a small zip', () => {
    expect(validatePortfolioZip({ name: 'CourseX.zip', size: 10 * 1024 * 1024 }, MAX)).toBeNull()
    expect(validatePortfolioZip({ name: 'CourseX.ZIP', size: 300 * 1024 * 1024 }, MAX)).toBeNull()
  })

  it('rejects non-zip files', () => {
    expect(validatePortfolioZip({ name: 'notes.pdf', size: 10 }, MAX)).toMatch(/folder or a single \.zip/)
  })

  it('rejects zips over 300 MB with a human message', () => {
    expect(validatePortfolioZip({ name: 'big.zip', size: 301 * 1024 * 1024 }, MAX)).toMatch(/301\.0 MB.*300 MB limit/)
  })

  it('uses the configured limit (400 MB)', () => {
    const max = 400 * 1024 * 1024
    expect(validatePortfolioZip({ name: 'a.zip', size: 350 * 1024 * 1024 }, max)).toBeNull()
    expect(validatePortfolioZip({ name: 'a.zip', size: 401 * 1024 * 1024 }, max)).toMatch(/400 MB limit/)
  })
})
