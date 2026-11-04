import { expect, test, request, type APIRequestContext, type Page } from '@playwright/test'
import { mkdirSync, readFileSync, readdirSync, statSync } from 'node:fs'
import { randomBytes } from 'node:crypto'
import { join } from 'node:path'
import JSZip from 'jszip'
import { BASE_URL, MAIN_API, REPO_ROOT, compose } from './env.ts'
import { json } from './api.ts'

// System Config (D29/T12): custom course template. Self-contained on a dev DB with real data: temp department + instructor
// are created through the API and removed in afterAll; the template is restored to the default in afterAll.
const SHOTS = process.env.SYSCFG_SHOTS
const shot = async (page: Page, name: string) => {
  if (!SHOTS) return
  mkdirSync(SHOTS, { recursive: true })
  await page.screenshot({ path: `${SHOTS}/${name}.png` })
}

const TPL = `${MAIN_API}/config/course-template`
const tag = `T12${Date.now().toString(36).toUpperCase()}`
const mail = `t12-instr-${tag.toLowerCase()}@chula.ac.th`
const PKG = join(REPO_ROOT, 'portfolio-service', 'templates', 'instructor-package')
const CORE = '90-grades/instructor_course_grading_template.csv'

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

function addDir(zip: JSZip, dir: string, prefix: string) {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) addDir(zip, p, `${prefix}${name}/`)
    else zip.file(`${prefix}${name}`, readFileSync(p))
  }
}
/** The default package folder plus README_CUSTOM.txt; `skip` drops an entry, `extra` adds some, `padMb` adds incompressible bytes. */
async function buildZip(opts: { skip?: string; extra?: Record<string, string>; padMb?: number } = {}): Promise<Buffer> {
  const zip = new JSZip()
  addDir(zip, PKG, '')
  zip.file('README_CUSTOM.txt', `custom template ${tag}\n`)
  if (opts.skip) zip.remove(opts.skip)
  for (const [k, v] of Object.entries(opts.extra ?? {})) zip.file(k, v)
  if (opts.padMb) zip.file('pad.bin', randomBytes(opts.padMb * 1024 * 1024), { compression: 'STORE' })
  return zip.generateAsync({ type: 'nodebuffer', compression: 'DEFLATE' })
}
const part = (buf: Buffer, name = 'template.zip') => ({ file: { name, mimeType: 'application/zip', buffer: buf } })

let sup: APIRequestContext
let instr: APIRequestContext
let superSession = ''
let instrSession = ''
let dept: { id: string } | undefined
let curr: { id: string } | undefined
let userId = ''

test.describe.configure({ mode: 'serial' })

async function cleanup() {
  if (sup) {
    await sup.delete(TPL).catch(() => undefined) // back to the default
    if (userId) await sup.delete(`${MAIN_API}/users/${userId}`).catch(() => undefined)
    if (curr) await sup.delete(`${MAIN_API}/curricula/${curr.id}`).catch(() => undefined)
    if (dept) await sup.delete(`${MAIN_API}/departments/${dept.id}`).catch(() => undefined)
    await sup.dispose()
  }
  if (instr) await instr.dispose()
}

test.beforeAll(async () => {
  superSession = session(superEmail())
  sup = await request.newContext({ baseURL: BASE_URL, storageState: cookieState(superSession) })
  try {
    dept = await json(await sup.post(`${MAIN_API}/departments`, { data: { code: tag, name: `T12 Dept ${tag}` } }), 201)
    curr = await json(await sup.post(`${MAIN_API}/curricula`, { data: { code: tag, name: `T12 Curr ${tag}`, department_id: dept!.id } }), 201)
    const u = await json(await sup.post(`${MAIN_API}/users`, { data: { email: mail, role: 'instructor', department_id: dept!.id, curriculum_id: curr!.id } }), 201)
    userId = u.id
    instrSession = session(mail)
    instr = await request.newContext({ baseURL: BASE_URL, storageState: cookieState(instrSession) })
  } catch (e) {
    await cleanup()
    throw e
  }
})
test.afterAll(cleanup)

test('API smoke on real MinIO: default → PUT → GET custom + uploaded_by → download equal → DELETE → default', async () => {
  const before = await json(await sup.get(TPL), 200)
  expect(before).toEqual({ source: 'default', size_bytes: null, uploaded_at: null, uploaded_by: null })
  const zip = await buildZip()
  const put = await json(await sup.put(TPL, { multipart: part(zip) }), 200)
  expect(put.source).toBe('custom')
  expect(put.uploaded_by).toBe(superEmail())
  expect(put.size_bytes).toBe(zip.length)
  expect(await json(await sup.get(TPL), 200)).toEqual(put)
  const dl = await sup.get(`${MAIN_API}/templates/instructor-package.zip`)
  expect(dl.headers()['content-disposition']).toContain('course-template.zip')
  expect(dl.headers()['content-type']).toContain('application/zip')
  expect((await dl.body()).equals(zip)).toBe(true)
  expect(await json(await sup.delete(TPL), 200)).toEqual(before)
  expect(await json(await sup.delete(TPL), 200)).toEqual(before) // idempotent
})

test('large upload through the gateway: ~19 MB passes, >20 MB → 413', async () => {
  const big = await buildZip({ padMb: 19 })
  expect(big.length).toBeLessThan(20 * 1024 * 1024)
  const ok = await json(await sup.put(TPL, { multipart: part(big), timeout: 110_000 }), 200)
  expect(ok.size_bytes).toBe(big.length)
  await json(await sup.delete(TPL), 200)
  const huge = await buildZip({ padMb: 21 })
  const res = await sup.put(TPL, { multipart: part(huge), timeout: 110_000 })
  expect(res.status()).toBe(413)
  expect((await (await sup.get(TPL)).json()).source).toBe('default')
})

test('permissions and validation through :8000', async () => {
  const z = await buildZip()
  expect((await instr.get(TPL)).status()).toBe(403)
  expect((await instr.put(TPL, { multipart: part(z) })).status()).toBe(403)
  expect((await instr.delete(TPL)).status()).toBe(403)
  const anon = await request.newContext({ baseURL: BASE_URL })
  expect((await anon.get(TPL)).status()).toBe(401)
  await anon.dispose()

  for (const [name, entry] of Object.entries({ dotdot: '../evil.txt', abs: '/etc/evil.txt', backslash: 'a\\evil.txt' })) {
    const res = await sup.put(TPL, { multipart: part(await buildZip({ extra: { [entry]: 'x' } })) })
    expect(res.status(), name).toBe(400)
  }
  expect((await sup.put(TPL, { multipart: part(Buffer.from('this is not a zip'), 'x.zip') })).status()).toBe(400)
  const missing = await sup.put(TPL, { multipart: part(await buildZip({ skip: CORE })) })
  expect(missing.status()).toBe(400)
  expect((await missing.json()).error).toContain(CORE)
  expect((await (await sup.get(TPL)).json()).source).toBe('default')
})

test('UI: nav → Default → errors → upload → Custom → instructor download equal, no nav, /config blocked → restore', async ({ browser }) => {
  const zip = await buildZip()
  const sctx = await browser.newContext({ storageState: cookieState(superSession) })
  const page = await sctx.newPage()
  await page.goto('/')
  await page.getByRole('navigation', { name: 'Main' }).getByRole('link', { name: 'System Config' }).click()
  await expect(page).toHaveURL(/\/config$/)
  await expect(page.getByRole('heading', { name: 'System Config' })).toBeVisible()
  const card = page.locator('section').filter({ hasText: 'Course portfolio template' })
  await expect(card.getByText('Default', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Restore default' })).toHaveCount(0)
  await shot(page, '01-default')

  // zip missing a core CSV → error naming the file, still Default
  await page.getByLabel('Template zip file').setInputFiles({ name: 'bad.zip', mimeType: 'application/zip', buffer: await buildZip({ skip: CORE }) })
  await expect(page.getByText(CORE, { exact: false })).toBeVisible()
  await expect(card.getByText('Default', { exact: true })).toBeVisible()
  await shot(page, '02-error-missing-csv')
  await page.getByLabel('Template zip file').setInputFiles({ name: 'notzip.txt', mimeType: 'text/plain', buffer: Buffer.from('x') })
  await expect(page.getByText('The template must be a .zip file.')).toBeVisible()

  // valid upload → Custom + uploader
  await page.getByLabel('Template zip file').setInputFiles({ name: 'my-template.zip', mimeType: 'application/zip', buffer: zip })
  await expect(card.getByText('Custom', { exact: true })).toBeVisible()
  await expect(card.getByText(superEmail(), { exact: false })).toBeVisible()
  await shot(page, '03-custom')

  // instructor: no System Config nav, /config blocked, Download Template = uploaded bytes
  const ictx = await browser.newContext({ storageState: cookieState(instrSession) })
  const ip = await ictx.newPage()
  await ip.goto('/portfolios')
  const nav = ip.getByRole('navigation', { name: 'Main' })
  await expect(nav.getByRole('link', { name: 'Course Portfolios' })).toBeVisible()
  await expect(nav.getByRole('link', { name: 'System Config' })).toHaveCount(0)
  const [dl] = await Promise.all([ip.waitForEvent('download'), ip.getByRole('button', { name: 'Download Template' }).click()])
  expect(dl.suggestedFilename()).toBe('course-template.zip')
  expect(readFileSync((await dl.path())!).equals(zip)).toBe(true)
  await ip.goto('/config')
  await expect(ip.getByRole('heading', { name: 'System Config' })).toHaveCount(0)
  await expect(ip).not.toHaveURL(/\/config$/)
  await shot(ip, '04-instructor-blocked')
  await ictx.close()

  // restore default (confirm dialog)
  await page.getByRole('button', { name: 'Restore default' }).click()
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await shot(page, '05-restore-confirm')
  await dialog.getByRole('button', { name: 'Restore default' }).click()
  await expect(card.getByText('Default', { exact: true })).toBeVisible()
  await expect(page.getByRole('button', { name: 'Restore default' })).toHaveCount(0)
  const dflt = await sup.get(`${MAIN_API}/templates/instructor-package.zip`)
  const names = Object.keys((await JSZip.loadAsync(await dflt.body())).files)
  expect(names.some((n) => n.endsWith('README_CUSTOM.txt'))).toBe(false)
  expect(names.some((n) => n.endsWith('README_FIRST.txt'))).toBe(true)

  // mobile width, with the custom state for the screenshot
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/config')
  await expect(page.getByRole('heading', { name: 'System Config' })).toBeVisible()
  await shot(page, '06-mobile')
  await sctx.close()
})
