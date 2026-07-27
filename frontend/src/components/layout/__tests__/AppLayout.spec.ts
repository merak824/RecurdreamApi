import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const layoutSource = readFileSync(resolve(dir, '../AppLayout.vue'), 'utf8')
const globalStyles = readFileSync(resolve(dir, '../../../style.css'), 'utf8')

describe('AppLayout branded background', () => {
  it('uses the same blue and orange ambient layers as the public pages', () => {
    expect(layoutSource).toContain('class="app-ambient-background"')
    expect(layoutSource).toContain(
      'radial-gradient(circle 18rem at calc(25% + 12rem) calc(25% + 12rem)'
    )
    expect(layoutSource).toContain(
      'radial-gradient(circle 16rem at calc(75% - 10rem) calc(75% - 10rem)'
    )
    expect(layoutSource).not.toContain('bg-mesh-gradient')
  })

  it('keeps the backend shell readable in both themes', () => {
    expect(layoutSource).toContain('--app-page: #f8fafc;')
    expect(layoutSource).toContain('--app-page: #0b0b10;')
    expect(layoutSource).toContain(':global(html.dark .app-layout-shell)')
    expect(layoutSource).toContain(':global(html.dark .app-ambient-background::before)')
    expect(globalStyles).toContain('background: rgba(255, 255, 255, 0.86);')
    expect(globalStyles).toContain('background: rgba(17, 24, 39, 0.86);')
    expect(globalStyles).toContain('backdrop-filter: blur(20px);')
  })
})
