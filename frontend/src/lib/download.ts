import { downloadBlob, type QueryParams } from './http'

export function saveBlob(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}

/** Fetch a file from the main API (cookie auth) and hand it to the browser. Returns the saved file name. */
export async function downloadAndSave(
  path: string,
  fallbackName: string,
  init?: { method?: 'GET' | 'POST'; body?: unknown; params?: QueryParams },
): Promise<string> {
  const { blob, filename } = await downloadBlob(path, init)
  const name = filename ?? fallbackName
  saveBlob(blob, name)
  return name
}
