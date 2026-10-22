import { groupIssues } from './issues'
import type { Issue } from './types'

const issue = (severity: Issue['severity'], code: string): Issue => ({
  severity,
  stage: 'structure',
  code,
  title: code,
  description: '',
  location: '',
  suggested_fix: '',
})

describe('groupIssues', () => {
  it('puts errors before warnings and keeps order inside each group', () => {
    const groups = groupIssues([issue('warning', 'W1'), issue('error', 'E1'), issue('error', 'E2'), issue('warning', 'W2')])
    expect(groups.map((g) => g.severity)).toEqual(['error', 'warning'])
    expect(groups[0].items.map((i) => i.issue.code)).toEqual(['E1', 'E2'])
    expect(groups[1].items.map((i) => i.issue.code)).toEqual(['W1', 'W2'])
  })

  it('omits empty groups', () => {
    expect(groupIssues([issue('warning', 'W1')]).map((g) => g.severity)).toEqual(['warning'])
    expect(groupIssues([])).toEqual([])
  })
})
