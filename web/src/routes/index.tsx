import { createFileRoute } from '@tanstack/react-router'
import { useAuth } from '@/hooks/useAuth'

export const Route = createFileRoute('/')({
  component: Dashboard,
})

function Dashboard() {
  const { isAuthenticated, user } = useAuth()

  return (
    <div className="space-y-8">
      {/* Hero section for non-authenticated users */}
      {!isAuthenticated && (
        <div className="card bg-gradient-to-r from-primary-600 to-primary-800 text-white">
          <h1 className="text-3xl font-bold mb-4">
            namazu - 地震速報 Webhook 中継サービス
          </h1>
          <p className="text-lg text-primary-100 mb-6">
            P2P地震情報から受信した地震速報を、あなたの Webhook エンドポイントに即座に配信します。
          </p>
          <div className="flex space-x-4">
            <a
              href="https://www.p2pquake.net/"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center px-4 py-2 bg-white text-primary-700 rounded-lg font-medium hover:bg-primary-50 transition-colors"
            >
              P2P地震情報について
              <ExternalLinkIcon className="ml-2 w-4 h-4" />
            </a>
          </div>
        </div>
      )}

      {/* Welcome section for authenticated users */}
      {isAuthenticated && (
        <div className="card">
          <h1 className="text-2xl font-bold text-gray-900 mb-2">
            ようこそ、{user?.displayName || 'ユーザー'}さん
          </h1>
          <p className="text-gray-600">
            地震速報の Webhook 配信を管理できます。
          </p>
        </div>
      )}

      {/* Features section */}
      <div className="grid md:grid-cols-3 gap-6">
        <FeatureCard
          title="リアルタイム配信"
          description="P2P地震情報から受信した地震速報を即座に Webhook に配信します。"
          icon={<BoltIcon className="w-8 h-8 text-yellow-500" />}
        />
        <FeatureCard
          title="フィルタリング"
          description="震度や地域でフィルタリングし、必要な情報のみを受信できます。"
          icon={<FilterIcon className="w-8 h-8 text-blue-500" />}
        />
        <FeatureCard
          title="自動リトライ"
          description="配信失敗時は指数バックオフで自動リトライします。"
          icon={<RefreshIcon className="w-8 h-8 text-green-500" />}
        />
      </div>

      {/* Earthquake scale reference */}
      <div className="card">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">震度スケール</h2>
        <div className="grid grid-cols-5 md:grid-cols-10 gap-2">
          <ScaleBadge scale={1} />
          <ScaleBadge scale={2} />
          <ScaleBadge scale={3} />
          <ScaleBadge scale={4} />
          <ScaleBadge scale="5弱" color="earthquake-5weak" />
          <ScaleBadge scale="5強" color="earthquake-5strong" />
          <ScaleBadge scale="6弱" color="earthquake-6weak" />
          <ScaleBadge scale="6強" color="earthquake-6strong" />
          <ScaleBadge scale={7} color="earthquake-7" />
        </div>
      </div>
    </div>
  )
}

function FeatureCard({
  title,
  description,
  icon,
}: {
  title: string
  description: string
  icon: React.ReactNode
}) {
  return (
    <div className="card hover:shadow-md transition-shadow">
      <div className="mb-4">{icon}</div>
      <h3 className="text-lg font-semibold text-gray-900 mb-2">{title}</h3>
      <p className="text-gray-600 text-sm">{description}</p>
    </div>
  )
}

function ScaleBadge({
  scale,
  color,
}: {
  scale: number | string
  color?: string
}) {
  const bgColor = color ? `bg-${color}` : getScaleColor(scale as number)
  return (
    <div
      className={`flex items-center justify-center h-10 rounded-lg font-bold text-white ${bgColor}`}
    >
      {scale}
    </div>
  )
}

function getScaleColor(scale: number): string {
  switch (scale) {
    case 1:
      return 'bg-sky-300'
    case 2:
      return 'bg-sky-400'
    case 3:
      return 'bg-sky-500'
    case 4:
      return 'bg-yellow-400'
    default:
      return 'bg-gray-400'
  }
}

// Simple icons
function ExternalLinkIcon({ className }: { className?: string }) {
  return (
    <svg className={className} fill="none" viewBox="0 0 24 24" stroke="currentColor">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M10 6H6a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2v-4M14 4h6m0 0v6m0-6L10 14" />
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
