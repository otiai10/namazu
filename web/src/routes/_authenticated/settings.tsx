import { createFileRoute, Link } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { useAuth } from '@/hooks/useAuth'
import { api, type UserProfile } from '@/lib/api'
import { LoadingSpinner } from '@/components/LoadingSpinner'

export const Route = createFileRoute('/_authenticated/settings')({
  component: SettingsPage,
})

function SettingsPage() {
  const { user } = useAuth()
  const [profile, setProfile] = useState<UserProfile | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    loadProfile()
  }, [])

  async function loadProfile() {
    try {
      setIsLoading(true)
      setError(null)
      const data = await api.getProfile()
      setProfile(data)
    } catch (err) {
      // If 404, user profile doesn't exist yet (will be created on first API call)
      if (err instanceof Error && err.message.includes('404')) {
        setProfile(null)
      } else {
        setError(err instanceof Error ? err.message : '読み込みに失敗しました')
      }
    } finally {
      setIsLoading(false)
    }
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">設定</h1>
        <p className="text-gray-600">アカウント情報とプラン設定</p>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {error}
        </div>
      )}

      {/* Profile Section */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">プロファイル</h2>

        {isLoading ? (
          <LoadingSpinner />
        ) : (
          <div className="space-y-4">
            <div className="flex items-center space-x-4">
              {user?.photoURL && (
                <img
                  src={user.photoURL}
                  alt=""
                  className="w-16 h-16 rounded-full"
                />
              )}
              <div>
                <p className="text-lg font-medium text-gray-900">
                  {user?.displayName || 'ユーザー'}
                </p>
                <p className="text-gray-500">{user?.email}</p>
              </div>
            </div>

            <div className="border-t border-gray-200 pt-4 space-y-3">
              <InfoRow label="ユーザー ID" value={profile?.id || user?.uid || '-'} />
              <InfoRow
                label="登録日"
                value={profile?.createdAt ? formatDate(profile.createdAt) : '-'}
              />
              <InfoRow
                label="最終更新"
                value={profile?.updatedAt ? formatDate(profile.updatedAt) : '-'}
              />
            </div>
          </div>
        )}
      </div>

      {/* Plan Section */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">プラン</h2>

        <div className="flex items-center justify-between">
          <div>
            <div className="flex items-center space-x-2">
              <span className="text-lg font-medium text-gray-900">
                {profile?.plan === 'pro' ? 'Pro' : 'Free'}
              </span>
              {profile?.plan === 'pro' ? (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-primary-100 text-primary-800">
                  Pro
                </span>
              ) : (
                <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-gray-100 text-gray-800">
                  Free
                </span>
              )}
            </div>
            <p className="text-sm text-gray-500 mt-1">
              {profile?.plan === 'pro'
                ? 'サブスクリプション 50 個まで'
                : 'サブスクリプション 3 個まで'}
            </p>
          </div>

          {profile?.plan !== 'pro' && (
            <Link to="/billing" className="btn btn-primary">
              Pro にアップグレード
            </Link>
          )}
        </div>

        {/* Plan comparison */}
        <div className="mt-6 border-t border-gray-200 pt-6">
          <h3 className="text-sm font-medium text-gray-900 mb-4">プラン比較</h3>
          <div className="grid md:grid-cols-2 gap-4">
            <PlanCard
              name="Free"
              price="¥0"
              features={[
                'サブスクリプション 3 個',
                '全ての地震速報',
                'フィルタ機能',
                'リトライ機能',
              ]}
              current={profile?.plan !== 'pro'}
            />
            <PlanCard
              name="Pro"
              price="¥500/月"
              features={[
                'サブスクリプション 50 個',
                '全ての地震速報',
                'フィルタ機能',
                'リトライ機能',
                '優先サポート',
              ]}
              current={profile?.plan === 'pro'}
              highlighted
            />
          </div>
        </div>
      </div>

      {/* Linked Providers Section */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">
          連携済みアカウント
        </h2>

        <div className="space-y-3">
          {user?.providerData.map((provider) => (
            <div
              key={provider.providerId}
              className="flex items-center justify-between p-3 bg-gray-50 rounded-lg"
            >
              <div className="flex items-center space-x-3">
                <ProviderIcon providerId={provider.providerId} />
                <div>
                  <p className="font-medium text-gray-900">
                    {getProviderName(provider.providerId)}
                  </p>
                  <p className="text-sm text-gray-500">{provider.email}</p>
                </div>
              </div>
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                連携済み
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}

function InfoRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-gray-500">{label}</span>
      <span className="text-gray-900 font-mono text-sm">{value}</span>
    </div>
  )
}

function PlanCard({
  name,
  price,
  features,
  current,
  highlighted,
}: {
  name: string
  price: string
  features: string[]
  current?: boolean
  highlighted?: boolean
}) {
  return (
    <div
      className={`rounded-lg border-2 p-4 ${
        highlighted
          ? 'border-primary-500 bg-primary-50'
          : 'border-gray-200 bg-white'
      }`}
    >
      <div className="flex justify-between items-start mb-3">
        <div>
          <h4 className="font-semibold text-gray-900">{name}</h4>
          <p className="text-2xl font-bold text-gray-900">{price}</p>
        </div>
        {current && (
          <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
            現在のプラン
          </span>
        )}
      </div>
      <ul className="space-y-2">
        {features.map((feature, i) => (
          <li key={i} className="flex items-center text-sm text-gray-600">
            <svg
              className="w-4 h-4 mr-2 text-green-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M5 13l4 4L19 7"
              />
            </svg>
            {feature}
          </li>
        ))}
      </ul>
    </div>
  )
}

function ProviderIcon({ providerId }: { providerId: string }) {
  if (providerId === 'google.com') {
    return (
      <div className="w-8 h-8 bg-white rounded-full flex items-center justify-center shadow-sm">
        <svg className="w-5 h-5" viewBox="0 0 24 24">
          <path
            fill="#4285F4"
            d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
          />
          <path
            fill="#34A853"
            d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
          />
          <path
            fill="#FBBC05"
            d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
          />
          <path
            fill="#EA4335"
            d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
          />
        </svg>
      </div>
    )
  }
  return (
    <div className="w-8 h-8 bg-gray-200 rounded-full flex items-center justify-center">
      <svg className="w-4 h-4 text-gray-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
      </svg>
    </div>
  )
}

function getProviderName(providerId: string): string {
  switch (providerId) {
    case 'google.com':
      return 'Google'
    case 'apple.com':
      return 'Apple'
    case 'password':
      return 'メール/パスワード'
    default:
      return providerId
  }
}

function formatDate(dateString: string): string {
  return new Date(dateString).toLocaleDateString('ja-JP', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

