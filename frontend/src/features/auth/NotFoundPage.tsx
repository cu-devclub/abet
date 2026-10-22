import { Link } from 'react-router'

export function NotFoundPage() {
  return (
    <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center gap-3 text-center px-4 font-sans">
      <p className="text-5xl font-bold text-indigo-600">404</p>
      <h1 className="text-lg font-semibold text-gray-900">Page not found</h1>
      <p className="text-sm text-gray-500">The page you are looking for doesn't exist or has moved.</p>
      <Link to="/portfolios" className="mt-2 px-4 py-2 text-sm font-medium text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 transition-colors">
        Back to Course Portfolios
      </Link>
    </div>
  )
}
