import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const dir = dirname(fileURLToPath(import.meta.url))
const source = readFileSync(resolve(dir, '../HomeView.vue'), 'utf8')

describe('HomeView', () => {
  it('uses the RecurDream brand and technology logo defaults', () => {
    expect(source).toContain('resolveSiteName(appStore.cachedPublicSettings?.site_name || appStore.siteName)')
    expect(source).toContain('resolveSiteLogo(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo)')
    expect(source).toContain('<img :src="siteLogo" :alt="`${siteName} Logo`"')
  })

  it('preserves the configurable custom homepage fallback', () => {
    expect(source).toContain('v-if="homeContent"')
    expect(source).toContain('v-if="isHomeContentUrl"')
    expect(source).toContain('v-html="homeContent"')
  })

  it('renders the approved three-screen default homepage with live channel health', () => {
    expect(source).toContain('class="home-hero"')
    expect(source).toContain('id="status"')
    expect(source).toContain('id="models"')
    expect(source).toContain('v-for="(monitor, index) in channelMonitors"')
    expect(source).toContain('<ProviderIcon :provider="monitor.provider"')
    expect(source).toContain('formatAvailability(monitor.availability_7d, monitor.primary_status)')
    expect(source).not.toContain('GPT 模型渠道</h3>')
    expect(source.match(/class="supported-model-card reveal-card"/g)).toHaveLength(7)
  })

  it('loads redacted public monitor data and refreshes it once per minute', () => {
    expect(source).toContain('listPublic as listPublicChannelMonitors')
    expect(source).toContain('await listPublicChannelMonitors({ signal: controller.signal })')
    expect(source).toContain('const channelStatusRefreshMs = 60_000')
    expect(source).toContain('if (!document.hidden) void refreshChannelStatus(true)')
    expect(source).toContain('channelStatusAbortController?.abort()')
  })

  it('uses the approved navigation destinations', () => {
    expect(source).toContain('href="#status"')
    expect(source).toContain('href="#models"')
    expect(source).toContain('href="https://image.recurdream.com"')
    expect(source).toContain("'https://docs.recurdream.com'")
  })

  it('keeps production routes for login and authenticated dashboards', () => {
    expect(source).toContain(':to="isAuthenticated ? dashboardPath : \'/login\'"')
    expect(source).toContain("const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')")
  })

  it('adds a simple accessible transition from home to login', () => {
    expect(source).toContain('@click.capture="handlePrimaryNavigation"')
    expect(source).toContain("import('@/views/auth/LoginView.vue')")
    expect(source).toContain("'leaving-for-login': loginTransitionInProgress")
    expect(source).toContain('if (loginTransitionInProgress.value) return')
    expect(source).toContain('window.setTimeout(resolve, 180)')
    expect(source).toContain('.home-page.leaving-for-login { opacity: 0; }')
    expect(source).not.toContain('document.startViewTransition')
  })

  it('includes the live uptime and performant reveal motion', () => {
    expect(source).toContain("new Date('2025-12-18T02:26:18Z')")
    expect(source).toContain('setInterval(updateRuntime, 1000)')
    expect(source).toContain('new IntersectionObserver')
    expect(source).not.toMatch(/\.status-card,[\s\S]*?backdrop-filter/)
  })

  it('defaults the homepage to dark mode when no preference is saved', () => {
    expect(source).toContain("if (!savedTheme) setTheme(true)")
    expect(source).toContain("'dark-mode': isDark")
    expect(source).toContain('.home-page.dark-mode')
  })

  it('keeps the blue and orange ambient background within the hero viewport', () => {
    expect(source).toContain('.home-page::before')
    expect(source).toContain('height: max(100dvh, 760px);')
    expect(source).toContain(
      'radial-gradient(circle 18rem at calc(25% + 12rem) calc(25% + 12rem)'
    )
    expect(source).toContain(
      'radial-gradient(circle 16rem at calc(75% - 10rem) calc(75% - 10rem)'
    )
    expect(source).toContain('.home-page.dark-mode::before { opacity: 1; }')
  })
})
