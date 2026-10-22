import type { Role } from '../auth/types'

export interface AdminScopeInput {
  department_ids: string[]
  curriculum_ids: string[]
}

export interface UserCreateInput {
  email: string
  role: Role
  department_id: string | null
  curriculum_id: string | null
  admin_scope: AdminScopeInput | null
}

/** Email is immutable; every other field is optional. `admin_scope` is only sent by super admins. */
export type UserUpdateInput = Partial<Omit<UserCreateInput, 'email'>>

export interface UserListParams {
  search?: string
  role?: Role
  department_id?: string
}
