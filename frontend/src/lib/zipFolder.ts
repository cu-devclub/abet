import JSZip from 'jszip'

const SKIPPED_FILE_NAMES = new Set(['.ds_store', 'thumbs.db'])
const SKIPPED_DIRECTORIES = new Set(['__macosx'])

export interface FolderFile {
  /** Path inside the zip, e.g. `MyFolder/data/a.csv` (top-level folder name preserved). */
  path: string
  file: File
}

/** Normalise separators and strip leading/trailing slashes and `.` segments. */
export function normalizePath(path: string): string {
  return path
    .replace(/\\/g, '/')
    .split('/')
    .filter((segment) => segment !== '' && segment !== '.')
    .join('/')
}

/** True for macOS/Windows metadata that must not end up in the portfolio zip. */
export function shouldSkipPath(path: string): boolean {
  const segments = normalizePath(path).split('/')
  const name = segments[segments.length - 1]?.toLowerCase() ?? ''
  if (SKIPPED_FILE_NAMES.has(name)) return true
  return segments.some((segment) => SKIPPED_DIRECTORIES.has(segment.toLowerCase()))
}

export function topLevelFolder(paths: readonly string[]): string | null {
  for (const path of paths) {
    const segments = normalizePath(path).split('/')
    if (segments.length > 1) return segments[0]
  }
  return null
}

export function zipFileName(folder: string | null): string {
  return `${folder ?? 'portfolio'}.zip`
}

export function toFolderFiles(entries: readonly FolderFile[]): FolderFile[] {
  return entries
    .map((entry) => ({ ...entry, path: normalizePath(entry.path) }))
    .filter((entry) => entry.path !== '' && !shouldSkipPath(entry.path))
}

export async function buildZip(
  entries: readonly FolderFile[],
  onProgress?: (percent: number) => void,
): Promise<File> {
  const files = toFolderFiles(entries)
  if (files.length === 0) throw new Error('The selected folder has no files to upload.')
  const zip = new JSZip()
  for (const { path, file } of files) zip.file(path, file)
  const blob = await zip.generateAsync({ type: 'blob', compression: 'DEFLATE' }, (meta) =>
    onProgress?.(Math.round(meta.percent)),
  )
  const name = zipFileName(topLevelFolder(files.map((f) => f.path)))
  return new File([blob], name, { type: 'application/zip' })
}

/** `<input webkitdirectory>` FileList → zip. Paths come from `webkitRelativePath`. */
export function zipFromFileList(list: FileList | readonly File[], onProgress?: (percent: number) => void): Promise<File> {
  const entries = Array.from(list).map((file) => ({ path: file.webkitRelativePath || file.name, file }))
  return buildZip(entries, onProgress)
}

function readEntries(reader: FileSystemDirectoryReader): Promise<FileSystemEntry[]> {
  return new Promise((resolve, reject) => reader.readEntries(resolve, reject))
}

function fileOf(entry: FileSystemFileEntry): Promise<File> {
  return new Promise((resolve, reject) => entry.file(resolve, reject))
}

async function collect(entry: FileSystemEntry, out: FolderFile[]): Promise<void> {
  if (shouldSkipPath(entry.fullPath)) return
  if (entry.isFile) {
    out.push({ path: entry.fullPath, file: await fileOf(entry as FileSystemFileEntry) })
    return
  }
  const reader = (entry as FileSystemDirectoryEntry).createReader()
  // readEntries returns results in batches; keep reading until it yields an empty batch.
  for (let batch = await readEntries(reader); batch.length > 0; batch = await readEntries(reader)) {
    for (const child of batch) await collect(child, out)
  }
}

export function droppedEntries(dt: DataTransfer): FileSystemEntry[] {
  const entries: FileSystemEntry[] = []
  // webkitGetAsEntry must be called synchronously during the drop event.
  for (const item of Array.from(dt.items)) {
    const entry = item.kind === 'file' ? item.webkitGetAsEntry() : null
    if (entry) entries.push(entry)
  }
  return entries
}

export function hasDroppedDirectory(entries: readonly FileSystemEntry[]): boolean {
  return entries.some((entry) => entry.isDirectory)
}

/** Dropped folder(s) → zip. Files dropped alongside are included under their own names. */
export async function zipFromEntries(
  entries: readonly FileSystemEntry[],
  onProgress?: (percent: number) => void,
): Promise<File> {
  const collected: FolderFile[] = []
  for (const entry of entries) await collect(entry, collected)
  return buildZip(collected, onProgress)
}
