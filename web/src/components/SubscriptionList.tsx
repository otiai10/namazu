import type { Subscription } from '@/lib/api'
import { SubscriptionCard } from './SubscriptionCard'
import { LoadingSpinner } from './LoadingSpinner'

interface SubscriptionListProps {
  subscriptions: Subscription[]
  isLoading: boolean
  onEdit: (sub: Subscription) => void
  onDelete: (id: string) => Promise<void>
  onCreateNew: () => void
}

export function SubscriptionList({
  subscriptions,
  isLoading,
  onEdit,
  onDelete,
  onCreateNew,
}: SubscriptionListProps) {
  if (isLoading) {
    return <LoadingSpinner />
  }

  if (subscriptions.length === 0) {
    return (
      <div className="card text-center py-12">
        <div className="text-gray-400 mb-4">
          <svg
            className="w-16 h-16 mx-auto"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={1}
              d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4"
            />
          </svg>
        </div>
        <h3 className="text-lg font-medium text-gray-900 mb-2">
          サブスクリプションがありません
        </h3>
        <p className="text-gray-600 mb-4">
          最初のサブスクリプションを作成しましょう。
        </p>
        <button onClick={onCreateNew} className="btn btn-primary">
          + 新規作成
        </button>
      </div>
    )
  }

  return (
    <div className="grid gap-4">
      {subscriptions.map((sub) => (
        <SubscriptionCard
          key={sub.id}
          subscription={sub}
          onEdit={() => onEdit(sub)}
          onDelete={() => onDelete(sub.id)}
        />
      ))}
    </div>
  )
}
