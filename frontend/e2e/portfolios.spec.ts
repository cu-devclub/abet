import { readFileSync } from 'node:fs'
import { expect, test, type Page } from '@playwright/test'
import JSZip from 'jszip'
import { apiAs, createOrg, dropOrg, json, unique, type Org } from './api.ts'
import { FIXTURE_DIR, MAIN_API, ROLES, mcList, statePath } from './env.ts'
import { makePdf, pdfPages, pdfText } from './pdf.ts'

const YEAR = 2026

type Api = Awaited<ReturnType<typeof apiAs>>

/** A throw-away curriculum with one offering the seeded instructor may upload to (and staff1 may only read). */
async function setup(api: Api, tag: string, withCover: string | null) {
  const org = await createOrg(api, tag)
  // the T05 fixture's identity CSV says course 1234567 → the offering must carry that code
  const codes = { valid: '1234567', invalid: `${tag}2` }
  const offerings: Record<string, { id: string }> = {}
  for (const [key, code] of Object.entries(codes)) {
    offerings[key] = await json(
      await api.post(`${MAIN_API}/offerings`, { data: { course_code: code, course_name: `Portfolio ${key}`, curriculum_id: org.curr.id, academic_year: YEAR, semester: 1 } }),
      201,
    )
    const instructor = (await json(await api.get(`${MAIN_API}/users?search=${ROLES.instructor}`), 200)).items[0]
    await json(await api.put(`${MAIN_API}/offerings/${offerings[key].id}/access/${instructor.id}`, { data: { can_upload: true, can_download: true } }), 200)
  }
  if (withCover) {
    await json(
      await api.put(`${MAIN_API}/curricula/${org.curr.id}/cover`, { multipart: { file: { name: 'cover.pdf', mimeType: 'application/pdf', buffer: makePdf(withCover) } } }),
      200,
    )
  }
  return { org, codes, offerings }
}

const rowOf = (page: Page, code: string) => page.getByRole('row', { name: new RegExp(code) })

test.describe('instructor: upload a portfolio', () => {
  test.use({ storageState: statePath('instructor') })

  const tag = unique('P')
  const COVER_TEXT = `E2ECOVER${tag}`
  let api: Api
  let org: Org
  let codes: { valid: string; invalid: string }
  let offerings: Record<string, { id: string }>

  test.beforeAll(async () => {
    api = await apiAs('super')
    ;({ org, codes, offerings } = await setup(api, tag, COVER_TEXT))
  })
  test.afterAll(async () => {
    await dropOrg(api, org)
    await api.dispose()
  })

  test('invalid package → issues modal opens immediately, nothing is queued', async ({ page }) => {
    const uploadsBefore = mcList('uploads/')
    await page.goto(`/portfolios?year=${YEAR}`)
    await expect(rowOf(page, codes.invalid)).toContainText('No Files')
    await rowOf(page, codes.invalid).getByRole('button', { name: 'Upload Files' }).click()

    // the unfilled template is INVALID on purpose (SCHEDULE_EMPTY)
    const template = await page.request.get(`${MAIN_API}/templates/instructor-package.zip`)
    expect(template.status()).toBe(200)
    const dialog = page.getByRole('dialog')
    await dialog.getByLabel('Portfolio .zip', { exact: true }).setInputFiles({ name: 'template.zip', mimeType: 'application/zip', buffer: await template.body() })
    await dialog.getByRole('button', { name: 'Submit & Process' }).click()

    const issues = page.getByRole('dialog', { name: 'Processing & Validation Errors' })
    await expect(issues).toBeVisible()
    await expect(issues.getByRole('region', { name: 'Errors' }).or(issues.locator('section[aria-label="Errors"]'))).toContainText(/schedule/i)
    await issues.getByRole('button', { name: 'Close' }).first().click()

    await expect(rowOf(page, codes.invalid)).toContainText('Failed')
    // history API: latest submission is invalid with error_count > 0, and no PDF
    const list = await json(await api.get(`${MAIN_API}/portfolios?academic_year=${YEAR}&curriculum_id=${org.curr.id}`), 200)
    const row = list.items.find((r: { course_code: string }) => r.course_code === codes.invalid)
    expect(row.latest_submission.status).toBe('invalid')
    expect(row.latest_submission.error_count).toBeGreaterThan(0)
    expect(row.pdf_available).toBe(false)
    // storage: an invalid package never reaches MinIO
    expect(mcList('uploads/')).toEqual(uploadsBefore)
    expect(mcList(`uploads/${row.latest_submission.id}.zip`)).toEqual([])

    // the failed badge re-opens the modal with the stored issues; Re-upload is offered
    await rowOf(page, codes.invalid).getByRole('button', { name: 'View Processing Errors' }).click()
    await expect(page.getByRole('dialog', { name: 'Processing & Validation Errors' })).toBeVisible()
    await expect(page.getByRole('dialog', { name: 'Processing & Validation Errors' }).getByRole('button', { name: /Re-upload/ })).toBeVisible()
  })

  test('valid folder → Processing → Generated → cover is page 1 → PDF download + bulk zip', async ({ page }) => {
    test.setTimeout(6 * 60_000)
    await page.goto(`/portfolios?year=${YEAR}`)
    await expect(rowOf(page, codes.valid)).toContainText('No Files')
    await rowOf(page, codes.valid).getByRole('button', { name: 'Upload Files' }).click()

    // the folder input is the real browser path: the page zips the folder itself with jszip
    const dialog = page.getByRole('dialog')
    await dialog.getByLabel('Portfolio .zip (folder)').setInputFiles(FIXTURE_DIR)
    await expect(dialog.getByRole('button', { name: 'Submit & Process' })).toBeEnabled({ timeout: 90_000 }) // in-browser zipping of the folder
    await dialog.getByRole('button', { name: 'Submit & Process' }).click()

    await expect(page.getByText('Uploaded — generating PDF')).toBeVisible()
    await expect(page.getByRole('dialog')).toBeHidden()
    await expect(rowOf(page, codes.valid)).toContainText(/Processing|Generated/)
    // polling (5 s) flips the badge on its own
    await expect(rowOf(page, codes.valid)).toContainText('Generated', { timeout: 5 * 60_000 })

    // single download → application/pdf, named <code>_<year>_S<sem>_portfolio.pdf
    const pdfResponse = page.waitForResponse((r) => r.url().includes('/portfolio.pdf'))
    const [download] = await Promise.all([page.waitForEvent('download'), rowOf(page, codes.valid).getByRole('button', { name: /Download .* portfolio PDF/ }).click()])
    const response = await pdfResponse
    expect(response.headers()['content-type']).toContain('application/pdf')
    expect(download.suggestedFilename()).toBe(`${codes.valid}_${YEAR}_S1_portfolio.pdf`)
    const pdf = readFileSync((await download.path())!)
    expect(pdf.subarray(0, 5).toString()).toBe('%PDF-')

    // page 1 is the curriculum cover, the rest is the generated portfolio
    expect(pdfText(pdf, 1, 1)).toContain(COVER_TEXT)
    expect(pdfPages(pdf)).toBeGreaterThan(10)

    // bulk download: select both rows (one has no PDF) → zip with the PDF
    await page.getByRole('checkbox', { name: `Select ${codes.valid}` }).check()
    await page.getByRole('checkbox', { name: `Select ${codes.invalid}` }).check()
    const [zipDownload] = await Promise.all([page.waitForEvent('download'), page.getByRole('button', { name: /^Download Selected/ }).click()])
    expect(zipDownload.suggestedFilename()).toMatch(/^ABET_Course_Portfolios_.*\.zip$/)
    const zip = await JSZip.loadAsync(readFileSync((await zipDownload.path())!))
    const names = Object.keys(zip.files)
    expect(names.some((n) => n.includes(codes.valid) && n.endsWith('.pdf'))).toBe(true)

    // the API agrees, and staff of the department can read but not upload
    const list = await json(await api.get(`${MAIN_API}/portfolios?academic_year=${YEAR}&curriculum_id=${org.curr.id}`), 200)
    const row = list.items.find((r: { course_code: string }) => r.course_code === codes.valid)
    expect(row).toMatchObject({ pdf_available: true, latest_submission: { status: 'generated' } })
    expect(row.latest_submission.uploaded_by.email).toBe(ROLES.instructor)

    // storage (MinIO): uploads/ object is gone, the final PDF exists, outputs/ prefix is cleaned up
    const sid = row.latest_submission.id
    expect(mcList(`uploads/${sid}.zip`)).toEqual([])
    expect(mcList(`portfolios/${sid}.pdf`).length).toBe(1)
    expect(mcList(`outputs/${sid}/`)).toEqual([])
  })

  test('403 paths: instructor cannot use the courses/users/cover APIs', async () => {
    const instructor = await apiAs('instructor')
    expect((await instructor.get(`${MAIN_API}/offerings?academic_year=${YEAR}`)).status()).toBe(403)
    expect((await instructor.post(`${MAIN_API}/offerings`, { data: { course_code: 'X1', course_name: 'x', curriculum_id: org.curr.id, academic_year: YEAR, semester: 1 } })).status()).toBe(403)
    expect((await instructor.post(`${MAIN_API}/users`, { data: { email: 'x@chula.ac.th', role: 'staff' } })).status()).toBe(403)
    const other = await apiAs('instructor2') // no access row on these offerings
    expect((await other.get(`${MAIN_API}/offerings/${offerings.valid.id}/portfolio.pdf`)).status()).toBe(403)
    const upload = await other.post(`${MAIN_API}/offerings/${offerings.valid.id}/submissions`, {
      multipart: { file: { name: 'a.zip', mimeType: 'application/zip', buffer: Buffer.from('PK') } },
    })
    expect(upload.status()).toBe(403)
  })
})
