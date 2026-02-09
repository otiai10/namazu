/**
 * Maps normalized severity values (10-100) stored in Firestore EventRecord
 * to Japanese seismic intensity display strings and color classes.
 * NOTE: These are NOT the raw P2P scale values (10-70). See ScaleToSeverity in Go backend.
 */

const SEVERITY_MAP: Record<number, { display: string; colorClass: string; textClass: string }> = {
  10:  { display: '1',  colorClass: 'bg-earthquake-1', textClass: 'text-orange-900' },
  20:  { display: '2',  colorClass: 'bg-earthquake-2', textClass: 'text-orange-900' },
  30:  { display: '3',  colorClass: 'bg-earthquake-3', textClass: 'text-white' },
  40:  { display: '4',  colorClass: 'bg-earthquake-4', textClass: 'text-white' },
  50:  { display: '5弱', colorClass: 'bg-earthquake-5weak', textClass: 'text-white' },
  60:  { display: '5強', colorClass: 'bg-earthquake-5strong', textClass: 'text-white' },
  70:  { display: '6弱', colorClass: 'bg-earthquake-6weak', textClass: 'text-white' },
  80:  { display: '6強', colorClass: 'bg-earthquake-6strong', textClass: 'text-white' },
  100: { display: '7',  colorClass: 'bg-earthquake-7', textClass: 'text-white' },
}

export function severityToDisplay(severity: number): string {
  return SEVERITY_MAP[severity]?.display ?? String(severity)
}

export function severityToColorClass(severity: number): string {
  return SEVERITY_MAP[severity]?.colorClass ?? 'bg-gray-200'
}

export function severityToTextClass(severity: number): string {
  return SEVERITY_MAP[severity]?.textClass ?? 'text-gray-700'
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
