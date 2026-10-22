import { request, type APIRequestContext, type APIResponse } from '@playwright/test'
import { BASE_URL, MAIN_API, statePath, type RoleKey } from './env.ts'

/** Authenticated API client for a seeded role (talks to the gateway like the browser does). */
export async function apiAs(role: RoleKey): Promise<APIRequestContext> {
  return request.newContext({ baseURL: BASE_URL, storageState: statePath(role) })
}

/** Assert the status and return the parsed JSON body (null for 204). */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export async function json(res: APIResponse, expected: number): Promise<any> {
  if (res.status() !== expected) throw new Error(`${res.url()} → ${res.status()} (expected ${expected}): ${await res.text()}`)
  return res.status() === 204 ? null : await res.json()
}

export const unique = (prefix: string) => `${prefix}${Date.now().toString(36).toUpperCase()}`

export interface Org {
  dept: { id: string; code: string }
  curr: { id: string; code: string }
}

/** A throw-away department + curriculum (super admin API). */
export async function createOrg(api: APIRequestContext, tag: string): Promise<Org> {
  const dept = await json(await api.post(`${MAIN_API}/departments`, { data: { code: tag, name: `E2E Dept ${tag}` } }), 201)
  const curr = await json(await api.post(`${MAIN_API}/curricula`, { data: { code: tag, name: `E2E Curr ${tag}`, department_id: dept.id } }), 201)
  return { dept, curr }
}

/** Remove the org and every offering inside it (best effort, used in `finally`). */
export async function dropOrg(api: APIRequestContext, org: Org) {
  const offerings = await api.get(`${MAIN_API}/offerings?academic_year=2026&curriculum_id=${org.curr.id}`)
  if (offerings.ok()) {
    const ids = ((await offerings.json()).items as { id: string }[]).map((o) => o.id)
    if (ids.length > 0) await api.post(`${MAIN_API}/offerings/bulk-delete`, { data: { ids } })
  }
  await api.delete(`${MAIN_API}/curricula/${org.curr.id}`)
  await api.delete(`${MAIN_API}/departments/${org.dept.id}`)
}
