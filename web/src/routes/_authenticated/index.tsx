import { createFileRoute } from '@tanstack/react-router'
import { useAuth } from '@/hooks/useAuth'
import { useSubscriptions } from '@/hooks/useSubscriptions'
import { SubscriptionList } from '@/components/SubscriptionList'
import { SubscriptionForm } from '@/components/SubscriptionForm'
import { EventFeed } from '@/components/EventFeed'

export const Route = createFileRoute('/_authenticated/')({
  component: Dashboard,
})

function Dashboard() {
  const { user } = useAuth()
  const subs = useSubscriptions()

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-gray-900">ダッシュボード</h1>
        <p className="text-gray-600">
          ようこそ、{user?.displayName || 'ユーザー'}さん
        </p>
      </div>

      {subs.error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {subs.error}
        </div>
      )}

      {subs.showForm && (
        <SubscriptionForm
          subscription={subs.editingSubscription}
          onClose={subs.closeForm}
          onSuccess={subs.handleFormSuccess}
        />
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-1" id="event-feed-container">
          <EventFeed />
        </div>

        <div className="lg:col-span-2" id="subscriptions-container">
          <SubscriptionList
            subscriptions={subs.subscriptions}
            isLoading={subs.isLoading}
            onEdit={subs.openEditForm}
            onDelete={subs.handleDelete}
            onCreateNew={subs.openCreateForm}
          />
        </div>
      </div>
    </div>
  )
}
