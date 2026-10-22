import type { Curriculum } from '../curricula/types'
import type { Department } from '../departments/types'
import { scopeOrg } from './scope'

const ref = (id: string) => ({ id, code: id.toUpperCase(), name: id })
const dept = (id: string) => ({ ...ref(id), curriculum_count: 0, created_at: '', updated_at: '' }) as Department
const curr = (id: string, deptId: string) =>
  ({ ...ref(id), department: ref(deptId), cover: null, created_at: '', updated_at: '' }) as Curriculum

const departments = [dept('d1'), dept('d2'), dept('d3')]
const curricula = [curr('c1', 'd1'), curr('c2', 'd1'), curr('c3', 'd2'), curr('c4', 'd3')]

describe('scopeOrg', () => {
  it('returns everything without a scope', () => {
    const result = scopeOrg(null, departments, curricula)
    expect(result.departments).toHaveLength(3)
    expect(result.curricula).toHaveLength(4)
  })

  it('includes all curricula of scoped departments', () => {
    const result = scopeOrg({ departments: [ref('d1')], curricula: [] }, departments, curricula)
    expect(result.departments.map((d) => d.id)).toEqual(['d1'])
    expect(result.curricula.map((c) => c.id)).toEqual(['c1', 'c2'])
  })

  it('exposes the parent department of a scoped curriculum', () => {
    const result = scopeOrg({ departments: [], curricula: [ref('c3')] }, departments, curricula)
    expect(result.departments.map((d) => d.id)).toEqual(['d2'])
    expect(result.curricula.map((c) => c.id)).toEqual(['c3'])
  })
})
