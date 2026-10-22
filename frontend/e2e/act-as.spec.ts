import { expect, test, request, type APIRequestContext, type Browser, type BrowserContext, type Page } from '@playwright/test'
import { mkdirSync } from 'node:fs'
import { BASE_URL, MAIN_API, compose } from './env.ts'
import { json } from './api.ts'

// Dev-only "act as user" (D28). Self-contained: creates a temp department + curriculum and temp users through the API
// as the EXISTING super admin of the dev DB, and removes all of it in afterAll (even on failure). No seed, no uploads.
// Run with e2e/act-as.config.ts (no globalSetup, so seed.sql is not loaded). ACT_AS_SHOTS=<dir> also saves screenshots.
const SHOTS = process.env.ACT_AS_SHOTS
const shot = async (page: Page, name: string) => {
  if (!SHOTS) return
  mkdirSync(SHOTS, { recursive: true })
  await page.screenshot({ path: `${SHOTS}/${name}.png` })
}

const tag = `T11${Date.now().toString(36).toUpperCase()}`
const mail = (kind: string) => `t11-${kind}-${tag.toLowerCase()}@chula.ac.th`
const EMAILS = { admin: mail('admin'), instructor: mail('instr'), staff: mail('staff') }
const ALL = ['Course Portfolios', 'Courses', 'Cover Page', 'Users', 'Departments', 'Curriculums']

/** `abet_session` cookie for a provisioned user, exactly like `make dev-session`. */
function session(email: string): string {
  const out = compose(['exec', '-T', 'auth-service', 'go', 'run', './cmd/devsession', email]).trim()
  const m = /^abet_session=(.+)$/m.exec(out)
  if (!m) throw new Error('devsession printed no cookie')
  return m[1]
}

function superEmail(): string {
  const out = compose(['exec', '-T', 'postgres', 'sh', '-c',
    `PGPASSWORD="$POSTGRES_PASSWORD" psql -At -U "$POSTGRES_USER" -d "$MAIN_DB_NAME" -c "select email from users where role='super_admin' order by created_at limit 1"`])
  const email = out.trim().split('\n')[0]
  if (!email) throw new Error('no super_admin in the dev DB')
  return email
}

const cookieState = (value: string) => ({
  cookies: [{ name: 'abet_session', value, domain: new URL(BASE_URL).hostname, path: '/', expires: Math.floor(Date.now() / 1000) + 3600, httpOnly: true, secure: false, sameSite: 'Lax' as const }],
  origins: [],
})

let api: APIRequestContext
let superSession = ''
let superMail = ''
let dept: { id: string } | undefined
let curr: { id: string } | undefined
const created: string[] = []

test.describe.configure({ mode: 'serial' })

async function cleanup() {
  if (!api) return
  for (const id of created) await api.delete(`${MAIN_API}/users/${id}`).catch(() => undefined)
  if (curr) await api.delete(`${MAIN_API}/curricula/${curr.id}`).catch(() => undefined)
  if (dept) await api.delete(`${MAIN_API}/departments/${dept.id}`).catch(() => undefined)
  await api.dispose()
}

test.beforeAll(async () => {
  superMail = superEmail()
  superSession = session(superMail)
  api = await request.newContext({ baseURL: BASE_URL, storageState: cookieState(superSession) })
  try {
    dept = await json(await api.post(`${MAIN_API}/departments`, { data: { code: tag, name: `T11 Dept ${tag}` } }), 201)
    curr = await json(await api.post(`${MAIN_API}/curricula`, { data: { code: tag, name: `T11 Curr ${tag}`, department_id: dept!.id } }), 201)
    const mk = async (key: keyof typeof EMAILS, body: object) => {
      const u = await json(await api.post(`${MAIN_API}/users`, { data: { email: EMAILS[key], ...body } }), 201)
      created.push(u.id)
    }
    await mk('admin', { role: 'admin', admin_scope: { department_ids: [dept!.id], curriculum_ids: [] } })
    await mk('instructor', { role: 'instructor', department_id: dept!.id, curriculum_id: curr!.id })
    await mk('staff', { role: 'staff', department_id: dept!.id })
  } catch (e) {
    await cleanup()
    throw e
  }
})

test.afterAll(cleanup)

async function newSuperPage(browser: Browser, viewport?: { width: number; height: number }) {
  const ctx = await browser.newContext({ storageState: cookieState(superSession), viewport })
  const page = await ctx.newPage()
  return { ctx, page }
}

const actAsCookie = async (ctx: BrowserContext) => (await ctx.cookies()).find((c) => c.name === 'abet_act_as')
const bannerOf = (page: Page) => page.getByRole('status').filter({ hasText: 'Acting as' })
const rx = (s: string) => new RegExp(s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))

async function actAs(page: Page, role: 'Admin' | 'Instructor' | 'Staff' | 'Super admin', email: string) {
  await page.getByRole('button', { name: 'Switch role' }).click()
  const panel = page.getByRole('dialog', { name: 'Switch role panel' })
  await panel.getByRole('tab', { name: role, exact: true }).click()
  await panel.getByLabel('Search users').fill(email)
  await panel.getByRole('option', { name: rx(email) }).click()
  await expect(bannerOf(page)).toContainText(email)
}

async function menus(page: Page) {
  const nav = page.getByRole('navigation', { name: 'Main' })
  await expect(nav.getByRole('link', { name: 'Course Portfolios' })).toBeVisible()
  const present: string[] = []
  for (const item of ALL) if ((await nav.getByRole('link', { name: item }).count()) > 0) present.push(item)
  return present
}

test('switcher is visible for the super admin and lists the temp admin', async ({ browser }) => {
  const { ctx, page } = await newSuperPage(browser)
  await page.goto('/portfolios')
  const btn = page.getByRole('button', { name: 'Switch role' })
  await expect(btn).toBeVisible()
  await expect(bannerOf(page)).toHaveCount(0)
  await btn.click()
  const panel = page.getByRole('dialog', { name: 'Switch role panel' })
  await panel.getByLabel('Search users').fill(tag.toLowerCase())
  await expect(panel.getByRole('option')).toHaveCount(1) // Admin tab: only the temp admin
  await shot(page, '01-switcher-panel')
  await page.keyboard.press('Escape')
  await ctx.close()
})

test('act as admin → instructor → staff directly, then Back to Super Admin', async ({ browser }) => {
  const { ctx, page } = await newSuperPage(browser)
  await page.goto('/portfolios')
  expect(await menus(page)).toEqual(ALL)

  await actAs(page, 'Admin', EMAILS.admin)
  await expect(bannerOf(page)).toContainText('dev testing as')
  expect(await actAsCookie(ctx)).toBeTruthy()
  expect(await menus(page)).toEqual(['Course Portfolios', 'Courses', 'Cover Page', 'Users'])
  const me = await json(await page.request.get(`${MAIN_API}/me`), 200)
  expect(me.user.email).toBe(EMAILS.admin)
  expect(me.act_as.available).toBe(true)
  expect(me.act_as.real_user.role).toBe('super_admin')
  expect(me.user.admin_scope.departments.map((d: { id: string }) => d.id)).toEqual([dept!.id])
  await shot(page, '02-banner-as-admin')

  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'Users' }).click()
  await expect(page.getByRole('row', { name: rx(EMAILS.instructor) })).toBeVisible()
  await expect(page.getByRole('row', { name: rx(EMAILS.staff) })).toBeVisible()
  const users = await json(await page.request.get(`${MAIN_API}/users`), 200)
  expect(users.items.map((u: { role: string }) => u.role)).not.toContain('super_admin')
  expect(users.items.map((u: { email: string }) => u.email)).not.toContain(superMail)
  expect((await page.request.post(`${MAIN_API}/departments`, { data: { code: 'ZZNOPE', name: 'x' } })).status()).toBe(403) // no manage_org

  await actAs(page, 'Instructor', EMAILS.instructor) // direct switch, no Back in between
  expect(await menus(page)).toEqual(['Course Portfolios'])
  await shot(page, '03-banner-as-instructor')
  expect((await page.request.get(`${MAIN_API}/users`)).status()).toBe(403)

  await actAs(page, 'Staff', EMAILS.staff)
  expect(await menus(page)).toEqual(['Course Portfolios'])
  await shot(page, '04-banner-as-staff')

  await page.getByRole('button', { name: 'Back to Super Admin' }).click()
  await expect(bannerOf(page)).toHaveCount(0)
  expect(await menus(page)).toEqual(ALL)
  expect(await actAsCookie(ctx)).toBeUndefined()
  const back = await json(await page.request.get(`${MAIN_API}/me`), 200)
  expect(back.user.role).toBe('super_admin')
  expect(back.act_as.real_user).toBeNull()
  await shot(page, '05-back-as-super-admin')
  await ctx.close()
})

test('collapsed sidebar and a 390px viewport keep switcher and banner usable', async ({ browser }) => {
  const { ctx, page } = await newSuperPage(browser)
  await page.goto('/portfolios')
  await page.getByRole('button', { name: 'Collapse sidebar' }).click()
  await expect(page.getByRole('button', { name: 'Switch role' })).toBeVisible()
  await shot(page, '06-switcher-collapsed-sidebar')
  await ctx.close()

  const m = await newSuperPage(browser, { width: 390, height: 800 })
  await m.page.goto('/portfolios')
  await shot(m.page, '07-mobile-before')
  const mb = m.page.getByRole('button', { name: 'Switch role' })
  await expect(mb).toBeVisible()
  await mb.click()
  const panel = m.page.getByRole('dialog', { name: 'Switch role panel' })
  await expect(panel).toBeVisible()
  const box = (await panel.boundingBox())!
  expect(box.x).toBeGreaterThanOrEqual(0)
  expect(box.x + box.width).toBeLessThanOrEqual(390)
  await shot(m.page, '08-mobile-panel')
  await panel.getByLabel('Search users').fill(EMAILS.admin)
  await panel.getByRole('option').first().click()
  await expect(bannerOf(m.page)).toBeVisible()
  await shot(m.page, '09-mobile-banner')
  await m.page.getByRole('button', { name: 'Back to Super Admin' }).click()
  await expect(bannerOf(m.page)).toHaveCount(0)
  await m.ctx.close()
})
