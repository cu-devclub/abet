import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, sep } from 'node:path'
import { expect, test } from '@playwright/test'
import JSZip from 'jszip'
import { apiAs, createOrg, dropOrg, json, unique } from './api.ts'
import { FIXTURE_DIR, MAIN_API, mcList } from './env.ts'
import { makePdf } from './pdf.ts'

const YEAR = 2026

function addDir(zip: JSZip, dir: string, root = dir) {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name)
    if (statSync(full).isDirectory()) addDir(zip, full, root)
    else zip.file(relative(root, full).split(sep).join('/'), readFileSync(full))
  }
}

test.describe('MinIO object layout', () => {
  test('cover object lives at covers/<curriculum>.pdf; offering delete removes uploads/ + portfolios/ objects', async () => {
    test.setTimeout(6 * 60_000)
    const api = await apiAs('super')
    const org = await createOrg(api, unique('S'))
    try {
      await json(await api.put(`${MAIN_API}/curricula/${org.curr.id}/cover`, { multipart: { file: { name: 'c.pdf', mimeType: 'application/pdf', buffer: makePdf('STORAGE') } } }), 200)
      expect(mcList(`covers/${org.curr.id}.pdf`).length).toBe(1)

      const offering = await json(
        await api.post(`${MAIN_API}/offerings`, { data: { course_code: '1234567', course_name: 'Storage', curriculum_id: org.curr.id, academic_year: YEAR, semester: 1 } }),
        201,
      )
      const zip = new JSZip()
      addDir(zip, FIXTURE_DIR)
      const buffer = await zip.generateAsync({ type: 'nodebuffer' })
      const sub = await json(await api.post(`${MAIN_API}/offerings/${offering.id}/submissions`, { multipart: { file: { name: 'p.zip', mimeType: 'application/zip', buffer } } }), 201)
      const sid: string = sub.id ?? sub.submission?.id
      expect(sid).toBeTruthy()

      // the zip is in uploads/ while queued/processing (or already gone when the build finished)
      await expect
        .poll(async () => (await json(await api.get(`${MAIN_API}/portfolios?academic_year=${YEAR}&curriculum_id=${org.curr.id}`), 200)).items[0].latest_submission.status, { timeout: 5 * 60_000, intervals: [3000] })
        .toBe('generated')
      expect(mcList(`portfolios/${sid}.pdf`).length).toBe(1)

      // delete the offering → objects removed (best effort after commit)
      await json(await api.post(`${MAIN_API}/offerings/bulk-delete`, { data: { ids: [offering.id] } }), 200)
      await expect.poll(() => mcList(`portfolios/${sid}.pdf`), { timeout: 30_000 }).toEqual([])
      expect(mcList(`uploads/${sid}.zip`)).toEqual([])

      // removing the cover deletes the object
      await api.delete(`${MAIN_API}/curricula/${org.curr.id}/cover`)
      expect(mcList(`covers/${org.curr.id}.pdf`)).toEqual([])
    } finally {
      await dropOrg(api, org)
      await api.dispose()
    }
  })
})
