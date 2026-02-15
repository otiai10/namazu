import { useState } from 'react'
import { createRootRouteWithContext, Outlet, Link } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/router-devtools'
import { useAuth, type AuthContext } from '@/hooks/useAuth'

interface RouterContext {
  auth: AuthContext
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: RootLayout,
})

function RootLayout() {
  const auth = useAuth()

  return (
    <>
      <div className="min-h-screen bg-gray-50">
        {auth.isAuthenticated && <Navigation />}
        <main className={auth.isAuthenticated ? 'max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8' : ''}>
          <Outlet />
        </main>
      </div>
      {import.meta.env.DEV && <TanStackRouterDevtools />}
    </>
  )
}

function HamburgerIcon() {
  return (
    <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M4 6h16M4 12h16M4 18h16" />
    </svg>
  )
}

function CloseIcon() {
  return (
    <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
      <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
    </svg>
  )
}

function MobileMenuButton({ isOpen, onToggle }: { isOpen: boolean; onToggle: () => void }) {
  return (
    <button
      type="button"
      onClick={onToggle}
      className="sm:hidden inline-flex items-center justify-center p-2 rounded-md text-gray-500 hover:text-gray-700 hover:bg-gray-100 transition-colors"
      aria-expanded={isOpen}
      aria-controls="mobile-menu"
      aria-label={isOpen ? 'メニューを閉じる' : 'メニューを開く'}
    >
      {isOpen ? <CloseIcon /> : <HamburgerIcon />}
    </button>
  )
}

function MobileNavLink({ to, children, onClick }: { to: string; children: React.ReactNode; onClick: () => void }) {
  return (
    <Link
      to={to}
      onClick={onClick}
      className="block px-3 py-2 text-base font-medium text-gray-600 hover:text-gray-900 hover:bg-gray-100 rounded-md transition-colors"
      activeProps={{
        className: 'text-primary-600 bg-primary-50',
      }}
    >
      {children}
    </Link>
  )
}

function MobileUserInfo() {
  const auth = useAuth()

  return (
    <div className="border-t border-gray-200 pt-4 pb-2 px-3">
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
        type="button"
        onClick={() => auth.signOut()}
        className="mt-3 w-full text-left px-3 py-2 text-sm text-gray-500 hover:text-gray-700 hover:bg-gray-100 rounded-md transition-colors"
      >
        ログアウト
      </button>
    </div>
  )
}

function MobileMenu({ isOpen, onClose }: { isOpen: boolean; onClose: () => void }) {
  if (!isOpen) {
    return null
  }

  return (
    <div id="mobile-menu" className="sm:hidden border-t border-gray-200">
      <div className="px-2 pt-2 pb-3 space-y-1">
        <MobileNavLink to="/" onClick={onClose}>ダッシュボード</MobileNavLink>
        <MobileNavLink to="/settings" onClick={onClose}>設定</MobileNavLink>
      </div>
      <MobileUserInfo />
    </div>
  )
}

function Navigation() {
  const [isMobileMenuOpen, setIsMobileMenuOpen] = useState(false)

  const handleToggleMobileMenu = () => {
    setIsMobileMenuOpen((prev) => !prev)
  }

  const handleCloseMobileMenu = () => {
    setIsMobileMenuOpen(false)
  }

  return (
    <nav className="bg-white shadow-sm border-b border-gray-200">
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex justify-between h-16">
          <div className="flex">
            <div className="flex-shrink-0 flex items-center">
              <Link to="/" className="text-xl font-bold text-primary-600">
                namazu
              </Link>
              <span className="ml-2 text-sm text-gray-500 hidden sm:inline">
                地震速報 Webhook 中継
              </span>
            </div>
            <div className="hidden sm:ml-8 sm:flex sm:space-x-4">
              <NavLink to="/">ダッシュボード</NavLink>
              <NavLink to="/settings">設定</NavLink>
            </div>
          </div>
          <div className="hidden sm:flex sm:items-center">
            <UserMenu />
          </div>
          <div className="flex items-center sm:hidden">
            <MobileMenuButton isOpen={isMobileMenuOpen} onToggle={handleToggleMobileMenu} />
          </div>
        </div>
      </div>
      <MobileMenu isOpen={isMobileMenuOpen} onClose={handleCloseMobileMenu} />
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
  const auth = useAuth()

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
        type="button"
        onClick={() => auth.signOut()}
        className="text-sm text-gray-500 hover:text-gray-700"
      >
        ログアウト
      </button>
    </div>
  )
}
