import { createBrowserRouter, Navigate } from 'react-router'
import { AppShell } from '../components/layout/AppShell'
import { LoginPage } from '../features/auth/LoginPage'
import { NotFoundPage } from '../features/auth/NotFoundPage'
import { RequireAuth } from '../features/auth/RequireAuth'
import { RequireCapability } from '../features/auth/RequireCapability'
import { PortfoliosPage } from '../features/portfolios/PortfoliosPage'

export const router = createBrowserRouter([
  { path: '/login', element: <LoginPage /> },
  {
    element: <RequireAuth />,
    children: [
      {
        element: <AppShell />,
        children: [
          { path: '/', element: <Navigate to="/portfolios" replace /> },
          { path: '/portfolios', element: <PortfoliosPage /> },
          {
            element: <RequireCapability capability="manage_courses" />,
            children: [{ path: '/courses', lazy: async () => ({ Component: (await import('../features/courses/CoursesPage')).CoursesPage }) }],
          },
          {
            element: <RequireCapability capability="manage_covers" />,
            children: [{ path: '/covers', lazy: async () => ({ Component: (await import('../features/covers/CoversPage')).CoversPage }) }],
          },
          {
            element: <RequireCapability capability="manage_users" />,
            children: [{ path: '/users', lazy: async () => ({ Component: (await import('../features/users/UsersPage')).UsersPage }) }],
          },
          {
            element: <RequireCapability capability="manage_org" />,
            children: [
              { path: '/departments', lazy: async () => ({ Component: (await import('../features/departments/DepartmentsPage')).DepartmentsPage }) },
              { path: '/curricula', lazy: async () => ({ Component: (await import('../features/curricula/CurriculaPage')).CurriculaPage }) },
            ],
          },
        ],
      },
    ],
  },
  { path: '*', element: <NotFoundPage /> },
])
