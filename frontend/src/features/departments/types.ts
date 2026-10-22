export interface Department {
  id: string
  code: string
  name: string
  curriculum_count: number
  created_at: string
  updated_at: string
}

export interface DepartmentInput {
  code: string
  name: string
}
