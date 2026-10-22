import { AUTH_API, MAIN_API } from '../config/env'

export class ApiError extends Error {
  readonly status: number
  readonly body: unknown

  constructor(status: number, message: string, body: unknown = null) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
  }
}

type QueryValue = string | number | boolean | null | undefined
export type QueryParams = Record<string, QueryValue>

export interface RequestOptions {
  /** API base; defaults to the main-service base (`/auth/api/v1`). */
  base?: string
  /** Do not trigger the global 401 handler (used by `GET /me` and logout). */
  skipUnauthorized?: boolean
  signal?: AbortSignal
}

let unauthorizedHandler: (() => void) | null = null

/** Registered once by the app providers: invalidates `me` and navigates to /login. */
export function setUnauthorizedHandler(handler: (() => void) | null): void {
  unauthorizedHandler = handler
}

export function buildQuery(params?: QueryParams): string {
  if (!params) return ''
  const search = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === '') continue
    search.set(key, String(value))
  }
  const text = search.toString()
  return text ? `?${text}` : ''
}

function errorMessage(status: number, body: unknown): string {
  if (body && typeof body === 'object' && 'error' in body && typeof body.error === 'string') {
    return body.error
  }
  return status >= 500 ? 'Something went wrong on the server. Please try again.' : `Request failed (${status})`
}

async function readBody(res: Response): Promise<unknown> {
  const text = await res.text()
  if (!text) return null
  try {
    return JSON.parse(text)
  } catch {
    return null
  }
}

function toApiError(status: number, body: unknown, opts?: RequestOptions): ApiError {
  if (status === 401 && !opts?.skipUnauthorized) unauthorizedHandler?.()
  return new ApiError(status, errorMessage(status, body), body)
}

async function request<T>(method: string, path: string, body: unknown, opts?: RequestOptions): Promise<T> {
  const init: RequestInit = { method, credentials: 'include', signal: opts?.signal }
  if (body !== undefined) {
    init.headers = { 'Content-Type': 'application/json' }
    init.body = JSON.stringify(body)
  }
  const res = await fetch(`${opts?.base ?? MAIN_API}${path}`, init)
  const data = await readBody(res)
  if (!res.ok) throw toApiError(res.status, data, opts)
  return data as T
}

export function apiGet<T>(path: string, params?: QueryParams, opts?: RequestOptions): Promise<T> {
  return request<T>('GET', `${path}${buildQuery(params)}`, undefined, opts)
}

export function apiPost<T>(path: string, body?: unknown, opts?: RequestOptions): Promise<T> {
  return request<T>('POST', path, body ?? {}, opts)
}

export function apiPatch<T>(path: string, body: unknown, opts?: RequestOptions): Promise<T> {
  return request<T>('PATCH', path, body, opts)
}

export function apiPut<T>(path: string, body: unknown, opts?: RequestOptions): Promise<T> {
  return request<T>('PUT', path, body, opts)
}

export function apiDelete<T = void>(path: string, opts?: RequestOptions): Promise<T> {
  return request<T>('DELETE', path, undefined, opts)
}

/** Request options for the auth-service (`/public/api/v1`), e.g. logout. */
export const authApi: RequestOptions = { base: AUTH_API, skipUnauthorized: true }

export function uploadWithProgress<T>(
  path: string,
  formData: FormData,
  onProgress?: (percent: number) => void,
  method: 'POST' | 'PUT' = 'POST',
): Promise<T> {
  return new Promise<T>((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open(method, `${MAIN_API}${path}`)
    xhr.withCredentials = true
    xhr.responseType = 'text'
    xhr.upload.onprogress = (event) => {
      if (event.lengthComputable) onProgress?.(Math.round((event.loaded / event.total) * 100))
    }
    xhr.onerror = () => reject(new ApiError(0, 'Network error. Check your connection and try again.'))
    xhr.onabort = () => reject(new ApiError(0, 'Upload cancelled.'))
    xhr.onload = () => {
      let data: unknown = null
      try {
        data = xhr.responseText ? JSON.parse(xhr.responseText) : null
      } catch {
        data = null
      }
      if (xhr.status >= 200 && xhr.status < 300) {
        onProgress?.(100)
        resolve(data as T)
      } else {
        reject(toApiError(xhr.status, data))
      }
    }
    xhr.send(formData)
  })
}

export interface DownloadedBlob {
  blob: Blob
  filename: string | null
}

export function filenameFromDisposition(header: string | null): string | null {
  if (!header) return null
  const star = /filename\*=UTF-8''([^;]+)/i.exec(header)
  if (star) {
    try {
      return decodeURIComponent(star[1])
    } catch {
      return star[1]
    }
  }
  const plain = /filename="?([^";]+)"?/i.exec(header)
  return plain ? plain[1] : null
}

export async function downloadBlob(
  path: string,
  init?: { method?: 'GET' | 'POST'; body?: unknown; params?: QueryParams },
): Promise<DownloadedBlob> {
  const hasBody = init?.body !== undefined
  const res = await fetch(`${MAIN_API}${path}${buildQuery(init?.params)}`, {
    method: init?.method ?? 'GET',
    credentials: 'include',
    headers: hasBody ? { 'Content-Type': 'application/json' } : undefined,
    body: hasBody ? JSON.stringify(init?.body) : undefined,
  })
  if (!res.ok) throw toApiError(res.status, await readBody(res))
  return { blob: await res.blob(), filename: filenameFromDisposition(res.headers.get('Content-Disposition')) }
}
