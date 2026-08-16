import React, { useState } from 'react';
import { 
  Users, BookOpen, LogOut, Plus, X, Edit2, 
  Trash2, Download, UploadCloud, AlertCircle, CheckCircle2,
  Globe, Moon, Building2, GraduationCap, BookMarked,
  Search, ArrowUpDown, ChevronUp, ChevronDown, Filter, RotateCcw,
  ChevronRight, AlertTriangle, FileText, Eye, RefreshCw, Layers, Check,
  LayoutGrid, List, Info, ShieldCheck, FileCheck
} from 'lucide-react';

// --- MOCK DATA ---
const INITIAL_DEPARTMENTS = [
  { id: '1', name: 'Mathematics and Computer Science', code: 'MCS' },
  { id: '2', name: 'Electrical Engineering', code: 'EE' },
];

const INITIAL_CURRICULUMS = [
  { 
    id: '1', 
    name: 'Computer Science', 
    code: 'CS', 
    departmentId: '1', 
    coverFile: 'CS_Curriculum_Cover_2024.pdf', 
    coverSize: '240 KB', 
    coverUpdatedAt: '14/10/2024 15:30' 
  },
  { 
    id: '2', 
    name: 'Mathematics', 
    code: 'MATH', 
    departmentId: '1',
    coverFile: 'MATH_Curriculum_Cover.pdf',
    coverSize: '185 KB',
    coverUpdatedAt: '12/09/2024 11:20'
  },
  { 
    id: '3', 
    name: 'Electrical Engineering', 
    code: 'EE', 
    departmentId: '2' 
  }
];

const MOCK_COURSES = [
  { id: 'CS101', name: 'Introduction to Computer Science', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', year: '2024', semester: '1' },
  { id: 'MATH205', name: 'Calculus and Linear Algebra', department: 'Mathematics and Computer Science', curriculum: 'Mathematics', year: '2024', semester: '1' },
  { id: 'CS102', name: 'Data Structures and Algorithms', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', year: '2024', semester: '2' },
  { id: 'EE101', name: 'Circuit Theory I', department: 'Electrical Engineering', curriculum: 'Electrical Engineering', year: '2024', semester: '2' }
];

const MOCK_USERS = [
  { id: 1, name: 'Alice Smith', email: 'alice.admin@company.com', role: 'super_admin', department: 'Mathematics and Computer Science', curriculum: 'Computer Science' },
  { id: 2, name: 'Bob Johnson', email: 'bob.admin@company.com', role: 'admin', department: 'Mathematics and Computer Science', curriculum: 'Mathematics' },
  { id: 3, name: 'Charlie Davis', email: 'charlie.staff@company.com', role: 'staff', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', courses: [{ id: 'CS101', access: 'edit' }, { id: 'MATH205', access: 'view' }] },
  { id: 4, name: 'Diana Prince', email: 'diana.instructor@company.com', role: 'instructor', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', courses: [{ id: 'CS101', access: 'edit' }] },
];

const MOCK_FILES = [
  { 
    id: 1, 
    courseId: 'CS101', 
    name: 'CS101_Summary_Final.pdf', 
    year: '2024', 
    semester: '1', 
    time: '14/10/2024 15:30', 
    status: 'generated' 
  },
  { 
    id: 2, 
    courseId: 'MATH205', 
    name: 'MATH205_Grades_Raw.zip', 
    year: '2024', 
    semester: '1', 
    time: '15/01/2024 09:15', 
    status: 'processing' 
  },
  { 
    id: 3, 
    courseId: 'CS102', 
    name: 'CS102_Grades_Raw.zip', 
    year: '2024', 
    semester: '2', 
    time: '15/10/2024 16:45', 
    status: 'failed',
    errors: [
      {
        id: 'err-1',
        code: 'ERR_SCORE_OUT_OF_BOUNDS',
        stage: 'Validation',
        title: 'Score Value Exceeds Maximum (Row 42)',
        description: 'Student score exceeds maximum allowable points (100). Found value "105.5" for Student ID 64010482 in Quiz 2 column.',
        location: 'Sheet "Midterm_Scores", Cell D42',
        suggestedFix: 'Verify student score sheet and correct the entered mark to be within valid range [0, 100].'
      },
      {
        id: 'err-2',
        code: 'ERR_CLO_MAPPING_MISSING',
        stage: 'CLO Mapping',
        title: 'Unmapped Assessment Instrument for CLO 2.3',
        description: 'Course Learning Outcome (CLO 2.3 - Algorithm Efficiency Analysis) does not have any corresponding exam or assignment mapped in the syllabus matrix.',
        location: 'Sheet "CLO_Matrix", Row 8',
        suggestedFix: 'Ensure all active CLOs have at least one evaluation instrument assigned with non-zero weight.'
      },
      {
        id: 'err-3',
        code: 'ERR_RUBRIC_FILE_MISSING',
        stage: 'Score Calculation',
        title: 'Missing Evaluation Rubric Spreadsheet',
        description: 'Referenced rubric file "CS102_Rubric_ProgAssignment1.xlsx" is missing from the uploaded ZIP archive or is corrupt.',
        location: 'Archive root /rubrics/',
        suggestedFix: 'Re-upload the archive including all required rubric definition spreadsheets.'
      }
    ]
  }
];

// --- REUSABLE COMPONENTS ---
const Badge = ({ children, variant = 'gray', onClick, title }) => {
  const variants = {
    super_admin: 'bg-purple-100 text-purple-800 border-purple-200',
    admin: 'bg-blue-100 text-blue-800 border-blue-200',
    instructor: 'bg-teal-100 text-teal-800 border-teal-200',
    staff: 'bg-gray-100 text-gray-800 border-gray-200',
    processing: 'bg-yellow-100 text-yellow-800 border-yellow-200',
    generated: 'bg-green-100 text-green-800 border-green-200',
    failed: 'bg-red-50 text-red-700 border-red-200 hover:bg-red-100 hover:border-red-300 font-semibold',
  };

  if (onClick) {
    return (
      <button 
        type="button"
        onClick={onClick}
        title={title}
        className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border transition-all cursor-pointer ${variants[variant] || 'bg-gray-100 border-gray-200'}`}
      >
        {children}
      </button>
    );
  }

  return (
    <span title={title} className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium border ${variants[variant] || 'bg-gray-100 border-gray-200'}`}>
      {children}
    </span>
  );
};

const Modal = ({ isOpen, onClose, title, children, footer, maxWidth = 'max-w-lg' }) => {
  if (!isOpen) return null;
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
      <div className={`bg-white rounded-xl shadow-xl w-full ${maxWidth} overflow-hidden flex flex-col max-h-[90vh]`}>
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
  const [editingName, setEditingName] = useState('');
  const [editingEmail, setEditingEmail] = useState('');
  const [isNewUser, setIsNewUser] = useState(false);

  const handleEmailChange = (email) => {
    setEditingEmail(email);
    if (isNewUser) {
      if (email.includes('@')) {
        const prefix = email.split('@')[0];
        const formatted = prefix
          .split(/[._-]/)
          .filter(Boolean)
          .map(part => part.charAt(0).toUpperCase() + part.slice(1))
          .join(' ');
        setEditingName(formatted || 'Google Account User');
      } else if (email.trim().length > 0) {
        setEditingName('Fetching from Google / Backend...');
      } else {
        setEditingName('');
      }
    }
  };

  const openUserModal = (user = null) => {
    if (user) {
      setSelectedUser(user);
      setEditingName(user.name);
      setEditingEmail(user.email);
      setEditingRole(user.role);
      setEditingDepartment(user.department || '');
      setEditingCurriculum(user.curriculum || '');
      setIsNewUser(false);
    } else {
      setSelectedUser({ name: '', email: '', role: 'staff', department: '', curriculum: '', isNew: true });
      setEditingName('');
      setEditingEmail('');
      setEditingRole('staff');
      setEditingDepartment('Mathematics and Computer Science');
      setEditingCurriculum('Computer Science');
      setIsNewUser(true);
    }
  };

  const handleSaveUser = () => {
    if (isNewUser) {
      const newUser = {
        id: Date.now(),
        name: editingName || editingEmail.split('@')[0] || 'New User',
        email: editingEmail || 'user@company.com',
        role: editingRole,
        department: editingDepartment,
        curriculum: editingCurriculum
      };
      setUsers([...users, newUser]);
    } else {
      setUsers(users.map(u => u.id === selectedUser.id ? {
        ...u,
        role: editingRole,
        department: editingDepartment,
        curriculum: editingCurriculum
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
        title={isNewUser ? "Add New User" : "Edit User"}
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
                <div className="flex items-center justify-between mb-1">
                  <label className="block text-sm font-medium text-gray-700">Name</label>
                  <span className="text-[11px] text-gray-400">Synced via Google SSO</span>
                </div>
                <input 
                  type="text" 
                  disabled
                  value={editingName} 
                  placeholder={isNewUser ? "Auto-synced from email" : "e.g. John Doe"}
                  className="w-full border rounded-lg px-3 py-2 text-sm bg-gray-100 border-gray-300 text-gray-600 cursor-not-allowed select-none" 
                />
                <p className="text-xs text-gray-400 mt-1">
                  {isNewUser 
                    ? "Name is fetched from backend/Google after entering email." 
                    : "Automatically managed & synced by backend via Google login."}
                </p>
              </div>
              <div>
                <label className="block text-sm font-medium text-gray-700 mb-1">Email</label>
                <input 
                  type="email" 
                  disabled={!isNewUser}
                  value={editingEmail} 
                  onChange={(e) => handleEmailChange(e.target.value)}
                  placeholder="e.g. user@company.com"
                  className={`w-full border rounded-lg px-3 py-2 text-sm ${!isNewUser ? 'bg-gray-100 border-gray-300 text-gray-600 cursor-not-allowed' : 'bg-white border-gray-300 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500'}`} 
                />
                {!isNewUser && (
                  <p className="text-xs text-gray-400 mt-1">Email is linked to Google account.</p>
                )}
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
          </div>
        )}
      </Modal>
    </div>
  );
};

const CourseManagement = () => {
  const [courses, setCourses] = useState(MOCK_COURSES);
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingCourse, setEditingCourse] = useState(null);
  
  // Year selector - single choice, defaults to latest year (no 'all' choice)
  const [selectedYear, setSelectedYear] = useState('2024');
  
  // Multi-select filters next to course list (empty = all)
  const [selectedSemesters, setSelectedSemesters] = useState([]);
  const [selectedDepartments, setSelectedDepartments] = useState([]);
  const [selectedCurriculums, setSelectedCurriculums] = useState([]);

  // Search & Sort states
  const [searchQuery, setSearchQuery] = useState('');
  const [editingId, setEditingId] = useState('');
  const [editingName, setEditingName] = useState('');
  const [editingDepartment, setEditingDepartment] = useState('Mathematics and Computer Science');
  const [editingCurriculum, setEditingCurriculum] = useState('Computer Science');
  const [editingYear, setEditingYear] = useState('2024');
  const [editingSemester, setEditingSemester] = useState('1');
  const [selectedCourseIds, setSelectedCourseIds] = useState([]);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [courseToDelete, setCourseToDelete] = useState(null);

  const toggleSemester = (sem) => {
    if (selectedSemesters.includes(sem)) {
      setSelectedSemesters(selectedSemesters.filter(s => s !== sem));
    } else {
      setSelectedSemesters([...selectedSemesters, sem]);
    }
  };

  const toggleDepartment = (deptName) => {
    if (selectedDepartments.includes(deptName)) {
      setSelectedDepartments(selectedDepartments.filter(d => d !== deptName));
    } else {
      setSelectedDepartments([...selectedDepartments, deptName]);
    }
  };

  const toggleCurriculum = (currName) => {
    if (selectedCurriculums.includes(currName)) {
      setSelectedCurriculums(selectedCurriculums.filter(c => c !== currName));
    } else {
      setSelectedCurriculums([...selectedCurriculums, currName]);
    }
  };

  const clearAllFilters = () => {
    setSelectedSemesters([]);
    setSelectedDepartments([]);
    setSelectedCurriculums([]);
    setSearchQuery('');
  };

  const hasActiveFilters = selectedSemesters.length > 0 || selectedDepartments.length > 0 || selectedCurriculums.length > 0 || searchQuery !== '';

  const handleOpenAdd = () => {
    setEditingCourse(null);
    setEditingId('');
    setEditingName('');
    setEditingDepartment(INITIAL_DEPARTMENTS[0]?.name || '');
    setEditingCurriculum('');
    setEditingYear(selectedYear);
    setEditingSemester(selectedSemesters.length === 1 ? selectedSemesters[0] : '1');
    setIsAddModalOpen(true);
  };

  const handleEdit = (course) => {
    setEditingCourse(course);
    setEditingId(course.id);
    setEditingName(course.name);
    setEditingDepartment(course.department || '');
    setEditingCurriculum(course.curriculum || '');
    setEditingYear(course.year || selectedYear);
    setEditingSemester(course.semester || '1');
    setIsAddModalOpen(true);
  };

  const handleSaveCourse = () => {
    if (!editingId.trim() || !editingName.trim()) return;

    if (editingCourse) {
      setCourses(courses.map(c => c.id === editingCourse.id ? { 
        ...c,
        id: editingId.trim().toUpperCase(), 
        name: editingName.trim(),
        department: editingDepartment,
        curriculum: editingCurriculum,
        year: editingYear,
        semester: editingSemester,
      } : c));
    } else {
      setCourses([...courses, { 
        id: editingId.trim().toUpperCase(), 
        name: editingName.trim(),
        year: editingYear,
        semester: editingSemester,
        department: editingDepartment,
        curriculum: editingCurriculum,
      }]);
    }
    setIsAddModalOpen(false);
    setEditingCourse(null);
  };

  const handleToggleSelect = (courseId) => {
    if (selectedCourseIds.includes(courseId)) {
      setSelectedCourseIds(selectedCourseIds.filter(id => id !== courseId));
    } else {
      setSelectedCourseIds([...selectedCourseIds, courseId]);
    }
  };

  const handleSelectAllToggle = () => {
    if (selectedCourseIds.length === filteredCourses.length && filteredCourses.length > 0) {
      setSelectedCourseIds([]);
    } else {
      setSelectedCourseIds(filteredCourses.map(c => c.id));
    }
  };

  const handleDeleteSingle = (course) => {
    setCourseToDelete(course);
    setDeleteConfirmOpen(true);
  };

  const handleExecuteDelete = () => {
    const idsToDelete = courseToDelete ? [courseToDelete.id] : selectedCourseIds;
    setCourses(courses.filter(c => !idsToDelete.includes(c.id)));
    setSelectedCourseIds(selectedCourseIds.filter(id => !idsToDelete.includes(id)));
    setDeleteConfirmOpen(false);
    setCourseToDelete(null);
  };

  const filteredCourses = courses.filter(course => {
    const matchesYear = (course.year || '2024') === selectedYear;
    const matchesSemester = selectedSemesters.length === 0 || selectedSemesters.includes(course.semester || '1');
    const matchesDept = selectedDepartments.length === 0 || selectedDepartments.includes(course.department || '');
    const matchesCurriculum = selectedCurriculums.length === 0 || selectedCurriculums.includes(course.curriculum || '');

    const query = searchQuery.toLowerCase();
    const matchesSearch = (
      course.id.toLowerCase().includes(query) ||
      course.name.toLowerCase().includes(query) ||
      (course.department || '').toLowerCase().includes(query) ||
      (course.curriculum || '').toLowerCase().includes(query)
    );

    return matchesYear && matchesSemester && matchesDept && matchesCurriculum && matchesSearch;
  });

  return (
    <div className="p-6 space-y-6 w-full">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        {/* Left: Title + Year Dropdown */}
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="text-2xl font-bold text-gray-900">Course Management</h1>
          <div className="flex items-center bg-white border border-gray-300 rounded-lg shadow-sm px-3 py-1.5 ml-1">
            <span className="text-xs font-semibold text-gray-500 mr-2 uppercase tracking-wide">Year:</span>
            <select 
              value={selectedYear}
              onChange={(e) => setSelectedYear(e.target.value)}
              className="bg-transparent text-sm font-bold text-indigo-700 outline-none cursor-pointer"
            >
              <option value="2025">2025</option>
              <option value="2024">2024</option>
              <option value="2023">2023</option>
              <option value="2022">2022</option>
            </select>
          </div>
        </div>

        <div className="flex items-center gap-3">
          {selectedCourseIds.length > 0 && (
            <button 
              onClick={() => { setCourseToDelete(null); setDeleteConfirmOpen(true); }} 
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

      {/* Main Layout: Course List on Left (flex-1), Multi-Select Filter Panel on Right (w-72) */}
      <div className="flex flex-col lg:flex-row gap-6 items-start">
        {/* Left Area: Search Bar + Course Table */}
        <div className="flex-1 w-full min-w-0 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 bg-white p-3 rounded-xl border border-gray-200 shadow-sm">
            <div className="relative flex-1">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Search className="h-4 w-4 text-gray-400" />
              </div>
              <input
                type="text"
                placeholder="Search course ID, name, department..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="block w-full pl-9 pr-3 py-1.5 border border-gray-200 rounded-lg text-sm bg-gray-50 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:bg-white transition-colors"
              />
            </div>
            <div className="text-xs text-gray-500 px-2 whitespace-nowrap">
              Showing <span className="font-semibold text-gray-900">{filteredCourses.length}</span> course{filteredCourses.length === 1 ? '' : 's'} in <span className="font-semibold text-indigo-600">{selectedYear}</span>
            </div>
          </div>

          <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="w-12 px-6 py-3 text-left">
                    <input
                      type="checkbox"
                      checked={selectedCourseIds.length === filteredCourses.length && filteredCourses.length > 0}
                      onChange={handleSelectAllToggle}
                      className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                    />
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Course ID</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Course Name</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Semester</th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Department</th>
                  <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {filteredCourses.map(course => {
                  const isSelected = selectedCourseIds.includes(course.id);
                  return (
                    <tr key={course.id} className={`hover:bg-gray-50 transition-colors ${isSelected ? 'bg-indigo-50/40' : ''}`}>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <input 
                          type="checkbox" 
                          checked={isSelected} 
                          onChange={() => handleToggleSelect(course.id)}
                          className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                        />
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                        <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">{course.id}</span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                        <div className="flex items-center gap-2">
                          <span>{course.name}</span>
                          {(() => {
                            const parentCurr = INITIAL_CURRICULUMS.find(c => c.name === course.curriculum);
                            if (parentCurr?.coverFile) {
                              return (
                                <span className="inline-flex items-center gap-1 text-[11px] font-medium text-indigo-700 bg-indigo-50 border border-indigo-200/80 px-1.5 py-0.5 rounded" title={`Shared Cover: ${parentCurr.coverFile} (${parentCurr.name})`}>
                                  <FileText size={11} className="text-indigo-500" />
                                  <span>{parentCurr.code} Cover</span>
                                </span>
                              );
                            }
                            return null;
                          })()}
                        </div>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
                          {course.semester === 'Summer' ? 'Summer' : `Sem ${course.semester || '1'}`}
                        </span>
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{course.department || '-'}</td>
                      <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                        <button 
                          onClick={() => handleEdit(course)}
                          className="text-indigo-600 hover:text-indigo-900 bg-indigo-50 hover:bg-indigo-100 p-1.5 rounded-lg transition-colors inline-flex items-center"
                          title="Edit Course"
                        >
                          <Edit2 size={14} />
                        </button>
                        <button 
                          onClick={() => handleDeleteSingle(course)}
                          className="text-red-600 hover:text-red-900 bg-red-50 hover:bg-red-100 p-1.5 rounded-lg transition-colors inline-flex items-center"
                          title="Delete Course"
                        >
                          <Trash2 size={14} />
                        </button>
                      </td>
                    </tr>
                  );
                })}
                {filteredCourses.length === 0 && (
                  <tr>
                    <td colSpan={6} className="text-center py-8 text-gray-500 italic">
                      No courses found matching the selected filters.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Right Area: Sidebar Filter Panel */}
        <div className="w-full lg:w-72 shrink-0 bg-white rounded-xl shadow-sm border border-gray-200 p-5 space-y-6 sticky top-6">
          <div className="flex items-center justify-between pb-3 border-b border-gray-100">
            <div className="flex items-center gap-2 text-gray-900 font-semibold text-sm">
              <Filter size={16} className="text-indigo-600" />
              <span>Filters</span>
            </div>
            {hasActiveFilters && (
              <button 
                onClick={clearAllFilters}
                className="text-xs font-medium text-indigo-600 hover:text-indigo-800 flex items-center gap-1 transition-colors"
              >
                <RotateCcw size={12} /> Reset
              </button>
            )}
          </div>

          {/* Semester Multi-Select */}
          <div>
            <div className="flex items-center justify-between mb-2.5">
              <span className="text-xs font-bold text-gray-700 uppercase tracking-wider">Semester</span>
              <span className="text-[11px] text-gray-400 font-medium">
                {selectedSemesters.length === 0 ? 'All' : `${selectedSemesters.length} selected`}
              </span>
            </div>
            <div className="space-y-1.5">
              {[
                { id: '1', label: 'Semester 1' },
                { id: '2', label: 'Semester 2' },
                { id: 'Summer', label: 'Summer' }
              ].map(sem => {
                const isChecked = selectedSemesters.includes(sem.id);
                return (
                  <label 
                    key={sem.id}
                    className={`flex items-center justify-between p-2 rounded-lg text-sm cursor-pointer transition-colors ${
                      isChecked ? 'bg-indigo-50/70 text-indigo-900 font-medium' : 'hover:bg-gray-50 text-gray-700'
                    }`}
                  >
                    <div className="flex items-center gap-2.5">
                      <input 
                        type="checkbox"
                        checked={isChecked}
                        onChange={() => toggleSemester(sem.id)}
                        className="h-4 w-4 text-indigo-600 rounded border-gray-300 focus:ring-indigo-500 cursor-pointer"
                      />
                      <span>{sem.label}</span>
                    </div>
                  </label>
                );
              })}
            </div>
          </div>

          {/* Department Multi-Select */}
          <div className="pt-4 border-t border-gray-100">
            <div className="flex items-center justify-between mb-2.5">
              <span className="text-xs font-bold text-gray-700 uppercase tracking-wider">Department</span>
              <span className="text-[11px] text-gray-400 font-medium">
                {selectedDepartments.length === 0 ? 'All' : `${selectedDepartments.length} selected`}
              </span>
            </div>
            <div className="space-y-1.5">
              {INITIAL_DEPARTMENTS.map(dept => {
                const isChecked = selectedDepartments.includes(dept.name);
                return (
                  <label 
                    key={dept.id}
                    className={`flex items-start gap-2.5 p-2 rounded-lg text-sm cursor-pointer transition-colors ${
                      isChecked ? 'bg-indigo-50/70 text-indigo-900 font-medium' : 'hover:bg-gray-50 text-gray-700'
                    }`}
                  >
                    <input 
                      type="checkbox"
                      checked={isChecked}
                      onChange={() => toggleDepartment(dept.name)}
                      className="h-4 w-4 mt-0.5 text-indigo-600 rounded border-gray-300 focus:ring-indigo-500 cursor-pointer shrink-0"
                    />
                    <span className="leading-snug text-xs font-medium">{dept.name}</span>
                  </label>
                );
              })}
            </div>
          </div>

          {/* Curriculum Multi-Select */}
          <div className="pt-4 border-t border-gray-100">
            <div className="flex items-center justify-between mb-2.5">
              <span className="text-xs font-bold text-gray-700 uppercase tracking-wider">Curriculum</span>
              <span className="text-[11px] text-gray-400 font-medium">
                {selectedCurriculums.length === 0 ? 'All' : `${selectedCurriculums.length} selected`}
              </span>
            </div>
            <div className="space-y-1.5 max-h-60 overflow-y-auto">
              {INITIAL_CURRICULUMS
                .filter(curr => {
                  if (selectedDepartments.length === 0) return true;
                  const parentDept = INITIAL_DEPARTMENTS.find(d => d.id === curr.departmentId);
                  return parentDept && selectedDepartments.includes(parentDept.name);
                })
                .map(curr => {
                  const isChecked = selectedCurriculums.includes(curr.name);
                  return (
                    <label 
                      key={curr.id}
                      className={`flex items-start gap-2.5 p-2 rounded-lg text-sm cursor-pointer transition-colors ${
                        isChecked ? 'bg-indigo-50/70 text-indigo-900 font-medium' : 'hover:bg-gray-50 text-gray-700'
                      }`}
                    >
                      <input 
                        type="checkbox"
                        checked={isChecked}
                        onChange={() => toggleCurriculum(curr.name)}
                        className="h-4 w-4 mt-0.5 text-indigo-600 rounded border-gray-300 focus:ring-indigo-500 cursor-pointer shrink-0"
                      />
                      <span className="leading-snug text-xs font-medium">{curr.name}</span>
                    </label>
                  );
                })}
            </div>
          </div>

          <div className="pt-3 border-t border-gray-100">
            <p className="text-[11px] text-gray-400 text-center italic">
              Unselected categories default to showing all
            </p>
          </div>
        </div>
      </div>

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
              {editingCourse ? "Save Changes" : "Create Course"}
            </button>
          </>
        }
      >
        <div className="space-y-4">
          <div className="grid grid-cols-2 gap-4">
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
                value={editingName}
                onChange={(e) => setEditingName(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Academic Year</label>
              <select 
                value={editingYear}
                onChange={(e) => setEditingYear(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white"
              >
                <option value="2025">2025</option>
                <option value="2024">2024</option>
                <option value="2023">2023</option>
                <option value="2022">2022</option>
              </select>
            </div>
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">Semester</label>
              <select 
                value={editingSemester}
                onChange={(e) => setEditingSemester(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white"
              >
                <option value="1">Semester 1</option>
                <option value="2">Semester 2</option>
                <option value="Summer">Summer</option>
              </select>
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
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white"
              >
                <option value="">Select Department...</option>
                {INITIAL_DEPARTMENTS.map((dept) => (
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
                {INITIAL_CURRICULUMS
                  .filter(curr => {
                    const dept = INITIAL_DEPARTMENTS.find(d => d.name === editingDepartment);
                    return !dept || curr.departmentId === dept.id;
                  })
                  .map((curr) => (
                    <option key={curr.id} value={curr.name}>{curr.name}</option>
                  ))}
              </select>
            </div>
          </div>
        </div>
      </Modal>

      <Modal
        isOpen={deleteConfirmOpen}
        onClose={() => { setDeleteConfirmOpen(false); setCourseToDelete(null); }}
        title="Confirm Delete Course"
        footer={
          <>
            <button 
              onClick={() => { setDeleteConfirmOpen(false); setCourseToDelete(null); }} 
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
        <p className="text-sm text-gray-600">
          {courseToDelete 
            ? `Are you sure you want to delete course "${courseToDelete.id} - ${courseToDelete.name}"? This action cannot be undone.`
            : `Are you sure you want to delete the ${selectedCourseIds.length} selected course(s)? This action cannot be undone.`}
        </p>
      </Modal>
    </div>
  );
};

const CoverManagement = () => {
  const [curriculums, setCurriculums] = useState(INITIAL_CURRICULUMS);
  const [departments] = useState(INITIAL_DEPARTMENTS);
  const [courses] = useState(MOCK_COURSES);

  const [searchQuery, setSearchQuery] = useState('');
  const [selectedDepartments, setSelectedDepartments] = useState([]);
  const [selectedStatuses, setSelectedStatuses] = useState([]);
  const [sortField, setSortField] = useState('code');
  const [sortDirection, setSortDirection] = useState('asc');
  
  // Selection
  const [selectedCurrIds, setSelectedCurrIds] = useState([]);

  // Modal States
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalCurrId, setModalCurrId] = useState('');
  const [isFixedCurriculum, setIsFixedCurriculum] = useState(false);
  const [modalFile, setModalFile] = useState(null);
  const [isDragging, setIsDragging] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  // Preview and Delete modals
  const [previewCurr, setPreviewCurr] = useState(null);
  const [deleteConfirmCurr, setDeleteConfirmCurr] = useState(null);
  const [isBulkDeleteOpen, setIsBulkDeleteOpen] = useState(false);

  // Toast
  const [toastMessage, setToastMessage] = useState(null);

  const showToast = (text, type = 'success') => {
    setToastMessage({ text, type });
    setTimeout(() => setToastMessage(null), 3500);
  };

  const getCoursesForCurriculum = (currName) => {
    return courses.filter(c => c.curriculum === currName);
  };

  const toggleDepartment = (deptId) => {
    if (selectedDepartments.includes(deptId)) {
      setSelectedDepartments(selectedDepartments.filter(d => d !== deptId));
    } else {
      setSelectedDepartments([...selectedDepartments, deptId]);
    }
  };

  const toggleStatus = (status) => {
    if (selectedStatuses.includes(status)) {
      setSelectedStatuses(selectedStatuses.filter(s => s !== status));
    } else {
      setSelectedStatuses([...selectedStatuses, status]);
    }
  };

  const clearAllFilters = () => {
    setSelectedDepartments([]);
    setSelectedStatuses([]);
    setSearchQuery('');
  };

  const hasActiveFilters = selectedDepartments.length > 0 || selectedStatuses.length > 0 || searchQuery !== '';

  // Filtered & Sorted Curriculums
  const filteredCurriculums = curriculums.filter(curr => {
    const dept = departments.find(d => d.id === curr.departmentId);
    const deptName = dept ? dept.name.toLowerCase() : '';
    const query = searchQuery.toLowerCase();

    const matchesSearch = (
      curr.name.toLowerCase().includes(query) ||
      curr.code.toLowerCase().includes(query) ||
      deptName.includes(query) ||
      (curr.coverFile && curr.coverFile.toLowerCase().includes(query))
    );

    const matchesDept = selectedDepartments.length === 0 || selectedDepartments.includes(curr.departmentId);

    const matchesStatus = (
      selectedStatuses.length === 0 ||
      (selectedStatuses.includes('has_cover') && !!curr.coverFile) ||
      (selectedStatuses.includes('no_cover') && !curr.coverFile)
    );

    return matchesSearch && matchesDept && matchesStatus;
  });

  const sortedCurriculums = [...filteredCurriculums].sort((a, b) => {
    let aVal = '';
    let bVal = '';

    if (sortField === 'code') {
      aVal = a.code;
      bVal = b.code;
    } else if (sortField === 'name') {
      aVal = a.name;
      bVal = b.name;
    } else if (sortField === 'department') {
      aVal = departments.find(d => d.id === a.departmentId)?.name || '';
      bVal = departments.find(d => d.id === b.departmentId)?.name || '';
    } else if (sortField === 'courses') {
      aVal = getCoursesForCurriculum(a.name).length;
      bVal = getCoursesForCurriculum(b.name).length;
    } else if (sortField === 'cover') {
      aVal = a.coverFile || '';
      bVal = b.coverFile || '';
    } else if (sortField === 'time') {
      aVal = a.coverUpdatedAt || '';
      bVal = b.coverUpdatedAt || '';
    }

    if (aVal < bVal) return sortDirection === 'asc' ? -1 : 1;
    if (aVal > bVal) return sortDirection === 'asc' ? 1 : -1;
    return 0;
  });

  const handleSort = (field) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const handleToggleSelect = (id) => {
    if (selectedCurrIds.includes(id)) {
      setSelectedCurrIds(selectedCurrIds.filter(item => item !== id));
    } else {
      setSelectedCurrIds([...selectedCurrIds, id]);
    }
  };

  const handleSelectAllToggle = () => {
    if (selectedCurrIds.length === sortedCurriculums.length && sortedCurriculums.length > 0) {
      setSelectedCurrIds([]);
    } else {
      setSelectedCurrIds(sortedCurriculums.map(c => c.id));
    }
  };

  const handleOpenUpload = (curr) => {
    setErrorMsg('');
    if (curr) {
      setModalCurrId(curr.id);
      setIsFixedCurriculum(true);
      setModalFile(curr.coverFile ? { name: curr.coverFile, size: curr.coverSize || '240 KB' } : null);
    } else {
      setModalCurrId(curriculums[0]?.id || '');
      setIsFixedCurriculum(false);
      setModalFile(null);
    }
    setIsModalOpen(true);
  };

  const handleFileChange = (e) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      if (!file.name.toLowerCase().endsWith('.pdf') && file.type !== 'application/pdf') {
        setErrorMsg('Please select a valid PDF file (.pdf)');
        return;
      }
      setErrorMsg('');
      setModalFile({
        name: file.name,
        size: `${(file.size / 1024).toFixed(1)} KB`
      });
    }
  };

  const handleFileDrop = (e) => {
    e.preventDefault();
    setIsDragging(false);
    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const file = e.dataTransfer.files[0];
      if (!file.name.toLowerCase().endsWith('.pdf') && file.type !== 'application/pdf') {
        setErrorMsg('Please select a valid PDF file (.pdf)');
        return;
      }
      setErrorMsg('');
      setModalFile({
        name: file.name,
        size: `${(file.size / 1024).toFixed(1)} KB`
      });
    }
  };

  const handleSaveCover = () => {
    const targetCurr = curriculums.find(c => c.id === modalCurrId);
    if (!targetCurr) {
      setErrorMsg('Please select a curriculum.');
      return;
    }
    if (!modalFile) {
      setErrorMsg('Please upload a PDF cover sheet file.');
      return;
    }

    const now = new Date();
    const timeStr = `${String(now.getDate()).padStart(2, '0')}/${String(now.getMonth() + 1).padStart(2, '0')}/${now.getFullYear()} ${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`;

    setCurriculums(curriculums.map(c => c.id === targetCurr.id ? {
      ...c,
      coverFile: modalFile.name,
      coverSize: modalFile.size,
      coverUpdatedAt: timeStr
    } : c));

    setIsModalOpen(false);
    showToast(`Cover PDF saved for ${targetCurr.name}.`);
  };

  const handleExecuteDeleteSingle = () => {
    if (!deleteConfirmCurr) return;
    const name = deleteConfirmCurr.name;
    setCurriculums(curriculums.map(c => c.id === deleteConfirmCurr.id ? {
      ...c,
      coverFile: undefined,
      coverSize: undefined,
      coverUpdatedAt: undefined
    } : c));
    setDeleteConfirmCurr(null);
    showToast(`Cover removed from ${name}.`, 'info');
  };

  const handleExecuteBulkDelete = () => {
    setCurriculums(curriculums.map(c => selectedCurrIds.includes(c.id) ? {
      ...c,
      coverFile: undefined,
      coverSize: undefined,
      coverUpdatedAt: undefined
    } : c));
    setSelectedCurrIds([]);
    setIsBulkDeleteOpen(false);
    showToast(`Covers removed from selected curriculums.`, 'info');
  };

  const selectedWithCovers = curriculums.filter(c => selectedCurrIds.includes(c.id) && !!c.coverFile);
  const activeTargetCurr = curriculums.find(c => c.id === modalCurrId);
  const activeTargetCourses = activeTargetCurr ? getCoursesForCurriculum(activeTargetCurr.name) : [];

  return (
    <div className="p-6 space-y-6 w-full">
      {/* Toast Notification */}
      {toastMessage && (
        <div className="fixed bottom-6 right-6 z-50 animate-in fade-in slide-in-from-bottom-5 duration-300">
          <div className={`flex items-center gap-3 px-4 py-3 rounded-xl shadow-lg border text-sm font-medium ${
            toastMessage.type === 'error'
              ? 'bg-red-50 text-red-800 border-red-200'
              : toastMessage.type === 'info'
              ? 'bg-blue-50 text-blue-800 border-blue-200'
              : 'bg-emerald-50 text-emerald-800 border-emerald-200'
          }`}>
            {toastMessage.type === 'error' ? (
              <AlertCircle size={18} className="text-red-600 shrink-0" />
            ) : toastMessage.type === 'info' ? (
              <Info size={18} className="text-blue-600 shrink-0" />
            ) : (
              <CheckCircle2 size={18} className="text-emerald-600 shrink-0" />
            )}
            <span>{toastMessage.text}</span>
            <button 
              onClick={() => setToastMessage(null)} 
              className="ml-2 text-gray-400 hover:text-gray-600 p-0.5 rounded"
            >
              <X size={14} />
            </button>
          </div>
        </div>
      )}

      {/* Top Header: Title + Actions */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Cover Management</h1>
          <p className="text-sm text-gray-500 mt-1">
            Manage standard course cover PDFs by curriculum. All courses under the same curriculum share the same cover.
          </p>
        </div>

        <div className="flex items-center gap-3">
          {selectedWithCovers.length > 0 && (
            <button 
              onClick={() => setIsBulkDeleteOpen(true)} 
              className="flex items-center px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 font-medium text-sm transition-colors shadow-sm whitespace-nowrap"
            >
              <Trash2 size={16} className="mr-2" /> Remove Covers ({selectedWithCovers.length})
            </button>
          )}

          <button 
            onClick={() => handleOpenUpload()} 
            className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-sm whitespace-nowrap"
          >
            <Plus size={16} className="mr-2" /> Upload Cover PDF
          </button>
        </div>
      </div>

      {/* Main Layout: Table on Left (flex-1), Multi-Select Filter Panel on Right (w-72) */}
      <div className="flex flex-col lg:flex-row gap-6 items-start">
        {/* Left Area: Search Bar + Table */}
        <div className="flex-1 w-full min-w-0 space-y-4">
          {/* Search Bar + Results Status */}
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 bg-white p-3 rounded-xl border border-gray-200 shadow-sm">
            <div className="relative flex-1">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Search className="h-4 w-4 text-gray-400" />
              </div>
              <input
                type="text"
                placeholder="Search curriculum code, name, department, cover..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="block w-full pl-9 pr-3 py-1.5 border border-gray-200 rounded-lg text-sm bg-gray-50 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:bg-white transition-colors"
              />
            </div>
            <div className="text-xs text-gray-500 px-2 whitespace-nowrap">
              Showing <span className="font-semibold text-gray-900">{sortedCurriculums.length}</span> curriculum{sortedCurriculums.length === 1 ? '' : 's'}
            </div>
          </div>

          {/* Curriculums & Covers Table */}
          <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
            <div className="overflow-x-auto">
              <table className="min-w-full divide-y divide-gray-200">
                <thead className="bg-gray-50">
                  <tr>
                    <th className="w-12 px-6 py-3 text-left">
                      <input
                        type="checkbox"
                        checked={selectedCurrIds.length === sortedCurriculums.length && sortedCurriculums.length > 0}
                        onChange={handleSelectAllToggle}
                        className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                      />
                    </th>
                    <th 
                      onClick={() => handleSort('code')}
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                    >
                      <div className="flex items-center gap-1">
                        Code
                        {sortField === 'code' ? (
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
                        Curriculum Name
                        {sortField === 'name' ? (
                          sortDirection === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />
                        ) : (
                          <ArrowUpDown size={12} className="text-gray-400" />
                        )}
                      </div>
                    </th>
                    <th 
                      onClick={() => handleSort('department')}
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                    >
                      <div className="flex items-center gap-1">
                        Department
                        {sortField === 'department' ? (
                          sortDirection === 'asc' ? <ChevronUp size={14} /> : <ChevronDown size={14} />
                        ) : (
                          <ArrowUpDown size={12} className="text-gray-400" />
                        )}
                      </div>
                    </th>
                    <th 
                      onClick={() => handleSort('cover')}
                      className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                    >
                      <div className="flex items-center gap-1">
                        Cover PDF
                        {sortField === 'cover' ? (
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
                  {sortedCurriculums.map((curr) => {
                    const dept = departments.find(d => d.id === curr.departmentId);
                    const isSelected = selectedCurrIds.includes(curr.id);

                    return (
                      <tr 
                        key={curr.id}
                        onClick={() => handleToggleSelect(curr.id)}
                        className={`hover:bg-gray-50 transition-colors cursor-pointer ${
                          isSelected ? 'bg-indigo-50/40 hover:bg-indigo-50/60' : ''
                        }`}
                      >
                        <td className="px-6 py-4 whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                          <input 
                            type="checkbox" 
                            checked={isSelected} 
                            onChange={() => handleToggleSelect(curr.id)}
                            className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                          />
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                          <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">
                            {curr.code}
                          </span>
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                          {curr.name}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {dept?.name || '-'}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm">
                          {curr.coverFile ? (
                            <div className="flex items-center gap-2">
                              <div className="p-1 bg-rose-50 text-rose-600 rounded shrink-0">
                                <FileText size={15} />
                              </div>
                              <span 
                                className="font-medium text-gray-900 text-xs cursor-help"
                                title={curr.coverFile}
                              >
                                {curr.coverFile.length > 15 
                                  ? `${curr.coverFile.slice(0, 15)}...` 
                                  : curr.coverFile}
                              </span>
                            </div>
                          ) : (
                            <span className="text-gray-400 text-xs italic">No Cover</span>
                          )}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                          {curr.coverUpdatedAt ? (
                            <div className="flex items-center text-xs text-gray-500">
                              <CheckCircle2 size={12} className="mr-1 text-gray-400" />
                              {curr.coverUpdatedAt}
                            </div>
                          ) : (
                            '-'
                          )}
                        </td>
                        <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2" onClick={(e) => e.stopPropagation()}>
                          {curr.coverFile && (
                            <>
                              <button 
                                onClick={() => setPreviewCurr(curr)}
                                className="text-gray-600 hover:text-indigo-900 bg-gray-50 hover:bg-indigo-50 p-1.5 rounded-lg transition-colors inline-flex items-center"
                                title="Preview Cover PDF"
                              >
                                <Eye size={14} />
                              </button>
                              <button 
                                onClick={() => showToast(`Downloading ${curr.coverFile}...`)}
                                className="text-green-600 hover:text-green-900 bg-green-50 hover:bg-green-100 p-1.5 rounded-lg transition-colors inline-flex items-center"
                                title="Download PDF"
                              >
                                <Download size={14} />
                              </button>
                            </>
                          )}
                          <button 
                            onClick={() => handleOpenUpload(curr)}
                            className="text-indigo-600 hover:text-indigo-900 bg-indigo-50 hover:bg-indigo-100 p-1.5 rounded-lg transition-colors inline-flex items-center"
                            title={curr.coverFile ? "Replace Cover PDF" : "Upload Cover PDF"}
                          >
                            <UploadCloud size={14} />
                          </button>
                          {curr.coverFile && (
                            <button 
                              onClick={() => setDeleteConfirmCurr(curr)}
                              className="text-red-600 hover:text-red-900 bg-red-50 hover:bg-red-100 p-1.5 rounded-lg transition-colors inline-flex items-center"
                              title="Remove Cover"
                            >
                              <Trash2 size={14} />
                            </button>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                  {sortedCurriculums.length === 0 && (
                    <tr>
                      <td colSpan={7} className="text-center py-8 text-gray-500 italic">
                        No curriculums found matching your search and filter criteria.
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </div>
        </div>

        {/* Right Area: Multi-Select Filter Panel Sidebar */}
        <div className="w-full lg:w-72 shrink-0 bg-white rounded-xl shadow-sm border border-gray-200 p-5 space-y-6 sticky top-6">
          <div className="flex items-center justify-between pb-3 border-b border-gray-100">
            <div className="flex items-center gap-2 text-gray-900 font-semibold text-sm">
              <Filter size={16} className="text-indigo-600" />
              <span>Filters</span>
            </div>
            {hasActiveFilters && (
              <button 
                onClick={clearAllFilters}
                className="text-xs font-medium text-indigo-600 hover:text-indigo-800 flex items-center gap-1 transition-colors"
              >
                <RotateCcw size={12} /> Clear all
              </button>
            )}
          </div>

          {/* Department Filter */}
          <div>
            <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
              Department
            </label>
            <div className="space-y-1.5 max-h-48 overflow-y-auto pr-1">
              {departments.map((dept) => {
                const isChecked = selectedDepartments.includes(dept.id);
                const count = curriculums.filter(c => c.departmentId === dept.id).length;
                return (
                  <label 
                    key={dept.id} 
                    className="flex items-center justify-between p-1.5 rounded-lg hover:bg-gray-50 cursor-pointer text-sm"
                  >
                    <div className="flex items-center min-w-0 pr-2">
                      <input
                        type="checkbox"
                        checked={isChecked}
                        onChange={() => toggleDepartment(dept.id)}
                        className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                      />
                      <span className="ml-2 text-gray-700 truncate" title={dept.name}>
                        {dept.name}
                      </span>
                    </div>
                    <span className="text-xs text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded-full shrink-0">
                      {count}
                    </span>
                  </label>
                );
              })}
            </div>
          </div>

          {/* Cover Status Filter */}
          <div>
            <label className="block text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
              Cover Status
            </label>
            <div className="space-y-1.5">
              {[
                { id: 'has_cover', label: 'Cover Uploaded', count: curriculums.filter(c => !!c.coverFile).length },
                { id: 'no_cover', label: 'Missing Cover', count: curriculums.filter(c => !c.coverFile).length },
              ].map(status => {
                const isChecked = selectedStatuses.includes(status.id);
                return (
                  <label 
                    key={status.id} 
                    className="flex items-center justify-between p-1.5 rounded-lg hover:bg-gray-50 cursor-pointer text-sm"
                  >
                    <div className="flex items-center">
                      <input
                        type="checkbox"
                        checked={isChecked}
                        onChange={() => toggleStatus(status.id)}
                        className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                      />
                      <span className="ml-2 text-gray-700">{status.label}</span>
                    </div>
                    <span className="text-xs text-gray-400 bg-gray-100 px-1.5 py-0.5 rounded-full">
                      {status.count}
                    </span>
                  </label>
                );
              })}
            </div>
          </div>
        </div>
      </div>

      {/* Upload / Replace Cover Modal */}
      <Modal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
        title={isFixedCurriculum && activeTargetCurr ? `Upload Cover PDF - ${activeTargetCurr.name}` : "Upload Curriculum Cover PDF"}
        maxWidth="max-w-xl"
        footer={
          <>
            <button 
              onClick={() => setIsModalOpen(false)} 
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button 
              onClick={handleSaveCover} 
              disabled={!modalFile}
              className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              Save Cover
            </button>
          </>
        }
      >
        <div className="space-y-4">
          {errorMsg && (
            <div className="p-3 bg-red-50 text-red-700 rounded-xl text-sm flex items-start border border-red-100">
              <AlertCircle size={16} className="mt-0.5 mr-2 shrink-0" />
              <p>{errorMsg}</p>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Target Curriculum</label>
            {isFixedCurriculum && activeTargetCurr ? (
              <div className="w-full border border-gray-200 bg-gray-50 rounded-lg px-3.5 py-2.5 text-sm text-gray-900 font-medium flex items-center justify-between shadow-2xs">
                <div className="flex items-center gap-2 min-w-0">
                  <span className="font-mono bg-indigo-100 text-indigo-700 text-xs px-2.5 py-0.5 rounded font-bold uppercase shrink-0">
                    {activeTargetCurr.code}
                  </span>
                  <span className="truncate">{activeTargetCurr.name}</span>
                </div>
                <span className="text-xs text-gray-500 bg-gray-200/70 px-2 py-0.5 rounded shrink-0 ml-2">
                  {departments.find(d => d.id === activeTargetCurr.departmentId)?.name || 'No Dept'}
                </span>
              </div>
            ) : (
              <select
                value={modalCurrId}
                onChange={(e) => setModalCurrId(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white"
              >
                {curriculums.map((curr) => {
                  const dept = departments.find(d => d.id === curr.departmentId);
                  return (
                    <option key={curr.id} value={curr.id}>
                      [{curr.code}] {curr.name} — {dept?.name || 'No Dept'}
                    </option>
                  );
                })}
              </select>
            )}
          </div>

          {activeTargetCurr && (
            <div className="p-3 bg-indigo-50/70 border border-indigo-100 rounded-xl text-xs text-indigo-900 flex items-start gap-2">
              <Info size={15} className="text-indigo-600 mt-0.5 shrink-0" />
              <p>
                This cover will be automatically shared with all courses under <strong>{activeTargetCurr.name}</strong>.
              </p>
            </div>
          )}

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Course Cover PDF (cover.pdf)</label>
            {modalFile ? (
              <div className="flex items-center justify-between p-3 bg-rose-50/50 border border-rose-200 rounded-xl">
                <div className="flex items-center gap-3 min-w-0">
                  <div className="p-2 bg-rose-100 text-rose-700 rounded-lg shrink-0">
                    <FileText size={20} />
                  </div>
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-gray-900 truncate">
                      {modalFile.name}
                    </p>
                    <p className="text-xs text-gray-500">
                      Ready as curriculum cover
                    </p>
                  </div>
                </div>

                <div className="flex items-center gap-1.5 shrink-0 ml-2">
                  <label 
                    className="px-2.5 py-1 text-xs font-medium text-indigo-700 bg-white border border-indigo-200 rounded-lg hover:bg-indigo-50 cursor-pointer transition-colors shadow-xs"
                  >
                    Change
                    <input 
                      type="file" 
                      accept=".pdf,application/pdf" 
                      onChange={handleFileChange} 
                      className="hidden" 
                    />
                  </label>
                  <button 
                    type="button"
                    onClick={() => setModalFile(null)} 
                    className="p-1 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
                  >
                    <Trash2 size={16} />
                  </button>
                </div>
              </div>
            ) : (
              <div 
                onDragOver={(e) => { e.preventDefault(); setIsDragging(true); }}
                onDragLeave={() => setIsDragging(false)}
                onDrop={handleFileDrop}
                className={`relative border-2 border-dashed rounded-xl p-6 text-center transition-all cursor-pointer group ${
                  isDragging 
                    ? 'border-indigo-500 bg-indigo-50/40' 
                    : 'border-gray-300 bg-gray-50/50 hover:bg-indigo-50/20 hover:border-indigo-300'
                }`}
              >
                <input 
                  type="file" 
                  accept=".pdf,application/pdf" 
                  onChange={handleFileChange} 
                  className="absolute inset-0 w-full h-full opacity-0 cursor-pointer" 
                />
                <div className="space-y-1">
                  <UploadCloud className="mx-auto h-10 w-10 text-gray-400 group-hover:text-indigo-600 transition-colors" />
                  <p className="text-xs font-medium text-gray-700">
                    <span className="text-indigo-600 font-semibold">Click to upload</span> or drag and drop <span className="font-mono text-gray-600 font-medium">cover.pdf</span>
                  </p>
                  <p className="text-[11px] text-gray-400">
                    Standard A4 course cover sheet (PDF only)
                  </p>
                </div>
              </div>
            )}
          </div>
        </div>
      </Modal>

      {/* Delete Single Confirmation Modal */}
      <Modal
        isOpen={!!deleteConfirmCurr}
        onClose={() => setDeleteConfirmCurr(null)}
        title="Remove Cover PDF"
        footer={
          <>
            <button 
              onClick={() => setDeleteConfirmCurr(null)} 
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button 
              onClick={handleExecuteDeleteSingle} 
              className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
            >
              Remove Cover
            </button>
          </>
        }
      >
        {deleteConfirmCurr && (
          <p className="text-sm text-gray-600">
            Are you sure you want to remove the cover PDF from curriculum <strong>"{deleteConfirmCurr.name}"</strong>? The <strong>{getCoursesForCurriculum(deleteConfirmCurr.name).length}</strong> course(s) under this curriculum will no longer have a cover sheet attached until a new one is uploaded.
          </p>
        )}
      </Modal>

      {/* Bulk Delete Modal */}
      <Modal
        isOpen={isBulkDeleteOpen}
        onClose={() => setIsBulkDeleteOpen(false)}
        title="Remove Selected Covers"
        footer={
          <>
            <button 
              onClick={() => setIsBulkDeleteOpen(false)} 
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Cancel
            </button>
            <button 
              onClick={handleExecuteBulkDelete} 
              className="px-4 py-2 text-sm font-medium text-white bg-red-600 rounded-lg hover:bg-red-700 transition-colors"
            >
              Remove Covers ({selectedWithCovers.length})
            </button>
          </>
        }
      >
        <p className="text-sm text-gray-600">
          Are you sure you want to remove cover PDFs from the <strong>{selectedWithCovers.length}</strong> selected curriculum(s)? This action cannot be undone.
        </p>
      </Modal>

      {/* Realistic Preview Modal */}
      <Modal
        isOpen={!!previewCurr}
        onClose={() => setPreviewCurr(null)}
        title={`Cover Sheet Preview - ${previewCurr?.name}`}
        maxWidth="max-w-2xl"
        footer={
          <>
            <button
              onClick={() => setPreviewCurr(null)}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Close Preview
            </button>
            {previewCurr && (
              <button
                onClick={() => showToast(`Downloading ${previewCurr.coverFile}...`)}
                className="flex items-center px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors shadow-sm"
              >
                <Download size={16} className="mr-1.5" /> Download PDF
              </button>
            )}
          </>
        }
      >
        {previewCurr && (
          <div className="space-y-4">
            <div className="flex items-center justify-between bg-gray-900 text-white px-4 py-2 rounded-lg text-xs font-mono">
              <div className="flex items-center gap-2">
                <FileText size={14} className="text-rose-400" />
                <span className="font-semibold truncate max-w-[280px]">{previewCurr.coverFile || 'cover_sheet.pdf'}</span>
              </div>
              <div className="flex items-center gap-3 text-gray-300">
                <span>{previewCurr.coverSize || '240 KB'}</span>
                <span>•</span>
                <span>Page 1 of 1</span>
              </div>
            </div>

            <div className="bg-gray-200 p-4 rounded-xl flex justify-center">
              <div className="bg-white w-full max-w-[480px] min-h-[540px] shadow-md rounded border border-gray-300 p-8 flex flex-col justify-between text-gray-800 select-none">
                <div className="text-center space-y-2 border-b-2 border-indigo-900 pb-4">
                  <div className="mx-auto w-10 h-10 rounded-full bg-indigo-900 text-white flex items-center justify-center font-bold text-xs">
                    ABET
                  </div>
                  <h2 className="text-xs font-bold tracking-wider text-gray-900 uppercase">
                    FACULTY OF ENGINEERING & INFORMATION TECHNOLOGY
                  </h2>
                  <p className="text-[11px] font-semibold text-indigo-950 uppercase">
                    {departments.find(d => d.id === previewCurr.departmentId)?.name || 'DEPARTMENT'}
                  </p>
                  <div className="inline-block bg-indigo-50 border border-indigo-200 px-2 py-0.5 rounded text-[10px] font-bold text-indigo-700">
                    CURRICULUM: {previewCurr.name.toUpperCase()} ({previewCurr.code})
                  </div>
                </div>

                <div className="py-6 space-y-4 text-center">
                  <div>
                    <h3 className="text-base font-black text-gray-900 tracking-tight leading-snug uppercase">
                      ABET COURSE PORTFOLIO & ASSESSMENT DOSSIER
                    </h3>
                    <p className="text-[11px] text-gray-500 mt-1 italic">
                      Continuous Quality Improvement (CQI) Course Summary Sheet
                    </p>
                  </div>

                  <div className="bg-indigo-50/50 border border-dashed border-indigo-300 rounded-lg p-3 text-left space-y-1.5 text-xs">
                    <div className="flex justify-between items-center text-[10px] text-indigo-700 font-bold uppercase tracking-wider">
                      <span>Shared Course Placement Slot</span>
                      <span>All {previewCurr.code} Courses</span>
                    </div>
                    <div className="grid grid-cols-2 gap-2 text-[11px] pt-1">
                      <div>
                        <span className="text-gray-500 block text-[9px] uppercase font-semibold">Sample Course</span>
                        <span className="font-bold text-gray-900">
                          {getCoursesForCurriculum(previewCurr.name)[0]?.id || `${previewCurr.code}101`}:{' '}
                          {getCoursesForCurriculum(previewCurr.name)[0]?.name || 'Course Title'}
                        </span>
                      </div>
                      <div>
                        <span className="text-gray-500 block text-[9px] uppercase font-semibold">Academic Year</span>
                        <span className="font-bold text-gray-900">2024 / Semester 1</span>
                      </div>
                    </div>
                  </div>

                  <div className="flex items-center justify-center gap-2 pt-2 text-[10px] font-bold text-emerald-800 bg-emerald-50/80 border border-emerald-200/80 rounded py-1.5">
                    <ShieldCheck size={14} className="text-emerald-600" />
                    <span>Official ABET Accredited Program Template</span>
                  </div>
                </div>

                <div className="pt-4 border-t border-gray-200">
                  <div className="grid grid-cols-3 gap-2 text-center text-[9px] text-gray-500">
                    <div>
                      <div className="border-b border-gray-400 h-8 mb-1"></div>
                      <span className="font-semibold block text-gray-700">Course Instructor</span>
                      <span>Signature & Date</span>
                    </div>
                    <div>
                      <div className="border-b border-gray-400 h-8 mb-1"></div>
                      <span className="font-semibold block text-gray-700">Curriculum Chair</span>
                      <span>[{previewCurr.code}] Program</span>
                    </div>
                    <div>
                      <div className="border-b border-gray-400 h-8 mb-1"></div>
                      <span className="font-semibold block text-gray-700">Department Head</span>
                      <span>Approval Stamp</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        )}
      </Modal>
    </div>
  );
};

const CourseList = () => {
  const [uploadTargetCourse, setUploadTargetCourse] = useState(null);
  const [selectedErrorFile, setSelectedErrorFile] = useState(null);
  const [expandedErrorIds, setExpandedErrorIds] = useState([]);

  const [selectedYear, setSelectedYear] = useState('2024');
  const [selectedSemesters, setSelectedSemesters] = useState([]);
  const [selectedDepartments, setSelectedDepartments] = useState([]);
  const [selectedCurriculums, setSelectedCurriculums] = useState([]);
  const [searchQuery, setSearchQuery] = useState('');

  const toggleSemester = (sem) => {
    if (selectedSemesters.includes(sem)) {
      setSelectedSemesters(selectedSemesters.filter(s => s !== sem));
    } else {
      setSelectedSemesters([...selectedSemesters, sem]);
    }
  };

  const toggleDepartment = (deptName) => {
    if (selectedDepartments.includes(deptName)) {
      setSelectedDepartments(selectedDepartments.filter(d => d !== deptName));
    } else {
      setSelectedDepartments([...selectedDepartments, deptName]);
    }
  };

  const toggleCurriculum = (currName) => {
    if (selectedCurriculums.includes(currName)) {
      setSelectedCurriculums(selectedCurriculums.filter(c => c !== currName));
    } else {
      setSelectedCurriculums([...selectedCurriculums, currName]);
    }
  };

  const clearAllFilters = () => {
    setSelectedSemesters([]);
    setSelectedDepartments([]);
    setSelectedCurriculums([]);
    setSearchQuery('');
  };

  const hasActiveFilters = selectedSemesters.length > 0 || selectedDepartments.length > 0 || selectedCurriculums.length > 0 || searchQuery !== '';

  const handleOpenErrors = (course, file) => {
    setSelectedErrorFile({ course, file });
    setExpandedErrorIds([]);
  };

  const filteredCourses = MOCK_COURSES.filter(course => {
    const file = MOCK_FILES.find(f => f.courseId === course.id && f.year === selectedYear);
    const statusText = file ? file.status : 'No files';
    const query = searchQuery.toLowerCase();

    const matchesYear = (course.year || '2024') === selectedYear;
    const matchesSemester = selectedSemesters.length === 0 || selectedSemesters.includes(course.semester || '1');
    const matchesDept = selectedDepartments.length === 0 || selectedDepartments.includes(course.department || '');
    const matchesCurriculum = selectedCurriculums.length === 0 || selectedCurriculums.includes(course.curriculum || '');

    const matchesSearch = (
      course.id.toLowerCase().includes(query) ||
      course.name.toLowerCase().includes(query) ||
      (course.department || '').toLowerCase().includes(query) ||
      (course.curriculum || '').toLowerCase().includes(query) ||
      statusText.toLowerCase().includes(query)
    );

    return matchesYear && matchesSemester && matchesDept && matchesCurriculum && matchesSearch;
  });

  return (
    <div className="p-6 space-y-6 w-full">
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
        {/* Left: Title + Year Selector */}
        <div className="flex flex-wrap items-center gap-3">
          <h1 className="text-2xl font-bold text-gray-900">Courses</h1>
          <div className="flex items-center bg-white border border-gray-300 rounded-lg shadow-sm px-3 py-1.5 ml-1">
            <span className="text-xs font-semibold text-gray-500 mr-2 uppercase tracking-wide">Year:</span>
            <select 
              value={selectedYear}
              onChange={(e) => setSelectedYear(e.target.value)}
              className="bg-transparent text-sm font-bold text-indigo-700 outline-none cursor-pointer"
            >
              <option value="2025">2025</option>
              <option value="2024">2024</option>
              <option value="2023">2023</option>
              <option value="2022">2022</option>
            </select>
          </div>
        </div>
      </div>

      {/* Main Layout: Course List on Left (flex-1), Multi-Select Filter Panel on Right (w-72) */}
      <div className="flex flex-col lg:flex-row gap-6 items-start">
        {/* Left Area: Course Cards Grid */}
        <div className="flex-1 w-full min-w-0 space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3 bg-white p-3 rounded-xl border border-gray-200 shadow-sm">
            <div className="relative flex-1">
              <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                <Search className="h-4 w-4 text-gray-400" />
              </div>
              <input
                type="text"
                placeholder="Search course ID, name, status..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="block w-full pl-9 pr-3 py-1.5 border border-gray-200 rounded-lg text-sm bg-gray-50 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:bg-white transition-colors"
              />
            </div>
            <div className="text-xs text-gray-500 px-2 whitespace-nowrap">
              Showing <span className="font-semibold text-gray-900">{filteredCourses.length}</span> course{filteredCourses.length === 1 ? '' : 's'} in <span className="font-semibold text-indigo-600">{selectedYear}</span>
            </div>
          </div>

          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
            {filteredCourses.map(course => {
              const file = MOCK_FILES.find(f => f.courseId === course.id && f.year === selectedYear);

              return (
                <div key={course.id} className="group bg-white rounded-xl shadow-sm border border-gray-200 p-4 hover:shadow-md hover:border-indigo-300 transition-all flex flex-col justify-between">
                  <div>
                    {/* Top Row: Course ID + Semester + Status */}
                    <div className="flex justify-between items-start mb-2">
                      <div className="flex items-center gap-1.5">
                        <span className="text-xs font-bold tracking-wider text-indigo-700 bg-indigo-50 px-2.5 py-1 rounded-md uppercase">
                          {course.id}
                        </span>
                        <span className="text-[10px] font-medium text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded">
                          {course.semester === 'Summer' ? 'Summer' : `Sem ${course.semester || '1'}`}
                        </span>
                      </div>
                      {file && (
                        file.status === 'failed' ? (
                          <Badge 
                            variant="failed" 
                            onClick={() => handleOpenErrors(course, file)}
                            title="Click to view processing errors"
                          >
                            <AlertCircle size={12} className="mr-1 inline text-red-600 shrink-0" />
                            <span>Failed</span>
                            <span className="ml-1 text-[10px] text-red-700 font-semibold bg-red-100/90 px-1.5 py-0.2 rounded-full">
                              {file.errors?.length || 1}
                            </span>
                          </Badge>
                        ) : (
                          <Badge variant={file.status}>
                            {file.status.charAt(0).toUpperCase() + file.status.slice(1)}
                          </Badge>
                        )
                      )}
                    </div>

                    {/* Middle: Course Name */}
                    <h3 className="text-sm font-semibold text-gray-900 leading-snug mb-3">
                      {course.name}
                    </h3>
                  </div>

                  {/* Bottom: Timestamp & Action Buttons */}
                  <div className="pt-3 border-t border-gray-100 mt-2">
                    <div className="h-4 mb-2">
                      {file ? (
                        <p className="text-[11px] text-gray-400 font-medium flex items-center">
                          <CheckCircle2 size={12} className="mr-1 text-gray-400" />
                          Updated: {file.time}
                        </p>
                      ) : (
                        <p className="text-[11px] text-gray-400 font-medium italic">No files for {selectedYear}</p>
                      )}
                    </div>
                    
                    <div className="flex justify-between items-center">
                      <div className="flex items-center gap-2">
                        <button 
                          onClick={() => setUploadTargetCourse(course)}
                          className="flex items-center text-xs font-medium text-indigo-600 hover:text-indigo-800 transition-colors"
                        >
                          <UploadCloud size={14} className="mr-1" /> {file?.status === 'failed' ? 'Re-upload' : 'Upload'}
                        </button>
                        {file?.status === 'failed' && (
                          <button 
                            onClick={() => handleOpenErrors(course, file)}
                            className="flex items-center text-xs font-medium text-red-600 hover:text-red-800 transition-colors"
                            title="View Error Details"
                          >
                            <AlertTriangle size={14} className="mr-1" /> Errors
                          </button>
                        )}
                      </div>
                      
                      {file?.status === 'generated' && (
                        <button className="flex items-center text-xs font-medium text-green-600 hover:text-green-800 transition-colors">
                          <Download size={14} className="mr-1" /> Download
                        </button>
                      )}
                    </div>
                  </div>
                </div>
              );
            })}
            {filteredCourses.length === 0 && (
              <div className="col-span-full py-12 text-center text-gray-400 italic bg-white rounded-xl border border-gray-200">
                No courses found matching the selected filters.
              </div>
            )}
          </div>
        </div>

        {/* Right Area: Multi-Select Filter Panel Sidebar */}
        <div className="w-full lg:w-72 shrink-0 bg-white rounded-xl shadow-sm border border-gray-200 p-5 space-y-6 sticky top-6">
          <div className="flex items-center justify-between pb-3 border-b border-gray-100">
            <div className="flex items-center gap-2 text-gray-900 font-semibold text-sm">
              <Filter size={16} className="text-indigo-600" />
              <span>Filters</span>
            </div>
            {hasActiveFilters && (
              <button 
                onClick={clearAllFilters}
                className="text-xs font-medium text-indigo-600 hover:text-indigo-800 flex items-center gap-1 transition-colors"
              >
                <RotateCcw size={12} /> Reset
              </button>
            )}
          </div>

          {/* Semester Multi-Select */}
          <div>
            <div className="flex items-center justify-between mb-2.5">
              <span className="text-xs font-bold text-gray-700 uppercase tracking-wider">Semester</span>
              <span className="text-[11px] text-gray-400 font-medium">
                {selectedSemesters.length === 0 ? 'All' : `${selectedSemesters.length} selected`}
              </span>
            </div>
            <div className="space-y-1.5">
              {[
                { id: '1', label: 'Semester 1' },
                { id: '2', label: 'Semester 2' },
                { id: 'Summer', label: 'Summer' }
              ].map(sem => {
                const isChecked = selectedSemesters.includes(sem.id);
                return (
                  <label 
                    key={sem.id}
                    className={`flex items-center justify-between p-2 rounded-lg text-sm cursor-pointer transition-colors ${
                      isChecked ? 'bg-indigo-50/70 text-indigo-900 font-medium' : 'hover:bg-gray-50 text-gray-700'
                    }`}
                  >
                    <div className="flex items-center gap-2.5">
                      <input 
                        type="checkbox"
                        checked={isChecked}
                        onChange={() => toggleSemester(sem.id)}
                        className="h-4 w-4 text-indigo-600 rounded border-gray-300 focus:ring-indigo-500 cursor-pointer"
                      />
                      <span>{sem.label}</span>
                    </div>
                  </label>
                );
              })}
            </div>
          </div>

          {/* Department Multi-Select */}
          <div className="pt-4 border-t border-gray-100">
            <div className="flex items-center justify-between mb-2.5">
              <span className="text-xs font-bold text-gray-700 uppercase tracking-wider">Department</span>
              <span className="text-[11px] text-gray-400 font-medium">
                {selectedDepartments.length === 0 ? 'All' : `${selectedDepartments.length} selected`}
              </span>
            </div>
            <div className="space-y-1.5">
              {INITIAL_DEPARTMENTS.map(dept => {
                const isChecked = selectedDepartments.includes(dept.name);
                return (
                  <label 
                    key={dept.id}
                    className={`flex items-start gap-2.5 p-2 rounded-lg text-sm cursor-pointer transition-colors ${
                      isChecked ? 'bg-indigo-50/70 text-indigo-900 font-medium' : 'hover:bg-gray-50 text-gray-700'
                    }`}
                  >
                    <input 
                      type="checkbox"
                      checked={isChecked}
                      onChange={() => toggleDepartment(dept.name)}
                      className="h-4 w-4 mt-0.5 text-indigo-600 rounded border-gray-300 focus:ring-indigo-500 cursor-pointer shrink-0"
                    />
                    <span className="leading-snug text-xs font-medium">{dept.name}</span>
                  </label>
                );
              })}
            </div>
          </div>

          {/* Curriculum Multi-Select */}
          <div className="pt-4 border-t border-gray-100">
            <div className="flex items-center justify-between mb-2.5">
              <span className="text-xs font-bold text-gray-700 uppercase tracking-wider">Curriculum</span>
              <span className="text-[11px] text-gray-400 font-medium">
                {selectedCurriculums.length === 0 ? 'All' : `${selectedCurriculums.length} selected`}
              </span>
            </div>
            <div className="space-y-1.5 max-h-60 overflow-y-auto">
              {INITIAL_CURRICULUMS
                .filter(curr => {
                  if (selectedDepartments.length === 0) return true;
                  const parentDept = INITIAL_DEPARTMENTS.find(d => d.id === curr.departmentId);
                  return parentDept && selectedDepartments.includes(parentDept.name);
                })
                .map(curr => {
                  const isChecked = selectedCurriculums.includes(curr.name);
                  return (
                    <label 
                      key={curr.id}
                      className={`flex items-start gap-2.5 p-2 rounded-lg text-sm cursor-pointer transition-colors ${
                        isChecked ? 'bg-indigo-50/70 text-indigo-900 font-medium' : 'hover:bg-gray-50 text-gray-700'
                      }`}
                    >
                      <input 
                        type="checkbox"
                        checked={isChecked}
                        onChange={() => toggleCurriculum(curr.name)}
                        className="h-4 w-4 mt-0.5 text-indigo-600 rounded border-gray-300 focus:ring-indigo-500 cursor-pointer shrink-0"
                      />
                      <span className="leading-snug text-xs font-medium">{curr.name}</span>
                    </label>
                  );
                })}
            </div>
          </div>

          <div className="pt-3 border-t border-gray-100">
            <p className="text-[11px] text-gray-400 text-center italic">
              Unselected categories default to showing all
            </p>
          </div>
        </div>
      </div>

      {/* Processing Errors Modal */}
      <Modal
        isOpen={!!selectedErrorFile}
        onClose={() => setSelectedErrorFile(null)}
        title="Processing & Validation Errors"
        maxWidth="max-w-2xl"
        footer={
          <>
            <button 
              onClick={() => setSelectedErrorFile(null)} 
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors"
            >
              Close
            </button>
            <button 
              onClick={() => {
                const targetCourse = selectedErrorFile?.course;
                setSelectedErrorFile(null);
                if (targetCourse) setUploadTargetCourse(targetCourse);
              }} 
              className="flex items-center px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors shadow-sm"
            >
              <UploadCloud size={16} className="mr-2" /> Re-upload Files
            </button>
          </>
        }
      >
        {selectedErrorFile && (
          <div className="space-y-4">
            {/* Overview Banner */}
            <div className="bg-red-50 border border-red-200 rounded-xl p-4">
              <div className="flex items-start gap-3">
                <div className="p-2 bg-red-100 text-red-700 rounded-lg shrink-0">
                  <AlertCircle size={20} />
                </div>
                <div className="flex-1 min-w-0">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <h4 className="text-sm font-bold text-red-900">
                      {selectedErrorFile.course.id}: {selectedErrorFile.course.name}
                    </h4>
                    <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-bold bg-red-200 text-red-800">
                      {selectedErrorFile.file.errors?.length || 0} Error{(selectedErrorFile.file.errors?.length || 0) === 1 ? '' : 's'}
                    </span>
                  </div>
                  <p className="text-xs text-red-700 mt-1">
                    File: <span className="font-semibold">{selectedErrorFile.file.name}</span> • Processed: {selectedErrorFile.file.time}
                  </p>
                  <p className="text-xs text-red-600 mt-1">
                    The processing pipeline failed during file analysis. Click each error row below to expand details and suggestions.
                  </p>
                </div>
              </div>
            </div>

            {/* Expand/Collapse All Toolbar */}
            <div className="flex items-center justify-between px-1">
              <span className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
                Error Breakdown ({selectedErrorFile.file.errors?.length || 0})
              </span>
              <button
                onClick={() => {
                  const allIds = selectedErrorFile.file.errors?.map((e) => e.id) || [];
                  if (expandedErrorIds.length === allIds.length) {
                    setExpandedErrorIds([]);
                  } else {
                    setExpandedErrorIds(allIds);
                  }
                }}
                className="text-xs font-medium text-indigo-600 hover:text-indigo-800 transition-colors"
              >
                {expandedErrorIds.length === (selectedErrorFile.file.errors?.length || 0) ? 'Collapse All' : 'Expand All'}
              </button>
            </div>

            {/* Expandable Error Rows */}
            <div className="space-y-2.5">
              {selectedErrorFile.file.errors && selectedErrorFile.file.errors.length > 0 ? (
                selectedErrorFile.file.errors.map((error, index) => {
                  const isExpanded = expandedErrorIds.includes(error.id);
                  
                  const stageColors = {
                    'Validation': 'bg-amber-100 text-amber-800 border-amber-200',
                    'CLO Mapping': 'bg-purple-100 text-purple-800 border-purple-200',
                    'Score Calculation': 'bg-blue-100 text-blue-800 border-blue-200',
                    'Upload': 'bg-rose-100 text-rose-800 border-rose-200',
                    'Report Generation': 'bg-indigo-100 text-indigo-800 border-indigo-200',
                  };

                  return (
                    <div 
                      key={error.id || index}
                      className="border border-gray-200 rounded-xl overflow-hidden bg-white shadow-sm transition-all"
                    >
                      {/* Row Header (Clickable) */}
                      <div 
                        onClick={() => {
                          if (isExpanded) {
                            setExpandedErrorIds(expandedErrorIds.filter(id => id !== error.id));
                          } else {
                            setExpandedErrorIds([...expandedErrorIds, error.id]);
                          }
                        }}
                        className="flex items-center justify-between p-3.5 cursor-pointer hover:bg-gray-50/80 transition-colors gap-3 select-none"
                      >
                        <div className="flex items-center gap-2.5 min-w-0 flex-1">
                          <div className={`text-gray-400 transform transition-transform duration-150 ${isExpanded ? 'rotate-90 text-indigo-600' : ''}`}>
                            <ChevronRight size={16} />
                          </div>

                          <span className={`px-2 py-0.5 rounded text-[11px] font-semibold border shrink-0 ${stageColors[error.stage] || 'bg-gray-100 text-gray-700'}`}>
                            {error.stage}
                          </span>

                          <span className="font-mono text-xs text-gray-500 shrink-0 font-medium hidden sm:inline">
                            {error.code}
                          </span>

                          <span className="text-sm font-semibold text-gray-900 truncate">
                            {error.title}
                          </span>
                        </div>

                        <span className="text-xs text-indigo-600 font-medium shrink-0">
                          {isExpanded ? 'Hide' : 'Details'}
                        </span>
                      </div>

                      {/* Expanded Detail Panel */}
                      {isExpanded && (
                        <div className="p-4 pt-2 border-t border-gray-100 bg-gray-50/60 space-y-3">
                          <div>
                            <span className="text-xs font-semibold text-gray-700 uppercase tracking-wide">Issue Description</span>
                            <p className="text-sm text-gray-800 mt-1 leading-relaxed">
                              {error.description}
                            </p>
                          </div>

                          {error.location && (
                            <div className="flex items-center gap-2 text-xs text-gray-600">
                              <span className="font-semibold text-gray-700">Location:</span>
                              <span className="bg-gray-200/80 text-gray-800 font-mono px-2 py-0.5 rounded text-[11px]">
                                {error.location}
                              </span>
                            </div>
                          )}

                          {error.suggestedFix && (
                            <div className="bg-emerald-50 border border-emerald-200/80 rounded-lg p-3 text-xs text-emerald-900">
                              <span className="font-bold flex items-center gap-1.5 mb-1 text-emerald-800">
                                💡 Suggested Fix:
                              </span>
                              <p className="leading-relaxed text-emerald-800">{error.suggestedFix}</p>
                            </div>
                          )}
                        </div>
                      )}
                    </div>
                  );
                })
              ) : (
                <div className="p-4 text-center text-sm text-gray-500 bg-gray-50 rounded-xl border border-gray-200 italic">
                  An unexpected error occurred during processing. Please review your file format or re-upload.
                </div>
              )}
            </div>
          </div>
        )}
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
             <p>Uploading files for <span className="font-semibold">{uploadTargetCourse?.name}</span>, Academic Year <span className="font-semibold">{selectedYear}</span>.</p>
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
  const [activeTab, setActiveTab] = useState('courses'); // 'courses', 'course_management', 'cover_management', 'users'
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
            <>
              <button
                onClick={() => setActiveTab('course_management')}
                className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'course_management' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
              >
                <BookMarked size={18} className="mr-3" /> Course Management
              </button>
              <button
                onClick={() => setActiveTab('cover_management')}
                className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'cover_management' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
              >
                <Layers size={18} className="mr-3" /> Cover
              </button>
              <button
                onClick={() => setActiveTab('users')}
                className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'users' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
              >
                <Users size={18} className="mr-3" /> Users
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
        {activeTab === 'courses' && <CourseList />}
        {activeTab === 'course_management' && <CourseManagement />}
        {activeTab === 'cover_management' && <CoverManagement />}
        {activeTab === 'users' && <UserManagement />}
      </div>
    </div>
  );
}