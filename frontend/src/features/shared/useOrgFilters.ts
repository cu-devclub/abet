import { useCallback, useMemo } from 'react'
import { useSearchParams } from 'react-router'
import { useYears } from '../courses/api'

const KEYS = { department: 'dept', curriculum: 'cur', year: 'year', semesters: 'sem', search: 'q' } as const

function parseSemesters(raw: string | null): number[] {
  if (!raw) return []
  return raw
    .split(',')
    .map(Number)
    .filter((n) => n === 1 || n === 2 || n === 3)
}

/**
 * Department / curriculum / year / semester / search filters kept in the URL
 * search params (shareable, survive reload). Year defaults to `current` from GET /offerings/years.
 */
export function useOrgFilters() {
  const [params, setParams] = useSearchParams()
  const years = useYears()

  const departmentId = params.get(KEYS.department) ?? ''
  const curriculumId = params.get(KEYS.curriculum) ?? ''
  const search = params.get(KEYS.search) ?? ''
  const semesterParam = params.get(KEYS.semesters)
  const semesters = useMemo(() => parseSemesters(semesterParam), [semesterParam])
  const yearParam = Number(params.get(KEYS.year))
  const year = Number.isInteger(yearParam) && yearParam > 0 ? yearParam : years.data?.current

  const update = useCallback(
    (changes: Partial<Record<keyof typeof KEYS, string | null>>) =>
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          for (const [name, value] of Object.entries(changes)) {
            const key = KEYS[name as keyof typeof KEYS]
            if (value) next.set(key, value)
            else next.delete(key)
          }
          return next
        },
        { replace: true },
      ),
    [setParams],
  )

  return {
    departmentId,
    curriculumId,
    year,
    years: years.data?.items ?? [],
    currentYear: years.data?.current,
    semesters,
    search,
    /** Changing department drops the curriculum unless it still belongs to the new department. */
    setDepartment: (id: string, keepCurriculum = false) =>
      update({ department: id, ...(keepCurriculum ? {} : { curriculum: null }) }),
    setCurriculum: (id: string) => update({ curriculum: id }),
    setYear: (value: number) => update({ year: String(value) }),
    setSemesters: (values: number[]) => update({ semesters: values.length ? values.join(',') : null }),
    setSearch: (value: string) => update({ search: value }),
    clearAll: () => update({ department: null, curriculum: null, semesters: null, search: null }),
    hasActive: Boolean(departmentId || curriculumId || semesters.length || search),
  }
}

export type OrgFilters = ReturnType<typeof useOrgFilters>
