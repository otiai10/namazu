import { createFileRoute, Link } from '@tanstack/react-router'
import { useState, useEffect } from 'react'
import { useAuth } from '@/hooks/useAuth'
import { api, type Subscription, type CreateSubscriptionInput } from '@/lib/api'

export const Route = createFileRoute('/subscriptions')({
  component: SubscriptionsPage,
})

function SubscriptionsPage() {
  const { isAuthenticated, isLoading: authLoading } = useAuth()
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)

  useEffect(() => {
    if (isAuthenticated) {
      loadSubscriptions()
    }
  }, [isAuthenticated])

  async function loadSubscriptions() {
    try {
      setIsLoading(true)
      setError(null)
      const data = await api.listSubscriptions()
      setSubscriptions(data || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '読み込みに失敗しました')
    } finally {
      setIsLoading(false)
    }
  }

  async function handleDelete(id: string) {
    if (!confirm('このサブスクリプションを削除しますか？')) return
    try {
      await api.deleteSubscription(id)
      setSubscriptions((prev) => prev.filter((s) => s.id !== id))
    } catch (err) {
      setError(err instanceof Error ? err.message : '削除に失敗しました')
    }
  }

  function handleEdit(sub: Subscription) {
    setEditingId(sub.id)
    setShowForm(true)
  }

  function handleFormClose() {
    setShowForm(false)
    setEditingId(null)
  }

  function handleFormSuccess() {
    handleFormClose()
    loadSubscriptions()
  }

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
          サブスクリプションを管理するにはログインしてください。
        </p>
        <Link to="/" className="btn btn-primary">
          ホームへ戻る
        </Link>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex justify-between items-center">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Subscriptions</h1>
          <p className="text-gray-600">Webhook 配信先を管理します</p>
        </div>
        <button
          onClick={() => setShowForm(true)}
          className="btn btn-primary"
        >
          + 新規作成
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {error}
        </div>
      )}

      {showForm && (
        <SubscriptionForm
          subscription={editingId ? subscriptions.find((s) => s.id === editingId) : undefined}
          onClose={handleFormClose}
          onSuccess={handleFormSuccess}
        />
      )}

      {isLoading ? (
        <LoadingSpinner />
      ) : subscriptions.length === 0 ? (
        <div className="card text-center py-12">
          <div className="text-gray-400 mb-4">
            <svg className="w-16 h-16 mx-auto" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
            </svg>
          </div>
          <h3 className="text-lg font-medium text-gray-900 mb-2">
            サブスクリプションがありません
          </h3>
          <p className="text-gray-600 mb-4">
            最初のサブスクリプションを作成しましょう。
          </p>
          <button onClick={() => setShowForm(true)} className="btn btn-primary">
            + 新規作成
          </button>
        </div>
      ) : (
        <div className="grid gap-4">
          {subscriptions.map((sub) => (
            <SubscriptionCard
              key={sub.id}
              subscription={sub}
              onEdit={() => handleEdit(sub)}
              onDelete={() => handleDelete(sub.id)}
            />
          ))}
        </div>
      )}
    </div>
  )
}

function SubscriptionCard({
  subscription,
  onEdit,
  onDelete,
}: {
  subscription: Subscription
  onEdit: () => void
  onDelete: () => void
}) {
  return (
    <div className="card hover:shadow-md transition-shadow">
      <div className="flex justify-between items-start">
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-gray-900">{subscription.name}</h3>
          <p className="text-sm text-gray-500 font-mono mt-1 truncate max-w-lg">
            {subscription.delivery.url}
          </p>
          <div className="flex flex-wrap gap-2 mt-3">
            {subscription.filter?.min_scale && (
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-yellow-100 text-yellow-800">
                震度 {scaleToDisplay(subscription.filter.min_scale)} 以上
              </span>
            )}
            {subscription.filter?.prefectures?.map((pref) => (
              <span
                key={pref}
                className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800"
              >
                {pref}
              </span>
            ))}
            {subscription.delivery.retry?.enabled && (
              <span className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                リトライ有効
              </span>
            )}
          </div>
        </div>
        <div className="flex space-x-2 ml-4">
          <button
            onClick={onEdit}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
            title="編集"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
            </svg>
          </button>
          <button
            onClick={onDelete}
            className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
            title="削除"
          >
            <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  )
}

function SubscriptionForm({
  subscription,
  onClose,
  onSuccess,
}: {
  subscription?: Subscription
  onClose: () => void
  onSuccess: () => void
}) {
  const isEditing = !!subscription
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const [name, setName] = useState(subscription?.name || '')
  const [url, setUrl] = useState(subscription?.delivery.url || '')
  const [secret, setSecret] = useState(subscription?.delivery.secret || '')
  const [minScale, setMinScale] = useState(subscription?.filter?.min_scale || 0)
  const [prefectures, setPrefectures] = useState(
    subscription?.filter?.prefectures?.join(', ') || ''
  )

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError(null)
    setIsSubmitting(true)

    const input: CreateSubscriptionInput = {
      name,
      delivery: {
        type: 'webhook',
        url,
        secret,
      },
      filter: minScale > 0 || prefectures.trim()
        ? {
            min_scale: minScale > 0 ? minScale : undefined,
            prefectures: prefectures.trim()
              ? prefectures.split(',').map((p) => p.trim()).filter(Boolean)
              : undefined,
          }
        : undefined,
    }

    try {
      if (isEditing && subscription) {
        await api.updateSubscription(subscription.id, input)
      } else {
        await api.createSubscription(input)
      }
      onSuccess()
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存に失敗しました')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="card">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-xl font-semibold text-gray-900">
          {isEditing ? 'サブスクリプション編集' : '新規サブスクリプション'}
        </h2>
        <button
          onClick={onClose}
          className="p-2 text-gray-400 hover:text-gray-600 rounded-lg"
        >
          <svg className="w-5 h-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700 mb-4">
          {error}
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-4">
        <div>
          <label className="label">名前</label>
          <input
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            className="input"
            placeholder="My Webhook"
            required
          />
        </div>

        <div>
          <label className="label">Webhook URL</label>
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            className="input font-mono text-sm"
            placeholder="https://example.com/webhook"
            required
          />
        </div>

        <div>
          <label className="label">シークレット (HMAC 署名用)</label>
          <input
            type="text"
            value={secret}
            onChange={(e) => setSecret(e.target.value)}
            className="input font-mono text-sm"
            placeholder="your-secret-key"
            required
          />
        </div>

        <div className="border-t border-gray-200 pt-4">
          <h3 className="text-sm font-medium text-gray-900 mb-3">フィルタ設定 (オプション)</h3>

          <div className="grid md:grid-cols-2 gap-4">
            <div>
              <label className="label">最小震度</label>
              <select
                value={minScale}
                onChange={(e) => setMinScale(Number(e.target.value))}
                className="input"
              >
                <option value={0}>フィルタなし</option>
                <option value={10}>震度1 以上</option>
                <option value={20}>震度2 以上</option>
                <option value={30}>震度3 以上</option>
                <option value={40}>震度4 以上</option>
                <option value={45}>震度5弱 以上</option>
                <option value={50}>震度5強 以上</option>
                <option value={55}>震度6弱 以上</option>
                <option value={60}>震度6強 以上</option>
                <option value={70}>震度7</option>
              </select>
            </div>

            <div>
              <label className="label">対象地域 (カンマ区切り)</label>
              <input
                type="text"
                value={prefectures}
                onChange={(e) => setPrefectures(e.target.value)}
                className="input"
                placeholder="東京都, 神奈川県"
              />
            </div>
          </div>
        </div>

        <div className="flex justify-end space-x-3 pt-4">
          <button type="button" onClick={onClose} className="btn btn-secondary">
            キャンセル
          </button>
          <button
            type="submit"
            disabled={isSubmitting}
            className="btn btn-primary disabled:opacity-50"
          >
            {isSubmitting ? '保存中...' : isEditing ? '更新' : '作成'}
          </button>
        </div>
      </form>
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

function scaleToDisplay(scale: number): string {
  switch (scale) {
    case 10: return '1'
    case 20: return '2'
    case 30: return '3'
    case 40: return '4'
    case 45: return '5弱'
    case 50: return '5強'
    case 55: return '6弱'
    case 60: return '6強'
    case 70: return '7'
    default: return String(scale)
  }
}
