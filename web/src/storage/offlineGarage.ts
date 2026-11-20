import { openDB, DBSchema, IDBPDatabase } from 'idb'
import { UserBikeSummary, BikeHealthDashboard } from '../api/types'

interface MotoSpecDB extends DBSchema {
  bikes: {
    key: string
    value: UserBikeSummary
  }
  dashboards: {
    key: string
    value: BikeHealthDashboard
  }
  settings: {
    key: string
    value: { id: string; value: string | boolean | number }
  }
}

const DB_NAME = 'motospec-offline-cache'
const DB_VERSION = 1

let dbPromise: Promise<IDBPDatabase<MotoSpecDB>> | null = null

function getDB(): Promise<IDBPDatabase<MotoSpecDB>> {
  if (!dbPromise) {
    dbPromise = openDB<MotoSpecDB>(DB_NAME, DB_VERSION, {
      upgrade(db) {
        if (!db.objectStoreNames.contains('bikes')) {
          db.createObjectStore('bikes', { keyPath: 'id' })
        }
        if (!db.objectStoreNames.contains('dashboards')) {
          db.createObjectStore('dashboards', { keyPath: 'id' })
        }
        if (!db.objectStoreNames.contains('settings')) {
          db.createObjectStore('settings', { keyPath: 'id' })
        }
      },
    })
  }
  return dbPromise
}

// Cache the list of user bikes for offline access
export async function cacheOfflineBikes(bikes: UserBikeSummary[]): Promise<void> {
  try {
    const db = await getDB()
    const tx = db.transaction('bikes', 'readwrite')
    await tx.store.clear()
    for (const bike of bikes) {
      await tx.store.put(bike)
    }
    await tx.done
  } catch (err) {
    console.warn('Failed to cache bikes to IndexedDB:', err)
  }
}

// Retrieve cached user bikes
export async function getCachedBikes(): Promise<UserBikeSummary[]> {
  try {
    const db = await getDB()
    return await db.getAll('bikes')
  } catch (err) {
    console.warn('Failed to read cached bikes from IndexedDB:', err)
    return []
  }
}

// Cache full health dashboard for a motorcycle
export async function cacheOfflineDashboard(dashboard: BikeHealthDashboard): Promise<void> {
  try {
    const db = await getDB()
    await db.put('dashboards', dashboard)
  } catch (err) {
    console.warn('Failed to cache dashboard to IndexedDB:', err)
  }
}

// Retrieve cached health dashboard for a motorcycle
export async function getCachedDashboard(bikeId: string): Promise<BikeHealthDashboard | undefined> {
  try {
    const db = await getDB()
    return await db.get('dashboards', bikeId)
  } catch (err) {
    console.warn('Failed to read cached dashboard from IndexedDB:', err)
    return undefined
  }
}
