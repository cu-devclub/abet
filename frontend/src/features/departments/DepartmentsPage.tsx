import { useState } from 'react'
import { Edit2, Plus } from 'lucide-react'
import { PageHeader } from '../../components/layout/PageHeader'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { IconButton } from '../../components/ui/IconButton'
import { PageSpinner } from '../../components/ui/Spinner'
import { Alert } from '../../components/ui/Alert'
import { TableCard, Th } from '../../components/ui/Table'
import { plural } from '../../lib/format'
import { useDepartments } from './api'
import { DepartmentFormModal } from './components/DepartmentFormModal'
import type { Department } from './types'

type Editing = { department: Department | null } | null

export function DepartmentsPage() {
  const { data: departments, isPending, error } = useDepartments()
  const [editing, setEditing] = useState<Editing>(null)

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Department Management"
        actions={
          <Button icon={Plus} onClick={() => setEditing({ department: null })}>
            Add New Department
          </Button>
        }
      />

      {error && <Alert>{error.message}</Alert>}
      {isPending ? (
        <PageSpinner />
      ) : (
        <TableCard>
          <thead className="bg-gray-50">
            <tr>
              <Th className="px-6">Department Code</Th>
              <Th className="px-6">Department Name</Th>
              <Th className="px-6">Curriculums</Th>
              <Th className="px-6" align="right">
                Action
              </Th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {(departments ?? []).map((dept) => (
              <tr key={dept.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-6 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                  <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">{dept.code}</span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{dept.name}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {dept.curriculum_count} {plural(dept.curriculum_count, 'Curriculum', 'Curriculums')}
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <IconButton icon={Edit2} label="Edit Department" size={13} onClick={() => setEditing({ department: dept })} />
                </td>
              </tr>
            ))}
            {departments?.length === 0 && (
              <tr>
                <td colSpan={4}>
                  <EmptyState>No departments found. Click "Add New Department" to create one.</EmptyState>
                </td>
              </tr>
            )}
          </tbody>
        </TableCard>
      )}

      {editing && <DepartmentFormModal key={editing.department?.id ?? 'new'} department={editing.department} onClose={() => setEditing(null)} />}
    </div>
  )
}
