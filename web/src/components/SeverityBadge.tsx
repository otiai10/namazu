import { severityToDisplay, severityToColorClass } from '@/lib/severity'

interface SeverityBadgeProps {
  severity: number
}

export function SeverityBadge({ severity }: SeverityBadgeProps) {
  const display = severityToDisplay(severity)
  const colorClass = severityToColorClass(severity)

  return (
    <div
      className={`flex-shrink-0 w-10 h-10 rounded-lg flex items-center justify-center text-white font-bold text-sm ${colorClass}`}
    >
      {display}
    </div>
  )
}
