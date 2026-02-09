import { StrictMode, useEffect } from 'react'
import { createRoot } from 'react-dom/client'
import { RouterProvider, createRouter } from '@tanstack/react-router'
import { routeTree } from './routeTree.gen'
import { AuthProvider, useAuth } from './hooks/useAuth'
import './index.css'

function PendingSpinner() {
  return (
    <div className="min-h-screen bg-gray-50 flex items-center justify-center">
      <div className="w-8 h-8 border-4 border-primary-200 border-t-primary-600 rounded-full animate-spin" />
    </div>
  )
}

// Create a new router instance
const router = createRouter({
  routeTree,
  context: {
    auth: undefined!,
  },
  defaultPendingComponent: PendingSpinner,
})

// Register the router instance for type safety
declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}

function InnerApp() {
  const auth = useAuth()

  // Navigate when auth state changes and current page doesn't match.
  // beforeLoad guards handle initial page load; this handles
  // runtime transitions (login success, logout).
  useEffect(() => {
    if (auth.isLoading) return
    const currentPath = router.state.location.pathname
    if (auth.isAuthenticated && currentPath === '/login') {
      router.navigate({ to: '/' })
    }
  }, [auth.isAuthenticated, auth.isLoading])

  return <RouterProvider router={router} context={{ auth }} />
}

function App() {
  return (
    <AuthProvider>
      <InnerApp />
    </AuthProvider>
  )
}

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)
