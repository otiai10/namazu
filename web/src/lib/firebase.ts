import { initializeApp, type FirebaseApp } from 'firebase/app'
import { getAuth, connectAuthEmulator, GoogleAuthProvider, type Auth } from 'firebase/auth'

// Firebase configuration from environment variables
const firebaseConfig = {
  apiKey: import.meta.env.VITE_FIREBASE_API_KEY,
  authDomain: import.meta.env.VITE_FIREBASE_AUTH_DOMAIN,
  projectId: import.meta.env.VITE_FIREBASE_PROJECT_ID,
  storageBucket: import.meta.env.VITE_FIREBASE_STORAGE_BUCKET,
  messagingSenderId: import.meta.env.VITE_FIREBASE_MESSAGING_SENDER_ID,
  appId: import.meta.env.VITE_FIREBASE_APP_ID,
}

// Auth Emulator configuration
const useAuthEmulator = import.meta.env.VITE_USE_AUTH_EMULATOR === 'true'
const authEmulatorHost = import.meta.env.VITE_AUTH_EMULATOR_HOST || 'http://127.0.0.1:9099'

// Tenant ID configuration
const tenantId = import.meta.env.VITE_FIREBASE_TENANT_ID || null

// Check if Firebase is configured
export const isFirebaseConfigured = !!firebaseConfig.apiKey

let app: FirebaseApp | null = null
let auth: Auth | null = null
let googleProvider: GoogleAuthProvider | null = null

if (isFirebaseConfigured) {
  app = initializeApp(firebaseConfig)
  auth = getAuth(app)

  // Connect to Auth Emulator if configured
  if (useAuthEmulator) {
    connectAuthEmulator(auth, authEmulatorHost, { disableWarnings: true })
    if (import.meta.env.DEV) {
      console.info(`[namazu] Connected to Firebase Auth Emulator at ${authEmulatorHost}`)
    }
  }

  // Set tenant ID if configured
  if (tenantId) {
    auth.tenantId = tenantId
    if (import.meta.env.DEV) {
      console.info(`[namazu] Using tenant ID: ${tenantId}`)
    }
  }

  googleProvider = new GoogleAuthProvider()
} else {
  console.warn('[namazu] Firebase not configured. Running in demo mode.')
  console.warn('[namazu] To enable auth, create web/.env.local with Firebase config.')
}

export { auth, googleProvider }
export default app
