import { validateCover } from './validateCover'

const MAX = 20 * 1024 * 1024

describe('validateCover', () => {
  it('accepts a PDF under the limit', () => {
    expect(validateCover({ name: 'cover.PDF', type: '', size: 1000 }, MAX)).toBeNull()
    expect(validateCover({ name: 'x', type: 'application/pdf', size: 1000 }, MAX)).toBeNull()
  })

  it('rejects non-PDF files', () => {
    expect(validateCover({ name: 'cover.docx', type: '', size: 1000 }, MAX)).toMatch(/valid PDF/)
  })

  it('rejects files over the configured limit', () => {
    expect(validateCover({ name: 'cover.pdf', type: 'application/pdf', size: 21 * 1024 * 1024 }, MAX)).toMatch(/at most 20\.0 MB/)
  })
})

describe('validateCover with a custom limit', () => {
  it('uses the given limit in the message', () => {
    expect(validateCover({ name: 'c.pdf', type: 'application/pdf', size: 11 * 1024 * 1024 }, 10 * 1024 * 1024)).toMatch(/at most 10\.0 MB/)
  })
})
