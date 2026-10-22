import { expect, test } from '@playwright/test'
import { apiAs, json, unique } from './api.ts'
import { MAIN_API, statePath } from './env.ts'

const confirmDelete = async (page: import('@playwright/test').Page) => {
  await page.getByRole('dialog').filter({ hasText: 'Are you sure' }).getByRole('button', { name: 'Delete', exact: true }).click()
}

test.describe('super admin: departments & curriculums', () => {
  test.use({ storageState: statePath('super') })

  test('create → duplicate refused → delete order enforced → delete', async ({ page }) => {
    const tag = unique('E2E')
    const api = await apiAs('super')
    try {
      await page.goto('/departments')
      await page.getByRole('button', { name: 'Add New Department' }).click()
      const dialog = page.getByRole('dialog')
      await dialog.getByLabel('Department Code').fill(tag.toLowerCase())
      await dialog.getByLabel('Department Name').fill(`Dept ${tag}`)
      await dialog.getByRole('button', { name: 'Create Department' }).click()
      await expect(page.getByRole('row', { name: new RegExp(`Dept ${tag}`) })).toContainText(tag) // code stored upper-cased

      // duplicate code → API 409 message stays inside the modal
      await page.getByRole('button', { name: 'Add New Department' }).click()
      await page.getByRole('dialog').getByLabel('Department Code').fill(tag)
      await page.getByRole('dialog').getByLabel('Department Name').fill('Duplicate')
      await page.getByRole('dialog').getByRole('button', { name: 'Create Department' }).click()
      await expect(page.getByRole('dialog').getByRole('alert')).toBeVisible()
      await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()

      // curriculum under the new department
      await page.goto('/curricula')
      await page.getByRole('button', { name: 'Add New Curriculum' }).click()
      const cd = page.getByRole('dialog')
      await cd.getByLabel('Curriculum Code').fill(tag)
      await cd.getByLabel('Curriculum Name').fill(`Curr ${tag}`)
      await cd.getByLabel('Department').selectOption({ label: `Dept ${tag}` })
      await cd.getByRole('button', { name: 'Create Curriculum' }).click()
      await expect(page.getByRole('row', { name: new RegExp(`Curr ${tag}`) })).toBeVisible()

      // department is referenced by the curriculum → refused with the API message
      await page.goto('/departments')
      await page.getByRole('row', { name: new RegExp(`Dept ${tag}`) }).getByRole('button', { name: 'Edit Department' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete Department' }).click()
      await confirmDelete(page)
      await expect(page.getByText(/referenced/i).first()).toBeVisible()
      await page.keyboard.press('Escape')

      // delete curriculum, then department
      await page.goto('/curricula')
      await page.getByRole('row', { name: new RegExp(`Curr ${tag}`) }).getByRole('button', { name: /Edit/ }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete Curriculum' }).click()
      await confirmDelete(page)
      await expect(page.getByRole('row', { name: new RegExp(`Curr ${tag}`) })).toHaveCount(0)

      await page.goto('/departments')
      await page.getByRole('row', { name: new RegExp(`Dept ${tag}`) }).getByRole('button', { name: 'Edit Department' }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete Department' }).click()
      await confirmDelete(page)
      await expect(page.getByRole('row', { name: new RegExp(`Dept ${tag}`) })).toHaveCount(0)
    } finally {
      const curricula = await json(await api.get(`${MAIN_API}/curricula`), 200)
      for (const c of curricula.items.filter((c: { code: string }) => c.code === tag)) await api.delete(`${MAIN_API}/curricula/${c.id}`)
      const depts = await json(await api.get(`${MAIN_API}/departments`), 200)
      for (const d of depts.items.filter((d: { code: string }) => d.code === tag)) await api.delete(`${MAIN_API}/departments/${d.id}`)
    }
  })
})

test.describe('super admin: users', () => {
  test.use({ storageState: statePath('super') })

  test('creates a scoped admin (department + curriculum scope), edits and deletes it', async ({ page }) => {
    const email = `${unique('e2e.').toLowerCase()}@chula.ac.th`
    const api = await apiAs('super')
    try {
      await page.goto('/users')
      await page.getByRole('button', { name: 'Add New User' }).click()
      const dialog = page.getByRole('dialog')
      await dialog.getByLabel('Email').fill(email)
      await dialog.getByLabel('Role').selectOption({ label: 'Admin' })
      await dialog.getByRole('checkbox', { name: /CS — Computer Science/ }).check()
      await dialog.getByRole('checkbox', { name: /MATH-B — B\.Sc\. Mathematics/ }).check()
      await dialog.getByRole('button', { name: /Create User|Save/ }).click()
      await expect(page.getByRole('row', { name: new RegExp(email.replace(/\./g, '\\.')) })).toContainText(/admin/i)

      const created = (await json(await api.get(`${MAIN_API}/users?search=${encodeURIComponent(email)}`), 200)).items[0]
      expect(created.role).toBe('admin')
      expect(created.admin_scope.departments.map((d: { code: string }) => d.code)).toEqual(['CS'])
      expect(created.admin_scope.curricula.map((c: { code: string }) => c.code)).toEqual(['MATH-B'])

      // the protected (non-editable) super admin has no edit button for the viewer's own row
      await expect(page.getByRole('row', { name: /super1@chula\.ac\.th/ }).getByRole('button', { name: /Edit/ })).toHaveCount(0)

      await page.getByRole('row', { name: new RegExp(email.replace(/\./g, '\\.')) }).getByRole('button', { name: /Edit/ }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete User' }).click()
      await confirmDelete(page)
      await expect(page.getByRole('row', { name: new RegExp(email.replace(/\./g, '\\.')) })).toHaveCount(0)
    } finally {
      const users = await json(await api.get(`${MAIN_API}/users?search=${encodeURIComponent(email)}`), 200)
      for (const u of users.items.filter((u: { email: string }) => u.email === email)) await api.delete(`${MAIN_API}/users/${u.id}`)
    }
  })
})

test.describe('admin: users', () => {
  test.use({ storageState: statePath('admin') })

  test('admin creates an instructor, sees it listed, then deletes it', async ({ page }) => {
    const email = `${unique('e2e.').toLowerCase()}@chula.ac.th`
    const api = await apiAs('admin')
    try {
      await page.goto('/users')
      await page.getByRole('button', { name: 'Add New User' }).click()
      const dialog = page.getByRole('dialog')
      await dialog.getByLabel('Email').fill(email)
      await dialog.getByLabel('Department').selectOption({ label: 'Computer Science' })
      await dialog.getByLabel('Curriculum').selectOption({ label: 'B.Sc. Computer Science' })
      await expect(dialog.getByLabel('Role').locator('option')).toHaveText(['Instructor', 'Staff']) // nothing above instructor/staff
      await dialog.getByLabel('Role').selectOption({ label: 'Instructor' })
      await dialog.getByRole('button', { name: /Create User|Save/ }).click()
      const row = page.getByRole('row', { name: new RegExp(email.replace(/\./g, '\\.')) })
      await expect(row).toBeVisible()
      await expect(row).toContainText(/instructor/i)

      // domain not allowed → error stays in the modal
      await page.getByRole('button', { name: 'Add New User' }).click()
      await page.getByRole('dialog').getByLabel('Email').fill('someone@gmail.com')
      await page.getByRole('dialog').getByLabel('Department').selectOption({ label: 'Computer Science' })
      await page.getByRole('dialog').getByRole('button', { name: /Create User|Save/ }).click()
      await expect(page.getByRole('dialog').getByRole('alert')).toBeVisible()
      await page.getByRole('dialog').getByRole('button', { name: 'Cancel' }).click()

      await row.getByRole('button', { name: /Edit/ }).click()
      await page.getByRole('dialog').getByRole('button', { name: 'Delete User' }).click()
      await confirmDelete(page)
      await expect(row).toHaveCount(0)
    } finally {
      const users = await json(await api.get(`${MAIN_API}/users?search=${encodeURIComponent(email)}`), 200)
      for (const u of users.items.filter((u: { email: string }) => u.email === email)) await api.delete(`${MAIN_API}/users/${u.id}`)
    }
  })

  test('admin cannot create another admin; sees no MATH offerings (API)', async () => {
    const api = await apiAs('admin')
    const rogue = await api.post(`${MAIN_API}/users`, { data: { email: 'rogue-admin@chula.ac.th', role: 'admin', department_id: null, curriculum_id: null } })
    expect([400, 403]).toContain(rogue.status())
    const sup = await apiAs('super')
    const math = (await json(await sup.get(`${MAIN_API}/departments`), 200)).items.find((d: { code: string }) => d.code === 'MATH')
    const offerings = await json(await api.get(`${MAIN_API}/offerings?academic_year=2026&department_id=${math.id}`), 200)
    expect(offerings.items).toHaveLength(0)
  })
})

test.describe('admin limited to one curriculum', () => {
  test.use({ storageState: statePath('admin2') })

  test('cannot create a course in a curriculum outside the scope', async () => {
    const api = await apiAs('admin2')
    const sup = await apiAs('super')
    const cs = (await json(await sup.get(`${MAIN_API}/curricula`), 200)).items.find((c: { code: string }) => c.code === 'CS-B')
    const res = await api.post(`${MAIN_API}/offerings`, {
      data: { course_code: '9999999', course_name: 'Nope', curriculum_id: cs.id, academic_year: 2026, semester: 1 },
    })
    expect(res.status()).toBe(403)
  })
})
