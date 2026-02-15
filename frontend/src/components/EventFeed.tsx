import { useState, useEffect, useRef } from 'react'
import { useEvents } from '@/hooks/useEvents'
import { EventCard } from './EventCard'
import { LoadingSpinner } from './LoadingSpinner'

const NEW_EVENT_DURATION_MS = 10_000

export function EventFeed() {
  const { events, isLoading, error } = useEvents()
  const [newEventIds, setNewEventIds] = useState<Set<string>>(new Set())
  const knownIdsRef = useRef<Set<string> | null>(null)

  useEffect(() => {
    if (isLoading || events.length === 0) return

    const currentIds = new Set(events.map((e) => e.id))

    if (knownIdsRef.current === null) {
      knownIdsRef.current = currentIds
      return
    }

    const freshIds = new Set<string>()
    for (const id of currentIds) {
      if (!knownIdsRef.current.has(id)) {
        freshIds.add(id)
      }
    }

    knownIdsRef.current = currentIds

    if (freshIds.size === 0) return

    setNewEventIds((prev) => new Set([...prev, ...freshIds]))

    const timer = setTimeout(() => {
      setNewEventIds((prev) => {
        const next = new Set(prev)
        for (const id of freshIds) {
          next.delete(id)
        }
        return next
      })
    }, NEW_EVENT_DURATION_MS)

    return () => clearTimeout(timer)
  }, [events, isLoading])

  return (
    <div className="card p-0 overflow-hidden">
      <div className="px-4 py-2 border-b border-gray-200 bg-gray-50 rounded-t-xl flex items-center gap-2">
        <h2 className="text-base font-semibold text-gray-900">地震速報</h2>
        <span className="text-xs text-gray-400">- リアルタイム更新</span>
      </div>

      <div className="max-h-[400px] lg:max-h-[600px] overflow-y-auto divide-y divide-gray-100">
        {isLoading ? (
          <LoadingSpinner />
        ) : error ? (
          <div className="p-6 text-center text-red-600 text-sm">{error}</div>
        ) : events.length === 0 ? (
          <div className="p-6 text-center text-gray-400 text-sm">
            地震情報はまだありません
          </div>
        ) : (
          events.map((event) => (
            <EventCard
              key={event.id}
              event={event}
              isNew={newEventIds.has(event.id)}
            />
          ))
        )}
      </div>
    </div>
  )
}
