import { SeverityBadge } from './SeverityBadge'
import { formatRelativeTime } from '@/lib/severity'
import type { EarthquakeEvent } from '@/hooks/useEvents'

interface EventCardProps {
  event: EarthquakeEvent
}

export function EventCard({ event }: EventCardProps) {
  return (
    <div className="px-4 py-3 hover:bg-gray-50 transition-colors">
      <div className="flex items-start space-x-3">
        <SeverityBadge severity={event.severity} />
        <div className="flex-1 min-w-0">
          <p className="text-sm font-medium text-gray-900 truncate">
            {event.type === 'earthquake' ? '地震情報' : event.type}
          </p>
          {event.affectedAreas.length > 0 && (
            <p className="text-xs text-gray-500 truncate mt-0.5">
              {event.affectedAreas.slice(0, 3).join(', ')}
              {event.affectedAreas.length > 3 &&
                ` 他${event.affectedAreas.length - 3}地域`}
            </p>
          )}
          <p className="text-xs text-gray-400 mt-1">
            {formatRelativeTime(event.occurredAt)}
          </p>
        </div>
      </div>
    </div>
  )
}
