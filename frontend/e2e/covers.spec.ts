import { expect, test } from '@playwright/test'
import { apiAs, createOrg, dropOrg, json, unique, type Org } from './api.ts'
import { MAIN_API, statePath } from './env.ts'
import { makePdf } from './pdf.ts'

test.describe('super admin: cover pages', () => {
  test.use({ storageState: statePath('super') })

  let api: Awaited<ReturnType<typeof apiAs>>
  let org: Org
  const tag = unique('V')

  test.beforeAll(async () => {
    api = await apiAs('super')
    org = await createOrg(api, tag)
  })
  test.afterAll(async () => {
    await dropOrg(api, org)
    await api.dispose()
  })

  test('upload → preview iframe → download → replace → remove', async ({ page }) => {
    const row = () => page.getByRole('row', { name: new RegExp(`E2E Curr ${tag}`) })
    await page.goto('/covers')
    await row().getByRole('button', { name: 'Upload Cover PDF' }).click()
    const dialog = page.getByRole('dialog')

    // not a PDF → refused in the browser, nothing is sent
    await dialog.getByLabel('Cover PDF', { exact: true }).setInputFiles({ name: 'notes.txt', mimeType: 'text/plain', buffer: Buffer.from('hello') })
    await expect(dialog.getByText(/pdf/i).first()).toBeVisible()
    await expect(dialog.getByRole('button', { name: 'Save Cover' })).toBeDisabled()

    await dialog.getByLabel('Cover PDF', { exact: true }).setInputFiles({ name: 'cover-v1.pdf', mimeType: 'application/pdf', buffer: makePdf('COVER ONE') })
    await dialog.getByRole('button', { name: 'Save Cover' }).click()
    await expect(row()).toContainText('cover-v1.pdf')

    // preview: the iframe URL must be served inline as PDF and be frameable
    const coverResponse = page.waitForResponse((r) => r.url().includes(`/curricula/${org.curr.id}/cover`) && r.request().resourceType() === 'document')
    await row().getByRole('button', { name: 'Preview Cover PDF' }).click()
    const cover = await coverResponse
    expect(cover.status()).toBe(200)
    expect(cover.headers()['content-type']).toContain('application/pdf')
    expect(cover.headers()['content-disposition']).toContain('inline')
    expect(cover.headers()['x-frame-options'] ?? '').not.toMatch(/deny/i)
    await expect(page.getByTitle(`Cover PDF of E2E Curr ${tag}`)).toBeVisible()

    // download from the preview
    const [download] = await Promise.all([page.waitForEvent('download'), page.getByRole('dialog').getByRole('button', { name: 'Download PDF' }).click()])
    expect(download.suggestedFilename()).toBe('cover-v1.pdf')
    await page.getByRole('dialog').getByRole('button', { name: 'Close Preview' }).click()

    // replace
    await row().getByRole('button', { name: 'Replace Cover PDF' }).click()
    await page.getByRole('dialog').getByLabel('Cover PDF', { exact: true }).setInputFiles({ name: 'cover-v2.pdf', mimeType: 'application/pdf', buffer: makePdf('COVER TWO') })
    await page.getByRole('dialog').getByRole('button', { name: 'Save Cover' }).click()
    await expect(row()).toContainText('cover-v2.pdf')

    // remove
    await row().getByRole('button', { name: 'Remove Cover' }).click()
    await page.getByRole('dialog').getByRole('button', { name: 'Remove Cover', exact: true }).click()
    await expect(row()).not.toContainText('cover-v2.pdf')
    const gone = await api.get(`${MAIN_API}/curricula/${org.curr.id}/cover`)
    expect(gone.status()).toBe(404)
  })

  test('API rejects a non-PDF disguised as .pdf', async () => {
    const res = await api.put(`${MAIN_API}/curricula/${org.curr.id}/cover`, {
      multipart: { file: { name: 'fake.pdf', mimeType: 'application/pdf', buffer: Buffer.from('not a pdf at all') } },
    })
    expect(res.status()).toBe(400)
  })
})

test.describe('admin: cover scope', () => {
  test.use({ storageState: statePath('admin2') })

  test('sees only curricula in scope; API refuses others', async ({ page }) => {
    await page.goto('/covers')
    await expect(page.getByRole('row', { name: /B\.Sc\. Mathematics/ })).toBeVisible()
    await expect(page.getByRole('row', { name: /B\.Sc\. Computer Science/ })).toHaveCount(0)

    const sup = await apiAs('super')
    const cs = (await json(await sup.get(`${MAIN_API}/curricula`), 200)).items.find((c: { code: string }) => c.code === 'CS-B')
    const api = await apiAs('admin2')
    const res = await api.put(`${MAIN_API}/curricula/${cs.id}/cover`, {
      multipart: { file: { name: 'x.pdf', mimeType: 'application/pdf', buffer: makePdf('X') } },
    })
    expect(res.status()).toBe(403)
  })
})

test.describe('instructor: no cover API', () => {
  test.use({ storageState: statePath('instructor') })

  test('403 on cover endpoints', async () => {
    const api = await apiAs('instructor')
    const sup = await apiAs('super')
    const cs = (await json(await sup.get(`${MAIN_API}/curricula`), 200)).items.find((c: { code: string }) => c.code === 'CS-B')
    expect((await api.get(`${MAIN_API}/curricula/${cs.id}/cover`)).status()).toBe(403)
  })
})
