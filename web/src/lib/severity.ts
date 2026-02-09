/**
 * Maps normalized severity values (10-100) stored in Firestore EventRecord
 * to Japanese seismic intensity display strings and color classes.
 * NOTE: These are NOT the raw P2P scale values (10-70). See ScaleToSeverity in Go backend.
 */

const SEVERITY_MAP: Record<number, { display: string; colorClass: string }> = {
  10:  { display: '1',  colorClass: 'bg-earthquake-1' },
  20:  { display: '2',  colorClass: 'bg-earthquake-2' },
  30:  { display: '3',  colorClass: 'bg-earthquake-3' },
  40:  { display: '4',  colorClass: 'bg-earthquake-4' },
  50:  { display: '5弱', colorClass: 'bg-earthquake-5weak' },
  60:  { display: '5強', colorClass: 'bg-earthquake-5strong' },
  70:  { display: '6弱', colorClass: 'bg-earthquake-6weak' },
  80:  { display: '6強', colorClass: 'bg-earthquake-6strong' },
  100: { display: '7',  colorClass: 'bg-earthquake-7' },
}

export function severityToDisplay(severity: number): string {
  return SEVERITY_MAP[severity]?.display ?? String(severity)
}

export function severityToColorClass(severity: number): string {
  return SEVERITY_MAP[severity]?.colorClass ?? 'bg-gray-200'
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
