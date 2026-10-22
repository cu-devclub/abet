import { latestUpdate, statusLabel } from './rowStatus'
import type { PortfolioRow } from './types'

describe('statusLabel', () => {
  it('maps submission statuses to badge labels', () => {
    expect(statusLabel('generated')).toBe('Generated')
    expect(statusLabel('queued')).toBe('Processing')
    expect(statusLabel('processing')).toBe('Processing')
    expect(statusLabel('invalid')).toBe('Failed')
    expect(statusLabel('failed')).toBe('Failed')
    expect(statusLabel(null)).toBe('No Files')
  })
})

describe('latestUpdate', () => {
  it('prefers completion time over creation time', () => {
    const row = { latest_submission: { created_at: 'a', completed_at: 'b' } } as unknown as PortfolioRow
    expect(latestUpdate(row)).toBe('b')
    const pending = { latest_submission: { created_at: 'a', completed_at: null } } as unknown as PortfolioRow
    expect(latestUpdate(pending)).toBe('a')
    expect(latestUpdate({ latest_submission: null } as PortfolioRow)).toBeNull()
  })
})
