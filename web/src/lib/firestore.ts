import { getFirestore, connectFirestoreEmulator, type Firestore } from 'firebase/firestore'
import app, { isFirebaseConfigured } from './firebase'

const firestoreEmulatorHost = import.meta.env.VITE_FIRESTORE_EMULATOR_HOST || ''

let db: Firestore | null = null

if (isFirebaseConfigured && app) {
  db = getFirestore(app)

  if (firestoreEmulatorHost) {
    const [host, portStr] = firestoreEmulatorHost.split(':')
    const port = parseInt(portStr, 10)
    connectFirestoreEmulator(db, host, port)
    if (import.meta.env.DEV) {
      console.info(`[namazu] Connected to Firestore Emulator at ${firestoreEmulatorHost}`)
    }
  }
}

export { db }
export const isFirestoreConfigured = !!db
