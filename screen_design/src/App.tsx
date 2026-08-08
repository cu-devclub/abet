import React, { useState } from 'react';
import { 
  Users, BookOpen, LogOut, Plus, X, Edit2, 
  Trash2, Download, UploadCloud, AlertCircle, CheckCircle2,
  Globe, Moon, Building2, GraduationCap,
  Search, ArrowUpDown, ChevronUp, ChevronDown
} from 'lucide-react';


interface CourseAccess {
  id: string;
  access: string;
}

interface User {
  id: number;
  name: string;
  email: string;
  role: string;
  department: string;
  curriculum: string;
  courses?: CourseAccess[];
}

interface Department {
  id: string;
  name: string;
  code: string;
}

interface Curriculum {
  id: string;
  name: string;
  code: string;
  departmentId: string;
}

const INITIAL_DEPARTMENTS: Department[] = [
  { id: '1', name: 'Mathematics and Computer Science', code: 'MCS' },
  { id: '2', name: 'Electrical Engineering', code: 'EE' }
];

const INITIAL_CURRICULUMS: Curriculum[] = [
  { id: '1', name: 'Computer Science', code: 'CS', departmentId: '1' },
  { id: '2', name: 'Mathematics', code: 'MATH', departmentId: '1' },
  { id: '3', name: 'Electrical Engineering', code: 'EE', departmentId: '2' }
];

// --- MOCK DATA ---
const MOCK_COURSES = [
  { id: 'CS101', name: 'Introduction to Computer Science' },
  { id: 'MATH205', name: 'Calculus and Linear Algebra' },
  { id: 'ENG102', name: 'Academic Writing' }
];

const MOCK_USERS = [
  { id: 1, name: 'Alice Smith', email: 'alice.admin@company.com', role: 'super_admin', department: 'Mathematics and Computer Science', curriculum: 'Computer Science' },
  { id: 2, name: 'Bob Johnson', email: 'bob.admin@company.com', role: 'admin', department: 'Mathematics and Computer Science', curriculum: 'Mathematics' },
  { id: 3, name: 'Charlie Davis', email: 'charlie.staff@company.com', role: 'staff', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', courses: [{ id: 'CS101', access: 'edit' }, { id: 'MATH205', access: 'view' }] },
  { id: 4, name: 'Diana Prince', email: 'diana.instructor@company.com', role: 'instructor', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', courses: [{ id: 'CS101', access: 'edit' }] },
];

const MOCK_FILES = [
  { id: 1, courseId: 'CS101', name: 'CS101_Summary_Final.pdf', year: '2024', time: '14/10/2024 15:30', status: 'generated' },
  { id: 2, courseId: 'MATH205', name: 'MATH205_Grades_Raw.zip', year: '2024', time: '15/01/2024 09:15', status: 'processing' },
  { id: 3, courseId: 'ENG102', name: 'ENG102_Summary_Final.pdf', year: '2023', time: '10/10/2023 11:00', status: 'generated' }
];

// --- REUSABLE COMPONENTS ---
const Badge = ({ children, variant = 'gray' }: { children: React.ReactNode; variant: string }) => {
  const variants: Record<string, string> = {
    super_admin: 'bg-purple-100 text-purple-800 border-purple-200',
    admin: 'bg-blue-100 text-blue-800 border-blue-200',
    instructor: 'bg-teal-100 text-teal-800 border-teal-200',
    staff: 'bg-gray-100 text-gray-800 border-gray-200',
    processing: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    generated: 'bg-green-100 text-green-800 border-green-200',
  };
  return (
    <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium border ${variants[variant] || 'bg-gray-100 border-gray-200'}`}>
      {children}
    </span>
  );
};

const Modal = ({ isOpen, onClose, title, children, footer }: {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}) => {
  if (!isOpen) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div className="bg-white rounded-xl shadow-xl w-full max-w-lg overflow-hidden flex flex-col max-h-[90vh]">
        <div className="flex justify-between items-center p-4 border-b">
          <h3 className="text-lg font-semibold text-gray-900">{title}</h3>
          <button onClick={onClose} className="text-gray-400 hover:text-gray-600 rounded-lg p-1 hover:bg-gray-100 transition-colors">
            <X size={20} />
          </button>
        </div>
        <div className="p-4 overflow-y-auto flex-1">
          {children}
        </div>
        {footer && <div className="p-4 border-t bg-gray-50 flex justify-end gap-3">{footer}</div>}
      </div>
    </div>
  );
};

// --- PAGES ---

const LoginPage = ({ onLogin, onSimulateError }: { onLogin: () => void; onSimulateError: () => void }) => (
  <div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8 relative">
    
    {/* Theme & Language Selectors */}
    <div className="absolute top-4 right-4 sm:top-6 sm:right-6 flex gap-3">
      <div className="flex items-center bg-white border border-gray-200 rounded-lg px-2.5 py-1.5 shadow-sm">
        <Globe size={16} className="text-gray-500 mr-2" />
        <select className="bg-transparent text-sm text-gray-700 outline-none cursor-pointer">
          <option value="en">English</option>
          <option value="th">ภาษาไทย</option>
        </select>
      </div>
      <div className="flex items-center bg-white border border-gray-200 rounded-lg px-2.5 py-1.5 shadow-sm">
        <Moon size={16} className="text-gray-500 mr-2" />
        <select className="bg-transparent text-sm text-gray-700 outline-none cursor-pointer">
          <option value="light">Light</option>
          <option value="dark">Dark</option>
          <option value="system">System</option>
        </select>
      </div>
    </div>

    <div className="sm:mx-auto sm:w-full sm:max-w-md">
      <div className="bg-white py-8 px-4 shadow-xl shadow-gray-200/50 sm:rounded-2xl sm:px-10 border border-gray-100">
        <div className="text-center mb-8">
          <div className="mx-auto h-12 w-12 bg-indigo-600 rounded-xl flex items-center justify-center mb-4 shadow-lg shadow-indigo-200">
            <BookOpen className="h-6 w-6 text-white" />
          </div>
          <h2 className="text-2xl font-bold tracking-tight text-gray-900">ABET</h2>
          <p className="mt-2 text-sm text-gray-600">Please sign in to access your courses</p>
        </div>
        <div className="space-y-4">
          <button
            onClick={onLogin}
            className="w-full flex justify-center items-center py-2.5 px-4 border border-gray-300 rounded-lg shadow-sm bg-white text-sm font-medium text-gray-700 hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-indigo-500 transition-all"
          >
            <svg className="h-5 w-5 mr-2" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
              <path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/>
              <path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/>
              <path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/>
              <path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/>
            </svg>
            Sign in with Google
          </button>
          
          <div className="pt-4 text-center">
            <button onClick={onSimulateError} className="text-xs text-gray-400 hover:text-gray-600 underline">
              Simulate Permission Error
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
);

const UserManagement = ({
  users,
  setUsers,
  departments,
  curriculums
}: {
  users: User[];
  setUsers: React.Dispatch<React.SetStateAction<User[]>>;
  departments: Department[];
  curriculums: Curriculum[];
}) => {
  const [selectedUser, setSelectedUser] = useState<any>(null);
  const [editingRole, setEditingRole] = useState('staff');
  const [editingDepartment, setEditingDepartment] = useState('');
  const [editingCurriculum, setEditingCurriculum] = useState('');
  const [courseAccess, setCourseAccess] = useState<any[]>([]);
  
  const [editingName, setEditingName] = useState('');
  const [editingEmail, setEditingEmail] = useState('');
  const [isNewUser, setIsNewUser] = useState(false);

  const openUserModal = (user: any = null) => {
    if (user) {
      setSelectedUser(user);
      setEditingName(user.name);
      setEditingEmail(user.email);
      setEditingRole(user.role);
      setEditingDepartment(user.department || '');
      setEditingCurriculum(user.curriculum || '');
      setCourseAccess(user.courses || []);
      setIsNewUser(false);
    } else {
      setSelectedUser({ name: '', email: '', role: 'staff', department: '', curriculum: '', isNew: true });
      setEditingName('');
      setEditingEmail('');
      setEditingRole('staff');
      setEditingDepartment(departments[0]?.name || '');
      setEditingCurriculum('');
      setCourseAccess([]);
      setIsNewUser(true);
    }
  };

  const handleSaveUser = () => {
    if (isNewUser) {
      const newUser: User = {
        id: Date.now(),
        name: editingName || 'New User',
        email: editingEmail || 'user@company.com',
        role: editingRole,
        department: editingDepartment,
        curriculum: editingCurriculum,
        courses: ['staff', 'instructor'].includes(editingRole) ? courseAccess : []
      };
      setUsers([...users, newUser]);
    } else {
      setUsers(users.map(u => u.id === selectedUser.id ? {
        ...u,
        name: editingName,
        email: editingEmail,
        role: editingRole,
        department: editingDepartment,
        curriculum: editingCurriculum,
        courses: ['staff', 'instructor'].includes(editingRole) ? courseAccess : []
      } : u));
    }
    setSelectedUser(null);
  };

  const handleDeleteUser = () => {
    if (selectedUser && !isNewUser) {
      setUsers(users.filter(u => u.id !== selectedUser.id));
    }
    setSelectedUser(null);
  };

  const addCourseAccess = () => {
    setCourseAccess([...courseAccess, { id: '', access: 'view' }]);
  };

  const removeCourseAccess = (index: number) => {
    setCourseAccess(courseAccess.filter((_, i) => i !== index));
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900">User Management</h1>
        <button 
          onClick={() => openUserModal()} 
          className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-sm"
        >
          <Plus size={16} className="mr-2" /> Add User
        </button>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Email</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Department</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Curriculum</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Role</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {users.map((user) => (
              <tr key={user.id} className="hover:bg-gray-50 transition-colors">
                <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{user.name}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{user.email}</td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <div className="max-w-[160px] truncate" title={user.department || ''}>
                    {user.department || '-'}
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  <div className="max-w-[140px] truncate" title={user.curriculum || ''}>
                    {user.curriculum || '-'}
                  </div>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-sm">
                  <Badge variant={user.role}>{user.role.replace('_', ' ').toUpperCase()}</Badge>
                </td>
                <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                  <button 
                    onClick={() => openUserModal(user)}
                    className="text-indigo-600 hover:text-indigo-900 bg-indigo-50 px-3 py-1.5 rounded-lg transition-colors"
                  >
                    Edit User
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Modal 
        isOpen={!!selectedUser} 
        onClose={() => setSelectedUser(null)} 
        title={isNewUser ? "Add New User" : "Edit User Access"}
        footer={
          <>
            {!isNewUser && (
              <button onClick={handleDeleteUser} className="px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 rounded-lg mr-auto transition-colors">
                Delete User
              </button>
            )}
            <button onClick={() => setSelectedUser(null)} className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
              Cancel
            </button>
            <button onClick={handleSaveUser} className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors">
              {isNewUser ? "Create User" : "Save Changes"}
            </button>
          </>
        }
      >
        {selectedUser && (
          <div className="space-y-5">
            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Name</label>
                <input 
                  type="text" 
                  value={editingName} 
                  onChange={(e) => setEditingName(e.target.value)}
                  placeholder="e.g. John Doe"
                  className="w-full border rounded-lg px-3 py-2 text-sm bg-white border-gray-300 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
                <input 
                  type="email" 
                  value={editingEmail} 
                  onChange={(e) => setEditingEmail(e.target.value)}
                  placeholder="e.g. user@company.com"
                  className="w-full border rounded-lg px-3 py-2 text-sm bg-white border-gray-300 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Department</label>
                <select 
                  value={editingDepartment}
                  onChange={(e) => {
                    setEditingDepartment(e.target.value);
                    setEditingCurriculum('');
                  }}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                >
                  <option value="">Select Department...</option>
                  {departments.map((dept) => (
                    <option key={dept.id} value={dept.name}>{dept.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Curriculum</label>
                <select 
                  value={editingCurriculum}
                  onChange={(e) => setEditingCurriculum(e.target.value)}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                >
                  <option value="">Select Curriculum...</option>
                  {curriculums
                    .filter(curr => {
                      const dept = departments.find(d => d.name === editingDepartment);
                      return !dept || curr.departmentId === dept.id;
                    })
                    .map((curr) => (
                      <option key={curr.id} value={curr.name}>{curr.name}</option>
                    ))}
                </select>
              </div>
            </div>
            
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Role</label>
              <select 
                value={editingRole}
                onChange={(e) => setEditingRole(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                <option value="super_admin">Super Admin</option>
                <option value="admin">Admin</option>
                <option value="instructor">Instructor</option>
                <option value="staff">Staff</option>
              </select>
            </div>

            {['staff', 'instructor'].includes(editingRole) && (
              <div className="pt-2 border-t mt-4">
                <div className="flex justify-between items-center mb-3">
                  <h4 className="text-sm font-semibold text-gray-900">Course Access</h4>
                  <button onClick={addCourseAccess} className="text-xs flex items-center text-indigo-600 hover:text-indigo-800 font-medium bg-indigo-50 px-2 py-1 rounded">
                    <Plus size={14} className="mr-1" /> Add Course
                  </button>
                </div>
                {courseAccess.length === 0 ? (
                  <p className="text-sm text-gray-500 text-center py-4 bg-gray-50 rounded-lg border border-dashed">No course access granted.</p>
                ) : (
                  <div className="space-y-2">
                    {courseAccess.map((access, index) => (
                      <div key={index} className="flex gap-2 items-center">
                        <select className="flex-1 border border-gray-300 rounded-lg px-3 py-2 text-sm" defaultValue={access.id}>
                          <option value="" disabled>Select Course...</option>
                          {MOCK_COURSES.map(c => <option key={c.id} value={c.id}>{c.id} - {c.name}</option>)}
                        </select>
                        <select className="w-32 border border-gray-300 rounded-lg px-3 py-2 text-sm" defaultValue={access.access}>
                          <option value="view">View</option>
                          <option value="edit">Edit</option>
                        </select>
                        <button onClick={() => removeCourseAccess(index)} className="p-2 text-red-500 hover:bg-red-50 rounded-lg">
                          <Trash2 size={16} />
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            )}
            
            {['super_admin', 'admin'].includes(editingRole) && (
              <div className="p-4 bg-blue-50 text-blue-800 rounded-lg text-sm flex items-start mt-4 border border-blue-100">
                <AlertCircle size={16} className="mt-0.5 mr-2 flex-shrink-0" />
                <p>Super Admins and Admins automatically have full access to all courses. Course specific access rules do not apply.</p>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  );
};

const DepartmentManagement = ({
  departments,
  setDepartments,
  curriculums,
  users
}: {
  departments: Department[];
  setDepartments: React.Dispatch<React.SetStateAction<Department[]>>;
  curriculums: Curriculum[];
  users: User[];
}) => {
  const [selectedDept, setSelectedDept] = useState<Department | null>(null);
  const [editingName, setEditingName] = useState('');
  const [editingCode, setEditingCode] = useState('');
  const [isNewDept, setIsNewDept] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  const openDeptModal = (dept: Department | null = null) => {
    setErrorMsg('');
    if (dept) {
      setSelectedDept(dept);
      setEditingName(dept.name);
      setEditingCode(dept.code);
      setIsNewDept(false);
    } else {
      setSelectedDept({ id: '', name: '', code: '' });
      setEditingName('');
      setEditingCode('');
      setIsNewDept(true);
    }
  };

  const handleSaveDept = () => {
    if (!editingName.trim() || !editingCode.trim()) {
      setErrorMsg('Name and Code are required.');
      return;
    }

    if (isNewDept) {
      const newDept: Department = {
        id: Date.now().toString(),
        name: editingName.trim(),
        code: editingCode.trim().toUpperCase()
      };
      setDepartments([...departments, newDept]);
    } else if (selectedDept) {
      setDepartments(departments.map(d => d.id === selectedDept.id ? {
        ...d,
        name: editingName.trim(),
        code: editingCode.trim().toUpperCase()
      } : d));
    }
    setSelectedDept(null);
  };

  const handleDeleteDept = (dept: Department) => {
    const hasCurriculums = curriculums.some(c => c.departmentId === dept.id);
    const hasUsers = users.some(u => u.department === dept.name);

    if (hasCurriculums || hasUsers) {
      alert(`Cannot delete department "${dept.name}". It is currently assigned to ${hasCurriculums ? 'one or more curriculums' : ''} ${hasCurriculums && hasUsers ? 'and' : ''} ${hasUsers ? 'one or more users' : ''}.`);
      return;
    }

    if (confirm(`Are you sure you want to delete the department "${dept.name}"?`)) {
      setDepartments(departments.filter(d => d.id !== dept.id));
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Department Management</h1>
        <button 
          onClick={() => openDeptModal()} 
          className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-sm"
        >
          <Plus size={16} className="mr-2" /> Add Department
        </button>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Code</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Department Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Curriculums</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {departments.map((dept) => {
              const deptCurriculums = curriculums.filter(c => c.departmentId === dept.id);
              return (
                <tr key={dept.id} className="hover:bg-gray-50 transition-colors">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                    <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">{dept.code}</span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{dept.name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {deptCurriculums.length} {deptCurriculums.length === 1 ? 'Curriculum' : 'Curriculums'}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                    <button 
                      onClick={() => openDeptModal(dept)}
                      className="text-indigo-600 hover:text-indigo-900 bg-indigo-50 px-3 py-1.5 rounded-lg transition-colors"
                    >
                      Edit
                    </button>
                    <button 
                      onClick={() => handleDeleteDept(dept)}
                      className="text-red-600 hover:text-red-900 bg-red-50 px-3 py-1.5 rounded-lg transition-colors"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              );
            })}
            {departments.length === 0 && (
              <tr>
                <td colSpan={4} className="text-center py-8 text-gray-500 italic">No departments found. Click "Add Department" to create one.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <Modal 
        isOpen={!!selectedDept} 
        onClose={() => setSelectedDept(null)} 
        title={isNewDept ? "Add New Department" : "Edit Department"}
        footer={
          <>
            <button onClick={() => setSelectedDept(null)} className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
              Cancel
            </button>
            <button onClick={handleSaveDept} className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors">
              {isNewDept ? "Create Department" : "Save Changes"}
            </button>
          </>
        }
      >
        {selectedDept && (
          <div className="space-y-4">
            {errorMsg && (
              <div className="p-3 bg-red-50 text-red-700 rounded-lg text-sm flex items-start border border-red-100">
                <AlertCircle size={16} className="mt-0.5 mr-2 flex-shrink-0" />
                <p>{errorMsg}</p>
              </div>
            )}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Department Code</label>
              <input 
                type="text" 
                value={editingCode} 
                onChange={(e) => setEditingCode(e.target.value)}
                placeholder="e.g. MCS"
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Department Name</label>
              <input 
                type="text" 
                value={editingName} 
                onChange={(e) => setEditingName(e.target.value)}
                placeholder="e.g. Mathematics and Computer Science"
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
              />
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

const CurriculumManagement = ({
  curriculums,
  setCurriculums,
  departments,
  users
}: {
  curriculums: Curriculum[];
  setCurriculums: React.Dispatch<React.SetStateAction<Curriculum[]>>;
  departments: Department[];
  users: User[];
}) => {
  const [selectedCurr, setSelectedCurr] = useState<Curriculum | null>(null);
  const [editingName, setEditingName] = useState('');
  const [editingCode, setEditingCode] = useState('');
  const [editingDeptId, setEditingDeptId] = useState('');
  const [isNewCurr, setIsNewCurr] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  const openCurrModal = (curr: Curriculum | null = null) => {
    setErrorMsg('');
    if (curr) {
      setSelectedCurr(curr);
      setEditingName(curr.name);
      setEditingCode(curr.code);
      setEditingDeptId(curr.departmentId);
      setIsNewCurr(false);
    } else {
      setSelectedCurr({ id: '', name: '', code: '', departmentId: '' });
      setEditingName('');
      setEditingCode('');
      setEditingDeptId(departments[0]?.id || '');
      setIsNewCurr(true);
    }
  };

  const handleSaveCurr = () => {
    if (!editingName.trim() || !editingCode.trim() || !editingDeptId) {
      setErrorMsg('Name, Code, and Department are required.');
      return;
    }

    if (isNewCurr) {
      const newCurr: Curriculum = {
        id: Date.now().toString(),
        name: editingName.trim(),
        code: editingCode.trim().toUpperCase(),
        departmentId: editingDeptId
      };
      setCurriculums([...curriculums, newCurr]);
    } else if (selectedCurr) {
      setCurriculums(curriculums.map(c => c.id === selectedCurr.id ? {
        ...c,
        name: editingName.trim(),
        code: editingCode.trim().toUpperCase(),
        departmentId: editingDeptId
      } : c));
    }
    setSelectedCurr(null);
  };

  const handleDeleteCurr = (curr: Curriculum) => {
    const hasUsers = users.some(u => u.curriculum === curr.name);

    if (hasUsers) {
      alert(`Cannot delete curriculum "${curr.name}". It is currently assigned to one or more users.`);
      return;
    }

    if (confirm(`Are you sure you want to delete the curriculum "${curr.name}"?`)) {
      setCurriculums(curriculums.filter(c => c.id !== curr.id));
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Curriculum Management</h1>
        <button 
          onClick={() => openCurrModal()} 
          className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-sm"
          disabled={departments.length === 0}
          title={departments.length === 0 ? "Create a department first" : ""}
        >
          <Plus size={16} className="mr-2" /> Add Curriculum
        </button>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Code</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Curriculum Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Department</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody className="bg-white divide-y divide-gray-200">
            {curriculums.map((curr) => {
              const dept = departments.find(d => d.id === curr.departmentId);
              return (
                <tr key={curr.id} className="hover:bg-gray-50 transition-colors">
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                    <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">{curr.code}</span>
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">{curr.name}</td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                    {dept ? dept.name : <span className="text-red-500 italic">No Department</span>}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                    <button 
                      onClick={() => openCurrModal(curr)}
                      className="text-indigo-600 hover:text-indigo-900 bg-indigo-50 px-3 py-1.5 rounded-lg transition-colors"
                    >
                      Edit
                    </button>
                    <button 
                      onClick={() => handleDeleteCurr(curr)}
                      className="text-red-600 hover:text-red-900 bg-red-50 px-3 py-1.5 rounded-lg transition-colors"
                    >
                      Delete
                    </button>
                  </td>
                </tr>
              );
            })}
            {curriculums.length === 0 && (
              <tr>
                <td colSpan={4} className="text-center py-8 text-gray-500 italic">No curriculums found. Click "Add Curriculum" to create one.</td>
              </tr>
            )}
          </tbody>
        </table>
      </div>

      <Modal 
        isOpen={!!selectedCurr} 
        onClose={() => setSelectedCurr(null)} 
        title={isNewCurr ? "Add New Curriculum" : "Edit Curriculum"}
        footer={
          <>
            <button onClick={() => setSelectedCurr(null)} className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
              Cancel
            </button>
            <button onClick={handleSaveCurr} className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors">
              {isNewCurr ? "Create Curriculum" : "Save Changes"}
            </button>
          </>
        }
      >
        {selectedCurr && (
          <div className="space-y-4">
            {errorMsg && (
              <div className="p-3 bg-red-50 text-red-700 rounded-lg text-sm flex items-start border border-red-100">
                <AlertCircle size={16} className="mt-0.5 mr-2 flex-shrink-0" />
                <p>{errorMsg}</p>
              </div>
            )}
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Curriculum Code</label>
              <input 
                type="text" 
                value={editingCode} 
                onChange={(e) => setEditingCode(e.target.value)}
                placeholder="e.g. CS"
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Curriculum Name</label>
              <input 
                type="text" 
                value={editingName} 
                onChange={(e) => setEditingName(e.target.value)}
                placeholder="e.g. Computer Science"
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
              />
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Department</label>
              <select 
                value={editingDeptId}
                onChange={(e) => setEditingDeptId(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
              >
                <option value="" disabled>Select Department...</option>
                {departments.map((dept) => (
                  <option key={dept.id} value={dept.id}>{dept.name}</option>
                ))}
              </select>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

const CourseList = ({
  courses,
  setCourses,
  files,
  setFiles,
  departments,
  curriculums
}: {
  courses: any[];
  setCourses: React.Dispatch<React.SetStateAction<any[]>>;
  files: any[];
  setFiles: React.Dispatch<React.SetStateAction<any[]>>;
  departments: Department[];
  curriculums: Curriculum[];
}) => {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingCourse, setEditingCourse] = useState<any>(null);
  const [uploadTargetCourse, setUploadTargetCourse] = useState<any>(null);
  const [globalYear, setGlobalYear] = useState('2024');
  const [selectedDeptFilter, setSelectedDeptFilter] = useState('');
  const [selectedCurriculumFilter, setSelectedCurriculumFilter] = useState('');

  // Search & Sort states
  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState<string>('id');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  // Creation method select tabs
  const [activeCreationMethod, setActiveCreationMethod] = useState<'single' | 'bulk'>('single');
  const [bulkFileUploaded, setBulkFileUploaded] = useState(false);
  const [bulkFileName, setBulkFileName] = useState('');
  const [bulkMockCourses, setBulkMockCourses] = useState<any[]>([]);

  // Input states for Add/Edit
  const [editingId, setEditingId] = useState('');
  const [editingNameInput, setEditingNameInput] = useState('');
  const [editingDepartment, setEditingDepartment] = useState('');
  const [editingCurriculum, setEditingCurriculum] = useState('');

  // Selection states
  const [selectedCourseIds, setSelectedCourseIds] = useState<string[]>([]);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);

  // Helper to get file details for a course
  const getCourseFile = (courseId: string) => {
    return files.find(f => f.courseId === courseId && f.year === globalYear);
  };

  // Filtered & Sorted courses
  const filteredCourses = courses.filter(course => {
    const file = getCourseFile(course.id);
    const statusText = file ? file.status : 'No files';
    const query = searchQuery.toLowerCase();
    
    const matchesSearch = (
      course.id.toLowerCase().includes(query) ||
      course.name.toLowerCase().includes(query) ||
      (course.department || '').toLowerCase().includes(query) ||
      (course.curriculum || '').toLowerCase().includes(query) ||
      statusText.toLowerCase().includes(query)
    );

    const matchesDept = !selectedDeptFilter || course.department === selectedDeptFilter;
    const matchesCurriculum = !selectedCurriculumFilter || course.curriculum === selectedCurriculumFilter;

    return matchesSearch && matchesDept && matchesCurriculum;
  });

  const handleDeptFilterChange = (deptName: string) => {
    setSelectedDeptFilter(deptName);
    setSelectedCurriculumFilter('');
  };

  const sortedCourses = [...filteredCourses].sort((a, b) => {
    let aVal: any = '';
    let bVal: any = '';

    if (sortField === 'id') {
      aVal = a.id;
      bVal = b.id;
    } else if (sortField === 'name') {
      aVal = a.name;
      bVal = b.name;
    } else if (sortField === 'department') {
      aVal = a.department || '';
      bVal = b.department || '';
    } else if (sortField === 'curriculum') {
      aVal = a.curriculum || '';
      bVal = b.curriculum || '';
    } else if (sortField === 'status') {
      const fileA = getCourseFile(a.id);
      const fileB = getCourseFile(b.id);
      aVal = fileA ? fileA.status : 'no_files';
      bVal = fileB ? fileB.status : 'no_files';
    } else if (sortField === 'time') {
      const fileA = getCourseFile(a.id);
      const fileB = getCourseFile(b.id);
      aVal = fileA ? fileA.time : '';
      bVal = fileB ? fileB.time : '';
    }

    if (aVal < bVal) return sortDirection === 'asc' ? -1 : 1;
    if (aVal > bVal) return sortDirection === 'asc' ? 1 : -1;
    return 0;
  });

  const handleSort = (field: string) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const handleEdit = (course: any) => {
    setEditingCourse(course);
    setEditingId(course.id);
    setEditingNameInput(course.name);
    setEditingDepartment(course.department || '');
    setEditingCurriculum(course.curriculum || '');
    setActiveCreationMethod('single');
  };

  const handleOpenAdd = () => {
    setEditingCourse(null);
    setEditingId('');
    setEditingNameInput('');
    setEditingDepartment(departments[0]?.name || '');
    setEditingCurriculum('');
    setActiveCreationMethod('single');
    setBulkFileUploaded(false);
    setBulkFileName('');
    setBulkMockCourses([]);
    setIsAddModalOpen(true);
  };

  const handleSaveCourse = () => {
    if (activeCreationMethod === 'bulk') {
      if (bulkFileUploaded && bulkMockCourses.length > 0) {
        setCourses([...courses, ...bulkMockCourses]);
      }
    } else {
      if (!editingId.trim() || !editingNameInput.trim()) return;

      if (editingCourse) {
        setCourses(courses.map(c => c.id === editingCourse.id ? { 
          id: editingId.trim().toUpperCase(), 
          name: editingNameInput.trim(),
          department: editingDepartment,
          curriculum: editingCurriculum
        } : c));
        if (editingId.trim().toUpperCase() !== editingCourse.id) {
          setFiles(files.map(f => f.courseId === editingCourse.id ? { ...f, courseId: editingId.trim().toUpperCase() } : f));
        }
      } else {
        setCourses([...courses, { 
          id: editingId.trim().toUpperCase(), 
          name: editingNameInput.trim(),
          department: editingDepartment,
          curriculum: editingCurriculum
        }]);
      }
    }
    setIsAddModalOpen(false);
    setEditingCourse(null);
  };

  const handleToggleSelect = (courseId: string) => {
    if (selectedCourseIds.includes(courseId)) {
      setSelectedCourseIds(selectedCourseIds.filter(id => id !== courseId));
    } else {
      setSelectedCourseIds([...selectedCourseIds, courseId]);
    }
  };

  const handleSelectAllToggle = () => {
    if (selectedCourseIds.length === sortedCourses.length) {
      setSelectedCourseIds([]);
    } else {
      setSelectedCourseIds(sortedCourses.map(c => c.id));
    }
  };

  const handleExecuteDelete = () => {
    setCourses(courses.filter(c => !selectedCourseIds.includes(c.id)));
    setFiles(files.filter(f => !selectedCourseIds.includes(f.courseId)));
    setSelectedCourseIds([]);
    setDeleteConfirmOpen(false);
  };

  const selectedCoursesWithGeneratedFiles = courses.filter(c => 
    selectedCourseIds.includes(c.id) && 
    files.some(f => f.courseId === c.id && f.year === globalYear && f.status === 'generated')
  );

  return (
    <div className="p-6 max-w-7xl mx-auto">
      {/* Page Header: Title + Actions */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
        <h1 className="text-2xl font-bold text-gray-900">My Courses</h1>
        
        {/* Actions */}
        <div className="flex items-center gap-3">
          {selectedCourseIds.length > 0 && (
            <button 
              onClick={() => setDeleteConfirmOpen(true)} 
              className="flex items-center px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 font-medium text-sm transition-colors shadow-sm whitespace-nowrap"
            >
              <Trash2 size={16} className="mr-2" /> Delete Selected ({selectedCourseIds.length})
            </button>
          )}
          
          <button onClick={handleOpenAdd} className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-sm whitespace-nowrap">
            <Plus size={16} className="mr-2" /> Add Course
          </button>
        </div>
      </div>

      {/* Filter and Search Bar */}
      <div className="flex flex-col md:flex-row md:items-center gap-3 mb-6">
        <div className="flex flex-wrap items-center gap-3">
          <div className="flex items-center bg-white border border-gray-300 rounded-lg shadow-sm px-3 py-1.5">
            <span className="text-sm text-gray-500 mr-2">Year:</span>
            <select 
              value={globalYear}
              onChange={(e) => setGlobalYear(e.target.value)}
              className="bg-transparent text-sm font-medium text-gray-900 outline-none cursor-pointer"
            >
              <option value="2024">2024</option>
              <option value="2023">2023</option>
              <option value="2022">2022</option>
            </select>
          </div>

          <div className="flex items-center bg-white border border-gray-300 rounded-lg shadow-sm px-3 py-1.5">
            <span className="text-sm text-gray-500 mr-2">Department:</span>
            <select 
              value={selectedDeptFilter}
              onChange={(e) => handleDeptFilterChange(e.target.value)}
              className="bg-transparent text-sm font-medium text-gray-900 outline-none cursor-pointer"
            >
              <option value="">All Departments</option>
              {departments.map((dept) => (
                <option key={dept.id} value={dept.name}>{dept.name}</option>
              ))}
            </select>
          </div>

          <div className="flex items-center bg-white border border-gray-300 rounded-lg shadow-sm px-3 py-1.5">
            <span className="text-sm text-gray-500 mr-2">Curriculum:</span>
            <select 
              value={selectedCurriculumFilter}
              onChange={(e) => setSelectedCurriculumFilter(e.target.value)}
              className="bg-transparent text-sm font-medium text-gray-900 outline-none cursor-pointer"
            >
              <option value="">All Curriculums</option>
              {curriculums
                .filter(curr => {
                  if (!selectedDeptFilter) return true;
                  const dept = departments.find(d => d.name === selectedDeptFilter);
                  return dept && curr.departmentId === dept.id;
                })
                .map((curr) => (
                  <option key={curr.id} value={curr.name}>{curr.name}</option>
                ))}
            </select>
          </div>
        </div>

        <div className="relative w-full md:w-64 md:ml-auto">
          <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
            <Search className="h-4 w-4 text-gray-400" />
          </div>
          <input
            type="text"
            placeholder="Search ID, name, dept, status..."
            value={searchQuery}
            onChange={(e) => setSearchQuery(e.target.value)}
            className="block w-full pl-9 pr-3 py-1.5 border border-gray-300 rounded-lg text-sm bg-white placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 transition-colors shadow-sm"
          />
        </div>
      </div>

      {/* Course Table */}
      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="w-12 px-6 py-3 text-left">
                  <input
                    type="checkbox"
                    checked={selectedCourseIds.length === sortedCourses.length && sortedCourses.length > 0}
                    onChange={handleSelectAllToggle}
                    className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                  />
                </th>
                <th 
                  onClick={() => handleSort('id')}
                  className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-1">
                    Course ID
                    {sortField === 'id' ? (
                      sortDirection === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />
                    ) : (
                      <ArrowUpDown size={12} className="text-gray-400" />
                    )}
                  </div>
                </th>
                <th 
                  onClick={() => handleSort('name')}
                  className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-1">
                    Course Name
                    {sortField === 'name' ? (
                      sortDirection === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />
                    ) : (
                      <ArrowUpDown size={12} className="text-gray-400" />
                    )}
                  </div>
                </th>

                <th 
                  onClick={() => handleSort('status')}
                  className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-1">
                    Status
                    {sortField === 'status' ? (
                      sortDirection === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />
                    ) : (
                      <ArrowUpDown size={12} className="text-gray-400" />
                    )}
                  </div>
                </th>
                <th 
                  onClick={() => handleSort('time')}
                  className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-1">
                    Updated Time
                    {sortField === 'time' ? (
                      sortDirection === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />
                    ) : (
                      <ArrowUpDown size={12} className="text-gray-400" />
                    )}
                  </div>
                </th>
                <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {sortedCourses.map(course => {
                const file = files.find(f => f.courseId === course.id && f.year === globalYear);
                const isSelected = selectedCourseIds.includes(course.id);

                return (
                  <tr 
                    key={course.id} 
                    onClick={() => handleToggleSelect(course.id)}
                    className={`hover:bg-gray-50 transition-colors cursor-pointer ${
                      isSelected ? 'bg-indigo-50/40 hover:bg-indigo-50/60' : ''
                    }`}
                  >
                    <td className="px-6 py-4 whitespace-nowrap">
                      <input 
                        type="checkbox" 
                        checked={isSelected} 
                        onChange={() => {}} // toggled by row onClick
                        className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                      />
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                      <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">
                        {course.id}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      {course.name}
                    </td>

                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {file ? (
                        <Badge variant={file.status}>
                          {file.status.charAt(0).toUpperCase() + file.status.slice(1)}
                        </Badge>
                      ) : (
                        <span className="text-gray-400 text-xs italic">No Files</span>
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                      {file ? (
                        <div className="flex items-center text-xs text-gray-500">
                          <CheckCircle2 size={12} className="mr-1 text-gray-400" />
                          {file.time}
                        </div>
                      ) : (
                        '-'
                      )}
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2" onClick={(e) => e.stopPropagation()}>
                      {file?.status === 'generated' && (
                        <button 
                          className="text-green-600 hover:text-green-900 bg-green-50 hover:bg-green-100 p-1.5 rounded-lg transition-colors inline-flex items-center"
                          title="Download PDF"
                        >
                          <Download size={14} />
                        </button>
                      )}
                      <button 
                        onClick={() => setUploadTargetCourse(course)}
                        className="text-indigo-600 hover:text-indigo-900 bg-indigo-50 hover:bg-indigo-100 p-1.5 rounded-lg transition-colors inline-flex items-center"
                        title="Upload Files"
                      >
                        <UploadCloud size={14} />
                      </button>
                      <button 
                        onClick={() => handleEdit(course)}
                        className="text-gray-600 hover:text-indigo-900 bg-gray-100 hover:bg-indigo-50 p-1.5 rounded-lg transition-colors inline-flex items-center"
                        title="Edit Course"
                      >
                        <Edit2 size={14} />
                      </button>
                    </td>
                  </tr>
                );
              })}
              {sortedCourses.length === 0 && (
                <tr>
                  <td colSpan={6} className="text-center py-8 text-gray-500 italic">
                    No courses found.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Add/Edit Course Modal */}
      <Modal 
        isOpen={isAddModalOpen || !!editingCourse} 
        onClose={() => { setIsAddModalOpen(false); setEditingCourse(null); }} 
        title={editingCourse ? "Edit Course" : "Create New Course"}
        footer={
          <>
            <button onClick={() => { setIsAddModalOpen(false); setEditingCourse(null); }} className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
              Cancel
            </button>
            <button onClick={handleSaveCourse} className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors">
              Save Course
            </button>
          </>
        }
      >
        <div className="space-y-4">
          {!editingCourse && (
            <div className="flex border-b border-gray-200 mb-4">
              <button
                type="button"
                onClick={() => setActiveCreationMethod('single')}
                className={`flex-1 pb-2.5 text-sm font-medium border-b-2 text-center transition-colors ${
                  activeCreationMethod === 'single'
                    ? 'border-indigo-600 text-indigo-600 font-semibold'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Single Course
              </button>
              <button
                type="button"
                onClick={() => setActiveCreationMethod('bulk')}
                className={`flex-1 pb-2.5 text-sm font-medium border-b-2 text-center transition-colors ${
                  activeCreationMethod === 'bulk'
                    ? 'border-indigo-600 text-indigo-600 font-semibold'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Upload Excel (Bulk)
              </button>
            </div>
          )}

          {activeCreationMethod === 'single' ? (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Course ID</label>
                <input 
                  type="text" 
                  placeholder="e.g. ENG101" 
                  value={editingId}
                  onChange={(e) => setEditingId(e.target.value)}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Course Name</label>
                <input 
                  type="text" 
                  placeholder="e.g. Academic Writing" 
                  value={editingNameInput}
                  onChange={(e) => setEditingNameInput(e.target.value)}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Department</label>
                <select 
                  value={editingDepartment}
                  onChange={(e) => {
                    setEditingDepartment(e.target.value);
                    setEditingCurriculum('');
                  }}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white"
                >
                  <option value="">Select Department...</option>
                  {departments.map((dept) => (
                    <option key={dept.id} value={dept.name}>{dept.name}</option>
                  ))}
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Curriculum</label>
                <select 
                  value={editingCurriculum}
                  onChange={(e) => setEditingCurriculum(e.target.value)}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white"
                >
                  <option value="">Select Curriculum...</option>
                  {curriculums
                    .filter(curr => {
                      const dept = departments.find(d => d.name === editingDepartment);
                      return !dept || curr.departmentId === dept.id;
                    })
                    .map((curr) => (
                      <option key={curr.id} value={curr.name}>{curr.name}</option>
                    ))}
                </select>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div 
                onClick={() => {
                  setBulkFileUploaded(true);
                  setBulkFileName('courses_template_import.xlsx');
                  setBulkMockCourses([
                    { id: 'CS102', name: 'Data Structures and Algorithms', department: 'Mathematics and Computer Science', curriculum: 'Computer Science' },
                    { id: 'MATH301', name: 'Abstract Algebra', department: 'Mathematics and Computer Science', curriculum: 'Mathematics' }
                  ]);
                }}
                className={`border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition-colors ${
                  bulkFileUploaded 
                    ? 'border-green-300 bg-green-50/20' 
                    : 'border-gray-300 bg-gray-50 hover:bg-indigo-50/20 hover:border-indigo-300'
                }`}
              >
                <UploadCloud className={`mx-auto h-12 w-12 mb-2 ${bulkFileUploaded ? 'text-green-500' : 'text-gray-400'}`} />
                {bulkFileUploaded ? (
                  <div>
                    <p className="text-sm font-semibold text-gray-800">File Selected: {bulkFileName}</p>
                    <p className="text-xs text-green-600 mt-1">Ready to import (2 courses parsed successfully)</p>
                  </div>
                ) : (
                  <div>
                    <p className="text-sm font-medium text-gray-700">Click to upload or drag Excel template here</p>
                    <p className="text-xs text-gray-500 mt-1">Supports .xlsx, .xls, .csv templates</p>
                  </div>
                )}
              </div>

              {bulkFileUploaded && bulkMockCourses.length > 0 && (
                <div className="border rounded-lg overflow-hidden bg-gray-50">
                  <div className="px-3 py-2 border-b bg-gray-100 text-xs font-semibold text-gray-700">
                    Preview (Rows to Import)
                  </div>
                  <div className="max-h-[160px] overflow-y-auto divide-y text-xs">
                    {bulkMockCourses.map((c, index) => (
                      <div key={index} className="p-2.5 flex justify-between items-center bg-white">
                        <div>
                          <p className="font-bold text-gray-900">{c.id}</p>
                          <p className="text-gray-500">{c.name}</p>
                        </div>
                        <div className="text-right text-[10px] text-gray-400">
                          <p>{c.department}</p>
                          <p>{c.curriculum}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </Modal>

      {/* Upload File Modal */}
      <Modal 
        isOpen={!!uploadTargetCourse} 
        onClose={() => setUploadTargetCourse(null)} 
        title={`Upload Files - ${uploadTargetCourse?.id}`}
        footer={
          <>
            <button onClick={() => setUploadTargetCourse(null)} className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors">
              Cancel
            </button>
            <button onClick={() => setUploadTargetCourse(null)} className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors">
              Submit & Process
            </button>
          </>
        }
      >
        <div className="space-y-5">
          <div className="bg-indigo-50 text-indigo-800 p-3 rounded-lg text-sm mb-4 border border-indigo-100 flex items-start">
             <AlertCircle size={16} className="mt-0.5 mr-2 shrink-0" />
             <p>Uploading files for <span className="font-semibold">{uploadTargetCourse?.name}</span>, Academic Year <span className="font-semibold">{globalYear}</span>.</p>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Upload Folder or Files</label>
            <div className="mt-1 flex justify-center px-6 pt-5 pb-6 border-2 border-gray-300 border-dashed rounded-xl bg-gray-50 hover:bg-indigo-50 hover:border-indigo-300 transition-colors cursor-pointer group">
              <div className="space-y-1 text-center">
                <UploadCloud className="mx-auto h-12 w-12 text-gray-400 group-hover:text-indigo-500 transition-colors" />
                <div className="flex text-sm text-gray-600 justify-center">
                  <span className="relative rounded-md font-medium text-indigo-600 focus-within:outline-none focus-within:ring-2 focus-within:ring-offset-2 focus-within:ring-indigo-500">
                    <span>Drag and drop</span>
                  </span>
                  <p className="pl-1">your files here</p>
                </div>
                <p className="text-xs text-gray-500">Files will be processed into a summary PDF</p>
              </div>
            </div>
          </div>
        </div>
      </Modal>

      {/* Delete Multiple Courses Confirmation Modal */}
      <Modal
        isOpen={deleteConfirmOpen}
        onClose={() => setDeleteConfirmOpen(false)}
        title="Confirm Delete Courses"
        footer={
          <>
            <button 
              onClick={() => setDeleteConfirmOpen(false)} 
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button 
              onClick={handleExecuteDelete} 
              className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
            >
              Delete
            </button>
          </>
        }
      >
        <div className="space-y-4">
          {selectedCoursesWithGeneratedFiles.length > 0 ? (
            <div className="p-4 bg-red-50 text-red-800 border border-red-200 rounded-xl space-y-3">
              <div className="flex items-start">
                <AlertCircle className="w-5 h-5 text-red-600 mr-2 shrink-0 mt-0.5" />
                <div>
                  <p className="font-semibold text-sm">Warning: Generated Files Affected</p>
                  <p className="text-xs text-red-700 mt-1">
                    The following selected courses have generated final report files. Deleting these courses will permanently delete their corresponding files:
                  </p>
                </div>
              </div>
              <ul className="list-disc list-inside text-xs font-semibold pl-1 space-y-1">
                {selectedCoursesWithGeneratedFiles.map(c => (
                  <li key={c.id}>{c.id} - {c.name}</li>
                ))}
              </ul>
            </div>
          ) : (
            <p className="text-sm text-gray-600">
              Are you sure you want to delete the {selectedCourseIds.length} selected courses? This action cannot be undone.
            </p>
          )}
        </div>
      </Modal>
    </div>
  );
};

// --- MAIN APP COMPONENT ---
export default function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [activeTab, setActiveTab] = useState('courses'); // 'courses', 'users', 'departments', 'curriculums'
  const [showErrorModal, setShowErrorModal] = useState(false);

  const [users, setUsers] = useState<User[]>(MOCK_USERS);
  const [departments, setDepartments] = useState<Department[]>(INITIAL_DEPARTMENTS);
  const [curriculums, setCurriculums] = useState<Curriculum[]>(INITIAL_CURRICULUMS);
  const [courses, setCourses] = useState<any[]>(MOCK_COURSES);
  const [files, setFiles] = useState<any[]>(MOCK_FILES);

  // Mock User Session
  const currentUser = { name: 'Admin User', role: 'super_admin' };

  if (!isAuthenticated) {
    return (
      <>
        <LoginPage 
          onLogin={() => setIsAuthenticated(true)} 
          onSimulateError={() => setShowErrorModal(true)} 
        />
        
        {/* Permission Error Modal */}
        <Modal 
          isOpen={showErrorModal} 
          onClose={() => setShowErrorModal(false)} 
          title="Access Denied"
        >
          <div className="flex flex-col items-center py-6 text-center">
            <div className="bg-red-100 p-3 rounded-full mb-4">
              <AlertCircle className="h-8 w-8 text-red-600" />
            </div>
            <h3 className="text-lg font-medium text-gray-900 mb-1">Insufficient Permissions</h3>
            <p className="text-sm text-gray-500">
              Your Google account was verified, but you do not have permission to access the ABET system. Please contact the system administrator.
            </p>
            <button 
              onClick={() => setShowErrorModal(false)}
              className="mt-6 px-6 py-2 bg-indigo-600 text-white text-sm font-medium rounded-lg hover:bg-indigo-700"
            >
              Back to Login
            </button>
          </div>
        </Modal>
      </>
    );
  }

  return (
    <div className="flex h-screen bg-gray-50 overflow-hidden font-sans">
      {/* Sidebar */}
      <div className="w-64 bg-white border-r border-gray-200 flex flex-col">
        <div className="h-16 flex items-center px-6 border-b border-gray-200">
          <BookOpen className="h-6 w-6 text-indigo-600 mr-2" />
          <span className="text-lg font-bold text-gray-900">ABET</span>
        </div>
        
        <nav className="flex-1 px-4 py-6 space-y-2 overflow-y-auto">
          <button
            onClick={() => setActiveTab('courses')}
            className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'courses' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
          >
            <BookOpen size={18} className="mr-3" /> Courses
          </button>
          
          {['super_admin', 'admin'].includes(currentUser.role) && (
            <>
              <button
                onClick={() => setActiveTab('users')}
                className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'users' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
              >
                <Users size={18} className="mr-3" /> User Management
              </button>

              <button
                onClick={() => setActiveTab('departments')}
                className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'departments' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
              >
                <Building2 size={18} className="mr-3" /> Departments
              </button>

              <button
                onClick={() => setActiveTab('curriculums')}
                className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'curriculums' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
              >
                <GraduationCap size={18} className="mr-3" /> Curriculums
              </button>
            </>
          )}
        </nav>

        {/* Global Settings (Language & Theme) for Authenticated Users */}
        <div className="px-4 py-4 space-y-3 border-t border-gray-200">
          <div className="flex items-center bg-gray-50 border border-gray-200 rounded-lg px-2.5 py-2">
            <Globe size={16} className="text-gray-500 mr-2 shrink-0" />
            <select className="w-full bg-transparent text-sm text-gray-700 outline-none cursor-pointer">
              <option value="en">English</option>
              <option value="th">ภาษาไทย</option>
            </select>
          </div>
          <div className="flex items-center bg-gray-50 border border-gray-200 rounded-lg px-2.5 py-2">
            <Moon size={16} className="text-gray-500 mr-2 shrink-0" />
            <select className="w-full bg-transparent text-sm text-gray-700 outline-none cursor-pointer">
              <option value="light">Light Mode</option>
              <option value="dark">Dark Mode</option>
              <option value="system">System Default</option>
            </select>
          </div>
        </div>

        <div className="p-4 border-t border-gray-200">
          <div className="flex items-center mb-4 px-2">
            <div className="h-8 w-8 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center font-bold text-sm mr-3">
              {currentUser.name.charAt(0)}
            </div>
            <div className="overflow-hidden">
              <p className="text-sm font-medium text-gray-900 truncate">{currentUser.name}</p>
              <p className="text-xs text-gray-500 capitalize">{currentUser.role.replace('_', ' ')}</p>
            </div>
          </div>
          <button 
            onClick={() => setIsAuthenticated(false)}
            className="w-full flex items-center justify-center px-3 py-2 border border-gray-300 rounded-lg text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 transition-colors"
          >
            <LogOut size={16} className="mr-2" /> Sign Out
          </button>
        </div>
      </div>

      {/* Main Content Area */}
      <div className="flex-1 overflow-auto">
        {activeTab === 'courses' && (
          <CourseList 
            courses={courses} 
            setCourses={setCourses} 
            files={files} 
            setFiles={setFiles} 
            departments={departments}
            curriculums={curriculums}
          />
        )}
        {activeTab === 'users' && (
          <UserManagement 
            users={users} 
            setUsers={setUsers} 
            departments={departments} 
            curriculums={curriculums} 
          />
        )}
        {activeTab === 'departments' && (
          <DepartmentManagement 
            departments={departments} 
            setDepartments={setDepartments} 
            curriculums={curriculums} 
            users={users} 
          />
        )}
        {activeTab === 'curriculums' && (
          <CurriculumManagement 
            curriculums={curriculums} 
            setCurriculums={setCurriculums} 
            departments={departments} 
            users={users} 
          />
        )}
      </div>
    </div>
  );
}