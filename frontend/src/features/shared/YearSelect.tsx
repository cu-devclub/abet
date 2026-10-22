import { CalendarDays } from 'lucide-react'
import { SearchableSelect } from '../../components/ui/SearchableSelect'

interface YearSelectProps {
  value: number | undefined
  years: number[]
  onChange: (year: number) => void
}

export function YearSelect({ value, years, onChange }: YearSelectProps) {
  const options = value !== undefined && !years.includes(value) ? [value, ...years] : years
  return (
    <SearchableSelect
      label="Year"
      value={value === undefined ? '' : String(value)}
      onChange={(next) => onChange(Number(next))}
      options={options.map((year) => ({
        value: String(year),
        label: String(year),
      }))}
      allLabel="…"
      icon={CalendarDays}
      required
      searchable={options.length > 8}
    />
  )
}
