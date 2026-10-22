import { expect, test, type Page } from '@playwright/test'
import { apiAs, createOrg, dropOrg, json, unique, type Org } from './api.ts'
import { MAIN_API, statePath } from './env.ts'

const YEAR = 2026
const HEADER = 'course id,course name,semester,user email,user role,upload permission,download permission'
const csv = (...rows: string[]) => Buffer.from([HEADER, ...rows].join('\n') + '\n')

async function openImport(page: Page, org: Org) {
  await page.goto(`/courses?cur=${org.curr.id}&year=${YEAR}`)
  await page.getByRole('button', { name: 'Add New Course' }).click()
  await page.getByRole('dialog').getByRole('tab', { name: 'Upload CSV' }).click()
}

async function offeringsOf(api: Awaited<ReturnType<typeof apiAs>>, org: Org) {
  const res = await json(await api.get(`${MAIN_API}/offerings?academic_year=${YEAR}&curriculum_id=${org.curr.id}`), 200)
  return res.items as { id: string; course_code: string; course_name: string; semester: number; has_submissions: boolean }[]
}

test.describe('super admin: courses & CSV import', () => {
  test.use({ storageState: statePath('super') })

  let api: Awaited<ReturnType<typeof apiAs>>
  let org: Org
  const tag = unique('C')

  test.beforeAll(async () => {
    api = await apiAs('super')
    org = await createOrg(api, tag)
  })
  test.afterAll(async () => {
    await dropOrg(api, org)
    await api.dispose()
  })

  test('add a single course, edit it, delete it', async ({ page }) => {
    await page.goto(`/courses?cur=${org.curr.id}&year=${YEAR}`)
    await page.getByRole('button', { name: 'Add New Course' }).click()
    const dialog = page.getByRole('dialog')
    await dialog.getByLabel('Course ID').fill(`${tag}1`)
    await dialog.getByLabel('Course Name').fill('Single Course')
    await dialog.getByLabel('Curriculum').selectOption({ label: `E2E Curr ${tag}` })
    await dialog.getByRole('button', { name: 'Add Course' }).click()
    const row = page.getByRole('row', { name: new RegExp(`${tag}1`) })
    await expect(row).toContainText('Single Course')
    await expect(row).toContainText(/0 assigned|0 users/i) // warning chip for courses without users

    await row.getByRole('button', { name: `Edit ${tag}1` }).click()
    await page.getByRole('dialog').getByLabel('Course Name').fill('Renamed Course')
    await page.getByRole('dialog').getByRole('button', { name: 'Save Changes' }).click()
    await expect(page.getByRole('row', { name: new RegExp(`${tag}1`) })).toContainText('Renamed Course')

    await page.getByRole('row', { name: new RegExp(`${tag}1`) }).getByRole('button', { name: `Edit ${tag}1` }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Delete Course' }).click()
    await page.getByRole('dialog').filter({ hasText: 'Are you sure' }).getByRole('button', { name: 'Delete', exact: true }).click()
    await expect(page.getByRole('row', { name: new RegExp(`${tag}1`) })).toHaveCount(0)
  })

  test('assign / edit / remove access on an offering', async ({ page }) => {
    const created = await json(
      await api.post(`${MAIN_API}/offerings`, {
        data: { course_code: `${tag}2`, course_name: 'Access Course', curriculum_id: org.curr.id, academic_year: YEAR, semester: 1 },
      }),
      201,
    )
    await page.goto(`/courses?cur=${org.curr.id}&year=${YEAR}`)
    await page.getByRole('button', { name: `Expand assigned users of ${tag}2` }).click()
    await page.getByRole('button', { name: 'Assign New User' }).click()
    const dialog = page.getByRole('dialog')
    const instructor2 = (await json(await api.get(`${MAIN_API}/users?search=instructor2@chula.ac.th`), 200)).items[0]
    await dialog.getByLabel('Select Instructor or Staff Member').selectOption({ value: instructor2.id })
    await dialog.getByRole('button', { name: 'Assign', exact: true }).click()
    await expect(dialog).toBeHidden()
    const explicit = async () =>
      (await json(await api.get(`${MAIN_API}/offerings/${created.id}/access`), 200)).items.filter((a: { is_default: boolean }) => !a.is_default)
    expect(await explicit()).toMatchObject([{ email: 'instructor2@chula.ac.th', can_upload: true, can_download: true }])

    await page.getByRole('button', { name: 'Edit access of instructor2@chula.ac.th' }).click()
    await page.getByRole('dialog').getByLabel('Download', { exact: true }).check()
    await page.getByRole('dialog').getByRole('button', { name: 'Save Changes' }).click()
    await expect.poll(async () => (await explicit())[0]?.can_upload).toBe(false)

    await page.getByRole('button', { name: 'Edit access of instructor2@chula.ac.th' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Remove Access' }).click()
    await expect.poll(async () => (await explicit()).length).toBe(0)
    await api.delete(`${MAIN_API}/offerings/${created.id}`)
  })

  test('CSV with errors: preview lists row errors and Apply stays disabled', async ({ page }) => {
    await openImport(page, org)
    const dialog = page.getByRole('dialog')
    await dialog.getByLabel('Mapping CSV', { exact: true }).setInputFiles({
      name: 'bad.csv',
      mimeType: 'text/csv',
      buffer: csv(`${tag}X,Bad Semester,9,person@chula.ac.th,Instructor,Yes,No`, `${tag}Y,Bad Domain,1,person@gmail.com,Instructor,Yes,No`),
    })
    await expect(dialog.getByText(/errors? found/i)).toBeVisible()
    await expect(dialog.getByText(/Row 2:/)).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Apply', exact: true })).toBeDisabled()
    expect(await offeringsOf(api, org)).toHaveLength(0)
  })

  test('header-only CSV is reported as "no data rows"', async ({ page }) => {
    await openImport(page, org)
    const dialog = page.getByRole('dialog')
    await dialog.getByLabel('Mapping CSV', { exact: true }).setInputFiles({ name: 'empty.csv', mimeType: 'text/csv', buffer: csv() })
    await expect(dialog.getByText(/no data rows/i)).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Apply', exact: true })).toBeDisabled()
  })

  test('import: create → (upload) → re-import shows update/create/delete/keep_has_uploads and applies', async ({ page }) => {
    const newUser = `${tag.toLowerCase()}.new@chula.ac.th`
    const ghost = `${tag.toLowerCase()}.ghost@chula.ac.th`

    // 1. first import: three courses, a new user, a both-No user that must NOT be created
    await openImport(page, org)
    let dialog = page.getByRole('dialog')
    await dialog.getByLabel('Mapping CSV', { exact: true }).setInputFiles({
      name: 'one.csv',
      mimeType: 'text/csv',
      buffer: csv(
        `${tag}A,Alpha,1,instructor1@chula.ac.th,Instructor,Yes,Yes`,
        `${tag}B,Beta,1,${newUser},Instructor,Yes,No`,
        `${tag}C,Gamma,1,${ghost},Staff,No,No`,
        `${tag}D,Delta,2,,,,`,
      ),
    })
    await expect(dialog.getByRole('row', { name: new RegExp(`Create.*${tag}A`) })).toBeVisible()
    await expect(dialog.getByText(`Users (1)`)).toBeVisible() // only newUser: the both-No user is not created
    await expect(dialog.getByRole('row').filter({ hasText: newUser }).filter({ hasText: 'Create user' })).toHaveCount(1)
    await dialog.getByRole('button', { name: 'Apply', exact: true }).click()
    await expect(dialog).toBeHidden()
    const first = await offeringsOf(api, org)
    expect(first.map((o) => `${o.course_code}/${o.semester}`).sort()).toEqual([`${tag}A/1`, `${tag}B/1`, `${tag}C/1`, `${tag}D/2`])
    const users = await json(await api.get(`${MAIN_API}/users?search=${tag.toLowerCase()}`), 200)
    expect(users.items.map((u: { email: string }) => u.email)).toEqual([newUser])

    // 2. B gets a (rejected) submission so it counts as "has uploads"
    const template = await api.get(`${MAIN_API}/templates/instructor-package.zip`)
    expect(template.headers()['content-type']).toContain('zip')
    const offB = first.find((o) => o.course_code === `${tag}B`)!
    const sub = await json(
      await api.post(`${MAIN_API}/offerings/${offB.id}/submissions`, {
        multipart: { file: { name: 'package.zip', mimeType: 'application/zip', buffer: await template.body() } },
      }),
      201,
    )
    expect(sub.status).toBe('invalid')

    // 3. second import: A renamed, E created, B omitted (has uploads), C+D omitted (deleted)
    await openImport(page, org)
    dialog = page.getByRole('dialog')
    await dialog.getByLabel('Mapping CSV', { exact: true }).setInputFiles({
      name: 'two.csv',
      mimeType: 'text/csv',
      buffer: csv(`${tag}A,Alpha Renamed,1,instructor1@chula.ac.th,Instructor,Yes,Yes`, `${tag}E,Epsilon,1,,,,`),
    })
    await expect(dialog.getByRole('row', { name: new RegExp(`Update name.*${tag}A`) })).toBeVisible()
    await expect(dialog.getByRole('row', { name: new RegExp(`Create.*${tag}E`) })).toBeVisible()
    await expect(dialog.getByRole('row', { name: new RegExp(`Keep \\(has uploads\\).*${tag}B`) })).toBeVisible()
    await expect(dialog.getByRole('row', { name: new RegExp(`Delete.*${tag}C`) })).toBeVisible()
    await expect(dialog.getByRole('row', { name: new RegExp(`Delete.*${tag}D`) })).toBeVisible()
    await dialog.getByRole('button', { name: 'Apply', exact: true }).click()
    await expect(dialog).toBeHidden()

    const second = await offeringsOf(api, org)
    expect(second.map((o) => o.course_code).sort()).toEqual([`${tag}A`, `${tag}B`, `${tag}E`])
    expect(second.find((o) => o.course_code === `${tag}A`)?.course_name).toBe('Alpha Renamed')
    expect(second.find((o) => o.course_code === `${tag}B`)?.has_submissions).toBe(true)
    // D9: B kept but its access rows were cleared
    expect((await json(await api.get(`${MAIN_API}/offerings/${offB.id}/access`), 200)).items.filter((a: { is_default: boolean }) => !a.is_default)).toHaveLength(0)

    // the page reflects the result
    await page.reload()
    await expect(page.getByRole('row', { name: new RegExp(`${tag}A`) })).toContainText('Alpha Renamed')
    await expect(page.getByRole('row', { name: new RegExp(`${tag}C`) })).toHaveCount(0)

    // cleanup of the user the import created
    const created = await json(await api.get(`${MAIN_API}/users?search=${newUser}`), 200)
    for (const u of created.items) await api.delete(`${MAIN_API}/users/${u.id}`)
  })

  test('bulk delete asks for confirmation and warns about uploads', async ({ page }) => {
    const rows = await offeringsOf(api, org)
    test.skip(rows.length === 0, 'nothing to delete')
    await page.goto(`/courses?cur=${org.curr.id}&year=${YEAR}`)
    await page.getByRole('checkbox', { name: 'Select all' }).check()
    await page.getByRole('button', { name: /Delete Selected/ }).click()
    const dialog = page.getByRole('dialog')
    if (rows.some((r) => r.has_submissions)) await expect(dialog.getByText(/uploads?|submissions?/i).first()).toBeVisible()
    await dialog.getByRole('button', { name: /Delete/ }).last().click()
    await expect(page.getByRole('row', { name: new RegExp(tag) })).toHaveCount(0)
  })
})
