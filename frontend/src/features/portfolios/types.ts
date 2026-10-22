import type { Ref } from '../auth/types'

export type SubmissionStatus = 'invalid' | 'queued' | 'processing' | 'generated' | 'failed'
export type IssueSeverity = 'error' | 'warning'
export type IssueStage = 'missing' | 'structure' | 'naming' | 'csv' | 'build'

export interface Issue {
  severity: IssueSeverity
  stage: IssueStage
  code: string
  title: string
  description: string
  location: string
  suggested_fix: string
}

export interface Submission {
  id: string
  offering_id: string
  status: SubmissionStatus
  original_filename: string
  size_bytes: number
  uploaded_by: { id: string; name: string; email: string } | null
  error_count: number
  warning_count: number
  created_at: string
  completed_at: string | null
  /** Only present on GET /submissions/{id} and the upload response; [] in lists. */
  issues: Issue[]
}

export interface PortfolioRow {
  offering_id: string
  course_code: string
  course_name: string
  academic_year: number
  semester: number
  curriculum: Ref
  department: Ref
  can_upload: boolean
  can_download: boolean
  latest_submission: Omit<Submission, 'issues'> | null
  pdf_available: boolean
}

export interface PortfolioListParams {
  academic_year: number
  department_id?: string
  curriculum_id?: string
}
