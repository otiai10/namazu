import { useState } from 'react'
import { api, type Subscription, type CreateSubscriptionInput, type CreateSubscriptionResponse } from '@/lib/api'
import { SecretDisplay } from './SecretDisplay'

interface SubscriptionFormProps {
  subscription?: Subscription
  onClose: () => void
  onSuccess: () => void
}

export function SubscriptionForm({
  subscription,
  onClose,
  onSuccess,
}: SubscriptionFormProps) {
  const isEditing = !!subscription
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [createdSecret, setCreatedSecret] = useState<string | null>(null)

  const [name, setName] = useState(subscription?.name || '')
  const [url, setUrl] = useState(subscription?.delivery.url || '')
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
      },
      filter:
        minScale > 0 || prefectures.trim()
          ? {
              min_scale: minScale > 0 ? minScale : undefined,
              prefectures: prefectures.trim()
                ? prefectures
                    .split(',')
                    .map((p) => p.trim())
                    .filter(Boolean)
                : undefined,
            }
          : undefined,
    }

    try {
      if (isEditing && subscription) {
        await api.updateSubscription(subscription.id, input)
        onSuccess()
      } else {
        const response: CreateSubscriptionResponse = await api.createSubscription(input)
        setCreatedSecret(response.delivery.secret)
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存に失敗しました')
    } finally {
      setIsSubmitting(false)
    }
  }

  function handleSecretDismiss() {
    setCreatedSecret(null)
    onSuccess()
  }

  if (createdSecret) {
    return <SecretDisplay secret={createdSecret} onDismiss={handleSecretDismiss} />
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
          <svg
            className="w-5 h-5"
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

        <div className="border-t border-gray-200 pt-4">
          <h3 className="text-sm font-medium text-gray-900 mb-3">
            フィルタ設定 (オプション)
          </h3>

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
