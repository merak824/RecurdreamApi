import type { RouterScrollBehavior } from 'vue-router'

type ScrollPosition = { left?: number; top?: number }

export function resolveScrollBehavior(
  hash: string,
  savedPosition: ScrollPosition | null,
  reducedMotion = false
): ReturnType<RouterScrollBehavior> {
  if (savedPosition) return savedPosition
  if (hash) return { el: hash, behavior: reducedMotion ? 'auto' : 'smooth' }
  return { top: 0 }
}
