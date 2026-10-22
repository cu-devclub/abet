import { mergeRowOrg } from './useOrgOptions'

const dept = (id: string) => ({ id, code: id.toUpperCase(), name: `${id} name`, curriculum_count: 1, created_at: '', updated_at: '' })
const curr = (id: string, departmentId: string) => ({
  id,
  code: id.toUpperCase(),
  name: `${id} name`,
  department: { id: departmentId, code: departmentId.toUpperCase(), name: `${departmentId} name` },
  cover: null,
  created_at: '',
  updated_at: '',
})
const ref = (id: string) => ({ id, code: id.toUpperCase(), name: `${id} name` })

describe('mergeRowOrg', () => {
  it('adds the departments/curricula of rows outside the caller’s own org, without duplicates', () => {
    const merged = mergeRowOrg(
      [dept('cs')],
      [curr('csb', 'cs')],
      [
        { department: ref('cs'), curriculum: ref('csb') },
        { department: ref('math'), curriculum: ref('mathb') },
        { department: ref('math'), curriculum: ref('mathb') },
      ],
    )
    expect(merged.departments.map((d) => d.id)).toEqual(['cs', 'math'])
    expect(merged.curricula.map((c) => [c.id, c.department.id])).toEqual([['csb', 'cs'], ['mathb', 'math']])
  })
})
