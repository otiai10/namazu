import { createFileRoute, Link } from '@tanstack/react-router'
import { useAuth } from '@/hooks/useAuth'

export const Route = createFileRoute('/billing')({
  component: BillingPage,
})

function BillingPage() {
  const { isAuthenticated, isLoading: authLoading } = useAuth()

  if (authLoading) {
    return <LoadingSpinner />
  }

  if (!isAuthenticated) {
    return (
      <div className="card text-center py-12">
        <h2 className="text-xl font-semibold text-gray-900 mb-4">
          ログインが必要です
        </h2>
        <p className="text-gray-600 mb-6">
          課金設定を確認するにはログインしてください。
        </p>
        <Link to="/" className="btn btn-primary">
          ホームへ戻る
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-6 max-w-2xl">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">課金・プラン管理</h1>
        <p className="text-gray-600">サブスクリプションプランの変更</p>
      </div>

      {/* Coming Soon Notice */}
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
                  d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z"
                />
              </svg>
            </div>
          </div>
          <div>
            <h3 className="text-lg font-semibold text-gray-900">
              Pro プラン - 近日公開予定
            </h3>
            <p className="text-gray-600 mt-1">
              Stripe を使用した課金システムを準備中です。
              Pro プランでは、より多くのサブスクリプションを登録できるようになります。
            </p>
          </div>
        </div>
      </div>

      {/* Plan Details */}
      <div className="grid md:grid-cols-2 gap-6">
        {/* Free Plan */}
        <div className="card border-2 border-gray-200">
          <div className="flex justify-between items-start mb-4">
            <div>
              <h3 className="text-xl font-bold text-gray-900">Free</h3>
              <p className="text-3xl font-bold text-gray-900 mt-1">¥0</p>
            </div>
            <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800">
              現在のプラン
            </span>
          </div>
          <ul className="space-y-3 mb-6">
            <PlanFeature included>サブスクリプション 3 個</PlanFeature>
            <PlanFeature included>全ての地震速報を受信</PlanFeature>
            <PlanFeature included>震度・地域フィルタ</PlanFeature>
            <PlanFeature included>自動リトライ</PlanFeature>
            <PlanFeature included>HMAC 署名付き配信</PlanFeature>
          </ul>
          <button
            disabled
            className="w-full btn bg-gray-100 text-gray-500 cursor-not-allowed"
          >
            現在のプラン
          </button>
        </div>

        {/* Pro Plan */}
        <div className="card border-2 border-primary-500 bg-primary-50/30">
          <div className="flex justify-between items-start mb-4">
            <div>
              <h3 className="text-xl font-bold text-gray-900">Pro</h3>
              <p className="text-3xl font-bold text-gray-900 mt-1">
                ¥500<span className="text-base font-normal text-gray-500">/月</span>
              </p>
            </div>
            <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-primary-100 text-primary-800">
              おすすめ
            </span>
          </div>
          <ul className="space-y-3 mb-6">
            <PlanFeature included>サブスクリプション 50 個</PlanFeature>
            <PlanFeature included>全ての地震速報を受信</PlanFeature>
            <PlanFeature included>震度・地域フィルタ</PlanFeature>
            <PlanFeature included>自動リトライ</PlanFeature>
            <PlanFeature included>HMAC 署名付き配信</PlanFeature>
            <PlanFeature included>優先サポート</PlanFeature>
          </ul>
          <button
            disabled
            className="w-full btn btn-primary opacity-50 cursor-not-allowed"
          >
            近日公開
          </button>
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
          ← 設定に戻る
        </Link>
      </div>
    </div>
  )
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

function LoadingSpinner() {
  return (
    <div className="flex justify-center py-12">
      <div className="w-8 h-8 border-4 border-primary-200 border-t-primary-600 rounded-full animate-spin" />
    </div>
  )
}
