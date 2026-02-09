import {
  createContext,
  useContext,
  useEffect,
  useState,
  useCallback,
  type ReactNode,
} from 'react'
import {
  User,
  signInWithPopup,
  signOut as firebaseSignOut,
  onAuthStateChanged,
} from 'firebase/auth'
import { auth, googleProvider, isFirebaseConfigured } from '@/lib/firebase'

export interface AuthContext {
  user: User | null
  isLoading: boolean
  isAuthenticated: boolean
  isDemoMode: boolean
  signInWithGoogle: () => Promise<void>
  signOut: () => Promise<void>
  getIdToken: () => Promise<string | null>
  waitUntilReady: () => Promise<void>
}

const AuthContext = createContext<AuthContext | null>(null)

// Demo user for --test-mode (no Firebase)
const DEMO_USER = {
  uid: 'demo-user',
  email: 'demo@example.com',
  displayName: 'Demo User',
} as unknown as User

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  // Stable promise that resolves when auth state is first determined.
  // Created once via useState initializer (StrictMode safe).
  const [authReadyPromise] = useState(() => {
    if (!isFirebaseConfigured || !auth) return Promise.resolve()
    return auth.authStateReady()
  })

  const waitUntilReady = useCallback(() => authReadyPromise, [authReadyPromise])

  useEffect(() => {
    // Demo mode: Firebase not configured
    if (!isFirebaseConfigured || !auth) {
      console.log('[namazu] Demo mode: Using mock user')
      setUser(DEMO_USER)
      setIsLoading(false)
      return
    }

    // Subscribe to auth state changes
    const unsubscribe = onAuthStateChanged(auth, (user) => {
      setUser(user)
      setIsLoading(false)
    })

    return () => unsubscribe()
  }, [])

  const signInWithGoogle = useCallback(async () => {
    if (!isFirebaseConfigured || !auth || !googleProvider) {
      console.log('[namazu] Demo mode: Sign in skipped')
      setUser(DEMO_USER)
      return
    }

    try {
      await signInWithPopup(auth, googleProvider)
    } catch (error) {
      console.error('Failed to sign in with Google:', error)
      throw error
    }
  }, [])

  const signOut = useCallback(async () => {
    if (!isFirebaseConfigured || !auth) {
      console.log('[namazu] Demo mode: Sign out skipped')
      // In demo mode, keep user logged in
      return
    }

    try {
      await firebaseSignOut(auth)
      window.location.href = '/login'
    } catch (error) {
      console.error('Failed to sign out:', error)
      throw error
    }
  }, [])

  const getIdToken = useCallback(async (): Promise<string | null> => {
    // Demo mode: No token needed (--test-mode backend doesn't verify)
    if (!isFirebaseConfigured) {
      return null
    }

    if (!user) return null
    try {
      return await user.getIdToken()
    } catch (error) {
      console.error('Failed to get ID token:', error)
      return null
    }
  }, [user])

  const value: AuthContext = {
    user,
    isLoading,
    isAuthenticated: !!user,
    isDemoMode: !isFirebaseConfigured,
    signInWithGoogle,
    signOut,
    getIdToken,
    waitUntilReady,
  }

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}

export function useAuth(): AuthContext {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth must be used within an AuthProvider')
  }
  return context
}
