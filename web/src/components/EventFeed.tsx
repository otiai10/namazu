import { useEvents } from '@/hooks/useEvents'
import { EventCard } from './EventCard'
import { LoadingSpinner } from './LoadingSpinner'

export function EventFeed() {
  const { events, isLoading, error } = useEvents()

  return (
    <div className="card p-0 overflow-hidden">
      <div className="px-4 py-2 border-b border-gray-200 bg-gray-50 rounded-t-xl flex items-center gap-2">
        <h2 className="text-base font-semibold text-gray-900">地震速報</h2>
        <span className="text-xs text-gray-400">- リアルタイム更新</span>
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
