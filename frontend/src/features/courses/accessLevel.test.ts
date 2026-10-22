import { accessInputOf, accessLevelOf } from './accessLevel'

describe('access level mapping', () => {
  it('maps flags to a level', () => {
    expect(accessLevelOf({ can_upload: true, can_download: true })).toBe('both')
    expect(accessLevelOf({ can_upload: false, can_download: true })).toBe('download')
    expect(accessLevelOf({ can_upload: true, can_download: false })).toBe('upload')
  })

  it('round-trips and never produces an empty permission set', () => {
    for (const level of ['both', 'download', 'upload'] as const) {
      const input = accessInputOf(level)
      expect(input.can_upload || input.can_download).toBe(true)
      expect(accessLevelOf(input)).toBe(level)
    }
  })
})
