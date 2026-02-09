import { useState, useEffect } from 'react'
import {
  collection,
  query,
  orderBy,
  limit,
  onSnapshot,
  type Timestamp,
} from 'firebase/firestore'
import { db, isFirestoreConfigured } from '@/lib/firestore'
import { api } from '@/lib/api'

export interface EarthquakeEvent {
  id: string
  type: string
  source: string
  severity: number
  affectedAreas: string[]
  occurredAt: Date
  receivedAt: Date
  createdAt: Date
}

interface UseEventsResult {
  events: EarthquakeEvent[]
  isLoading: boolean
  error: string | null
}

const EVENT_LIMIT = 20

function firestoreDocToEvent(
  id: string,
  data: Record<string, unknown>
): EarthquakeEvent {
  const toDate = (field: unknown): Date => {
    if (field && typeof (field as Timestamp).toDate === 'function') {
      return (field as Timestamp).toDate()
    }
    return new Date(0)
  }

  return {
    id,
    type: typeof data.type === 'string' ? data.type : 'unknown',
    source: typeof data.source === 'string' ? data.source : 'unknown',
    severity: typeof data.severity === 'number' ? data.severity : 0,
    affectedAreas: Array.isArray(data.affectedAreas) ? data.affectedAreas : [],
    occurredAt: toDate(data.occurredAt),
    receivedAt: toDate(data.receivedAt),
    createdAt: toDate(data.createdAt),
  }
}

export function useEvents(): UseEventsResult {
  const [events, setEvents] = useState<EarthquakeEvent[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    if (isFirestoreConfigured && db) {
      const q = query(
        collection(db, 'events'),
        orderBy('occurredAt', 'desc'),
        limit(EVENT_LIMIT)
      )

      const unsubscribe = onSnapshot(
        q,
        (snapshot) => {
          const newEvents = snapshot.docs.map((doc) =>
            firestoreDocToEvent(doc.id, doc.data())
          )
          setEvents(newEvents)
          setIsLoading(false)
          setError(null)
        },
        (err) => {
          console.error('[namazu] Firestore onSnapshot error:', err)
          setError('イベントの取得に失敗しました')
          setIsLoading(false)
        }
      )

      return () => unsubscribe()
    }

    // Demo mode fallback: fetch from REST API once
    let cancelled = false
    async function fetchEvents() {
      try {
        const data = await api.listEvents()
        if (cancelled) return
        const mapped = (data as Array<Record<string, string | number | string[]>>).map(
          (e) => ({
            id: e.id as string,
            type: e.type as string,
            source: e.source as string,
            severity: e.severity as number,
            affectedAreas: (e.affectedAreas as string[]) || [],
            occurredAt: new Date(e.occurredAt as string),
            receivedAt: new Date(e.receivedAt as string),
            createdAt: new Date(e.createdAt as string),
          })
        )
        setEvents(mapped)
      } catch (err) {
        if (cancelled) return
        setError(
          err instanceof Error ? err.message : 'イベントの取得に失敗しました'
        )
      } finally {
        if (!cancelled) setIsLoading(false)
      }
    }
    fetchEvents()
    return () => { cancelled = true }
  }, [])

  return { events, isLoading, error }
}
