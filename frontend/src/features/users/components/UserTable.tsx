import { Edit2, Lock } from 'lucide-react'
import { RoleBadge } from '../../../components/ui/Badge'
import { ColumnFilterTh, type ColumnFilterOption } from '../../../components/ui/ColumnFilterPopover'
import { EmptyState } from '../../../components/ui/EmptyState'
import { IconButton } from '../../../components/ui/IconButton'
import { SortableTh, TableCard, Th } from '../../../components/ui/Table'
import { Button } from '../../../components/ui/Button'
import { displayName } from '../../../lib/format'
import type { SortState } from '../../../lib/useSort'
import type { Role, User } from '../../auth/types'

export type UserSortField = 'name' | 'email' | 'department' | 'role'

const ROLE_OPTIONS: ColumnFilterOption<Role>[] = [
  { value: 'super_admin', label: 'Super Admin', chip: { text: 'SUPER', className: 'bg-purple-100 text-purple-800 border-purple-200' } },
  { value: 'admin', label: 'Admin', chip: { text: 'ADMIN', className: 'bg-blue-100 text-blue-800 border-blue-200' } },
  { value: 'instructor', label: 'Instructor', chip: { text: 'INSTRUCTOR', className: 'bg-teal-100 text-teal-800 border-teal-200' } },
  { value: 'staff', label: 'Staff', chip: { text: 'STAFF', className: 'bg-gray-100 text-gray-800 border-gray-200' } },
]

function affiliation(user: User): string {
  if (user.role === 'admin' && user.admin_scope) {
    const names = [...user.admin_scope.departments.map((d) => d.name), ...user.admin_scope.curricula.map((c) => c.name)]
    if (names.length > 0) return names.join(', ')
  }
  return user.department?.name ?? '-'
}

interface UserTableProps {
  users: User[]
  sort: SortState<UserSortField>
  roles: Role[]
  onRolesChange: (roles: Role[]) => void
  viewerIsAdmin: boolean
  onEdit: (user: User) => void
  onClearFilters: () => void
}

export function UserTable({ users, sort, roles, onRolesChange, viewerIsAdmin, onEdit, onClearFilters }: UserTableProps) {
  return (
    <TableCard>
      <thead className="bg-gray-50">
        <tr>
          <SortableTh label="Name" field="name" sort={sort} />
          <SortableTh label="Email" field="email" sort={sort} />
          <SortableTh label="Affiliation" field="department" sort={sort} />
          <ColumnFilterTh label="Role" field="role" sort={sort} options={ROLE_OPTIONS} selected={roles} onChange={onRolesChange} popoverWidth="w-56" />
          <Th align="right">Action</Th>
        </tr>
      </thead>
      <tbody className="bg-white divide-y divide-gray-200">
        {users.map((user) => (
          <tr key={user.id} className="hover:bg-gray-50 transition-colors">
            <td className="px-5 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
              <div className="flex items-center gap-2">
                <span>{displayName(user)}</span>
                {!user.editable && viewerIsAdmin && (
                  <span
                    className="text-[10px] font-semibold bg-gray-100 text-gray-500 border border-gray-200 px-1.5 rounded"
                    title="Admins cannot edit this role"
                  >
                    Protected
                  </span>
                )}
              </div>
            </td>
            <td className="px-5 py-4 whitespace-nowrap text-sm text-gray-500">{user.email}</td>
            <td className="px-5 py-4 whitespace-nowrap text-sm text-gray-500">
              <div className="max-w-[220px] truncate" title={affiliation(user)}>
                {affiliation(user)}
              </div>
            </td>
            <td className="px-5 py-4 whitespace-nowrap text-sm">
              <RoleBadge role={user.role} />
            </td>
            <td className="px-5 py-4 whitespace-nowrap text-right text-sm font-medium">
              {user.editable ? (
                <IconButton icon={Edit2} label="Edit User" size={13} onClick={() => onEdit(user)} />
              ) : (
                <span
                  className="inline-flex items-center justify-center p-1.5 text-gray-400 bg-gray-100 border border-gray-200 rounded-md cursor-not-allowed select-none"
                  title="You cannot edit this user"
                  role="img"
                  aria-label="Locked: you cannot edit this user"
                >
                  <Lock size={13} />
                </span>
              )}
            </td>
          </tr>
        ))}
        {users.length === 0 && (
          <tr>
            <td colSpan={5}>
              <EmptyState
                action={
                  <Button variant="link" onClick={onClearFilters}>
                    Clear filters
                  </Button>
                }
              >
                No users found matching your search or role filter.
              </EmptyState>
            </td>
          </tr>
        )}
      </tbody>
    </TableCard>
  )
}
