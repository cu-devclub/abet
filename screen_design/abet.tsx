import React, { useState, useEffect } from 'react';
import { 
  Users, BookOpen, LogOut, Search, Plus, X, Edit2, 
  Trash2, Download, UploadCloud, AlertCircle, CheckCircle2, MoreVertical,
  Globe, Moon, UserPlus
} from 'lucide-react';

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
const Badge = ({ children, variant = 'gray' }) => {
  const variants = {
    super_admin: 'bg-purple-100 text-purple-800 border-purple-200',
    admin: 'bg-blue-100 text-blue-800 border-blue-200',
    instructor: 'bg-teal-100 text-teal-800 border-teal-200',
    staff: 'bg-gray-100 text-gray-800 border-gray-200',
    processing: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    generated: 'bg-green-100 text-green-800 border-green-200',
  };
  return (
    <span className={`px-2.5 py-0.5 rounded-full text-xs font-medium border ${variants[variant]}`}>
      {children}
    </span>
  );
};

const Modal = ({ isOpen, onClose, title, children, footer }) => {
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

const LoginPage = ({ onLogin, onSimulateError }) => (
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

const UserManagement = () => {
  const [users, setUsers] = useState(MOCK_USERS);
  const [selectedUser, setSelectedUser] = useState(null);
  const [editingRole, setEditingRole] = useState('staff');
  const [editingDepartment, setEditingDepartment] = useState('');
  const [editingCurriculum, setEditingCurriculum] = useState('');
  const [courseAccess, setCourseAccess] = useState([]);
  
  const [editingName, setEditingName] = useState('');
  const [editingEmail, setEditingEmail] = useState('');
  const [isNewUser, setIsNewUser] = useState(false);

  const openUserModal = (user = null) => {
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
      setEditingDepartment('Mathematics and Computer Science');
      setEditingCurriculum('Computer Science');
      setCourseAccess([]);
      setIsNewUser(true);
    }
  };

  const handleSaveUser = () => {
    if (isNewUser) {
      const newUser = {
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

  const removeCourseAccess = (index) => {
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
                  disabled={!isNewUser}
                  value={editingName} 
                  onChange={(e) => setEditingName(e.target.value)}
                  placeholder="e.g. John Doe"
                  className={`w-full border rounded-lg px-3 py-2 text-sm ${!isNewUser ? 'bg-gray-100 border-gray-300 text-gray-600 cursor-not-allowed' : 'bg-white border-gray-300 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500'}`} 
                />
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
                <input 
                  type="email" 
                  disabled={!isNewUser}
                  value={editingEmail} 
                  onChange={(e) => setEditingEmail(e.target.value)}
                  placeholder="e.g. user@company.com"
                  className={`w-full border rounded-lg px-3 py-2 text-sm ${!isNewUser ? 'bg-gray-100 border-gray-300 text-gray-600 cursor-not-allowed' : 'bg-white border-gray-300 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500'}`} 
                />
              </div>
            </div>

            <div className="grid grid-cols-2 gap-4">
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Department</label>
                <select 
                  value={editingDepartment}
                  onChange={(e) => setEditingDepartment(e.target.value)}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                >
                  <option value="" disabled>Select Department...</option>
                  <option value="Mathematics and Computer Science">Mathematics and Computer Science</option>
                </select>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Curriculum</label>
                <select 
                  value={editingCurriculum}
                  onChange={(e) => setEditingCurriculum(e.target.value)}
                  className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500"
                >
                  <option value="" disabled>Select Curriculum...</option>
                  <option value="Computer Science">Computer Science</option>
                  <option value="Mathematics">Mathematics</option>
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

const CourseList = () => {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingCourse, setEditingCourse] = useState(null);
  const [uploadTargetCourse, setUploadTargetCourse] = useState(null);
  const [globalYear, setGlobalYear] = useState('2024');

  const handleEdit = (course) => {
    setEditingCourse(course);
  };

  return (
    <div className="p-6 max-w-7xl mx-auto">
      <div className="flex justify-between items-center mb-8">
        <h1 className="text-2xl font-bold text-gray-900">My Courses</h1>
        
        {/* Global Year Selector & Add Course Button */}
        <div className="flex items-center gap-4">
          <div className="flex items-center bg-white border border-gray-300 rounded-lg shadow-sm px-3 py-2">
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
          
          <button onClick={() => setIsAddModalOpen(true)} className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-sm">
            <Plus size={16} className="mr-2" /> Add Course
          </button>
        </div>
      </div>

      {/* Small Course Cards Grid */}
      <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
        {MOCK_COURSES.map(course => {
          const file = MOCK_FILES.find(f => f.courseId === course.id && f.year === globalYear);

          return (
            <div key={course.id} className="group bg-white rounded-xl shadow-sm border border-gray-200 p-5 hover:shadow-md hover:border-indigo-300 transition-all flex flex-col min-h-[160px]">
              
              {/* Top Row: Course ID (Left) & Status (Right) */}
              <div className="flex justify-between items-start mb-3">
                <div className="flex items-center gap-2">
                  <span className="text-xs font-bold tracking-wider text-indigo-700 bg-indigo-50 px-2.5 py-1.5 rounded-md uppercase">
                    {course.id}
                  </span>
                  <button 
                    onClick={() => handleEdit(course)}
                    className="text-gray-400 hover:text-indigo-600 p-1 rounded hover:bg-indigo-50 transition-colors opacity-0 group-hover:opacity-100"
                    title="Edit Course"
                  >
                    <Edit2 size={14} />
                  </button>
                </div>
                {file && (
                  <Badge variant={file.status}>
                    {file.status.charAt(0).toUpperCase() + file.status.slice(1)}
                  </Badge>
                )}
              </div>

              {/* Middle: Course Name */}
              <h3 className="text-[15px] font-semibold text-gray-900 leading-snug mb-5">
                {course.name}
              </h3>

              {/* Bottom: Timestamp & Action Buttons */}
              <div className="mt-auto">
                <div className="h-5 mb-3">
                  {file ? (
                    <p className="text-[11px] text-gray-400 font-medium flex items-center">
                      <CheckCircle2 size={12} className="mr-1 text-gray-400" />
                      Updated: {file.time}
                    </p>
                  ) : (
                    <p className="text-[11px] text-gray-400 font-medium italic">No files for {globalYear}</p>
                  )}
                </div>
                
                <div className="flex justify-between items-center pt-3 border-t border-gray-100">
                  <button 
                    onClick={() => setUploadTargetCourse(course)}
                    className="flex items-center text-sm font-medium text-indigo-600 hover:text-indigo-800 transition-colors"
                  >
                    <UploadCloud size={16} className="mr-1.5" /> Upload
                  </button>
                  
                  {file?.status === 'generated' && (
                    <button className="flex items-center text-sm font-medium text-green-600 hover:text-green-800 transition-colors">
                      <Download size={16} className="mr-1.5" /> Download
                    </button>
                  )}
                </div>
              </div>

            </div>
          );
        })}
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
            <button onClick={() => { setIsAddModalOpen(false); setEditingCourse(null); }} className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors">
              Save Course
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Course ID</label>
            <input 
              type="text" 
              placeholder="e.g. ENG101" 
              defaultValue={editingCourse?.id || ''}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Course Name</label>
            <input 
              type="text" 
              placeholder="e.g. Academic Writing" 
              defaultValue={editingCourse?.name || ''}
              className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
            />
          </div>
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
    </div>
  );
};

// --- MAIN APP COMPONENT ---
export default function App() {
  const [isAuthenticated, setIsAuthenticated] = useState(false);
  const [activeTab, setActiveTab] = useState('courses'); // 'courses', 'users'
  const [showErrorModal, setShowErrorModal] = useState(false);

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
        
        <nav className="flex-1 px-4 py-6 space-y-2">
          <button
            onClick={() => setActiveTab('courses')}
            className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'courses' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
          >
            <BookOpen size={18} className="mr-3" /> Courses
          </button>
          
          {['super_admin', 'admin'].includes(currentUser.role) && (
            <button
              onClick={() => setActiveTab('users')}
              className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'users' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
            >
              <Users size={18} className="mr-3" /> User Management
            </button>
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
        {activeTab === 'courses' ? (
          <CourseList />
        ) : (
          <UserManagement />
        )}
      </div>
    </div>
  );
}