import type { Ref, Role } from '../auth/types'

export interface Offering {
  id: string
  course_id: string
  course_code: string
  course_name: string
  academic_year: number
  semester: number
  curriculum: Ref
  department: Ref
  instructor_count: number
  staff_count: number
  has_submissions: boolean
  created_at: string
  updated_at: string
}

export interface OfferingListParams {
  academic_year: number
  department_id?: string
  curriculum_id?: string
}

export interface OfferingInput {
  course_code: string
  course_name: string
  curriculum_id: string
  academic_year: number
  semester: number
}

export interface YearsResponse {
  items: number[]
  current: number
}

export type AccessSource = 'manual' | 'csv' | 'default'

export interface AccessEntry {
  user_id: string
  name: string
  email: string
  role: Role
  can_upload: boolean
  can_download: boolean
  is_default: boolean
  default_reason: string | null
  source: AccessSource
}

export interface AccessInput {
  can_upload: boolean
  can_download: boolean
}

export type CourseAction = 'create' | 'update' | 'keep' | 'delete' | 'keep_has_uploads'
export type AccessAction = 'add' | 'update' | 'remove'
export type UserAction = 'create' | 'raise_role'

export interface ImportSummary {
  rows: number
  courses_create: number
  courses_update: number
  courses_keep: number
  courses_delete: number
  courses_kept_with_uploads: number
  access_add: number
  access_update: number
  access_remove: number
  users_create: number
  users_role_raised: number
}

export interface ImportCourseChange {
  action: CourseAction
  course_code: string
  course_name: string
  semester: number
  note: string
}

export interface ImportAccessChange {
  action: AccessAction
  course_code: string
  semester: number
  email: string
  role: Role
  can_upload: boolean
  can_download: boolean
}

export interface ImportUserChange {
  action: UserAction
  email: string
  role: Role
}

export interface ImportRowError {
  row: number
  message: string
}

export interface ImportPlan {
  applied: boolean
  summary: ImportSummary
  courses: ImportCourseChange[]
  access: ImportAccessChange[]
  users: ImportUserChange[]
  errors: ImportRowError[]
}

export interface ImportRequest {
  file: File
  curriculumId: string
  academicYear: number
}
