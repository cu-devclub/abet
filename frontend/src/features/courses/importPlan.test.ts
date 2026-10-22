import { isImportPlan, summaryItems } from './importPlan'
import type { ImportSummary } from './types'

const summary: ImportSummary = {
  rows: 12,
  courses_create: 2,
  courses_update: 0,
  courses_keep: 5,
  courses_delete: 1,
  courses_kept_with_uploads: 0,
  access_add: 6,
  access_update: 0,
  access_remove: 3,
  users_create: 0,
  users_role_raised: 0,
}

describe('isImportPlan', () => {
  it('recognises a plan body', () => {
    expect(isImportPlan({ applied: false, summary, courses: [], access: [], users: [], errors: [] })).toBe(true)
  })

  it('rejects plain error bodies', () => {
    expect(isImportPlan({ error: 'forbidden' })).toBe(false)
    expect(isImportPlan(null)).toBe(false)
  })
})

describe('summaryItems', () => {
  it('keeps the row count and hides zero counters', () => {
    const keys = summaryItems(summary).map((i) => i.key)
    expect(keys).toEqual(['rows', 'courses_create', 'courses_keep', 'courses_delete', 'access_add', 'access_remove'])
  })
})
