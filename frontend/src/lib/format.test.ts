import { elapsedText, formatBytes, formatDateTime, semesterLabel, semesterText } from './format'

describe('formatDateTime', () => {
  it('formats local time as DD/MM/YYYY HH:mm', () => {
    const d = new Date(2026, 9, 5, 7, 3)
    expect(formatDateTime(d.toISOString())).toBe('05/10/2026 07:03')
  })

  it('handles missing and invalid values', () => {
    expect(formatDateTime(null)).toBe('-')
    expect(formatDateTime('nope')).toBe('-')
  })
})

describe('semester labels', () => {
  it('maps 3 to Summer', () => {
    expect(semesterLabel(1)).toBe('1')
    expect(semesterLabel(2)).toBe('2')
    expect(semesterLabel(3)).toBe('Summer')
    expect(semesterText(3)).toBe('Summer')
    expect(semesterText(2)).toBe('Sem 2')
  })
})

describe('formatBytes', () => {
  it('scales units', () => {
    expect(formatBytes(512)).toBe('512 B')
    expect(formatBytes(245760)).toBe('240.0 KB')
    expect(formatBytes(5 * 1024 * 1024)).toBe('5.0 MB')
  })
})

describe('elapsedText', () => {
  const now = Date.parse('2026-10-10T12:00:00Z')
  it('formats minutes and hours', () => {
    expect(elapsedText('2026-10-10T11:57:00Z', now)).toBe('3 min')
    expect(elapsedText('2026-10-10T10:55:00Z', now)).toBe('1 h 5 min')
    expect(elapsedText('2026-10-10T10:00:00Z', now)).toBe('2 h')
  })
  it('is empty under a minute or for bad input', () => {
    expect(elapsedText('2026-10-10T11:59:30Z', now)).toBe('')
    expect(elapsedText('nope', now)).toBe('')
  })
})
