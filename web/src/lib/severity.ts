/**
 * Maps normalized severity values (10-100) stored in Firestore EventRecord
 * to Japanese seismic intensity display strings.
 * NOTE: These are NOT the raw P2P scale values (10-70). See ScaleToSeverity in Go backend.
 */
export function severityToDisplay(severity: number): string {
  switch (severity) {
    case 10: return '1'
    case 20: return '2'
    case 30: return '3'
    case 40: return '4'
    case 50: return '5弱'
    case 60: return '5強'
    case 70: return '6弱'
    case 80: return '6強'
    case 100: return '7'
    default: return String(severity)
  }
}

export function severityToColorClass(severity: number): string {
  switch (severity) {
    case 10: return 'bg-earthquake-1'
    case 20: return 'bg-earthquake-2'
    case 30: return 'bg-earthquake-3'
    case 40: return 'bg-earthquake-4'
    case 50: return 'bg-earthquake-5weak'
    case 60: return 'bg-earthquake-5strong'
    case 70: return 'bg-earthquake-6weak'
    case 80: return 'bg-earthquake-6strong'
    case 100: return 'bg-earthquake-7'
    default: return 'bg-gray-400'
  }
}

export function formatRelativeTime(date: Date): string {
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMins = Math.floor(diffMs / 60000)

  if (diffMins < 1) return 'たった今'
  if (diffMins < 60) return `${diffMins}分前`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}時間前`
  const diffDays = Math.floor(diffHours / 24)
  return `${diffDays}日前`
}
