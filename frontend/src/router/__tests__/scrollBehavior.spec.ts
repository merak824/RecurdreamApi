import { describe, expect, it } from 'vitest'

import { resolveScrollBehavior } from '../scrollBehavior'

describe('resolveScrollBehavior', () => {
  it('restores the saved position for browser back and forward navigation', () => {
    expect(resolveScrollBehavior('', { left: 12, top: 240 })).toEqual({ left: 12, top: 240 })
  })

  it('smoothly scrolls hash navigation to the target element', () => {
    expect(resolveScrollBehavior('#status', null)).toEqual({ el: '#status', behavior: 'smooth' })
  })

  it('disables smooth motion when reduced motion is requested', () => {
    expect(resolveScrollBehavior('#models', null, true)).toEqual({ el: '#models', behavior: 'auto' })
  })

  it('scrolls ordinary route navigation to the page top', () => {
    expect(resolveScrollBehavior('', null)).toEqual({ top: 0 })
  })
})
