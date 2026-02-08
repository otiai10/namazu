import { useState, useEffect, useCallback } from 'react'
import { api, type Subscription } from '@/lib/api'

interface UseSubscriptionsResult {
  subscriptions: Subscription[]
  isLoading: boolean
  error: string | null
  showForm: boolean
  editingSubscription: Subscription | undefined
  openCreateForm: () => void
  openEditForm: (sub: Subscription) => void
  closeForm: () => void
  handleDelete: (id: string) => Promise<void>
  handleFormSuccess: () => void
}

export function useSubscriptions(): UseSubscriptionsResult {
  const [subscriptions, setSubscriptions] = useState<Subscription[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [showForm, setShowForm] = useState(false)
  const [editingId, setEditingId] = useState<string | null>(null)

  const loadSubscriptions = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const data = await api.listSubscriptions()
      setSubscriptions(data || [])
    } catch (err) {
      setError(err instanceof Error ? err.message : '読み込みに失敗しました')
    } finally {
      setIsLoading(false)
    }
  }, [])

  useEffect(() => {
    loadSubscriptions()
  }, [loadSubscriptions])

  const handleDelete = useCallback(async (id: string) => {
    if (!confirm('このサブスクリプションを削除しますか？')) return
    try {
      await api.deleteSubscription(id)
      setSubscriptions((prev) => prev.filter((s) => s.id !== id))
    } catch (err) {
      setError(err instanceof Error ? err.message : '削除に失敗しました')
    }
  }, [])

  const editingSubscription = editingId
    ? subscriptions.find((s) => s.id === editingId)
    : undefined

  const openCreateForm = useCallback(() => setShowForm(true), [])

  const openEditForm = useCallback((sub: Subscription) => {
    setEditingId(sub.id)
    setShowForm(true)
  }, [])

  const closeForm = useCallback(() => {
    setShowForm(false)
    setEditingId(null)
  }, [])

  const handleFormSuccess = useCallback(() => {
    setShowForm(false)
    setEditingId(null)
    loadSubscriptions()
  }, [loadSubscriptions])

  return {
    subscriptions,
    isLoading,
    error,
    showForm,
    editingSubscription,
    openCreateForm,
    openEditForm,
    closeForm,
    handleDelete,
    handleFormSuccess,
  }
}
