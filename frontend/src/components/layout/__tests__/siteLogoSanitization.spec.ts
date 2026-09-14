import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const sidebarSource = readFileSync(resolve(dir, '../AppSidebar.vue'), 'utf8')
const homeViewSource = readFileSync(resolve(dir, '../../../views/HomeView.vue'), 'utf8')
const keyUsageViewSource = readFileSync(resolve(dir, '../../../views/KeyUsageView.vue'), 'utf8')
const logoSource = readFileSync(resolve(dir, '../../../../public/recurdream-logo.svg'), 'utf8')

describe('site_logo sanitization', () => {
  it('AppSidebar imports sanitizeUrl and applies it to siteLogo', () => {
    expect(sidebarSource).toContain("import { sanitizeUrl } from '@/utils/url'")
    expect(sidebarSource).toContain('sanitizeUrl(resolveSiteLogo(appStore.siteLogo)')
  })

  it('HomeView applies sanitizeUrl to siteLogo', () => {
    expect(homeViewSource).toContain('resolveSiteLogo(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo)')
  })

  it('KeyUsageView applies sanitizeUrl to siteLogo', () => {
    expect(keyUsageViewSource).toContain('resolveSiteLogo(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo)')
  })

  it('all three pass allowRelative and allowDataUrl options', () => {
    for (const src of [sidebarSource, homeViewSource, keyUsageViewSource]) {
      expect(src).toContain('allowRelative: true')
      expect(src).toContain('allowDataUrl: true')
    }
  })

  it('renders the sidebar brand as a transparent vector without a square frame', () => {
    expect(logoSource).toContain('width="64" height="64" viewBox="0 0 64 64"')
    expect(logoSource).not.toMatch(/<rect[^>]+fill="(?:#000(?:000)?|black)"/i)
    expect(sidebarSource).toContain('class="sidebar-logo-image h-full w-full object-contain"')
    expect(sidebarSource).not.toContain('shadow-glow')
  })
})
