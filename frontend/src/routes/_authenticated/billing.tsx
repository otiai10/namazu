import { createFileRoute, Link } from '@tanstack/react-router'
import { useState, useEffect, useCallback } from 'react'
import { api, BillingStatus } from '@/lib/api'

export const Route = createFileRoute('/_authenticated/billing')({
  component: BillingPage,
})

function BillingPage() {
  const [billingStatus, setBillingStatus] = useState<BillingStatus | null>(null)
  const [isLoadingStatus, setIsLoadingStatus] = useState(false)
  const [isProcessing, setIsProcessing] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const fetchBillingStatus = useCallback(async () => {
    setIsLoadingStatus(true)
    setError(null)
    try {
      const status = await api.getBillingStatus()
      setBillingStatus(status)
    } catch (err) {
      console.error('Failed to fetch billing status:', err)
      setError('課金ステータスの取得に失敗しました。')
    } finally {
      setIsLoadingStatus(false)
    }
  }, [])

  useEffect(() => {
    fetchBillingStatus()
  }, [fetchBillingStatus])

  const handleUpgrade = async () => {
    setIsProcessing(true)
    setError(null)
    try {
      const session = await api.createCheckoutSession()
      // Redirect to Stripe Checkout
      window.location.href = session.sessionUrl
    } catch (err) {
      console.error('Failed to create checkout session:', err)
      setError('チェックアウトセッションの作成に失敗しました。')
      setIsProcessing(false)
    }
  }

  const handleManageSubscription = async () => {
    setIsProcessing(true)
    setError(null)
    try {
      const returnUrl = window.location.href
      const portal = await api.getPortalSession(returnUrl)
      // Redirect to Stripe Customer Portal
      window.location.href = portal.url
    } catch (err) {
      console.error('Failed to get portal session:', err)
      setError('カスタマーポータルへのアクセスに失敗しました。')
      setIsProcessing(false)
    }
  }

  const isPro = billingStatus?.plan === 'pro' && billingStatus?.hasActiveSubscription
  const isFree = !isPro

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">課金・プラン管理</h1>
        <p className="text-gray-600">サブスクリプションプランの変更</p>
      </div>

      {/* Error Message */}
      {error && (
        <div className="card bg-red-50 border-red-200">
          <div className="flex items-center space-x-3">
            <svg
              className="w-5 h-5 text-red-500"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
              />
            </svg>
            <p className="text-red-700">{error}</p>
          </div>
        </div>
      )}

      {/* Current Subscription Status */}
      {isLoadingStatus ? (
        <div className="card">
          <div className="flex items-center justify-center py-8">
            <div className="w-6 h-6 border-2 border-primary-200 border-t-primary-600 rounded-full animate-spin" />
            <span className="ml-3 text-gray-600">ステータスを読み込み中...</span>
          </div>
        </div>
      ) : billingStatus && isPro ? (
        <div className="card bg-gradient-to-r from-primary-50 to-blue-50 border-primary-200">
          <div className="flex items-start space-x-4">
            <div className="flex-shrink-0">
              <div className="w-12 h-12 bg-primary-100 rounded-full flex items-center justify-center">
                <svg
                  className="w-6 h-6 text-primary-600"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </div>
            </div>
            <div className="flex-1">
              <h3 className="text-lg font-semibold text-gray-900">
                Pro プラン利用中
              </h3>
              <p className="text-gray-600 mt-1">
                ステータス: {formatSubscriptionStatus(billingStatus.subscriptionStatus)}
              </p>
              {billingStatus.subscriptionEndsAt && (
                <p className="text-gray-600 mt-1">
                  次回請求日: {formatDate(billingStatus.subscriptionEndsAt)}
                </p>
              )}
            </div>
          </div>
        </div>
      ) : null}

      {/* Plan Details */}
      <div className="grid md:grid-cols-2 gap-6">
        {/* Free Plan */}
        <div className={`card border-2 ${isFree ? 'border-primary-500 bg-primary-50/30' : 'border-gray-200'}`}>
          <div className="flex justify-between items-start mb-4">
            <div>
              <h3 className="text-xl font-bold text-gray-900">Free</h3>
              <p className="text-3xl font-bold text-gray-900 mt-1">¥0</p>
            </div>
            {isFree && (
              <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800">
                現在のプラン
              </span>
            )}
          </div>
          <ul className="space-y-3 mb-6">
            <PlanFeature included>サブスクリプション 1 個</PlanFeature>
            <PlanFeature included>全ての地震速報を受信</PlanFeature>
            <PlanFeature included>震度・地域フィルタ</PlanFeature>
            <PlanFeature included>自動リトライ</PlanFeature>
            <PlanFeature included>HMAC 署名付き配信</PlanFeature>
          </ul>
          <button
            disabled
            className="w-full btn bg-gray-100 text-gray-500 cursor-not-allowed"
          >
            {isFree ? '現在のプラン' : 'Free プラン'}
          </button>
        </div>

        {/* Pro Plan */}
        <div className={`card border-2 ${isPro ? 'border-primary-500 bg-primary-50/30' : 'border-gray-200'}`}>
          <div className="flex justify-between items-start mb-4">
            <div>
              <h3 className="text-xl font-bold text-gray-900">Pro</h3>
              <p className="text-3xl font-bold text-gray-900 mt-1">
                ¥500<span className="text-base font-normal text-gray-500">/月</span>
              </p>
            </div>
            {isPro ? (
              <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800">
                現在のプラン
              </span>
            ) : (
              <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-primary-100 text-primary-800">
                おすすめ
              </span>
            )}
          </div>
          <ul className="space-y-3 mb-6">
            <PlanFeature included>サブスクリプション 12 個</PlanFeature>
            <PlanFeature included>全ての地震速報を受信</PlanFeature>
            <PlanFeature included>震度・地域フィルタ</PlanFeature>
            <PlanFeature included>自動リトライ</PlanFeature>
            <PlanFeature included>HMAC 署名付き配信</PlanFeature>
            <PlanFeature included>優先サポート</PlanFeature>
          </ul>
          {isPro ? (
            <button
              onClick={handleManageSubscription}
              disabled={isProcessing}
              className="w-full btn btn-secondary flex items-center justify-center"
            >
              {isProcessing ? (
                <>
                  <div className="w-4 h-4 border-2 border-gray-300 border-t-gray-600 rounded-full animate-spin mr-2" />
                  処理中...
                </>
              ) : (
                'サブスクリプションを管理'
              )}
            </button>
          ) : (
            <button
              onClick={handleUpgrade}
              disabled={isProcessing || isLoadingStatus}
              className="w-full btn btn-primary flex items-center justify-center"
            >
              {isProcessing ? (
                <>
                  <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin mr-2" />
                  処理中...
                </>
              ) : (
                'Pro にアップグレード'
              )}
            </button>
          )}
        </div>
      </div>

      {/* FAQ */}
      <div className="card">
        <h2 className="text-lg font-semibold text-gray-900 mb-4">よくある質問</h2>
        <div className="space-y-4">
          <FaqItem
            question="Free プランでも全ての地震速報を受信できますか？"
            answer="はい、Free プランでも Pro プランと同じ地震速報データを受信できます。違いは登録可能なサブスクリプション数のみです。"
          />
          <FaqItem
            question="Pro プランのキャンセルはいつでもできますか？"
            answer="はい、Pro プランはいつでもキャンセル可能です。キャンセル後も期間終了まではご利用いただけます。"
          />
          <FaqItem
            question="支払い方法は何が使えますか？"
            answer="Stripe を通じて、主要なクレジットカード（Visa、Mastercard、American Express、JCB）をご利用いただけます。"
          />
        </div>
      </div>

      {/* Back to Settings */}
      <div className="text-center">
        <Link to="/settings" className="text-primary-600 hover:text-primary-700">
          設定に戻る
        </Link>
      </div>
    </div>
  )
}

function formatSubscriptionStatus(status?: string): string {
  switch (status) {
    case 'active':
      return '有効'
    case 'trialing':
      return 'トライアル中'
    case 'past_due':
      return '支払い遅延'
    case 'canceled':
      return 'キャンセル済み'
    case 'unpaid':
      return '未払い'
    default:
      return status || '不明'
  }
}

function formatDate(dateString: string): string {
  try {
    const date = new Date(dateString)
    return date.toLocaleDateString('ja-JP', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    })
  } catch {
    return dateString
  }
}

function PlanFeature({
  children,
  included,
}: {
  children: React.ReactNode
  included?: boolean
}) {
  return (
    <li className="flex items-center text-sm">
      {included ? (
        <svg
          className="w-5 h-5 mr-2 text-green-500"
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
      ) : (
        <svg
          className="w-5 h-5 mr-2 text-gray-300"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M6 18L18 6M6 6l12 12"
          />
        </svg>
      )}
      <span className={included ? 'text-gray-700' : 'text-gray-400'}>
        {children}
      </span>
    </li>
  )
}

function FaqItem({ question, answer }: { question: string; answer: string }) {
  return (
    <div className="border-b border-gray-100 pb-4 last:border-0 last:pb-0">
      <h4 className="font-medium text-gray-900">{question}</h4>
      <p className="text-sm text-gray-600 mt-1">{answer}</p>
    </div>
  )
}
