import { expect, test } from '@playwright/test'
import { statePath, type RoleKey } from './env.ts'

const ALL = ['Course Portfolios', 'Courses', 'Cover Page', 'Users', 'Departments', 'Curriculums']
const EXPECTED: Record<string, { role: RoleKey; items: string[] }> = {
  'super admin': { role: 'super', items: ALL },
  admin: { role: 'admin', items: ['Course Portfolios', 'Courses', 'Cover Page', 'Users'] },
  instructor: { role: 'instructor', items: ['Course Portfolios'] },
  staff: { role: 'staff', items: ['Course Portfolios'] },
}

for (const [label, { role, items }] of Object.entries(EXPECTED)) {
  test.describe(label, () => {
    test.use({ storageState: statePath(role) })

    test(`sees exactly: ${items.join(', ')}`, async ({ page }) => {
      await page.goto('/portfolios')
      const nav = page.getByRole('navigation', { name: 'Main' })
      for (const item of ALL) {
        const link = nav.getByRole('link', { name: item })
        if (items.includes(item)) await expect(link).toBeVisible()
        else await expect(link).toHaveCount(0)
      }
    })

    if (items.length === 1) {
      test('is bounced from admin pages (403 view → not-found/portfolios)', async ({ page }) => {
        await page.goto('/users')
        await expect(page.getByRole('heading', { name: 'Users', exact: true })).toHaveCount(0)
        const api = await page.request.get('/auth/api/v1/offerings?academic_year=2026')
        expect(api.status()).toBe(403)
        expect((await page.request.get('/auth/api/v1/users')).status()).toBe(403)
      })
    }
  })
}
