import React, { useState } from 'react';
import { 
  Users, BookOpen, LogOut, Plus, X, Edit2, 
  Trash2, Download, UploadCloud, AlertCircle, CheckCircle2,
  Moon, Building2, GraduationCap, BookMarked,
  Search, ArrowUpDown, ChevronUp, ChevronDown, Filter, RotateCcw,
  ChevronRight, AlertTriangle, FileText, Eye, Layers,
  Info, ShieldCheck, UserPlus, Check
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
  coverFile?: string;
  coverSize?: string;
  coverUpdatedAt?: string;
}

export interface ProcessingError {
  id: string;
  code: string;
  stage: 'Upload' | 'Validation' | 'CLO Mapping' | 'Score Calculation' | 'Report Generation';
  title: string;
  description: string;
  location?: string;
  suggestedFix?: string;
  rawSnippet?: string;
}

export interface CourseFile {
  id: number;
  courseId: string;
  name: string;
  year: string;
  semester: string;
  time: string;
  status: 'generated' | 'processing' | 'failed';
  errors?: ProcessingError[];
}

const INITIAL_DEPARTMENTS: Department[] = [
  { id: '1', name: 'Mathematics and Computer Science', code: 'MCS' },
  { id: '2', name: 'Electrical Engineering', code: 'EE' }
];

const INITIAL_CURRICULUMS: Curriculum[] = [
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

// --- MOCK DATA ---
const MOCK_COURSES = [
  { id: 'CS101', name: 'Introduction to Computer Science', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', year: '2024', semester: '1' },
  { id: 'MATH205', name: 'Calculus and Linear Algebra', department: 'Mathematics and Computer Science', curriculum: 'Mathematics', year: '2024', semester: '1' },
  { id: 'CS102', name: 'Data Structures and Algorithms', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', year: '2024', semester: '2' },
  { id: 'CS305', name: 'Software Engineering', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', year: '2024', semester: '1' },
  { id: 'EE101', name: 'Circuit Theory I', department: 'Electrical Engineering', curriculum: 'Electrical Engineering', year: '2024', semester: '2' }
];

const MOCK_USERS = [
  { id: 1, name: 'Alice Smith', email: 'alice.admin@company.com', role: 'super_admin', department: 'Mathematics and Computer Science', curriculum: 'Computer Science' },
  { id: 2, name: 'Bob Johnson', email: 'bob.admin@company.com', role: 'admin', department: 'Mathematics and Computer Science', curriculum: 'Mathematics' },
  { id: 3, name: 'Charlie Davis', email: 'charlie.staff@company.com', role: 'staff', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', courses: [{ id: 'CS101', access: 'edit' }, { id: 'MATH205', access: 'view' }] },
  { id: 4, name: 'Diana Prince', email: 'diana.instructor@company.com', role: 'instructor', department: 'Mathematics and Computer Science', curriculum: 'Computer Science', courses: [{ id: 'CS101', access: 'edit' }] },
  { id: 5, name: 'Evan Wright', email: 'evan.qa@company.com', role: 'staff', department: 'Mathematics and Computer Science', curriculum: 'Computer Science' },
  { id: 6, name: 'Fiona Gallagher', email: 'fiona.ee@company.com', role: 'instructor', department: 'Electrical Engineering', curriculum: 'Electrical Engineering' },
];

export interface CourseUserAccess {
  id: string;
  userId: number;
  userName: string;
  userEmail: string;
  userRole: string;
  userDept?: string;
  access: 'Edit and Download' | 'Download';
  isDefault?: boolean;
  defaultReason?: string;
}

const INITIAL_COURSE_ASSIGNMENTS: Record<string, CourseUserAccess[]> = {
  'CS101': [
    {
      id: 'a1',
      userId: 4,
      userName: 'Diana Prince',
      userEmail: 'diana.instructor@company.com',
      userRole: 'instructor',
      userDept: 'Mathematics and Computer Science',
      access: 'Edit and Download',
      isDefault: true,
      defaultReason: 'Primary Course Instructor'
    },
    {
      id: 'a2',
      userId: 3,
      userName: 'Charlie Davis',
      userEmail: 'charlie.staff@company.com',
      userRole: 'staff',
      userDept: 'Mathematics and Computer Science',
      access: 'Edit and Download',
      isDefault: false,
      defaultReason: 'Teaching Assistant'
    },
    {
      id: 'a3',
      userId: 1,
      userName: 'Alice Smith',
      userEmail: 'alice.admin@company.com',
      userRole: 'super_admin',
      userDept: 'Mathematics and Computer Science',
      access: 'Download',
      isDefault: true,
      defaultReason: 'Department Oversight (Default)'
    },
    {
      id: 'a4',
      userId: 5,
      userName: 'Evan Wright',
      userEmail: 'evan.qa@company.com',
      userRole: 'staff',
      userDept: 'Mathematics and Computer Science',
      access: 'Download',
      isDefault: false,
      defaultReason: 'External Reviewer'
    }
  ],
  'MATH205': [
    {
      id: 'a5',
      userId: 2,
      userName: 'Bob Johnson',
      userEmail: 'bob.admin@company.com',
      userRole: 'admin',
      userDept: 'Mathematics and Computer Science',
      access: 'Edit and Download',
      isDefault: true,
      defaultReason: 'Lead Mathematics Instructor'
    },
    {
      id: 'a6',
      userId: 3,
      userName: 'Charlie Davis',
      userEmail: 'charlie.staff@company.com',
      userRole: 'staff',
      userDept: 'Mathematics and Computer Science',
      access: 'Download',
      isDefault: false
    }
  ],
  'CS102': [
    {
      id: 'a7',
      userId: 4,
      userName: 'Diana Prince',
      userEmail: 'diana.instructor@company.com',
      userRole: 'instructor',
      userDept: 'Mathematics and Computer Science',
      access: 'Edit and Download',
      isDefault: false
    },
    {
      id: 'a8',
      userId: 1,
      userName: 'Alice Smith',
      userEmail: 'alice.admin@company.com',
      userRole: 'super_admin',
      userDept: 'Mathematics and Computer Science',
      access: 'Download',
      isDefault: true
    }
  ],
  'EE101': [
    {
      id: 'a9',
      userId: 6,
      userName: 'Fiona Gallagher',
      userEmail: 'fiona.ee@company.com',
      userRole: 'instructor',
      userDept: 'Electrical Engineering',
      access: 'Edit and Download',
      isDefault: true
    },
    {
      id: 'a10',
      userId: 1,
      userName: 'Alice Smith',
      userEmail: 'alice.admin@company.com',
      userRole: 'super_admin',
      userDept: 'Electrical Engineering',
      access: 'Download',
      isDefault: false
    }
  ]
};

const MOCK_FILES: CourseFile[] = [
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
const Badge = ({ 
  children, 
  variant = 'gray',
  onClick,
  title
}: { 
  children: React.ReactNode; 
  variant: string;
  onClick?: () => void;
  title?: string;
}) => {
  const variants: Record<string, string> = {
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

const Modal = ({ isOpen, onClose, title, children, footer, maxWidth = 'max-w-lg' }: {
  isOpen: boolean;
  onClose: () => void;
  title: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  maxWidth?: string;
}) => {
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

const SearchableSelect = ({
  label,
  value,
  onChange,
  options,
  allLabel = 'All',
  icon: Icon,
  disabled = false,
  className = '',
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  options: { value: string; label: string; subtext?: string }[];
  allLabel?: string;
  icon?: React.ElementType;
  disabled?: boolean;
  className?: string;
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const containerRef = React.useRef<HTMLDivElement>(null);
  const searchInputRef = React.useRef<HTMLInputElement>(null);

  React.useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      setTimeout(() => {
        searchInputRef.current?.focus();
      }, 50);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  const filteredOptions = options.filter(opt =>
    opt.label.toLowerCase().includes(search.toLowerCase()) ||
    (opt.subtext && opt.subtext.toLowerCase().includes(search.toLowerCase()))
  );

  const selectedOption = options.find(o => o.value === value);

  return (
    <div className={`relative ${className}`} ref={containerRef}>
      <button
        type="button"
        disabled={disabled}
        onClick={() => {
          if (!disabled) {
            setIsOpen(!isOpen);
            setSearch('');
          }
        }}
        className={`flex items-center gap-2 bg-white border rounded-lg shadow-2xs px-3 py-1.5 text-sm transition-all cursor-pointer ${
          disabled ? 'opacity-60 cursor-not-allowed bg-gray-50 border-gray-200' : 'hover:border-indigo-300 hover:bg-gray-50/50 border-gray-300'
        } ${isOpen ? 'ring-2 ring-indigo-500/20 border-indigo-500' : ''}`}
      >
        {Icon && <Icon size={14} className="text-gray-400 shrink-0" />}
        <span className="text-xs font-semibold text-gray-500 uppercase tracking-wide shrink-0">
          {label}:
        </span>
        <span
          className={`font-semibold truncate max-w-[130px] sm:max-w-[190px] text-left ${
            value ? 'text-indigo-700' : 'text-gray-700'
          }`}
          title={selectedOption ? selectedOption.label : allLabel}
        >
          {selectedOption ? selectedOption.label : allLabel}
        </span>
        {value ? (
          <span
            onClick={(e) => {
              e.stopPropagation();
              onChange('');
              setIsOpen(false);
            }}
            className="p-0.5 hover:bg-gray-200 rounded-full text-gray-400 hover:text-gray-600 ml-1 transition-colors cursor-pointer"
            title="Clear selection"
          >
            <X size={13} />
          </span>
        ) : (
          <ChevronDown size={14} className={`text-gray-400 transition-transform ${isOpen ? 'rotate-180 text-indigo-600' : ''}`} />
        )}
      </button>

      {isOpen && (
        <div className="absolute left-0 mt-1.5 w-72 bg-white rounded-xl shadow-xl border border-gray-200 py-2 z-50 animate-in fade-in zoom-in-95 duration-100">
          {/* Search Box */}
          <div className="px-2.5 pb-2 border-b border-gray-100">
            <div className="relative">
              <Search size={14} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-gray-400" />
              <input
                ref={searchInputRef}
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder={`Search ${label.toLowerCase()}...`}
                className="w-full pl-8 pr-7 py-1.5 text-xs bg-gray-50 border border-gray-200 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:bg-white transition-colors"
              />
              {search && (
                <button
                  type="button"
                  onClick={() => setSearch('')}
                  className="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600"
                >
                  <X size={12} />
                </button>
              )}
            </div>
          </div>

          {/* Options list */}
          <div className="max-h-56 overflow-y-auto py-1 text-xs">
            {(!search || allLabel.toLowerCase().includes(search.toLowerCase())) && (
              <button
                type="button"
                onClick={() => {
                  onChange('');
                  setIsOpen(false);
                }}
                className={`w-full text-left px-3 py-2 flex items-center justify-between hover:bg-indigo-50/70 transition-colors cursor-pointer ${
                  !value ? 'bg-indigo-50 text-indigo-700 font-semibold' : 'text-gray-700'
                }`}
              >
                <span>{allLabel}</span>
                {!value && <CheckCircle2 size={14} className="text-indigo-600 shrink-0" />}
              </button>
            )}

            {filteredOptions.map((opt) => {
              const isSelected = opt.value === value;
              return (
                <button
                  key={opt.value}
                  type="button"
                  onClick={() => {
                    onChange(opt.value);
                    setIsOpen(false);
                  }}
                  className={`w-full text-left px-3 py-2 flex items-center justify-between hover:bg-indigo-50/70 transition-colors cursor-pointer ${
                    isSelected ? 'bg-indigo-50 text-indigo-700 font-semibold' : 'text-gray-700'
                  }`}
                >
                  <div className="flex items-center gap-2 min-w-0 pr-2">
                    {opt.subtext && (
                      <span className="bg-gray-100 text-gray-600 font-mono text-[10px] px-1.5 py-0.5 rounded font-semibold uppercase shrink-0">
                        {opt.subtext}
                      </span>
                    )}
                    <span className="truncate">{opt.label}</span>
                  </div>
                  {isSelected && <CheckCircle2 size={14} className="text-indigo-600 shrink-0" />}
                </button>
              );
            })}

            {filteredOptions.length === 0 && (
              <div className="px-3 py-4 text-center text-gray-400 italic">
                No matching results
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
};

const SemesterHeaderFilter = ({
  sortField,
  sortDirection,
  onSort,
  selectedSemesters,
  onToggleSemester,
  onSelectAllSemesters,
  onClearSemesters,
}: {
  sortField: string;
  sortDirection: 'asc' | 'desc';
  onSort: () => void;
  selectedSemesters: string[];
  onToggleSemester: (sem: string) => void;
  onSelectAllSemesters: () => void;
  onClearSemesters: () => void;
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const dropdownRef = React.useRef<HTMLDivElement>(null);

  React.useEffect(() => {
    const handleClickOutside = (e: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(e.target as Node)) {
        setIsOpen(false);
      }
    };
    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  const hasFilter = selectedSemesters.length > 0;
  const isSorted = sortField === 'semester';

  return (
    <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider relative select-none">
      <div className="flex items-center gap-1.5" ref={dropdownRef}>
        {/* Sortable Header Label */}
        <div 
          onClick={onSort} 
          className="flex items-center gap-1 cursor-pointer hover:text-gray-900 transition-colors"
          title="Click to sort by semester"
        >
          <span>Semester</span>
          {isSorted ? (
            sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
          ) : (
            <ArrowUpDown size={12} className="text-gray-400" />
          )}
        </div>

        {/* Filter Trigger Button */}
        <div className="relative">
          <button
            type="button"
            onClick={(e) => {
              e.stopPropagation();
              setIsOpen(!isOpen);
            }}
            className={`p-1 rounded-md transition-all cursor-pointer flex items-center gap-0.5 ${
              hasFilter 
                ? 'bg-indigo-100 text-indigo-700 hover:bg-indigo-200' 
                : 'text-gray-400 hover:text-gray-600 hover:bg-gray-200/60'
            }`}
            title="Filter by Semester"
          >
            <Filter size={13} className={hasFilter ? 'fill-indigo-600 text-indigo-600' : ''} />
            {hasFilter && (
              <span className="text-[10px] font-bold leading-none px-1 py-0.2 bg-indigo-600 text-white rounded-full">
                {selectedSemesters.length}
              </span>
            )}
          </button>

          {/* Filter Popover Dropdown */}
          {isOpen && (
            <div 
              onClick={(e) => e.stopPropagation()}
              className="absolute left-0 mt-2 w-48 bg-white rounded-xl shadow-xl border border-gray-200 p-2.5 z-50 normal-case font-normal text-gray-700 animate-in fade-in zoom-in-95 duration-100"
            >
              <div className="flex items-center justify-between pb-2 mb-2 border-b border-gray-100">
                <span className="text-xs font-bold text-gray-800">Filter Semester</span>
                {hasFilter ? (
                  <button
                    type="button"
                    onClick={onClearSemesters}
                    className="text-[11px] text-indigo-600 hover:text-indigo-800 font-semibold cursor-pointer"
                  >
                    Clear
                  </button>
                ) : (
                  <button
                    type="button"
                    onClick={onSelectAllSemesters}
                    className="text-[11px] text-gray-500 hover:text-gray-700 cursor-pointer"
                  >
                    Select All
                  </button>
                )}
              </div>

              <div className="space-y-1">
                {[
                  { id: '1', label: 'Semester 1' },
                  { id: '2', label: 'Semester 2' },
                  { id: 'Summer', label: 'Summer' },
                ].map((sem) => {
                  const isChecked = selectedSemesters.includes(sem.id);
                  return (
                    <label
                      key={sem.id}
                      className={`flex items-center gap-2 px-2 py-1.5 rounded-lg text-xs cursor-pointer transition-colors ${
                        isChecked ? 'bg-indigo-50 text-indigo-900 font-medium' : 'hover:bg-gray-50 text-gray-700'
                      }`}
                    >
                      <input
                        type="checkbox"
                        checked={isChecked}
                        onChange={() => onToggleSemester(sem.id)}
                        className="h-3.5 w-3.5 text-indigo-600 rounded border-gray-300 focus:ring-indigo-500 cursor-pointer"
                      />
                      <span>{sem.label}</span>
                    </label>
                  );
                })}
              </div>

              <div className="pt-2 mt-2 border-t border-gray-100 flex justify-end">
                <button
                  type="button"
                  onClick={() => setIsOpen(false)}
                  className="px-2.5 py-1 bg-indigo-600 text-white rounded-md text-[11px] font-semibold hover:bg-indigo-700 transition-colors cursor-pointer"
                >
                  Done
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </th>
  );
};

// --- PAGES ---

const LoginPage = ({ onLogin, onSimulateError }: { onLogin: () => void; onSimulateError: () => void }) => (
  <div className="min-h-screen bg-gray-50 flex flex-col justify-center py-12 sm:px-6 lg:px-8 relative">
    
    {/* Theme Selector */}
    <div className="absolute top-4 right-4 sm:top-6 sm:right-6 flex gap-3">
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
  const [editingName, setEditingName] = useState('');
  const [editingEmail, setEditingEmail] = useState('');
  const [isNewUser, setIsNewUser] = useState(false);

  const handleEmailChange = (email: string) => {
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

  const openUserModal = (user: any = null) => {
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
      setEditingDepartment(departments[0]?.name || '');
      setEditingCurriculum('');
      setIsNewUser(true);
    }
  };

  const handleSaveUser = () => {
    if (isNewUser) {
      const newUser: User = {
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
        name: editingName,
        email: editingEmail,
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
          className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-sm cursor-pointer"
        >
          <Plus size={16} className="mr-2" /> Add New User
        </button>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Email</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Affiliation</th>
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
                    type="button"
                    onClick={() => openUserModal(user)}
                    className="inline-flex items-center justify-center p-1.5 text-indigo-700 bg-indigo-50 hover:bg-indigo-100 border border-indigo-200/70 rounded-md transition-colors cursor-pointer"
                    title="Edit User"
                  >
                    <Edit2 size={13} />
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
              <button 
                type="button"
                onClick={handleDeleteUser} 
                className="flex items-center px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 rounded-lg mr-auto transition-colors cursor-pointer"
              >
                <Trash2 size={15} className="mr-1.5" /> Delete User
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
                    ? "Name is fetched from backend/Google after entering email." : ""}
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
              </div>
            </div>
            <div>
                {!isNewUser && (
                  <p className="text-xs text-gray-400 mt-1">Both Name and Email are automatically synced with the Google account.</p>
                )}
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

  const handleDeleteDept = () => {
    if (!selectedDept) return;
    const dept = selectedDept;
    const hasCurriculums = curriculums.some(c => c.departmentId === dept.id);
    const hasUsers = users.some(u => u.department === dept.name);

    if (hasCurriculums || hasUsers) {
      alert(`Cannot delete department "${dept.name}". It is currently assigned to ${hasCurriculums ? 'one or more curriculums' : ''} ${hasCurriculums && hasUsers ? 'and' : ''} ${hasUsers ? 'one or more users' : ''}.`);
      return;
    }

    if (confirm(`Are you sure you want to delete the department "${dept.name}"?`)) {
      setDepartments(departments.filter(d => d.id !== dept.id));
      setSelectedDept(null);
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Department Management</h1>
        <button 
          onClick={() => openDeptModal()} 
          className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-sm cursor-pointer"
        >
          <Plus size={16} className="mr-2" /> Add New Department
        </button>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Department Code</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Department Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Curriculums</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
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
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button 
                      type="button"
                      onClick={() => openDeptModal(dept)}
                      className="inline-flex items-center justify-center p-1.5 text-indigo-700 bg-indigo-50 hover:bg-indigo-100 border border-indigo-200/70 rounded-md transition-colors cursor-pointer"
                      title="Edit Department"
                    >
                      <Edit2 size={13} />
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
            {!isNewDept && (
              <button 
                type="button"
                onClick={handleDeleteDept} 
                className="flex items-center px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 rounded-lg mr-auto transition-colors cursor-pointer"
              >
                <Trash2 size={15} className="mr-1.5" /> Delete Department
              </button>
            )}
            <button 
              type="button"
              onClick={() => setSelectedDept(null)} 
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button 
              type="button"
              onClick={handleSaveDept} 
              className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors cursor-pointer"
            >
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

  const handleDeleteCurr = () => {
    if (!selectedCurr) return;
    const curr = selectedCurr;
    const hasUsers = users.some(u => u.curriculum === curr.name);

    if (hasUsers) {
      alert(`Cannot delete curriculum "${curr.name}". It is currently assigned to one or more users.`);
      return;
    }

    if (confirm(`Are you sure you want to delete the curriculum "${curr.name}"?`)) {
      setCurriculums(curriculums.filter(c => c.id !== curr.id));
      setSelectedCurr(null);
    }
  };

  return (
    <div className="p-6">
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Curriculum Management</h1>
        <button 
          onClick={() => openCurrModal()} 
          className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-sm cursor-pointer"
          disabled={departments.length === 0}
          title={departments.length === 0 ? "Create a department first" : ""}
        >
          <Plus size={16} className="mr-2" /> Add New Curriculum
        </button>
      </div>

      <div className="bg-white rounded-xl shadow-sm border border-gray-200 overflow-hidden">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Curriculum Code</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Curriculum Name</th>
              <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Department</th>
              <th className="px-6 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">Action</th>
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
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <button 
                      type="button"
                      onClick={() => openCurrModal(curr)}
                      className="inline-flex items-center justify-center p-1.5 text-indigo-700 bg-indigo-50 hover:bg-indigo-100 border border-indigo-200/70 rounded-md transition-colors cursor-pointer"
                      title="Edit Curriculum"
                    >
                      <Edit2 size={13} />
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
            {!isNewCurr && (
              <button 
                type="button"
                onClick={handleDeleteCurr} 
                className="flex items-center px-4 py-2 text-sm font-medium text-red-600 hover:bg-red-50 rounded-lg mr-auto transition-colors cursor-pointer"
              >
                <Trash2 size={15} className="mr-1.5" /> Delete Curriculum
              </button>
            )}
            <button 
              type="button"
              onClick={() => setSelectedCurr(null)} 
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button 
              type="button"
              onClick={handleSaveCurr} 
              className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors cursor-pointer"
            >
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

const CourseManagement = ({
  courses,
  setCourses,
  departments,
  curriculums,
  files,
  setFiles,
  users = MOCK_USERS
}: {
  courses: any[];
  setCourses: React.Dispatch<React.SetStateAction<any[]>>;
  departments: Department[];
  curriculums: Curriculum[];
  files: any[];
  setFiles: React.Dispatch<React.SetStateAction<any[]>>;
  users?: User[];
}) => {
  const [isAddModalOpen, setIsAddModalOpen] = useState(false);
  const [editingCourse, setEditingCourse] = useState<any>(null);
  
  // Year selector - single choice, defaults to latest year (no 'all' choice)
  const [selectedYear, setSelectedYear] = useState('2024');
  
  // Top Dropdown Filters: Department & Curriculum
  const [selectedDepartment, setSelectedDepartment] = useState<string>('');
  const [selectedCurriculum, setSelectedCurriculum] = useState<string>('');

  // Column Filter: Semester (empty = all)
  const [selectedSemesters, setSelectedSemesters] = useState<string[]>([]);

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
  const [editingYear, setEditingYear] = useState('2024');
  const [editingSemester, setEditingSemester] = useState('1');
  const [errorMsg, setErrorMsg] = useState('');

  // Selection states
  const [selectedCourseIds, setSelectedCourseIds] = useState<string[]>([]);
  const [deleteConfirmOpen, setDeleteConfirmOpen] = useState(false);
  const [courseToDelete, setCourseToDelete] = useState<any>(null);

  // --- Course User Access State ---
  const [courseAssignments, setCourseAssignments] = useState<Record<string, CourseUserAccess[]>>(INITIAL_COURSE_ASSIGNMENTS);
  // Default all course access sub-tables to collapsed
  const [expandedCourseIds, setExpandedCourseIds] = useState<string[]>([]);

  // Edit Access Modal State
  const [editingAccessData, setEditingAccessData] = useState<{ courseId: string; courseName: string; department?: string; curriculum?: string; year?: string; userAccess: CourseUserAccess } | null>(null);
  const [editingAccessLevel, setEditingAccessLevel] = useState<'Edit and Download' | 'Download'>('Edit and Download');
  const [editingIsDefault, setEditingIsDefault] = useState(false);

  // Assign User to Course Modal State
  const [assigningCourse, setAssigningCourse] = useState<any | null>(null);
  const [assignUserId, setAssignUserId] = useState<string>('');
  const [assignAccessLevel, setAssignAccessLevel] = useState<'Edit and Download' | 'Download'>('Edit and Download');
  const [assignIsDefault, setAssignIsDefault] = useState(false);

  // Toast message
  const [toastMsg, setToastMsg] = useState<{ text: string; type: 'success' | 'error' | 'info' } | null>(null);

  const showToastMsg = (text: string, type: 'success' | 'error' | 'info' = 'success') => {
    setToastMsg({ text, type });
    setTimeout(() => setToastMsg(null), 3500);
  };

  const toggleCourseExpand = (courseId: string) => {
    if (expandedCourseIds.includes(courseId)) {
      setExpandedCourseIds(expandedCourseIds.filter(id => id !== courseId));
    } else {
      setExpandedCourseIds([...expandedCourseIds, courseId]);
    }
  };

  const toggleSemester = (sem: string) => {
    if (selectedSemesters.includes(sem)) {
      setSelectedSemesters(selectedSemesters.filter(s => s !== sem));
    } else {
      setSelectedSemesters([...selectedSemesters, sem]);
    }
  };

  const handleSelectAllSemesters = () => {
    setSelectedSemesters(['1', '2', 'Summer']);
  };

  const handleClearSemesters = () => {
    setSelectedSemesters([]);
  };

  const handleDepartmentChange = (deptName: string) => {
    setSelectedDepartment(deptName);
    if (deptName && selectedCurriculum) {
      const parentDept = departments.find(d => d.name === deptName);
      const curr = curriculums.find(c => c.name === selectedCurriculum);
      if (curr && parentDept && curr.departmentId !== parentDept.id) {
        setSelectedCurriculum('');
      }
    }
  };

  const clearAllFilters = () => {
    setSelectedDepartment('');
    setSelectedCurriculum('');
    setSelectedSemesters([]);
    setSearchQuery('');
  };

  const hasActiveFilters = Boolean(
    selectedDepartment || selectedCurriculum || selectedSemesters.length > 0 || searchQuery
  );

  // Filtered & Sorted courses
  const filteredCourses = courses.filter(course => {
    const matchesYear = (course.year || '2024') === selectedYear;
    const matchesSemester = selectedSemesters.length === 0 || selectedSemesters.includes(course.semester || '1');
    const matchesDept = !selectedDepartment || course.department === selectedDepartment;
    const matchesCurriculum = !selectedCurriculum || course.curriculum === selectedCurriculum;

    const query = searchQuery.toLowerCase();
    const matchesSearch = (
      course.id.toLowerCase().includes(query) ||
      course.name.toLowerCase().includes(query) ||
      (course.department || '').toLowerCase().includes(query) ||
      (course.curriculum || '').toLowerCase().includes(query)
    );

    return matchesYear && matchesSemester && matchesDept && matchesCurriculum && matchesSearch;
  });

  const sortedCourses = [...filteredCourses].sort((a, b) => {
    let aVal: any = '';
    let bVal: any = '';

    if (sortField === 'id') {
      aVal = a.id;
      bVal = b.id;
    } else if (sortField === 'name') {
      aVal = a.name;
      bVal = b.name;
    } else if (sortField === 'semester') {
      aVal = a.semester || '1';
      bVal = b.semester || '1';
    } else if (sortField === 'department') {
      aVal = a.department || '';
      bVal = b.department || '';
    } else if (sortField === 'curriculum') {
      aVal = a.curriculum || '';
      bVal = b.curriculum || '';
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
    setErrorMsg('');
    setEditingCourse(course);
    setEditingId(course.id);
    setEditingNameInput(course.name);
    setEditingDepartment(course.department || '');
    setEditingCurriculum(course.curriculum || '');
    setEditingYear(course.year || selectedYear);
    setEditingSemester(course.semester || '1');
    setActiveCreationMethod('single');
    setIsAddModalOpen(true);
  };

  const handleOpenAdd = () => {
    setErrorMsg('');
    setEditingCourse(null);
    setEditingId('');
    setEditingNameInput('');
    const activeDept = selectedDepartment || (departments[0]?.name || '');
    setEditingDepartment(activeDept);
    const availableCurrs = curriculums.filter(c => {
      const dept = departments.find(d => d.name === activeDept);
      return !dept || c.departmentId === dept.id;
    });
    setEditingCurriculum(selectedCurriculum || (availableCurrs[0]?.name || ''));
    setEditingYear(selectedYear);
    setEditingSemester(selectedSemesters.length === 1 ? selectedSemesters[0] : '1');
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
        setIsAddModalOpen(false);
      } else {
        setErrorMsg('Please upload a valid spreadsheet file first.');
      }
    } else {
      if (!editingId.trim() || !editingNameInput.trim()) {
        setErrorMsg('Course ID and Course Name are required.');
        return;
      }

      if (editingCourse) {
        setCourses(courses.map(c => c.id === editingCourse.id ? { 
          id: editingId.trim().toUpperCase(), 
          name: editingNameInput.trim(),
          department: editingDepartment,
          curriculum: editingCurriculum,
          year: editingYear,
          semester: editingSemester,
        } : c));
        if (editingId.trim().toUpperCase() !== editingCourse.id) {
          setFiles(files.map(f => f.courseId === editingCourse.id ? { ...f, courseId: editingId.trim().toUpperCase() } : f));
          if (courseAssignments[editingCourse.id]) {
            const currentAssignments = courseAssignments[editingCourse.id];
            const updatedAssignments = { ...courseAssignments };
            delete updatedAssignments[editingCourse.id];
            updatedAssignments[editingId.trim().toUpperCase()] = currentAssignments;
            setCourseAssignments(updatedAssignments);
          }
        }
      } else {
        const newCourseId = editingId.trim().toUpperCase();
        setCourses([...courses, { 
          id: newCourseId, 
          name: editingNameInput.trim(),
          department: editingDepartment,
          curriculum: editingCurriculum,
          year: selectedYear || editingYear,
          semester: editingSemester,
        }]);
        // Automatically assign creator/instructor with default access
        setCourseAssignments({
          ...courseAssignments,
          [newCourseId]: [
            {
              id: 'a_' + Date.now(),
              userId: 1,
              userName: 'Alice Smith',
              userEmail: 'alice.admin@company.com',
              userRole: 'super_admin',
              userDept: editingDepartment,
              access: 'Edit and Download',
              isDefault: true,
              defaultReason: 'Creator / Department Admin'
            }
          ]
        });
      }
      setIsAddModalOpen(false);
      setEditingCourse(null);
    }
  };

  const handleToggleSelect = (courseId: string) => {
    if (selectedCourseIds.includes(courseId)) {
      setSelectedCourseIds(selectedCourseIds.filter(id => id !== courseId));
    } else {
      setSelectedCourseIds([...selectedCourseIds, courseId]);
    }
  };

  const handleSelectAllToggle = () => {
    if (selectedCourseIds.length === sortedCourses.length && sortedCourses.length > 0) {
      setSelectedCourseIds([]);
    } else {
      setSelectedCourseIds(sortedCourses.map(c => c.id));
    }
  };

  const handleExecuteDelete = () => {
    if (courseToDelete) {
      setCourses(courses.filter(c => c.id !== courseToDelete.id));
      setFiles(files.filter(f => f.courseId !== courseToDelete.id));
      setSelectedCourseIds(selectedCourseIds.filter(id => id !== courseToDelete.id));
      if (courseAssignments[courseToDelete.id]) {
        const updatedAssignments = { ...courseAssignments };
        delete updatedAssignments[courseToDelete.id];
        setCourseAssignments(updatedAssignments);
      }
      showToastMsg(`Deleted course ${courseToDelete.id} - ${courseToDelete.name}.`, 'info');
      setCourseToDelete(null);
    } else if (selectedCourseIds.length > 0) {
      setCourses(courses.filter(c => !selectedCourseIds.includes(c.id)));
      setFiles(files.filter(f => !selectedCourseIds.includes(f.courseId)));
      showToastMsg(`Deleted ${selectedCourseIds.length} course(s).`, 'info');
      setSelectedCourseIds([]);
    }
    setDeleteConfirmOpen(false);
  };

  // --- User Access Handlers ---
  const handleOpenEditAccess = (course: any, userAccess: CourseUserAccess) => {
    setEditingAccessData({
      courseId: course.id,
      courseName: course.name,
      department: course.department,
      curriculum: course.curriculum,
      year: course.year,
      userAccess
    });
    setEditingAccessLevel(userAccess.access);
    setEditingIsDefault(!!userAccess.isDefault);
  };

  const handleSaveEditAccess = () => {
    if (!editingAccessData) return;
    const { courseId, userAccess } = editingAccessData;
    const currentList = courseAssignments[courseId] || [];
    const updatedList = currentList.map(a => a.id === userAccess.id ? {
      ...a,
      access: editingAccessLevel,
      isDefault: editingIsDefault
    } : a);
    setCourseAssignments({ ...courseAssignments, [courseId]: updatedList });
    setEditingAccessData(null);
    showToastMsg(`Updated access level for ${userAccess.userName} to "${editingAccessLevel}".`);
  };

  const handleDeleteAccessFromModal = () => {
    if (!editingAccessData) return;
    const { courseId, userAccess } = editingAccessData;
    const currentList = courseAssignments[courseId] || [];
    const updatedList = currentList.filter(a => a.id !== userAccess.id);
    setCourseAssignments({ ...courseAssignments, [courseId]: updatedList });
    setEditingAccessData(null);
    showToastMsg(`Removed ${userAccess.userName}'s access from ${courseId}.`, 'info');
  };

  const handleOpenAssignUser = (course: any) => {
    setAssigningCourse(course);
    const existingUserIds = (courseAssignments[course.id] || []).map(a => a.userId);
    const assignableUsers = (users || MOCK_USERS).filter(u => u.role !== 'admin' && u.role !== 'super_admin');
    const availableUser = assignableUsers.find(u => !existingUserIds.includes(u.id));
    setAssignUserId(availableUser ? String(availableUser.id) : assignableUsers[0]?.id ? String(assignableUsers[0].id) : '');
    setAssignAccessLevel('Edit and Download');
    setAssignIsDefault(false);
  };

  const handleSaveAssignUser = () => {
    if (!assigningCourse || !assignUserId) return;
    const userObj = (users || MOCK_USERS).find(u => String(u.id) === String(assignUserId));
    if (!userObj) return;

    const newAccess: CourseUserAccess = {
      id: 'a_' + Date.now(),
      userId: userObj.id,
      userName: userObj.name,
      userEmail: userObj.email,
      userRole: userObj.role,
      userDept: userObj.department,
      access: assignAccessLevel,
      isDefault: assignIsDefault,
      defaultReason: assignIsDefault ? 'Manual Default Assignment' : undefined
    };

    const currentList = courseAssignments[assigningCourse.id] || [];
    setCourseAssignments({
      ...courseAssignments,
      [assigningCourse.id]: [...currentList, newAccess]
    });

    if (!expandedCourseIds.includes(assigningCourse.id)) {
      setExpandedCourseIds([...expandedCourseIds, assigningCourse.id]);
    }

    setAssigningCourse(null);
    showToastMsg(`Assigned ${userObj.name} to ${assigningCourse.id} (${assignAccessLevel}).`);
  };

  // Helper to identify courses with generated reports before deletion
  const coursesWithGeneratedFiles = (courseToDelete ? [courseToDelete] : courses.filter(c => selectedCourseIds.includes(c.id)))
    .filter(course => files.some(f => f.courseId === course.id && f.year === selectedYear && f.status === 'generated'));

  return (
    <div className="p-6 space-y-5 w-full">
      {/* Toast Notification */}
      {toastMsg && (
        <div className="fixed bottom-6 right-6 z-50 animate-in fade-in slide-in-from-bottom-5 duration-300">
          <div className={`flex items-center gap-3 px-4 py-3 rounded-xl shadow-lg border text-sm font-medium ${
            toastMsg.type === 'error'
              ? 'bg-red-50 text-red-800 border-red-200'
              : toastMsg.type === 'info'
              ? 'bg-blue-50 text-blue-800 border-blue-200'
              : 'bg-emerald-50 text-emerald-800 border-emerald-200'
          }`}>
            <CheckCircle2 size={18} className={toastMsg.type === 'error' ? 'text-red-600' : toastMsg.type === 'info' ? 'text-blue-600' : 'text-emerald-600'} />
            <span>{toastMsg.text}</span>
          </div>
        </div>
      )}

      {/* Top Header: Title + Actions on Row 1, Filters on Row 2 under Title */}
      <div className="space-y-4">
        {/* Row 1: Title + Action Buttons */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <h1 className="text-2xl font-bold text-gray-900">Curriculum Management</h1>
          
          {/* Actions */}
          <div className="flex items-center gap-3 shrink-0">
            {selectedCourseIds.length > 0 && (
              <button 
                onClick={() => { setCourseToDelete(null); setDeleteConfirmOpen(true); }} 
                className="flex items-center px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 font-medium text-sm transition-colors shadow-2xs whitespace-nowrap cursor-pointer"
              >
                <Trash2 size={16} className="mr-2" /> Delete Selected ({selectedCourseIds.length})
              </button>
            )}
            
            <button onClick={handleOpenAdd} className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-2xs whitespace-nowrap cursor-pointer">
              <Plus size={16} className="mr-2" /> Add New Course
            </button>
          </div>
        </div>

        {/* Row 2: Filters under page title */}
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Department Searchable Dropdown */}
          <SearchableSelect
            label="Department"
            value={selectedDepartment}
            onChange={handleDepartmentChange}
            options={departments.map(d => ({ value: d.name, label: d.name, subtext: d.code }))}
            allLabel="All Departments"
            icon={Building2}
          />

          {/* Curriculum Searchable Dropdown */}
          <SearchableSelect
            label="Curriculum"
            value={selectedCurriculum}
            onChange={setSelectedCurriculum}
            options={curriculums
              .filter(c => {
                if (!selectedDepartment) return true;
                const parentDept = departments.find(d => d.name === selectedDepartment);
                return parentDept && c.departmentId === parentDept.id;
              })
              .map(c => ({ value: c.name, label: c.name, subtext: c.code }))
            }
            allLabel="All Curriculums"
            icon={GraduationCap}
          />

          {/* Year Dropdown */}
          <div className="flex items-center bg-white border border-gray-300 rounded-lg shadow-2xs px-3 py-1.5">
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

      {/* Search Bar + Active Filters Status */}
      <div className="bg-white p-3 rounded-xl border border-gray-200 shadow-2xs space-y-2">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div className="relative flex-1">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <Search className="h-4 w-4 text-gray-400" />
            </div>
            <input
              type="text"
              placeholder="Search course ID, name, department, curriculum..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="block w-full pl-9 pr-3 py-1.5 border border-gray-200 rounded-lg text-sm bg-gray-50 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:bg-white transition-colors"
            />
          </div>
          <div className="text-xs text-gray-500 px-2 whitespace-nowrap">
            Showing <span className="font-semibold text-gray-900">{sortedCourses.length}</span> course{sortedCourses.length === 1 ? '' : 's'} in <span className="font-semibold text-indigo-600">{selectedYear}</span>
          </div>
        </div>

        {/* Active Filters Bar */}
        {hasActiveFilters && (
          <div className="flex flex-wrap items-center gap-2 pt-1 border-t border-gray-100 text-xs">
            <span className="text-gray-500 font-medium">Active filters:</span>
            {selectedDepartment && (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 px-2 py-0.5 rounded-full font-medium">
                Dept: {selectedDepartment}
                <button onClick={() => setSelectedDepartment('')} className="hover:text-indigo-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            {selectedCurriculum && (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 px-2 py-0.5 rounded-full font-medium">
                Curriculum: {selectedCurriculum}
                <button onClick={() => setSelectedCurriculum('')} className="hover:text-indigo-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            {selectedSemesters.length > 0 && (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 px-2 py-0.5 rounded-full font-medium">
                Semesters: {selectedSemesters.map(s => s === 'Summer' ? 'Summer' : `Sem ${s}`).join(', ')}
                <button onClick={() => setSelectedSemesters([])} className="hover:text-indigo-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            {searchQuery && (
              <span className="inline-flex items-center gap-1 bg-gray-100 text-gray-700 border border-gray-200 px-2 py-0.5 rounded-full font-medium">
                Search: "{searchQuery}"
                <button onClick={() => setSearchQuery('')} className="hover:text-gray-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            <button
              onClick={clearAllFilters}
              className="text-indigo-600 hover:text-indigo-800 font-semibold underline ml-1 cursor-pointer flex items-center gap-1"
            >
              <RotateCcw size={11} /> Clear all
            </button>
          </div>
        )}
      </div>

      {/* Course Table (Full Width) */}
      <div className="bg-white rounded-xl shadow-2xs border border-gray-200 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th className="w-9 px-3 py-3 text-center" title="Expand user assignments">
                  <span className="sr-only">Expand</span>
                </th>
                <th className="w-10 px-3 py-3 text-left">
                  <input
                    type="checkbox"
                    checked={selectedCourseIds.length === sortedCourses.length && sortedCourses.length > 0}
                    onChange={handleSelectAllToggle}
                    className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                  />
                </th>
                <th 
                  onClick={() => handleSort('id')}
                  className="px-5 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-1">
                    Course ID
                    {sortField === 'id' ? (
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
                    ) : (
                      <ArrowUpDown size={12} className="text-gray-400" />
                    )}
                  </div>
                </th>
                <th 
                  onClick={() => handleSort('name')}
                  className="px-5 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-1">
                    Course Name
                    {sortField === 'name' ? (
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
                    ) : (
                      <ArrowUpDown size={12} className="text-gray-400" />
                    )}
                  </div>
                </th>
                <SemesterHeaderFilter
                  sortField={sortField}
                  sortDirection={sortDirection}
                  onSort={() => handleSort('semester')}
                  selectedSemesters={selectedSemesters}
                  onToggleSemester={toggleSemester}
                  onSelectAllSemesters={handleSelectAllSemesters}
                  onClearSemesters={handleClearSemesters}
                />
                <th className="px-5 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                  Actions
                </th>
              </tr>
            </thead>
            <tbody className="bg-white divide-y divide-gray-200">
              {sortedCourses.map(course => {
                const isSelected = selectedCourseIds.includes(course.id);
                const isExpanded = expandedCourseIds.includes(course.id);
                const courseUsers = courseAssignments[course.id] || [];
                const visibleCourseUsers = courseUsers.filter(u => u.userRole !== 'admin' && u.userRole !== 'super_admin');
                const instructorCount = visibleCourseUsers.filter(u => u.userRole === 'instructor').length;
                const staffCount = visibleCourseUsers.filter(u => u.userRole === 'staff').length;
                const hasNoAssignedUsers = visibleCourseUsers.length === 0;

                return (
                  <React.Fragment key={course.id}>
                    <tr 
                      onClick={() => toggleCourseExpand(course.id)}
                      className={`transition-colors cursor-pointer ${
                        hasNoAssignedUsers 
                          ? (isSelected 
                              ? 'bg-amber-100/70 hover:bg-amber-100/90' 
                              : isExpanded 
                              ? 'bg-amber-50/90 hover:bg-amber-100/80' 
                              : 'bg-amber-50/50 hover:bg-amber-100/60')
                          : (isSelected 
                              ? 'bg-indigo-50/40 hover:bg-indigo-50/60' 
                              : isExpanded 
                              ? 'bg-slate-50/60' 
                              : 'hover:bg-gray-50/80')
                      }`}
                    >
                      {/* Expand Button Column */}
                      <td className="pl-3.5 pr-1 py-4 whitespace-nowrap text-center" onClick={(e) => e.stopPropagation()}>
                        <button
                          type="button"
                          onClick={() => toggleCourseExpand(course.id)}
                          className="p-1 rounded-md text-gray-400 hover:text-indigo-600 hover:bg-indigo-50 transition-colors cursor-pointer"
                          title={isExpanded ? "Collapse assigned users" : "Expand assigned users"}
                        >
                          <ChevronRight 
                            size={16} 
                            className={`transition-transform duration-200 ${isExpanded ? 'rotate-90 text-indigo-600' : 'text-gray-400'}`} 
                          />
                        </button>
                      </td>

                      {/* Checkbox Column */}
                      <td className="px-3 py-4 whitespace-nowrap" onClick={(e) => e.stopPropagation()}>
                        <input 
                          type="checkbox" 
                          checked={isSelected} 
                          onChange={() => handleToggleSelect(course.id)}
                          className="h-4 w-4 text-indigo-600 border-gray-300 rounded focus:ring-indigo-500 cursor-pointer"
                        />
                      </td>

                      {/* Course ID */}
                      <td className="px-5 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                        <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">
                          {course.id}
                        </span>
                      </td>

                      {/* Course Name + Members Badge */}
                      <td className="px-5 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                        <div className="flex items-center gap-2">
                          <span>{course.name}</span>

                          {/* Members Count Badge (Clickable to toggle expand) */}
                          <button
                            type="button"
                            onClick={(e) => {
                              e.stopPropagation();
                              toggleCourseExpand(course.id);
                            }}
                            className={`inline-flex items-center gap-1 text-[11px] font-medium px-2 py-0.5 rounded-full transition-colors cursor-pointer ${
                              hasNoAssignedUsers
                                ? 'bg-amber-100 text-amber-800 border border-amber-300/80 hover:bg-amber-200/80'
                                : isExpanded 
                                ? 'bg-indigo-100 text-indigo-800' 
                                : 'bg-gray-100 hover:bg-indigo-50 text-gray-600 hover:text-indigo-700'
                            }`}
                            title={hasNoAssignedUsers ? "Warning: 0 users assigned. Click to assign." : "Click to view assigned users with access"}
                          >
                            {hasNoAssignedUsers ? (
                              <>
                                <AlertTriangle size={11} className="text-amber-700 shrink-0" />
                                <span>0 assigned</span>
                              </>
                            ) : (
                              <>
                                <Users size={11} className={isExpanded ? "text-indigo-700" : "text-gray-500"} />
                                <span>{visibleCourseUsers.length} user{visibleCourseUsers.length === 1 ? '' : 's'}</span>
                              </>
                            )}
                          </button>
                        </div>
                      </td>

                      {/* Semester */}
                      <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                        <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
                          {course.semester === 'Summer' ? 'Summer' : `Sem ${course.semester || '1'}`}
                        </span>
                      </td>

                      {/* Course Actions */}
                      <td className="px-5 py-4 whitespace-nowrap text-right" onClick={(e) => e.stopPropagation()}>
                        <button 
                          type="button"
                          onClick={() => handleEdit(course)}
                          className="inline-flex items-center justify-center p-1.5 text-indigo-700 bg-indigo-50 hover:bg-indigo-100 border border-indigo-200/70 rounded-md transition-colors cursor-pointer"
                          title="Edit Course"
                        >
                          <Edit2 size={13} />
                        </button>
                      </td>
                    </tr>

                    {/* --- Expanded Sub-Table: Assigned Users with Course Access --- */}
                    {isExpanded && (
                      <tr className="bg-slate-50/70 border-b border-gray-200">
                        <td colSpan={6} className="p-0">
                          <div className="py-3.5 px-6 sm:px-8 bg-gradient-to-b from-slate-50 via-slate-100/50 to-slate-50 border-y border-slate-200/80 shadow-inner">
                            {/* Sub-table Header Bar */}
                            <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-3 mb-3">
                              <div className="flex items-center gap-2.5">
                                <div className="p-1.5 bg-indigo-100 text-indigo-700 rounded-lg">
                                  <Users size={16} />
                                </div>
                                <div>
                                  <h4 className="text-xs font-bold text-gray-900 uppercase tracking-wider flex flex-wrap items-center gap-2">
                                    <span>Assigned users with course portfolio access permissions</span>
                                    <span className="text-[11px] font-medium bg-emerald-50 text-emerald-800 border border-emerald-200/80 px-2 py-0.5 rounded-full font-sans">
                                      {instructorCount} Instructor{instructorCount === 1 ? '' : 's'}
                                    </span>
                                    <span className="text-[11px] font-medium bg-blue-50 text-blue-800 border border-blue-200/80 px-2 py-0.5 rounded-full font-sans">
                                      {staffCount} Staff
                                    </span>
                                  </h4>
                                  <p className="text-[11px] text-gray-500">
                                    For course <strong className="text-gray-700">{course.id}</strong> ({course.name})
                                  </p>
                                </div>
                              </div>

                              <button
                                type="button"
                                onClick={() => handleOpenAssignUser(course)}
                                className="inline-flex items-center gap-1.5 px-3 py-1.5 bg-white border border-indigo-200 hover:border-indigo-300 text-indigo-700 hover:bg-indigo-50 rounded-lg text-xs font-semibold shadow-2xs transition-colors cursor-pointer self-start sm:self-auto"
                              >
                                <UserPlus size={13} />
                                <span>Assign New User</span>
                              </button>
                            </div>

                            {/* Sub-Table Content */}
                            <div className="bg-white rounded-lg border border-slate-200 shadow-2xs overflow-hidden">
                              <table className="min-w-full divide-y divide-gray-200 text-xs">
                                <thead className="bg-slate-50/90 text-gray-600 uppercase font-semibold text-[10px] tracking-wider">
                                  <tr>
                                    <th className="px-4 py-2.5 text-left">Name</th>
                                    <th className="px-4 py-2.5 text-left">Email</th>
                                    <th className="px-4 py-2.5 text-center">Upload</th>
                                    <th className="px-4 py-2.5 text-center">Download</th>
                                    <th className="px-4 py-2.5 text-right">Action</th>
                                  </tr>
                                </thead>
                                <tbody className="divide-y divide-gray-100 bg-white">
                                  {visibleCourseUsers.map((userAccess) => (
                                    <tr key={userAccess.id} className="hover:bg-slate-50/70 transition-colors">
                                      {/* Name Column */}
                                      <td className="px-4 py-3 whitespace-nowrap">
                                        <div className="flex items-center gap-2.5">
                                          <div className="h-7 w-7 rounded-full bg-indigo-50 text-indigo-700 border border-indigo-200 flex items-center justify-center font-bold text-xs shrink-0">
                                            {userAccess.userName.charAt(0)}
                                          </div>
                                          <div>
                                            <div className="font-semibold text-gray-900">{userAccess.userName}</div>
                                            <div className="text-[10px] text-gray-400 capitalize">{userAccess.userRole.replace('_', ' ')}</div>
                                          </div>
                                        </div>
                                      </td>

                                      {/* Email Column */}
                                      <td className="px-4 py-3 whitespace-nowrap text-gray-600 font-mono text-[11px]">
                                        {userAccess.userEmail}
                                      </td>

                                      {/* Upload Column */}
                                      <td className="px-4 py-3 whitespace-nowrap text-center">
                                        {userAccess.access === 'Edit and Download' ? (
                                          <div className="inline-flex items-center gap-1.5 text-emerald-600">
                                            <span className="h-5 w-5 rounded-full bg-emerald-100 flex items-center justify-center">
                                              <Check size={13} className="text-emerald-700 stroke-[3]" />
                                            </span>
                                            {userAccess.isDefault && (
                                              <span className="text-[10px] font-medium text-emerald-700 bg-emerald-100/90 px-1.5 py-0.2 rounded" title={userAccess.defaultReason || 'Default role permission'}>
                                                (default)
                                              </span>
                                            )}
                                          </div>
                                        ) : (
                                          <span className="text-gray-300 font-semibold">—</span>
                                        )}
                                      </td>

                                      {/* Download Column */}
                                      <td className="px-4 py-3 whitespace-nowrap text-center">
                                        <div className="inline-flex items-center gap-1.5 text-emerald-600">
                                          <span className="h-5 w-5 rounded-full bg-emerald-100 flex items-center justify-center">
                                            <Check size={13} className="text-emerald-700 stroke-[3]" />
                                          </span>
                                          {userAccess.isDefault && (
                                            <span className="text-[10px] font-medium text-sky-700 bg-sky-100/90 px-1.5 py-0.2 rounded" title={userAccess.defaultReason || 'Default role permission'}>
                                              (default)
                                            </span>
                                          )}
                                        </div>
                                      </td>

                                      {/* Action Column */}
                                      <td className="px-4 py-3 whitespace-nowrap text-right">
                                        <button
                                          type="button"
                                          onClick={() => handleOpenEditAccess(course, userAccess)}
                                          className="inline-flex items-center justify-center p-1.5 text-indigo-700 bg-indigo-50 hover:bg-indigo-100 border border-indigo-200/70 rounded-md transition-colors cursor-pointer"
                                          title="Edit Access"
                                        >
                                          <Edit2 size={13} />
                                        </button>
                                      </td>
                                    </tr>
                                  ))}

                                  {visibleCourseUsers.length === 0 && (
                                    <tr>
                                      <td colSpan={5} className="text-center py-6 text-gray-400 italic">
                                        No instructors or staff currently assigned to this course. Click "Assign New User" to add access.
                                      </td>
                                    </tr>
                                  )}
                                </tbody>
                              </table>
                            </div>
                          </div>
                        </td>
                      </tr>
                    )}
                  </React.Fragment>
                );
              })}
              {sortedCourses.length === 0 && (
                <tr>
                  <td colSpan={6} className="text-center py-8 text-gray-500 italic">
                    No courses found matching the selected filters for academic year {selectedYear}.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
        </div>
      </div>

      {/* Edit User Course Access Modal */}
      <Modal
        isOpen={!!editingAccessData}
        onClose={() => setEditingAccessData(null)}
        title="Edit Course Access"
        maxWidth="max-w-lg"
        footer={
          <div className="flex items-center justify-between w-full">
            <button
              type="button"
              onClick={handleDeleteAccessFromModal}
              className="flex items-center px-3 py-2 text-sm font-medium text-red-600 hover:text-red-800 bg-red-50 hover:bg-red-100 border border-red-200 rounded-lg transition-colors cursor-pointer"
              title="Remove user access from this course"
            >
              <Trash2 size={15} className="mr-1.5" /> Remove Access
            </button>
            <div className="flex items-center gap-2">
              <button
                type="button"
                onClick={() => setEditingAccessData(null)}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSaveEditAccess}
                className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors shadow-sm cursor-pointer"
              >
                Save Changes
              </button>
            </div>
          </div>
        }
      >
        {editingAccessData && (
          <div className="space-y-4">
            {/* User & Course Context (Read-Only) */}
            <div className="bg-slate-50 border border-slate-200 rounded-xl p-3.5 space-y-2">
              <div className="flex items-center gap-3">
                <div className="h-9 w-9 rounded-full bg-indigo-100 text-indigo-700 flex items-center justify-center font-bold text-sm shrink-0">
                  {editingAccessData.userAccess.userName.charAt(0)}
                </div>
                <div className="min-w-0 flex-1">
                  <h4 className="text-sm font-bold text-gray-900 truncate">
                    {editingAccessData.userAccess.userName}
                  </h4>
                  <p className="text-xs text-gray-500 font-mono truncate">
                    {editingAccessData.userAccess.userEmail}
                  </p>
                </div>
                <span className="text-xs font-semibold px-2 py-0.5 rounded-md bg-white border border-gray-200 text-gray-700 shadow-2xs capitalize">
                  {editingAccessData.userAccess.userRole.replace('_', ' ')}
                </span>
              </div>

              <div className="pt-2 border-t border-slate-200/80 space-y-1.5 text-xs">
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <span className="text-gray-500">Course: </span>
                    <span className="font-bold text-indigo-700">
                      {editingAccessData.courseId}: {editingAccessData.courseName}
                    </span>
                  </div>
                  <div className="shrink-0 text-right whitespace-nowrap">
                    <span className="text-gray-500">Year: </span>
                    <span className="font-bold text-indigo-700">
                      {editingAccessData.year || selectedYear}
                    </span>
                  </div>
                </div>
                {editingAccessData.department && (
                  <div>
                    <span className="text-gray-500">Department: </span>
                    <span className="font-semibold text-gray-800">{editingAccessData.department}</span>
                  </div>
                )}
                {editingAccessData.curriculum && (
                  <div>
                    <span className="text-gray-500">Curriculum: </span>
                    <span className="font-semibold text-gray-800">{editingAccessData.curriculum}</span>
                  </div>
                )}
              </div>
            </div>

            {/* Editable Access Level Radio Options */}
            <div>
              <label className="block text-xs font-bold text-gray-700 uppercase tracking-wider mb-2">
                Select permission for this course portfolio
              </label>
              <div className="space-y-2.5">
                {/* Upload and Download */}
                <label
                  className={`flex items-center gap-3 p-3 rounded-xl border-2 cursor-pointer transition-all ${
                    editingAccessLevel === 'Edit and Download'
                      ? 'border-emerald-500 bg-emerald-50/50 shadow-2xs'
                      : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50/50'
                  }`}
                >
                  <input
                    type="radio"
                    name="editAccessLevel"
                    value="Edit and Download"
                    checked={editingAccessLevel === 'Edit and Download'}
                    onChange={() => setEditingAccessLevel('Edit and Download')}
                    className="h-4 w-4 text-emerald-600 border-gray-300 focus:ring-emerald-500 cursor-pointer"
                  />
                  <span className="text-sm font-bold text-gray-900">Upload and Download</span>
                </label>

                {/* Download only */}
                <label
                  className={`flex items-center gap-3 p-3 rounded-xl border-2 cursor-pointer transition-all ${
                    editingAccessLevel === 'Download'
                      ? 'border-blue-500 bg-blue-50/50 shadow-2xs'
                      : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50/50'
                  }`}
                >
                  <input
                    type="radio"
                    name="editAccessLevel"
                    value="Download"
                    checked={editingAccessLevel === 'Download'}
                    onChange={() => setEditingAccessLevel('Download')}
                    className="h-4 w-4 text-blue-600 border-gray-300 focus:ring-blue-500 cursor-pointer"
                  />
                  <span className="text-sm font-bold text-gray-900">Download only</span>
                </label>
              </div>
            </div>
          </div>
        )}
      </Modal>

      {/* Assign User to Course Modal */}
      <Modal
        isOpen={!!assigningCourse}
        onClose={() => setAssigningCourse(null)}
        title="Assign New User"
        maxWidth="max-w-lg"
        footer={
          <>
            <button
              type="button"
              onClick={() => setAssigningCourse(null)}
              className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
            >
              Cancel
            </button>
            <button
              type="button"
              onClick={handleSaveAssignUser}
              disabled={!assignUserId}
              className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors shadow-sm cursor-pointer disabled:opacity-50"
            >
              Assign
            </button>
          </>
        }
      >
        {assigningCourse && (
          <div className="space-y-4">
            {/* Target Course Banner */}
            <div className="bg-indigo-50/70 border border-indigo-100 rounded-xl p-3.5 space-y-1.5 text-xs">
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0 flex-1">
                  <span className="text-gray-500">Course: </span>
                  <span className="font-bold text-indigo-900 text-sm">
                    {assigningCourse.id}: {assigningCourse.name}
                  </span>
                </div>
                <div className="shrink-0 text-right whitespace-nowrap">
                  <span className="text-gray-500">Year: </span>
                  <span className="font-bold text-indigo-900 text-sm">
                    {assigningCourse.year || selectedYear}
                  </span>
                </div>
              </div>

              <div>
                <span className="text-gray-500">Department: </span>
                <span className="font-semibold text-gray-800">
                  {assigningCourse.department || '-'}
                </span>
              </div>

              <div>
                <span className="text-gray-500">Curriculum: </span>
                <span className="font-semibold text-gray-800">
                  {assigningCourse.curriculum || '-'}
                </span>
              </div>
            </div>

            {/* User Select Dropdown */}
            <div>
              <label className="block text-xs font-bold text-gray-700 uppercase tracking-wider mb-1.5">
                Select Instructor or Staff Member
              </label>
              <select
                value={assignUserId}
                onChange={(e) => setAssignUserId(e.target.value)}
                className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white cursor-pointer"
              >
                <option value="">Select a user to assign...</option>
                {(users || MOCK_USERS)
                  .filter((u) => u.role !== 'admin' && u.role !== 'super_admin')
                  .map((u) => {
                    const alreadyAssigned = (courseAssignments[assigningCourse.id] || []).some(a => a.userId === u.id);
                    return (
                      <option key={u.id} value={u.id} disabled={alreadyAssigned}>
                        {u.name} ({u.email}) - {u.role.replace('_', ' ')} {alreadyAssigned ? '— [Already Assigned]' : ''}
                      </option>
                    );
                  })}
              </select>
            </div>

            {/* Access Level Selector */}
            <div>
              <label className="block text-xs font-bold text-gray-700 uppercase tracking-wider mb-2">
                Select permission for this course portfolio
              </label>
              <div className="space-y-2.5">
                <label
                  className={`flex items-center gap-3 p-3 rounded-xl border-2 cursor-pointer transition-all ${
                    assignAccessLevel === 'Edit and Download'
                      ? 'border-emerald-500 bg-emerald-50/50 shadow-2xs'
                      : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50/50'
                  }`}
                >
                  <input
                    type="radio"
                    name="assignAccessLevel"
                    value="Edit and Download"
                    checked={assignAccessLevel === 'Edit and Download'}
                    onChange={() => setAssignAccessLevel('Edit and Download')}
                    className="h-4 w-4 text-emerald-600 border-gray-300 focus:ring-emerald-500 cursor-pointer"
                  />
                  <span className="text-sm font-bold text-gray-900">Upload and Download</span>
                </label>

                <label
                  className={`flex items-center gap-3 p-3 rounded-xl border-2 cursor-pointer transition-all ${
                    assignAccessLevel === 'Download'
                      ? 'border-blue-500 bg-blue-50/50 shadow-2xs'
                      : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50/50'
                  }`}
                >
                  <input
                    type="radio"
                    name="assignAccessLevel"
                    value="Download"
                    checked={assignAccessLevel === 'Download'}
                    onChange={() => setAssignAccessLevel('Download')}
                    className="h-4 w-4 text-blue-600 border-gray-300 focus:ring-blue-500 cursor-pointer"
                  />
                  <span className="text-sm font-bold text-gray-900">Download only</span>
                </label>
              </div>
            </div>
          </div>
        )}
      </Modal>

      {/* Add/Edit Course Modal */}
      <Modal 
        isOpen={isAddModalOpen || !!editingCourse} 
        onClose={() => { setIsAddModalOpen(false); setEditingCourse(null); }} 
        title={editingCourse ? "Edit Course" : "Add New Course"}
        footer={
          editingCourse ? (
            <div className="flex items-center justify-between w-full">
              <button
                type="button"
                onClick={() => {
                  const target = editingCourse;
                  setIsAddModalOpen(false);
                  setEditingCourse(null);
                  setCourseToDelete(target);
                  setDeleteConfirmOpen(true);
                }}
                className="flex items-center px-3 py-2 text-sm font-medium text-red-600 hover:text-red-800 bg-red-50 hover:bg-red-100 border border-red-200 rounded-lg transition-colors cursor-pointer"
                title="Delete this course"
              >
                <Trash2 size={15} className="mr-1.5" /> Delete Course
              </button>
              <div className="flex items-center gap-2">
                <button 
                  type="button"
                  onClick={() => { setIsAddModalOpen(false); setEditingCourse(null); }} 
                  className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
                >
                  Cancel
                </button>
                <button 
                  type="button"
                  onClick={handleSaveCourse} 
                  className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors shadow-sm cursor-pointer"
                >
                  Save Changes
                </button>
              </div>
            </div>
          ) : (
            <div className="flex items-center justify-end gap-2 w-full">
              <button 
                type="button"
                onClick={() => { setIsAddModalOpen(false); setEditingCourse(null); }} 
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors cursor-pointer"
              >
                Cancel
              </button>
              <button 
                type="button"
                onClick={handleSaveCourse} 
                className="px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors shadow-sm cursor-pointer"
              >
                Add Course
              </button>
            </div>
          )
        }
      >
        <div className="space-y-4">
          {errorMsg && (
            <div className="p-3 bg-red-50 text-red-700 rounded-lg text-sm flex items-start border border-red-100">
              <AlertCircle size={16} className="mt-0.5 mr-2 shrink-0" />
              <p>{errorMsg}</p>
            </div>
          )}

          {!editingCourse && (
            <div className="flex border-b border-gray-200 mb-4">
              <button
                type="button"
                onClick={() => { setActiveCreationMethod('single'); setErrorMsg(''); }}
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
                onClick={() => { setActiveCreationMethod('bulk'); setErrorMsg(''); }}
                className={`flex-1 pb-2.5 text-sm font-medium border-b-2 text-center transition-colors ${
                  activeCreationMethod === 'bulk'
                    ? 'border-indigo-600 text-indigo-600 font-semibold'
                    : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300'
                }`}
              >
                Upload Excel
              </button>
            </div>
          )}

          {activeCreationMethod === 'single' ? (
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
                    value={editingNameInput}
                    onChange={(e) => setEditingNameInput(e.target.value)}
                    className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500" 
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="block text-sm font-medium text-gray-700">Academic Year</label>
                  </div>
                  {!editingCourse ? (
                    <input
                      type="text"
                      readOnly
                      disabled
                      value={selectedYear}
                      placeholder="No year selected"
                      className="w-full border border-gray-200 bg-gray-100 text-gray-700 rounded-lg px-3 py-2 text-sm cursor-not-allowed select-none focus:outline-none"
                    />
                  ) : (
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
                  )}
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
                  <div className="flex items-center justify-between mb-1">
                    <label className="block text-sm font-medium text-gray-700">Department</label>
                  </div>
                  {!editingCourse ? (
                    <input
                      type="text"
                      readOnly
                      disabled
                      value={editingDepartment}
                      placeholder="No department selected"
                      className="w-full border border-gray-200 bg-gray-100 text-gray-700 rounded-lg px-3 py-2 text-sm cursor-not-allowed select-none focus:outline-none"
                    />
                  ) : (
                    <select 
                      value={editingDepartment}
                      onChange={(e) => {
                        setEditingDepartment(e.target.value);
                        setEditingCurriculum('');
                      }}
                      className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white text-gray-900"
                    >
                      <option value="">Select Department...</option>
                      {departments.map((dept) => (
                        <option key={dept.id} value={dept.name}>{dept.name}</option>
                      ))}
                    </select>
                  )}
                </div>
                <div>
                  <div className="flex items-center justify-between mb-1">
                    <label className="block text-sm font-medium text-gray-700">Curriculum</label>
                  </div>
                  {!editingCourse ? (
                    <input
                      type="text"
                      readOnly
                      disabled
                      value={editingCurriculum}
                      placeholder="No curriculum selected"
                      className="w-full border border-gray-200 bg-gray-100 text-gray-700 rounded-lg px-3 py-2 text-sm cursor-not-allowed select-none focus:outline-none"
                    />
                  ) : (
                    <select 
                      value={editingCurriculum}
                      onChange={(e) => setEditingCurriculum(e.target.value)}
                      className="w-full border border-gray-300 rounded-lg px-3 py-2 text-sm focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 bg-white text-gray-900"
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
                  )}
                </div>
              </div>
            </div>
          ) : (
            <div className="space-y-4">
              <div 
                onClick={() => {
                  setBulkFileUploaded(true);
                  setBulkFileName('courses_template_import.xlsx');
                  const activeDeptName = editingDepartment || selectedDepartment || (departments[0]?.name || 'Mathematics and Computer Science');
                  const activeCurrName = editingCurriculum || selectedCurriculum || 'Computer Science';
                  setBulkMockCourses([
                    { id: 'CS102', name: 'Data Structures and Algorithms', department: activeDeptName, curriculum: activeCurrName, year: selectedYear, semester: '2' },
                    { id: 'MATH301', name: 'Abstract Algebra', department: activeDeptName, curriculum: activeCurrName, year: selectedYear, semester: '1' }
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
                    <p className="text-xs text-gray-500 mt-1">Supports .xlsx, .xls, .csv templates with Year & Semester</p>
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
                          <p className="font-bold text-gray-900">{c.id} - {c.name}</p>
                          <p className="text-gray-500">Year: {c.year} | Sem: {c.semester}</p>
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

      {/* Delete Courses Confirmation Modal */}
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
        <div className="space-y-4">
          {coursesWithGeneratedFiles.length > 0 ? (
            <div className="p-4 bg-red-50 text-red-800 border border-red-200 rounded-xl space-y-3">
              <div className="flex items-start">
                <AlertCircle className="w-5 h-5 text-red-600 mr-2 shrink-0 mt-0.5" />
                <div>
                  <p className="font-semibold text-sm">Warning: Generated Files Affected</p>
                  <p className="text-xs text-red-700 mt-1">
                    The following course(s) have generated final report files. Deleting them will permanently delete their corresponding files:
                  </p>
                </div>
              </div>
              <ul className="list-disc list-inside text-xs font-semibold pl-1 space-y-1">
                {coursesWithGeneratedFiles.map(c => (
                  <li key={c.id}>{c.id} - {c.name}</li>
                ))}
              </ul>
            </div>
          ) : (
            <p className="text-sm text-gray-600">
              {courseToDelete 
                ? `Are you sure you want to delete course "${courseToDelete.id} - ${courseToDelete.name}"? This action cannot be undone.`
                : `Are you sure you want to delete the ${selectedCourseIds.length} selected course(s)? This action cannot be undone.`
              }
            </p>
          )}
        </div>
      </Modal>
    </div>
  );
};

const CoverManagement = ({
  curriculums,
  setCurriculums,
  departments,
  courses
}: {
  curriculums: Curriculum[];
  setCurriculums: React.Dispatch<React.SetStateAction<Curriculum[]>>;
  departments: Department[];
  courses: any[];
}) => {
  // Top Dropdown Filters: Department & Curriculum & Status
  const [selectedDepartment, setSelectedDepartment] = useState<string>('');
  const [selectedCurriculum, setSelectedCurriculum] = useState<string>('');
  const [selectedStatus, setSelectedStatus] = useState<string>('');

  // Search & Sort states
  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState<string>('code');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');
  
  // Selection
  const [selectedCurrIds, setSelectedCurrIds] = useState<string[]>([]);

  // Modal States
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modalCurrId, setModalCurrId] = useState<string>('');
  const [isFixedCurriculum, setIsFixedCurriculum] = useState(false);
  const [modalFile, setModalFile] = useState<{ name: string; size: string } | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  // Preview and Delete modals
  const [previewCurr, setPreviewCurr] = useState<Curriculum | null>(null);
  const [deleteConfirmCurr, setDeleteConfirmCurr] = useState<Curriculum | null>(null);
  const [isBulkDeleteOpen, setIsBulkDeleteOpen] = useState(false);

  // Toast
  const [toastMessage, setToastMessage] = useState<{ type: 'success' | 'info' | 'error'; text: string } | null>(null);

  const showToast = (text: string, type: 'success' | 'info' | 'error' = 'success') => {
    setToastMessage({ text, type });
    setTimeout(() => setToastMessage(null), 3500);
  };

  const getCoursesForCurriculum = (currName: string) => {
    return courses.filter(c => c.curriculum === currName);
  };

  const handleDepartmentChange = (deptName: string) => {
    setSelectedDepartment(deptName);
    if (deptName && selectedCurriculum) {
      const parentDept = departments.find(d => d.name === deptName);
      const curr = curriculums.find(c => c.name === selectedCurriculum);
      if (curr && parentDept && curr.departmentId !== parentDept.id) {
        setSelectedCurriculum('');
      }
    }
  };

  const clearAllFilters = () => {
    setSelectedDepartment('');
    setSelectedCurriculum('');
    setSelectedStatus('');
    setSearchQuery('');
  };

  const hasActiveFilters = Boolean(
    selectedDepartment || selectedCurriculum || selectedStatus || searchQuery
  );

  // Filtered & Sorted Curriculums
  const filteredCurriculums = curriculums.filter(curr => {
    const dept = departments.find(d => d.id === curr.departmentId);
    const deptName = dept ? dept.name.toLowerCase() : '';
    const query = searchQuery.toLowerCase();

    const matchesDept = !selectedDepartment || dept?.name === selectedDepartment;
    const matchesCurriculum = !selectedCurriculum || curr.name === selectedCurriculum;
    const matchesStatus = (
      !selectedStatus ||
      (selectedStatus === 'has_cover' && !!curr.coverFile) ||
      (selectedStatus === 'no_cover' && !curr.coverFile)
    );

    const matchesSearch = (
      curr.name.toLowerCase().includes(query) ||
      curr.code.toLowerCase().includes(query) ||
      deptName.includes(query) ||
      (curr.coverFile && curr.coverFile.toLowerCase().includes(query))
    );

    return matchesDept && matchesCurriculum && matchesStatus && matchesSearch;
  });

  const sortedCurriculums = [...filteredCurriculums].sort((a, b) => {
    let aVal: any = '';
    let bVal: any = '';

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

  const handleSort = (field: string) => {
    if (sortField === field) {
      setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortDirection('asc');
    }
  };

  const handleToggleSelect = (id: string) => {
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

  const handleOpenUpload = (curr?: Curriculum) => {
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

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
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

  const handleFileDrop = (e: React.DragEvent<HTMLDivElement>) => {
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

  return (
    <div className="p-6 space-y-5 w-full">
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

      {/* Top Header: Title + Actions on Row 1, Filters on Row 2 under Title */}
      <div className="space-y-4">
        {/* Row 1: Title + Action Buttons */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Cover Management</h1>
            <p className="text-sm text-gray-500 mt-1">
              Manage standard course cover PDFs by curriculum. All courses under the same curriculum share the same cover.
            </p>
          </div>

          <div className="flex items-center gap-3 shrink-0">
            {selectedWithCovers.length > 0 && (
              <button 
                onClick={() => setIsBulkDeleteOpen(true)} 
                className="flex items-center px-4 py-2 bg-red-600 text-white rounded-lg hover:bg-red-700 font-medium text-sm transition-colors shadow-2xs whitespace-nowrap cursor-pointer"
              >
                <Trash2 size={16} className="mr-2" /> Remove Covers ({selectedWithCovers.length})
              </button>
            )}

            <button 
              onClick={() => handleOpenUpload()} 
              className="flex items-center px-4 py-2 bg-indigo-600 text-white rounded-lg hover:bg-indigo-700 font-medium text-sm transition-colors shadow-2xs whitespace-nowrap cursor-pointer"
            >
              <Plus size={16} className="mr-2" /> Upload Cover PDF
            </button>
          </div>
        </div>

        {/* Row 2: Filters under page title */}
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Department Searchable Dropdown */}
          <SearchableSelect
            label="Department"
            value={selectedDepartment}
            onChange={handleDepartmentChange}
            options={departments.map(d => ({ value: d.name, label: d.name, subtext: d.code }))}
            allLabel="All Departments"
            icon={Building2}
          />

          {/* Curriculum Searchable Dropdown */}
          <SearchableSelect
            label="Curriculum"
            value={selectedCurriculum}
            onChange={setSelectedCurriculum}
            options={curriculums
              .filter(c => {
                if (!selectedDepartment) return true;
                const parentDept = departments.find(d => d.name === selectedDepartment);
                return parentDept && c.departmentId === parentDept.id;
              })
              .map(c => ({ value: c.name, label: c.name, subtext: c.code }))
            }
            allLabel="All Curriculums"
            icon={GraduationCap}
          />

          {/* Cover Status Searchable Dropdown */}
          <SearchableSelect
            label="Cover Status"
            value={selectedStatus}
            onChange={setSelectedStatus}
            options={[
              { value: 'has_cover', label: 'Cover Uploaded' },
              { value: 'no_cover', label: 'Missing Cover' },
            ]}
            allLabel="All Statuses"
            icon={FileText}
          />
        </div>
      </div>

      {/* Search Bar + Active Filters Status */}
      <div className="bg-white p-3 rounded-xl border border-gray-200 shadow-2xs space-y-2">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
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

        {/* Active Filters Bar */}
        {hasActiveFilters && (
          <div className="flex flex-wrap items-center gap-2 pt-1 border-t border-gray-100 text-xs">
            <span className="text-gray-500 font-medium">Active filters:</span>
            {selectedDepartment && (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 px-2 py-0.5 rounded-full font-medium">
                Dept: {selectedDepartment}
                <button onClick={() => setSelectedDepartment('')} className="hover:text-indigo-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            {selectedCurriculum && (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 px-2 py-0.5 rounded-full font-medium">
                Curriculum: {selectedCurriculum}
                <button onClick={() => setSelectedCurriculum('')} className="hover:text-indigo-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            {selectedStatus && (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 px-2 py-0.5 rounded-full font-medium">
                Status: {selectedStatus === 'has_cover' ? 'Cover Uploaded' : 'Missing Cover'}
                <button onClick={() => setSelectedStatus('')} className="hover:text-indigo-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            {searchQuery && (
              <span className="inline-flex items-center gap-1 bg-gray-100 text-gray-700 border border-gray-200 px-2 py-0.5 rounded-full font-medium">
                Search: "{searchQuery}"
                <button onClick={() => setSearchQuery('')} className="hover:text-gray-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            <button
              onClick={clearAllFilters}
              className="text-indigo-600 hover:text-indigo-800 font-semibold underline ml-1 cursor-pointer flex items-center gap-1"
            >
              <RotateCcw size={11} /> Clear all
            </button>
          </div>
        )}
      </div>

      {/* Curriculums & Covers Table (Full Width) */}
      <div className="bg-white rounded-xl shadow-2xs border border-gray-200 overflow-hidden">
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
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
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
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
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
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
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
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
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
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
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
                      <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase font-mono">
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
                            className="text-gray-600 hover:text-indigo-900 bg-gray-50 hover:bg-indigo-50 p-1.5 rounded-lg transition-colors inline-flex items-center cursor-pointer"
                            title="Preview Cover PDF"
                          >
                            <Eye size={14} />
                          </button>
                          <button 
                            onClick={() => showToast(`Downloading ${curr.coverFile}...`)}
                            className="text-green-600 hover:text-green-900 bg-green-50 hover:bg-green-100 p-1.5 rounded-lg transition-colors inline-flex items-center cursor-pointer"
                            title="Download PDF"
                          >
                            <Download size={14} />
                          </button>
                        </>
                      )}
                      <button 
                        onClick={() => handleOpenUpload(curr)}
                        className="text-indigo-600 hover:text-indigo-900 bg-indigo-50 hover:bg-indigo-100 p-1.5 rounded-lg transition-colors inline-flex items-center cursor-pointer"
                        title={curr.coverFile ? "Replace Cover PDF" : "Upload Cover PDF"}
                      >
                        <UploadCloud size={14} />
                      </button>
                      {curr.coverFile && (
                        <button 
                          onClick={() => setDeleteConfirmCurr(curr)}
                          className="text-red-600 hover:text-red-900 bg-red-50 hover:bg-red-100 p-1.5 rounded-lg transition-colors inline-flex items-center cursor-pointer"
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

const CourseList = ({
  courses,
  files,
  departments,
  curriculums
}: {
  courses: any[];
  files: any[];
  departments: Department[];
  curriculums: Curriculum[];
}) => {
  const [uploadTargetCourse, setUploadTargetCourse] = useState<any>(null);
  const [selectedErrorFile, setSelectedErrorFile] = useState<{ course: any; file: CourseFile } | null>(null);
  const [expandedErrorIds, setExpandedErrorIds] = useState<string[]>([]);
  
  // Year selector - single choice, defaults to latest year (no 'all' choice)
  const [selectedYear, setSelectedYear] = useState('2024');

  // Top Dropdown Filters: Department & Curriculum
  const [selectedDepartment, setSelectedDepartment] = useState<string>('');
  const [selectedCurriculum, setSelectedCurriculum] = useState<string>('');

  // Column Filter: Semester (empty = all)
  const [selectedSemesters, setSelectedSemesters] = useState<string[]>([]);

  // Search & Sort states
  const [searchQuery, setSearchQuery] = useState('');
  const [sortField, setSortField] = useState<string>('id');
  const [sortDirection, setSortDirection] = useState<'asc' | 'desc'>('asc');

  const toggleSemester = (sem: string) => {
    if (selectedSemesters.includes(sem)) {
      setSelectedSemesters(selectedSemesters.filter(s => s !== sem));
    } else {
      setSelectedSemesters([...selectedSemesters, sem]);
    }
  };

  const handleSelectAllSemesters = () => {
    setSelectedSemesters(['1', '2', 'Summer']);
  };

  const handleClearSemesters = () => {
    setSelectedSemesters([]);
  };

  const handleDepartmentChange = (deptName: string) => {
    setSelectedDepartment(deptName);
    if (deptName && selectedCurriculum) {
      const parentDept = departments.find(d => d.name === deptName);
      const curr = curriculums.find(c => c.name === selectedCurriculum);
      if (curr && parentDept && curr.departmentId !== parentDept.id) {
        setSelectedCurriculum('');
      }
    }
  };

  const clearAllFilters = () => {
    setSelectedDepartment('');
    setSelectedCurriculum('');
    setSelectedSemesters([]);
    setSearchQuery('');
  };

  const hasActiveFilters = Boolean(
    selectedDepartment || selectedCurriculum || selectedSemesters.length > 0 || searchQuery
  );

  const handleOpenErrors = (course: any, file: CourseFile) => {
    setSelectedErrorFile({ course, file });
    setExpandedErrorIds([]);
  };

  // Helper to get file details for a course
  const getCourseFile = (courseId: string) => {
    return files.find(f => f.courseId === courseId && f.year === selectedYear);
  };

  // Filtered & Sorted courses
  const filteredCourses = courses.filter(course => {
    const file = getCourseFile(course.id);
    const statusText = file ? file.status : 'No files';
    const query = searchQuery.toLowerCase();
    
    const matchesYear = (course.year || '2024') === selectedYear;
    const matchesSemester = selectedSemesters.length === 0 || selectedSemesters.includes(course.semester || '1');
    const matchesDept = !selectedDepartment || course.department === selectedDepartment;
    const matchesCurriculum = !selectedCurriculum || course.curriculum === selectedCurriculum;

    const matchesSearch = (
      course.id.toLowerCase().includes(query) ||
      course.name.toLowerCase().includes(query) ||
      (course.department || '').toLowerCase().includes(query) ||
      (course.curriculum || '').toLowerCase().includes(query) ||
      statusText.toLowerCase().includes(query)
    );

    return matchesYear && matchesSemester && matchesDept && matchesCurriculum && matchesSearch;
  });

  const sortedCourses = [...filteredCourses].sort((a, b) => {
    let aVal: any = '';
    let bVal: any = '';

    if (sortField === 'id') {
      aVal = a.id;
      bVal = b.id;
    } else if (sortField === 'name') {
      aVal = a.name;
      bVal = b.name;
    } else if (sortField === 'semester') {
      aVal = a.semester || '1';
      bVal = b.semester || '1';
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

  return (
    <div className="p-6 space-y-5 w-full">
      {/* Top Header: Title on Row 1, Filters on Row 2 under Title */}
      <div className="space-y-4">
        {/* Row 1: Title */}
        <div className="flex items-center justify-between">
          <h1 className="text-2xl font-bold text-gray-900">Course Portfolios</h1>
        </div>

        {/* Row 2: Filters under page title */}
        <div className="flex flex-wrap items-center gap-2.5">
          {/* Department Searchable Dropdown */}
          <SearchableSelect
            label="Department"
            value={selectedDepartment}
            onChange={handleDepartmentChange}
            options={departments.map(d => ({ value: d.name, label: d.name, subtext: d.code }))}
            allLabel="All Departments"
            icon={Building2}
          />

          {/* Curriculum Searchable Dropdown */}
          <SearchableSelect
            label="Curriculum"
            value={selectedCurriculum}
            onChange={setSelectedCurriculum}
            options={curriculums
              .filter(c => {
                if (!selectedDepartment) return true;
                const parentDept = departments.find(d => d.name === selectedDepartment);
                return parentDept && c.departmentId === parentDept.id;
              })
              .map(c => ({ value: c.name, label: c.name, subtext: c.code }))
            }
            allLabel="All Curriculums"
            icon={GraduationCap}
          />

          {/* Year Dropdown */}
          <div className="flex items-center bg-white border border-gray-300 rounded-lg shadow-2xs px-3 py-1.5">
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

      {/* Search Bar + Active Filters Status */}
      <div className="bg-white p-3 rounded-xl border border-gray-200 shadow-2xs space-y-2">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
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
            Showing <span className="font-semibold text-gray-900">{sortedCourses.length}</span> course{sortedCourses.length === 1 ? '' : 's'} in <span className="font-semibold text-indigo-600">{selectedYear}</span>
          </div>
        </div>

        {/* Active Filters Bar */}
        {hasActiveFilters && (
          <div className="flex flex-wrap items-center gap-2 pt-1 border-t border-gray-100 text-xs">
            <span className="text-gray-500 font-medium">Active filters:</span>
            {selectedDepartment && (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 px-2 py-0.5 rounded-full font-medium">
                Dept: {selectedDepartment}
                <button onClick={() => setSelectedDepartment('')} className="hover:text-indigo-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            {selectedCurriculum && (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 px-2 py-0.5 rounded-full font-medium">
                Curriculum: {selectedCurriculum}
                <button onClick={() => setSelectedCurriculum('')} className="hover:text-indigo-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            {selectedSemesters.length > 0 && (
              <span className="inline-flex items-center gap-1 bg-indigo-50 text-indigo-700 border border-indigo-200 px-2 py-0.5 rounded-full font-medium">
                Semesters: {selectedSemesters.map(s => s === 'Summer' ? 'Summer' : `Sem ${s}`).join(', ')}
                <button onClick={() => setSelectedSemesters([])} className="hover:text-indigo-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            {searchQuery && (
              <span className="inline-flex items-center gap-1 bg-gray-100 text-gray-700 border border-gray-200 px-2 py-0.5 rounded-full font-medium">
                Search: "{searchQuery}"
                <button onClick={() => setSearchQuery('')} className="hover:text-gray-900 cursor-pointer"><X size={12} /></button>
              </span>
            )}
            <button
              onClick={clearAllFilters}
              className="text-indigo-600 hover:text-indigo-800 font-semibold underline ml-1 cursor-pointer flex items-center gap-1"
            >
              <RotateCcw size={11} /> Clear all
            </button>
          </div>
        )}
      </div>

      {/* Course Table (Full Width) */}
      <div className="bg-white rounded-xl shadow-2xs border border-gray-200 overflow-hidden">
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-200">
            <thead className="bg-gray-50">
              <tr>
                <th 
                  onClick={() => handleSort('id')}
                  className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-1">
                    Course ID
                    {sortField === 'id' ? (
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
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
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
                    ) : (
                      <ArrowUpDown size={12} className="text-gray-400" />
                    )}
                  </div>
                </th>
                <SemesterHeaderFilter
                  sortField={sortField}
                  sortDirection={sortDirection}
                  onSort={() => handleSort('semester')}
                  selectedSemesters={selectedSemesters}
                  onToggleSemester={toggleSemester}
                  onSelectAllSemesters={handleSelectAllSemesters}
                  onClearSemesters={handleClearSemesters}
                />
                <th 
                  onClick={() => handleSort('status')}
                  className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider cursor-pointer hover:bg-gray-100 transition-colors"
                >
                  <div className="flex items-center gap-1">
                    Status
                    {sortField === 'status' ? (
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
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
                    Latest Update
                    {sortField === 'time' ? (
                      sortDirection === 'asc' ? <ChevronUp size={14} className="text-indigo-600" /> : <ChevronDown size={14} className="text-indigo-600" />
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
                const file = getCourseFile(course.id);

                return (
                  <tr 
                    key={course.id} 
                    className="hover:bg-gray-50 transition-colors"
                  >
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-semibold text-indigo-700">
                      <span className="bg-indigo-50 px-2.5 py-1 rounded-md text-xs tracking-wider uppercase">
                        {course.id}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                      <span>{course.name}</span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-600">
                      <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-gray-100 text-gray-700">
                        {course.semester === 'Summer' ? 'Summer' : `Sem ${course.semester || '1'}`}
                      </span>
                    </td>
                    <td className="px-6 py-4 whitespace-nowrap text-sm">
                      {file ? (
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
                    <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium space-x-2">
                      {file?.status === 'generated' && (
                        <button 
                          className="text-green-600 hover:text-green-900 bg-green-50 hover:bg-green-100 p-1.5 rounded-lg transition-colors inline-flex items-center cursor-pointer"
                          title="Download PDF"
                        >
                          <Download size={14} />
                        </button>
                      )}
                      {file?.status === 'failed' && (
                        <button 
                          onClick={() => handleOpenErrors(course, file)}
                          className="text-red-600 hover:text-red-900 bg-red-50 hover:bg-red-100 p-1.5 rounded-lg transition-colors inline-flex items-center cursor-pointer"
                          title="View Processing Errors"
                        >
                          <AlertTriangle size={14} />
                        </button>
                      )}
                      <button 
                        onClick={() => setUploadTargetCourse(course)}
                        className="text-indigo-600 hover:text-indigo-900 bg-indigo-50 hover:bg-indigo-100 p-1.5 rounded-lg transition-colors inline-flex items-center cursor-pointer"
                        title={file?.status === 'failed' ? "Re-upload Files" : "Upload Files"}
                      >
                        <UploadCloud size={14} />
                      </button>
                    </td>
                  </tr>
                );
              })}
              {sortedCourses.length === 0 && (
                <tr>
                  <td colSpan={6} className="text-center py-8 text-gray-500 italic">
                    No courses found matching the selected filters for academic year {selectedYear}.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
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
                  const allIds = selectedErrorFile.file.errors?.map((e: any) => e.id) || [];
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
                selectedErrorFile.file.errors.map((error: any, index: number) => {
                  const isExpanded = expandedErrorIds.includes(error.id);
                  
                  const stageColors: Record<string, string> = {
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
        title="Upload Course Portfolio Folder"
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
        <div className="space-y-4">
          {/* Target Course Banner */}
          <div className="bg-indigo-50/70 border border-indigo-100 rounded-xl p-3.5 space-y-1.5 text-xs">
            <div className="flex items-start justify-between gap-3">
              <div className="min-w-0 flex-1">
                <span className="text-gray-500">Course: </span>
                <span className="font-bold text-indigo-900 text-sm">
                  {uploadTargetCourse?.id}: {uploadTargetCourse?.name}
                </span>
              </div>
              <div className="shrink-0 text-right whitespace-nowrap">
                <span className="text-gray-500">Year: </span>
                <span className="font-bold text-indigo-900 text-sm">
                  {uploadTargetCourse?.year || selectedYear}
                </span>
              </div>
            </div>

            <div>
              <span className="text-gray-500">Department: </span>
              <span className="font-semibold text-gray-800">
                {uploadTargetCourse?.department || '-'}
              </span>
            </div>

            <div>
              <span className="text-gray-500">Curriculum: </span>
              <span className="font-semibold text-gray-800">
                {uploadTargetCourse?.curriculum || '-'}
              </span>
            </div>
          </div>
          
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">Upload Folder</label>
            <div className="mt-1 flex justify-center px-6 pt-5 pb-6 border-2 border-gray-300 border-dashed rounded-xl bg-gray-50 hover:bg-indigo-50 hover:border-indigo-300 transition-colors cursor-pointer group">
              <div className="space-y-1 text-center">
                <UploadCloud className="mx-auto h-12 w-12 text-gray-400 group-hover:text-indigo-500 transition-colors" />
                <div className="flex text-sm text-gray-600 justify-center">
                  <span className="relative rounded-md font-medium text-indigo-600 focus-within:outline-none focus-within:ring-2 focus-within:ring-offset-2 focus-within:ring-indigo-500">
                    <span>Drag and drop</span>
                  </span>
                  <p className="pl-1">your folder here</p>
                </div>
                <p className="text-xs text-gray-500">Files will be processed into a Course Portfolio PDF</p>
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
  const [activeTab, setActiveTab] = useState('courses'); // 'courses', 'course_management', 'cover_management', 'users', 'departments', 'curriculums'
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
            <BookOpen size={18} className="mr-3" /> Course Portfolios
          </button>
          
          {['super_admin', 'admin'].includes(currentUser.role) && (
            <>
              <button
                onClick={() => setActiveTab('course_management')}
                className={`w-full flex items-center px-3 py-2.5 rounded-lg text-sm font-medium transition-colors ${activeTab === 'course_management' ? 'bg-indigo-50 text-indigo-700' : 'text-gray-600 hover:bg-gray-100'}`}
              >
                <BookMarked size={18} className="mr-3" /> Curriculum Management
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

        {/* Theme Settings for Authenticated Users */}
        <div className="px-4 py-3 border-t border-gray-200">
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
            files={files} 
            departments={departments}
            curriculums={curriculums}
          />
        )}
        {activeTab === 'course_management' && (
          <CourseManagement 
            courses={courses} 
            setCourses={setCourses} 
            departments={departments} 
            curriculums={curriculums} 
            files={files} 
            setFiles={setFiles} 
            users={users}
          />
        )}
        {activeTab === 'cover_management' && (
          <CoverManagement 
            curriculums={curriculums} 
            setCurriculums={setCurriculums} 
            departments={departments} 
            courses={courses} 
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