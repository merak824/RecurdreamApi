import { getUpdateStatus, type UpdateResult } from '@/api/admin/system'

const TERMINAL_STATUSES = new Set<UpdateResult['status']>(['succeeded', 'failed'])

export interface PollUpdateStatusOptions {
  intervalMs?: number
  timeoutMs?: number
  maxTransientErrors?: number
  getStatus?: (operationId: string) => Promise<UpdateResult>
  sleep?: (delayMs: number) => Promise<void>
  now?: () => number
}

/**
 * Polls the host-agent operation while tolerating short outages during the
 * nginx blue/green switch.
 */
export async function pollUpdateStatus(
  operationId: string,
  options: PollUpdateStatusOptions = {}
): Promise<UpdateResult> {
  const intervalMs = options.intervalMs ?? 3000
  const timeoutMs = options.timeoutMs ?? 20 * 60 * 1000
  const maxTransientErrors = options.maxTransientErrors ?? 10
  const getStatus = options.getStatus ?? getUpdateStatus
  const sleep = options.sleep ?? ((delayMs: number) => new Promise((resolve) => setTimeout(resolve, delayMs)))
  const now = options.now ?? Date.now
  const deadline = now() + timeoutMs
  let transientErrors = 0

  while (true) {
    try {
      const result = await getStatus(operationId)
      transientErrors = 0
      if (TERMINAL_STATUSES.has(result.status)) {
        return result
      }
    } catch (error) {
      transientErrors += 1
      if (transientErrors > maxTransientErrors) {
        throw error
      }
    }

    const remainingMs = deadline - now()
    if (remainingMs <= 0) {
      throw new Error('update operation timed out')
    }
    await sleep(Math.min(intervalMs, remainingMs))
  }
}
