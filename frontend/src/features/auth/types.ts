export type Role = 'super_admin' | 'admin' | 'instructor' | 'staff'

export interface Ref {
  id: string
  code: string
  name: string
}

export interface AdminScope {
  departments: Ref[]
  curricula: Ref[]
}

export interface User {
  id: string
  email: string
  name: string
  picture_url: string | null
  role: Role
  department: Ref | null
  curriculum: Ref | null
  admin_scope: AdminScope | null
  last_login_at: string | null
  /** Caller may edit/delete this user. */
  editable: boolean
  created_at: string
  updated_at: string
}

export interface Capabilities {
  manage_org: boolean
  manage_users: boolean
  manage_courses: boolean
  manage_covers: boolean
}

export type Capability = keyof Capabilities

/** Dev-only "act as user" state (D28). Missing or `available: false` = feature off. */
export interface ActAs {
  available: boolean
  /** The real super admin while acting as someone else, otherwise null. */
  real_user: Pick<User, 'id' | 'email' | 'name' | 'role'> | null
}

export interface Me {
  user: User
  capabilities: Capabilities
  act_as?: ActAs
}

/** Row of `GET /dev/act-as/users`. */
export interface ActAsUser {
  id: string
  email: string
  name: string
  role: Role
  department_id: string | null
  curriculum_id: string | null
}
