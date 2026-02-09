import { auth, isFirebaseConfigured } from './firebase'

const API_BASE = '/api'

interface FetchOptions extends RequestInit {
  requireAuth?: boolean
}

class ApiError extends Error {
  constructor(
    public status: number,
    message: string
  ) {
    super(message)
    this.name = 'ApiError'
  }
}

async function fetchWithAuth(
  path: string,
  options: FetchOptions = {}
): Promise<Response> {
  const { requireAuth = true, ...fetchOptions } = options

  const headers = new Headers(fetchOptions.headers)
  headers.set('Content-Type', 'application/json')

  // Only add auth header if Firebase is configured and user is logged in
  if (requireAuth && isFirebaseConfigured && auth) {
    const user = auth.currentUser
    if (user) {
      const token = await user.getIdToken()
      headers.set('Authorization', `Bearer ${token}`)
    }
  }
  // Demo mode: No auth header needed (--test-mode backend doesn't require it)

  const response = await fetch(`${API_BASE}${path}`, {
    ...fetchOptions,
    headers,
  })

  if (!response.ok) {
    const errorText = await response.text().catch(() => 'Unknown error')
    throw new ApiError(response.status, errorText)
  }

  return response
}

// Subscription types
export interface Subscription {
  id: string
  userId?: string
  name: string
  delivery: {
    type: string
    url: string
    secret: string
    secret_prefix?: string
    verified?: boolean
    sign_version?: string
    retry?: {
      enabled: boolean
      max_retries: number
      initial_ms: number
      max_ms: number
    }
  }
  filter?: {
    min_scale?: number
    prefectures?: string[]
  }
}

export interface CreateSubscriptionInput {
  name: string
  delivery: {
    type: string
    url: string
  }
  filter?: {
    min_scale?: number
    prefectures?: string[]
  }
}

export interface CreateSubscriptionResponse {
  id: string
  name: string
  delivery: {
    type: string
    url: string
    secret: string
    secret_prefix?: string
    verified?: boolean
    sign_version?: string
  }
  filter?: {
    min_scale?: number
    prefectures?: string[]
  }
}

export interface UserProfile {
  id: string
  uid: string
  email: string
  displayName: string
  pictureUrl?: string
  plan: string
  createdAt: string
  updatedAt: string
}

// Billing types
export interface BillingStatus {
  plan: string
  hasActiveSubscription: boolean
  subscriptionStatus?: string
  subscriptionEndsAt?: string
  stripeCustomerId?: string
}

export interface CheckoutSessionResponse {
  sessionId: string
  sessionUrl: string
}

export interface PortalSessionResponse {
  url: string
}

// API functions
export const api = {
  // Subscriptions
  async listSubscriptions(): Promise<Subscription[]> {
    const response = await fetchWithAuth('/subscriptions')
    return response.json()
  },

  async getSubscription(id: string): Promise<Subscription> {
    const response = await fetchWithAuth(`/subscriptions/${id}`)
    return response.json()
  },

  async createSubscription(input: CreateSubscriptionInput): Promise<CreateSubscriptionResponse> {
    const response = await fetchWithAuth('/subscriptions', {
      method: 'POST',
      body: JSON.stringify(input),
    })
    return response.json()
  },

  async updateSubscription(id: string, input: CreateSubscriptionInput): Promise<void> {
    await fetchWithAuth(`/subscriptions/${id}`, {
      method: 'PUT',
      body: JSON.stringify(input),
    })
  },

  async deleteSubscription(id: string): Promise<void> {
    await fetchWithAuth(`/subscriptions/${id}`, {
      method: 'DELETE',
    })
  },

  // User profile
  async getProfile(): Promise<UserProfile> {
    const response = await fetchWithAuth('/me')
    return response.json()
  },

  // Events (public)
  async listEvents(): Promise<unknown[]> {
    const response = await fetchWithAuth('/events', { requireAuth: false })
    return response.json()
  },

  // Health check (public)
  async health(): Promise<{ status: string }> {
    const response = await fetch('/health')
    return response.json()
  },

  // Billing
  async getBillingStatus(): Promise<BillingStatus> {
    const response = await fetchWithAuth('/billing/status')
    return response.json()
  },

  async createCheckoutSession(): Promise<CheckoutSessionResponse> {
    const response = await fetchWithAuth('/billing/create-checkout-session', {
      method: 'POST',
    })
    return response.json()
  },

  async getPortalSession(returnUrl?: string): Promise<PortalSessionResponse> {
    const url = returnUrl
      ? `/billing/portal-session?return_url=${encodeURIComponent(returnUrl)}`
      : '/billing/portal-session'
    const response = await fetchWithAuth(url)
    return response.json()
  },
}

export { ApiError }
