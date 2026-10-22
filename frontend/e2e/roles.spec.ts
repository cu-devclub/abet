import { expect, test, type Page } from '@playwright/test'
import { apiAs, json } from './api.ts'
import { MAIN_API, statePath, type RoleKey } from './env.ts'

/** Every page a role can open must load without console errors or failing API calls. */
const PAGES: { role: RoleKey; paths: string[] }[] = [
  { role: 'super', paths: ['/portfolios', '/courses', '/covers', '/users', '/departments', '/curricula'] },
  { role: 'admin', paths: ['/portfolios', '/courses', '/covers', '/users'] },
  { role: 'instructor', paths: ['/portfolios'] },
  { role: 'staff', paths: ['/portfolios'] },
]

function watch(page: Page) {
  const problems: string[] = []
  page.on('console', (m) => {
    if (m.type() === 'error') problems.push(`console: ${m.text()}`)
  })
  page.on('pageerror', (e) => problems.push(`pageerror: ${e.message}`))
  page.on('response', (r) => {
    if (r.status() >= 400 && r.url().includes('/api/')) problems.push(`${r.status()} ${r.request().method()} ${r.url()}`)
  })
  return problems
}

for (const { role, paths } of PAGES) {
  test.describe(`${role}: pages load cleanly`, () => {
    test.use({ storageState: statePath(role) })

    for (const path of paths) {
      test(path, async ({ page }) => {
        const problems = watch(page)
        await page.goto(path)
        await expect(page.getByRole('heading', { level: 1 })).toBeVisible()
        await expect(page.getByRole('progressbar')).toHaveCount(0)
        await page.waitForLoadState('networkidle')
        expect(problems).toEqual([])
      })
    }
  })
}

test.describe('portfolio visibility per role', () => {
  test('instructor sees only offerings with access; staff gets download but no upload; admin is scoped', async () => {
    const instructor = await json(await (await apiAs('instructor')).get(`${MAIN_API}/portfolios?academic_year=2026`), 200)
    const mine = instructor.items.map((r: { course_code: string }) => r.course_code)
    expect(mine).toContain('2301101')
    expect(mine).not.toContain('2301301') // instructor2's Calculus

    const staff = await json(await (await apiAs('staff')).get(`${MAIN_API}/portfolios?academic_year=2026`), 200)
    expect(staff.items.length).toBeGreaterThan(0)
    for (const row of staff.items) expect(row).toMatchObject({ can_upload: false, can_download: true }) // department default (A3)

    const admin2 = await json(await (await apiAs('admin2')).get(`${MAIN_API}/portfolios?academic_year=2026`), 200)
    for (const row of admin2.items) expect(row.curriculum.code).toBe('MATH-B')
  })

  test('staff UI shows no upload buttons', async ({ browser }) => {
    const ctx = await browser.newContext({ storageState: statePath('staff') })
    const page = await ctx.newPage()
    await page.goto('/portfolios')
    await expect(page.getByRole('row', { name: /2301101/ }).first()).toBeVisible()
    await expect(page.getByRole('button', { name: /Upload Files|Re-upload Files/ })).toHaveCount(0)
    await ctx.close()
  })
})
