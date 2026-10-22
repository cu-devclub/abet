export const ADD_COURSE_TABS = [
  { id: 'single', label: 'Single Course' },
  { id: 'csv', label: 'Upload CSV' },
] as const

export type AddCourseTab = (typeof ADD_COURSE_TABS)[number]['id']
