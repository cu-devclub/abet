import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, sep } from 'node:path'
import { expect, test } from '@playwright/test'
import JSZip from 'jszip'
import { apiAs, createOrg, dropOrg, json, unique } from './api.ts'
import { compose, FIXTURE_DIR, MAIN_API, ROLES, statePath } from './env.ts'

const YEAR = 2026

test.describe('H4: only the allowlisted auth routes are reachable through the gateway', () => {
  const hidden = [
    '/public/api/v1/auth/verify',
    '/public/internal/v1/auth/verify',
    '/public/metrics',
    '/public/healthz',
    '/public/readyz',
    '/public/health',
    '/auth/internal/v1/auth/verify',
    '/auth/metrics',
    '/auth/healthz',
  ]
  for (const path of hidden) {
    test(`GET ${path} → 404 JSON (not the SPA)`, async ({ request }) => {
      const res = await request.get(path, { maxRedirects: 0 })
      expect(res.status()).toBe(404)
      expect(res.headers()['content-type']).toContain('application/json')
      expect(await res.json()).toEqual({ error: 'not found' })
    })
  }

  test('Google login is still a 302 and logout only accepts POST', async ({ request }) => {
    const login = await request.get('/public/api/v1/auth/google/login?redirect=/courses', { maxRedirects: 0 })
    if (login.status() === 503) test.skip(true, 'GOOGLE_CLIENT_ID not configured')
    expect(login.status()).toBe(302)
    expect((await request.get('/public/api/v1/auth/logout', { maxRedirects: 0 })).status()).toBe(404)
  })
})

test.describe('H1: instructors see their own org plus the orgs of their assigned offerings', () => {
  test.use({ storageState: statePath('instructor') })

  test('API lists only own department/curriculum; UI filter options add the assigned offering’s org', async ({ page }) => {
    const mine = await apiAs('instructor')
    const depts = await json(await mine.get(`${MAIN_API}/departments`), 200)
    expect(depts.items.map((d: { code: string }) => d.code)).toEqual(['CS'])
    const curricula = await json(await mine.get(`${MAIN_API}/curricula`), 200)
    expect(curricula.items.map((c: { code: string }) => c.code)).toEqual(['CS-B'])

    const api = await apiAs('super')
    const tag = unique('H')
    const org = await createOrg(api, tag)
    try {
      const offering = await json(
        await api.post(`${MAIN_API}/offerings`, { data: { course_code: `${tag}1`, course_name: 'Elsewhere', curriculum_id: org.curr.id, academic_year: YEAR, semester: 1 } }),
        201,
      )
      const instructor = (await json(await api.get(`${MAIN_API}/users?search=${ROLES.instructor}`), 200)).items[0]
      await json(await api.put(`${MAIN_API}/offerings/${offering.id}/access/${instructor.id}`, { data: { can_upload: true, can_download: true } }), 200)

      await page.goto(`/portfolios?year=${YEAR}`)
      await expect(page.getByRole('row', { name: new RegExp(`${tag}1`) })).toBeVisible()

      await page.getByRole('button', { name: /^Department:/ }).click()
      const options = page.getByRole('listbox', { name: 'Department' }).getByRole('option')
      await expect(options.filter({ hasText: `E2E Dept ${tag}` })).toHaveCount(1)
      await expect(options.filter({ hasText: 'Computer Science' })).toHaveCount(1)
      await expect(options.filter({ hasText: 'Mathematics' })).toHaveCount(0)
      await options.filter({ hasText: `E2E Dept ${tag}` }).click()
      await expect(page.getByRole('row', { name: new RegExp(`${tag}1`) })).toBeVisible()
      await expect(page).toHaveURL(new RegExp(`dept=${org.dept.id}`))
    } finally {
      await dropOrg(api, org)
      await api.dispose()
    }
  })
})

test.describe('H3: limits come from .env via GET /config', () => {
  test.use({ storageState: statePath('super') })

  test('the upload and cover dialogs show the configured limits', async ({ page }) => {
    const cfg = await json(await (await apiAs('super')).get(`${MAIN_API}/config`), 200)
    expect(cfg).toMatchObject({ max_upload_mb: expect.any(Number), max_cover_mb: expect.any(Number), max_import_mb: expect.any(Number), build_timeout_minutes: expect.any(Number) })

    await page.goto('/covers')
    await page.getByRole('button', { name: 'Upload Cover PDF' }).first().click()
    await expect(page.getByRole('dialog')).toContainText(`max ${cfg.max_cover_mb} MB`)
  })

  test('an instructor’s upload dialog shows the configured portfolio limit', async ({ browser }) => {
    const cfg = await json(await (await apiAs('instructor')).get(`${MAIN_API}/config`), 200)
    const ctx = await browser.newContext({ storageState: statePath('instructor') })
    const page = await ctx.newPage()
    await page.goto(`/portfolios?year=${YEAR}`)
    await page.getByRole('button', { name: 'Upload Files' }).first().click()
    await expect(page.getByRole('dialog')).toContainText(`max ${cfg.max_upload_mb} MB`)
    await ctx.close()
  })
})

function addDir(zip: JSZip, root: string, dir = root) {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name)
    if (statSync(full).isDirectory()) addDir(zip, root, full)
    else zip.file(relative(root, full).split(sep).join('/'), readFileSync(full))
  }
}

/**
 * H6: needs a stack started with BUILD_TIMEOUT_MINUTES=1 and RECONCILE_INTERVAL_SECONDS=10
 * (`E2E_BUILD_TIMEOUT=1 make e2e`); it stops and restarts portfolio-service itself.
 */
test.describe('H6: a stuck build never stays in Processing', () => {
  test.skip(!process.env.E2E_BUILD_TIMEOUT, 'set E2E_BUILD_TIMEOUT=1 on a stack with BUILD_TIMEOUT_MINUTES=1, RECONCILE_INTERVAL_SECONDS=10')
  test.use({ storageState: statePath('instructor') })

  test('worker down → Failed (BUILD_TIMEOUT) → worker up → Generated or Failed, never Processing', async ({ page }) => {
    test.setTimeout(8 * 60_000)
    const api = await apiAs('super')
    const tag = unique('T')
    const org = await createOrg(api, tag)
    const row = (code: string) => page.getByRole('row', { name: new RegExp(code) })
    try {
      const offering = await json(
        await api.post(`${MAIN_API}/offerings`, { data: { course_code: '1234567', course_name: 'Timeout', curriculum_id: org.curr.id, academic_year: YEAR, semester: 1 } }),
        201,
      )
      const instructor = (await json(await api.get(`${MAIN_API}/users?search=${ROLES.instructor}`), 200)).items[0]
      await json(await api.put(`${MAIN_API}/offerings/${offering.id}/access/${instructor.id}`, { data: { can_upload: true, can_download: true } }), 200)

      compose(['stop', 'portfolio-service'])
      const zip = new JSZip()
      addDir(zip, FIXTURE_DIR)
      const buffer = await zip.generateAsync({ type: 'nodebuffer' })

      await page.goto(`/portfolios?year=${YEAR}&cur=${org.curr.id}`)
      await row('1234567').getByRole('button', { name: 'Upload Files' }).click()
      await page.getByRole('dialog').getByLabel('Portfolio .zip', { exact: true }).setInputFiles({ name: 'p.zip', mimeType: 'application/zip', buffer })
      await page.getByRole('dialog').getByRole('button', { name: 'Submit & Process' }).click()
      await expect(row('1234567')).toContainText('Processing')

      await expect(row('1234567')).toContainText('Failed', { timeout: 3 * 60_000 })
      await row('1234567').getByRole('button', { name: 'View Processing Errors' }).click()
      const issues = page.getByRole('dialog', { name: 'Processing & Validation Errors' })
      await expect(issues).toContainText('BUILD_TIMEOUT')
      await expect(issues).toContainText('PDF generation timed out')
      await issues.getByRole('button', { name: 'Close' }).first().click()

      compose(['start', 'portfolio-service'])
      await page.reload()
      await expect(row('1234567')).toContainText(/Generated|Failed/, { timeout: 4 * 60_000 })
      await expect(row('1234567')).not.toContainText('Processing')
    } finally {
      compose(['start', 'portfolio-service'])
      await dropOrg(api, org)
      await api.dispose()
    }
  })
})
