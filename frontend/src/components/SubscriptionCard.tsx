import type { Subscription } from '@/lib/api'

interface SubscriptionCardProps {
  subscription: Subscription
  onEdit: () => void
  onDelete: () => void
}

export function SubscriptionCard({
  subscription,
  onEdit,
  onDelete,
}: SubscriptionCardProps) {
  return (
    <div className="card hover:shadow-md transition-shadow">
      <div className="flex flex-col sm:flex-row sm:justify-between sm:items-start gap-3">
        <div className="flex-1">
          <h3 className="text-lg font-semibold text-gray-900">
            {subscription.name}
          </h3>
          <p className="text-sm text-gray-500 font-mono mt-1 truncate">
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
        <div className="flex space-x-2 self-end sm:self-start sm:ml-4 shrink-0">
          <button
            onClick={onEdit}
            className="p-2 text-gray-400 hover:text-gray-600 hover:bg-gray-100 rounded-lg transition-colors"
            title="編集"
            aria-label="編集"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
              />
            </svg>
          </button>
          <button
            onClick={onDelete}
            className="p-2 text-gray-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-colors"
            title="削除"
            aria-label="削除"
          >
            <svg
              className="w-5 h-5"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              aria-hidden="true"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
              />
            </svg>
          </button>
        </div>
      </div>
    </div>
  )
}

/**
 * Maps raw P2P seismic scale values (10-70) used in subscription filter settings
 * to display strings. These differ from normalized severity values (10-100) used in events.
 */
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
