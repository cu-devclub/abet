import type { Ref } from '../auth/types'

export interface CoverInfo {
  filename: string
  size_bytes: number
  updated_at: string
}

export interface Curriculum {
  id: string
  code: string
  name: string
  department: Ref
  cover: CoverInfo | null
  created_at: string
  updated_at: string
}

export interface CurriculumInput {
  code: string
  name: string
  department_id: string
}
