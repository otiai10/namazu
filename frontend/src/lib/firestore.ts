import { initializeApp } from 'firebase/app'
import { getFirestore, connectFirestoreEmulator, type Firestore } from 'firebase/firestore'
import app, { isFirebaseConfigured } from './firebase'

const firestoreEmulatorHost = import.meta.env.VITE_FIRESTORE_EMULATOR_HOST || ''
const firestoreDatabase = import.meta.env.VITE_FIRESTORE_DATABASE || ''
const firestoreProjectId = import.meta.env.VITE_FIRESTORE_PROJECT_ID || ''

let db: Firestore | null = null

if (isFirebaseConfigured && app) {
  // When a separate project ID is specified for Firestore (e.g. local emulator),
  // create a dedicated Firebase app so Auth (namazu-live) and Firestore (namazu-local)
  // can use different project IDs.
  const firestoreApp = firestoreProjectId
    ? initializeApp({ projectId: firestoreProjectId }, 'firestore')
    : app

  db = firestoreDatabase
    ? getFirestore(firestoreApp, firestoreDatabase)
    : getFirestore(firestoreApp)

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
