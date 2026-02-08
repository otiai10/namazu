import { useEvents } from '@/hooks/useEvents'
import { EventCard } from './EventCard'
import { LoadingSpinner } from './LoadingSpinner'

export function EventFeed() {
  const { events, isLoading, error } = useEvents()

  return (
    <div className="card p-0 overflow-hidden">
      <div className="px-6 py-4 border-b border-gray-200 bg-gray-50 rounded-t-xl">
        <h2 className="text-lg font-semibold text-gray-900">地震速報</h2>
        <p className="text-sm text-gray-500">リアルタイム更新</p>
      </div>

      <div className="max-h-[600px] overflow-y-auto divide-y divide-gray-100">
        {isLoading ? (
          <LoadingSpinner />
        ) : error ? (
          <div className="p-6 text-center text-red-600 text-sm">{error}</div>
        ) : events.length === 0 ? (
          <div className="p-6 text-center text-gray-400 text-sm">
            地震情報はまだありません
          </div>
        ) : (
          events.map((event) => <EventCard key={event.id} event={event} />)
        )}
      </div>
    </div>
  )
}
