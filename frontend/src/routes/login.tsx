import { createFileRoute, redirect } from '@tanstack/react-router'
import { useAuth } from '@/hooks/useAuth'

interface LoginSearch {
  redirect?: string
}

export const Route = createFileRoute('/login')({
  validateSearch: (search: Record<string, unknown>): LoginSearch => ({
    redirect: typeof search.redirect === 'string' ? search.redirect : undefined,
  }),
  beforeLoad: async ({ context, search }) => {
    await context.auth.waitUntilReady()
    if (context.auth.isAuthenticated) {
      throw redirect({ to: search.redirect ?? '/' })
    }
  },
  component: LoginPage,
})

function LoginPage() {
  const { signInWithGoogle, isDemoMode } = useAuth()

  const handleLogin = async () => {
    try {
      await signInWithGoogle()
      // Navigation is handled automatically by router.invalidate()
      // in main.tsx when auth.isAuthenticated changes.
    } catch (error) {
      console.error('Login failed:', error)
    }
  }

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center px-4">
      <div className="w-full max-w-md space-y-8">
        {/* Logo */}
        <div className="text-center">
          <h1 className="text-4xl font-bold text-primary-600">namazu</h1>
          <p className="mt-2 text-sm sm:text-base text-gray-500">地震速報 Webhook 中継サービス</p>
        </div>

        {/* Description */}
        <div className="card bg-gradient-to-r from-primary-600 to-primary-800 text-white">
          <p className="text-primary-100">
            P2P地震情報から受信した地震速報を、あなたの Webhook エンドポイントに即座に配信します。
          </p>
        </div>

        {/* Login button */}
        <div className="card text-center space-y-4">
          <button
            onClick={handleLogin}
            className="w-full inline-flex items-center justify-center px-6 py-3 bg-white border border-gray-300 rounded-lg shadow-sm text-gray-700 font-medium hover:bg-gray-50 transition-colors"
          >
            <GoogleIcon className="w-5 h-5 mr-3" />
            Google でログイン
          </button>
          {isDemoMode && (
            <p className="text-sm text-gray-400">デモモードで動作中</p>
          )}
        </div>

        {/* Features */}
        <div className="grid grid-cols-3 gap-2 sm:gap-3 text-center">
          <div className="p-3">
            <BoltIcon className="w-6 h-6 text-yellow-500 mx-auto mb-1" />
            <p className="text-xs text-gray-500">リアルタイム配信</p>
          </div>
          <div className="p-3">
            <FilterIcon className="w-6 h-6 text-blue-500 mx-auto mb-1" />
            <p className="text-xs text-gray-500">フィルタリング</p>
          </div>
          <div className="p-3">
            <RefreshIcon className="w-6 h-6 text-green-500 mx-auto mb-1" />
            <p className="text-xs text-gray-500">自動リトライ</p>
          </div>
        </div>

        {/* P2P Earthquake info link */}
        <div className="text-center">
          <a
            href="https://www.p2pquake.net/"
            target="_blank"
            rel="noopener noreferrer"
            className="text-sm text-gray-400 hover:text-gray-600 transition-colors"
          >
            P2P地震情報について
          </a>
        </div>
      </div>
    </div>
  )
}

function GoogleIcon({ className }: { className?: string }) {
  return (
    <svg className={className} viewBox="0 0 24 24">
      <path fill="#4285F4" d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92a5.06 5.06 0 01-2.2 3.32v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.1z" />
      <path fill="#34A853" d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" />
      <path fill="#FBBC05" d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" />
      <path fill="#EA4335" d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" />
    </svg>
  )
}

function BoltIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
    </svg>
  )
}

function FilterIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M3 4a1 1 0 011-1h16a1 1 0 011 1v2.586a1 1 0 01-.293.707l-6.414 6.414a1 1 0 00-.293.707V17l-4 4v-6.586a1 1 0 00-.293-.707L3.293 7.293A1 1 0 013 6.586V4z" />
    </svg>
  )
}

function RefreshIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
    </svg>
  )
}
