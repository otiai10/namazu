import { severityToDisplay, severityToColorClass, severityToTextClass } from '@/lib/severity'

interface SeverityBadgeProps {
  severity: number
}

export function SeverityBadge({ severity }: SeverityBadgeProps) {
  const display = severityToDisplay(severity)
  const colorClass = severityToColorClass(severity)
  const textClass = severityToTextClass(severity)

  return (
    <div
      className={`flex-shrink-0 w-10 h-10 rounded-lg flex items-center justify-center font-bold text-sm ${colorClass} ${textClass}`}
    >
      {display}
    </div>
  )
}
