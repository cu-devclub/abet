const pad = (n: number) => String(n).padStart(2, '0')

/** RFC3339 timestamp → `DD/MM/YYYY HH:mm` in the browser's local time. */
export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return '-'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return '-'
  return `${pad(d.getDate())}/${pad(d.getMonth() + 1)}/${d.getFullYear()} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

/** Stored semester 1/2/3 → label "1" / "2" / "Summer" (A1). */
export function semesterLabel(semester: number): string {
  return semester === 3 ? 'Summer' : String(semester)
}

/** Badge text: "Sem 1", "Sem 2", "Summer". */
export function semesterText(semester: number): string {
  return semester === 3 ? 'Summer' : `Sem ${semester}`
}

export const SEMESTER_OPTIONS = [
  { value: 1, label: 'Semester 1' },
  { value: 2, label: 'Semester 2' },
  { value: 3, label: 'Summer' },
] as const

export function roleLabel(role: string): string {
  return role.replace('_', ' ')
}

export function plural(count: number, singular: string, pluralForm = `${singular}s`): string {
  return count === 1 ? singular : pluralForm
}

/** Users have `name: ""` until their first Google sign-in → show the e-mail instead. */
export function displayName(person: { name: string; email: string }): string {
  return person.name.trim() || person.email
}

/** "3 min" / "2 h 5 min" / "2 d" since `iso`; empty for under a minute or an unparseable date. Re-evaluated on each render (the list polls). */
export function elapsedText(iso: string, now: number = Date.now()): string {
  const started = Date.parse(iso)
  if (Number.isNaN(started)) return ''
  const minutes = Math.floor((now - started) / 60_000)
  if (minutes < 1) return ''
  if (minutes < 60) return `${minutes} min`
  if (minutes >= 24 * 60) return `${Math.floor(minutes / (24 * 60))} d`
  const rest = minutes % 60
  return rest ? `${Math.floor(minutes / 60)} h ${rest} min` : `${Math.floor(minutes / 60)} h`
}
