import { severityToDisplay, severityToColorClass, severityToTextClass } from '@/lib/severity'

interface SeverityBadgeProps {
  severity: number
  isNew?: boolean
}

export function SeverityBadge({ severity, isNew = false }: SeverityBadgeProps) {
  const display = severityToDisplay(severity)
  const colorClass = severityToColorClass(severity)
  const textClass = severityToTextClass(severity)

  return (
    <div className="relative flex-shrink-0 w-10 h-10">
      {isNew && (
        <div
          className={`absolute inset-0 rounded-lg ${colorClass} animate-ping opacity-75`}
        />
      )}
      <div
        className={`relative w-full h-full rounded-lg flex items-center justify-center font-bold text-sm ${colorClass} ${textClass}`}
      >
        {display}
      </div>
    </div>
  )
}
