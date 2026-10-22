import { apiGet, apiPost, ApiError, buildQuery, filenameFromDisposition, setUnauthorizedHandler } from './http'

const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })

afterEach(() => {
  setUnauthorizedHandler(null)
  vi.unstubAllGlobals()
})

describe('buildQuery', () => {
  it('skips empty values', () => {
    expect(buildQuery({ a: 1, b: '', c: undefined, d: null, e: 'x y' })).toBe('?a=1&e=x+y')
    expect(buildQuery({})).toBe('')
  })
})

describe('filenameFromDisposition', () => {
  it('reads plain and RFC 5987 names', () => {
    expect(filenameFromDisposition('attachment; filename="a_2026_S1_portfolio.pdf"')).toBe('a_2026_S1_portfolio.pdf')
    expect(filenameFromDisposition("attachment; filename*=UTF-8''%E0%B8%81.pdf")).toBe('ก.pdf')
    expect(filenameFromDisposition(null)).toBeNull()
  })
})

describe('request helpers', () => {
  it('sends cookies and parses JSON', async () => {
    const fetchMock = vi.fn(async () => json({ items: [1] }))
    vi.stubGlobal('fetch', fetchMock)
    await expect(apiGet('/offerings', { academic_year: 2026 })).resolves.toEqual({ items: [1] })
    expect(fetchMock).toHaveBeenCalledWith('/auth/api/v1/offerings?academic_year=2026', expect.objectContaining({ credentials: 'include', method: 'GET' }))
  })

  it('POSTs JSON bodies', async () => {
    const fetchMock = vi.fn(async () => json({ deleted: 2 }))
    vi.stubGlobal('fetch', fetchMock)
    await apiPost('/offerings/bulk-delete', { ids: ['a', 'b'] })
    const init = (fetchMock.mock.calls[0] as unknown as [string, RequestInit])[1]
    expect(init.body).toBe('{"ids":["a","b"]}')
    expect(init.headers).toEqual({ 'Content-Type': 'application/json' })
  })

  it('turns {"error"} bodies into ApiError', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => json({ error: 'department is referenced by curricula or users' }, 409)))
    await expect(apiGet('/departments')).rejects.toMatchObject({
      name: 'ApiError',
      status: 409,
      message: 'department is referenced by curricula or users',
    })
  })

  it('keeps the body so 422 import plans stay readable', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => json({ summary: {}, errors: [{ row: 3, message: 'bad' }] }, 422)))
    const error = await apiPost('/offerings/import/apply').catch((e: unknown) => e)
    expect(error).toBeInstanceOf(ApiError)
    expect((error as ApiError).body).toMatchObject({ errors: [{ row: 3 }] })
  })

  it('calls the unauthorized handler on 401 unless skipped', async () => {
    const handler = vi.fn()
    setUnauthorizedHandler(handler)
    vi.stubGlobal('fetch', vi.fn(async () => json({ error: 'unauthenticated' }, 401)))
    await expect(apiGet('/portfolios')).rejects.toMatchObject({ status: 401 })
    expect(handler).toHaveBeenCalledTimes(1)
    await expect(apiGet('/me', undefined, { skipUnauthorized: true })).rejects.toMatchObject({ status: 401 })
    expect(handler).toHaveBeenCalledTimes(1)
  })

  it('hides server error details', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('boom', { status: 500 })))
    await expect(apiGet('/x')).rejects.toThrow(/Something went wrong/)
  })
})
