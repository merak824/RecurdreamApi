import { onBeforeUnmount, onMounted, ref, watch, type Ref } from 'vue'
import { redPacketsAPI } from '@/api/redPackets'

const REFRESH_INTERVAL_MS = 60_000
const STORAGE_KEY_PREFIX = 'red-packet:last-seen-activity:'

type ReadonlyRef<T> = Readonly<Ref<T>>

function storageKey(userId: number): string {
  return `${STORAGE_KEY_PREFIX}${userId}`
}

export function useRedPacketReminder(
  userId: ReadonlyRef<number | null | undefined>,
  currentPath: ReadonlyRef<string>
) {
  const hasUnread = ref(false)
  const currentActivityId = ref<number | null>(null)
  let requestVersion = 0
  let resolvedUserId: number | null = null
  let refreshInterval: ReturnType<typeof setInterval> | null = null

  function readSeenActivityId(key: string): string | null {
    try {
      return localStorage.getItem(key)
    } catch {
      return null
    }
  }

  function markActivitySeen(key: string, activityId: number): void {
    try {
      localStorage.setItem(key, String(activityId))
    } catch {
      // Storage can be unavailable in restricted browser contexts.
    }
  }

  function updateUnreadFromStorage(seenActivityId?: string | null): void {
    const activeUserId = userId.value
    const activityId = currentActivityId.value
    if (activeUserId == null || activityId == null || currentPath.value === '/red-packets') {
      hasUnread.value = false
      return
    }

    const seenId = seenActivityId === undefined
      ? readSeenActivityId(storageKey(activeUserId))
      : seenActivityId
    hasUnread.value = seenId !== String(activityId)
  }

  async function refresh(): Promise<void> {
    const requestedUserId = userId.value
    const version = ++requestVersion

    if (requestedUserId == null) {
      resolvedUserId = null
      currentActivityId.value = null
      hasUnread.value = false
      return
    }

    if (resolvedUserId !== requestedUserId) {
      resolvedUserId = requestedUserId
      currentActivityId.value = null
      hasUnread.value = false
    }

    if (currentPath.value === '/red-packets') {
      hasUnread.value = false
    }

    try {
      const activity = await redPacketsAPI.getCurrent()
      if (version !== requestVersion || userId.value !== requestedUserId) return

      if (!activity || activity.status !== 'active') {
        currentActivityId.value = null
        hasUnread.value = false
        return
      }

      currentActivityId.value = activity.id
      const key = storageKey(requestedUserId)
      if (currentPath.value === '/red-packets') {
        markActivitySeen(key, activity.id)
        hasUnread.value = false
        return
      }

      updateUnreadFromStorage()
    } catch {
      if (version !== requestVersion || userId.value !== requestedUserId) return
      currentActivityId.value = null
      hasUnread.value = false
    }
  }

  function handleStorage(event: StorageEvent): void {
    const activeUserId = userId.value
    if (activeUserId == null || event.key !== storageKey(activeUserId)) return
    updateUnreadFromStorage(event.newValue)
  }

  function handleFocus(): void {
    void refresh()
  }

  watch(userId, () => {
    hasUnread.value = false
    currentActivityId.value = null
    void refresh()
  })

  watch(currentPath, (path) => {
    if (path === '/red-packets') {
      hasUnread.value = false
    }
    void refresh()
  })

  void refresh()

  onMounted(() => {
    refreshInterval = setInterval(() => {
      void refresh()
    }, REFRESH_INTERVAL_MS)
    window.addEventListener('focus', handleFocus)
    window.addEventListener('storage', handleStorage)
  })

  onBeforeUnmount(() => {
    requestVersion += 1
    if (refreshInterval !== null) {
      clearInterval(refreshInterval)
      refreshInterval = null
    }
    window.removeEventListener('focus', handleFocus)
    window.removeEventListener('storage', handleStorage)
  })

  return {
    hasUnread,
    refresh
  }
}
