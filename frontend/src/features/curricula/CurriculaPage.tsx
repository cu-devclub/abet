import { useState } from 'react'
import { Edit2, Plus } from 'lucide-react'
import { PageHeader } from '../../components/layout/PageHeader'
import { Alert } from '../../components/ui/Alert'
import { Button } from '../../components/ui/Button'
import { EmptyState } from '../../components/ui/EmptyState'
import { IconButton } from '../../components/ui/IconButton'
import { PageSpinner } from '../../components/ui/Spinner'
import { TableCard, Th } from '../../components/ui/Table'
import { useDepartments } from '../departments/api'
import { useCurricula } from './api'
import { CurriculumFormModal } from './components/CurriculumFormModal'
import type { Curriculum } from './types'

type Editing = { curriculum: Curriculum | null } | null

export function CurriculaPage() {
  const curricula = useCurricula()
  const departments = useDepartments()
  const [editing, setEditing] = useState<Editing>(null)
  const noDepartments = departments.data?.length === 0

  return (
    <div className="p-6 space-y-6">
      <PageHeader
        title="Curriculum Management"
        actions={
          <Button
            icon={Plus}
            onClick={() => setEditing({ curriculum: null })}
            disabled={!departments.data || noDepartments}
            title={noDepartments ? 'Create a department first' : undefined}
          >
            Add New Curriculum
          </Button>
        }
      />

      {curricula.error && <Alert>{curricula.error.message}</Alert>}
      {curricula.isPending ? (
        <PageSpinner />
      ) : (
        <TableCard>
          <thead className="bg-gray-50">
            <tr>
              <Th className="px-6">Curriculum Code</Th>
              <Th className="px-6">Curriculum Name</Th>
              <Th className="px-6">Department</Th>
              <Th className="px-6" align="right">
                Action
              </Th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {(curricula.data ?? []).map((curr) => (
              <tr key={curr.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-6 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                  <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">{curr.code}</span>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{curr.name}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{curr.department.name}</td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <IconButton icon={Edit2} label="Edit Curriculum" size={13} onClick={() => setEditing({ curriculum: curr })} />
                </td>
              </tr>
            ))}
            {curricula.data?.length === 0 && (
              <tr>
                <td colSpan={4}>
                  <EmptyState>No curriculums found. Click "Add New Curriculum" to create one.</EmptyState>
                </td>
              </tr>
            )}
          </tbody>
        </TableCard>
      )}

      {editing && (
        <CurriculumFormModal
          key={editing.curriculum?.id ?? 'new'}
          curriculum={editing.curriculum}
          departments={departments.data ?? []}
          onClose={() => setEditing(null)}
        />
      )}
    </div>
  )
}
