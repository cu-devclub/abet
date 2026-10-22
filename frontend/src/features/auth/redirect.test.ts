import { safeRedirect } from './redirect'

describe('safeRedirect', () => {
  it('keeps relative paths with query strings', () => {
    expect(safeRedirect('/courses?year=2026')).toBe('/courses?year=2026')
  })

  it('falls back for anything else', () => {
    expect(safeRedirect(undefined)).toBe('/portfolios')
    expect(safeRedirect('https://evil.example')).toBe('/portfolios')
    expect(safeRedirect('//evil.example')).toBe('/portfolios')
    expect(safeRedirect('/login')).toBe('/portfolios')
  })
})
