import { createRootRouteWithContext, Outlet, Link } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/router-devtools'
import type { AuthContext } from '@/hooks/useAuth'

interface RouterContext {
  auth: AuthContext
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
})

function RootLayout() {
  return (
    <>
      <div className="min-h-screen bg-gray-50">
        <Navigation />
        <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <Outlet />
        </main>
      </div>
      {import.meta.env.DEV && <TanStackRouterDevtools />}
    </>
  )
}

function Navigation() {
  const { auth } = Route.useRouteContext()

  return (
    <nav className="bg-white shadow-sm border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex">
            <div className="flex-shrink-0 flex items-center">
              <Link to="/" className="text-xl font-bold text-primary-600">
                namazu
              </Link>
              <span className="ml-2 text-sm text-gray-500">
                地震速報 Webhook 中継
              </span>
            </div>
            <div className="hidden sm:ml-8 sm:flex sm:space-x-4">
              <NavLink to="/">ダッシュボード</NavLink>
              <NavLink to="/subscriptions">Subscriptions</NavLink>
              <NavLink to="/settings">設定</NavLink>
            </div>
          </div>
          <div className="flex items-center">
            {auth.isLoading ? (
              <div className="w-8 h-8 rounded-full bg-gray-200 animate-pulse" />
            ) : auth.isAuthenticated ? (
              <UserMenu />
            ) : (
              <button
                onClick={() => auth.signInWithGoogle()}
                className="btn btn-primary"
              >
                ログイン
              </button>
            )}
          </div>
        </div>
      </div>
    </nav>
  )
}

function NavLink({ to, children }: { to: string; children: React.ReactNode }) {
  return (
    <Link
      to={to}
      className="inline-flex items-center px-3 py-2 text-sm font-medium text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-md transition-colors"
      activeProps={{
        className: 'text-primary-600 bg-primary-50',
      }}
    >
      {children}
    </Link>
  )
}

function UserMenu() {
  const { auth } = Route.useRouteContext()

  return (
    <div className="flex items-center space-x-4">
      <div className="flex items-center space-x-2">
        {auth.user?.photoURL && (
          <img
            src={auth.user.photoURL}
            alt=""
            className="w-8 h-8 rounded-full"
          />
        )}
        <span className="text-sm text-gray-700">
          {auth.user?.displayName || auth.user?.email}
        </span>
      </div>
      <button
        onClick={() => auth.signOut()}
        className="text-sm text-gray-500 hover:text-gray-700"
      >
        ログアウト
      </button>
    </div>
  )
}
