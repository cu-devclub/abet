import { useMemo, useState } from 'react'
import { Info, Plus } from 'lucide-react'
import { PageHeader } from '../../components/layout/PageHeader'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { Count, FilterToolbar, type ActiveFilter } from '../../components/ui/FilterToolbar'
import { PageSpinner } from '../../components/ui/Spinner'
import { displayName, roleLabel } from '../../lib/format'
import { sortRows, useSort } from '../../lib/useSort'
import { useMe } from '../auth/api'
import type { Role, User } from '../auth/types'
import { useOrgOptions } from '../shared/useOrgOptions'
import { useUsers } from './api'
import { UserFormModal } from './components/UserFormModal'
import { UserTable, type UserSortField } from './components/UserTable'

type Editing = { user: User | null } | null

function matches(user: User, query: string): boolean {
  const haystack = [user.name, user.email, user.department?.name, user.curriculum?.name, user.role]
  return haystack.some((value) => value?.toLowerCase().includes(query))
}

function sortValue(user: User, field: UserSortField): string {
  switch (field) {
    case 'name':
      return displayName(user)
    case 'email':
      return user.email
    case 'department':
      return user.department?.name ?? ''
    case 'role':
      return user.role
  }
}

export function UsersPage() {
  const me = useMe()
  const users = useUsers()
  const org = useOrgOptions()
  const sort = useSort<UserSortField>('name')
  const [search, setSearch] = useState('')
  const [roles, setRoles] = useState<Role[]>([])
  const [editing, setEditing] = useState<Editing>(null)

  const query = search.trim().toLowerCase()
  const rows = useMemo(() => {
    const filtered = (users.data ?? []).filter((u) => (roles.length === 0 || roles.includes(u.role)) && (!query || matches(u, query)))
    return sortRows(filtered, (u) => sortValue(u, sort.field), sort.direction)
  }, [users.data, roles, query, sort.field, sort.direction])

  const viewerIsAdmin = me.data?.user.role === 'admin'
  const clearAll = () => {
    setSearch('')
    setRoles([])
  }

  const filters: ActiveFilter[] = []
  if (roles.length > 0) {
    filters.push({ key: 'roles', label: `Roles: ${roles.map((r) => roleLabel(r).toUpperCase()).join(', ')}`, onClear: () => setRoles([]) })
  }
  if (search) filters.push({ key: 'q', label: `Search: "${search}"`, neutral: true, onClear: () => setSearch('') })

  return (
    <div className="p-6 space-y-4">
      <PageHeader
        title="User Management"
        subtitle="Manage system users, affiliations, and permission levels."
        actions={
          <Button icon={Plus} onClick={() => setEditing({ user: null })} disabled={!me.data || org.isLoading}>
            Add New User
          </Button>
        }
      />

      {viewerIsAdmin && (
        <div className="p-3 bg-blue-50 border border-blue-200 rounded-xl flex items-center justify-between text-xs text-blue-900">
          <div className="flex items-center gap-2.5">
            <Info size={18} className="text-blue-600 shrink-0" />
            <span>
              <strong>Admin Permissions:</strong> You can create and edit users with roles below Admin (<strong>Instructor</strong> and{' '}
              <strong>Staff</strong>). Super Admin and Admin accounts are view-only.
            </span>
          </div>
          <span className="bg-blue-100 text-blue-800 font-semibold px-2.5 py-0.5 rounded-full text-[10px] shrink-0 border border-blue-200 uppercase tracking-wider">
            Role Below Admin Only
          </span>
        </div>
      )}

      <FilterToolbar
        search={search}
        onSearch={setSearch}
        placeholder="Search user by name, email, affiliation, or role..."
        summary={
          <>
            Showing <Count>{rows.length}</Count> of <Count>{users.data?.length ?? 0}</Count> users
          </>
        }
        filters={filters}
        onClearAll={clearAll}
      />

      {users.error && <Alert>{users.error.message}</Alert>}
      {users.isPending ? (
        <PageSpinner />
      ) : (
        <UserTable
          users={rows}
          sort={sort}
          roles={roles}
          onRolesChange={setRoles}
          viewerIsAdmin={viewerIsAdmin}
          onEdit={(user) => setEditing({ user })}
          onClearFilters={clearAll}
        />
      )}

      {editing && me.data && (
        <UserFormModal
          key={editing.user?.id ?? 'new'}
          user={editing.user}
          me={me.data}
          departments={org.departments}
          curricula={org.curricula}
          onClose={() => setEditing(null)}
        />
      )}
    </div>
  )
}
