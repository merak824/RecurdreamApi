import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')
const zhLocalePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../i18n/locales/zh/common.ts')
const zhLocaleSource = readFileSync(zhLocalePath, 'utf8')
const enLocalePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../i18n/locales/en/common.ts')
const enLocaleSource = readFileSync(enLocalePath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar custom external navigation', () => {
  it('adds external links for the manual and image studio navigation items', () => {
    expect(componentSource).toContain("const feishuDocsUrl = 'https://recurdream.feishu.cn/wiki/HoSLwRZI0ilf4hkhs27crlUVnNe'")
    expect(componentSource).toContain("const imageStudioUrl = 'https://image.recurdream.com'")
    expect(componentSource).toContain("{ path: '/docs/feishu', label: '使用手册', icon: DocumentIcon, externalUrl: feishuDocsUrl }")
    expect(componentSource).toContain("{ path: '/image-studio', label: '图片工作台', icon: BatchImageIcon, externalUrl: imageStudioUrl }")
  })

  it('opens custom external navigation items in a new tab', () => {
    expect(componentSource).toContain('externalUrl?: string')
    expect(componentSource).toContain('<a')
    expect(componentSource).toContain('v-if="item.externalUrl"')
    expect(componentSource).toContain(':href="item.externalUrl"')
    expect(componentSource).toContain('target="_blank"')
    expect(componentSource).toContain('rel="noopener noreferrer"')
  })
})

describe('AppSidebar header styles', () => {
  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.sidebar-brand\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})

describe('AppSidebar red packet reminder', () => {
  it('connects browser-local unread state to the user-facing red packet item', () => {
    expect(componentSource).toContain("import { useRedPacketReminder } from '@/composables/useRedPacketReminder'")
    expect(componentSource).toContain('unread?: boolean')
    expect(componentSource).toContain('const { hasUnread: hasUnreadRedPacket } = useRedPacketReminder(')
    expect(componentSource).toContain('computed(() => authStore.user?.id)')
    expect(componentSource).toContain('computed(() => route.path)')
    expect(componentSource).toContain("path: '/red-packets'")
    expect(componentSource).toContain('unread: hasUnreadRedPacket.value')
  })

  it('renders an accessible red packet badge only in user-facing navigation branches', () => {
    const adminMenu = componentSource.match(/<!-- Admin Section -->[\s\S]*?<!-- Personal Section for Admin/)
    const adminPersonalMenu = componentSource.match(/<!-- Personal Section for Admin[\s\S]*?<!-- Regular User View -->/)
    const regularUserMenu = componentSource.match(/<!-- Regular User View -->[\s\S]*?<\/nav>/)

    expect(adminMenu?.[0]).not.toContain('red-packet-unread-badge')
    expect(adminPersonalMenu?.[0]).toContain('red-packet-unread-badge')
    expect(regularUserMenu?.[0]).toContain('red-packet-unread-badge')
    expect(componentSource.match(/class="red-packet-unread-badge /g) ?? []).toHaveLength(2)
    expect(componentSource).toContain("import Icon from '@/components/icons/Icon.vue'")
    expect(componentSource.match(/<Icon name="gift" size="sm"/g) ?? []).toHaveLength(2)
    expect(componentSource).not.toContain('red-packet-unread-dot')
    expect(componentSource).toContain(":aria-label=" + '"t(\'nav.newRedPacketActivity\')"')
    expect(componentSource).toContain('absolute right-2.5 top-1/2 flex h-6 w-6 -translate-y-1/2 items-center justify-center rounded-md bg-red-500')
  })

  it('does not add unread state to the admin activity management item', () => {
    expect(componentSource).toContain(
      "{ path: '/admin/red-packets', label: t('nav.redPacketManagement'), icon: GiftIcon, hideInSimpleMode: true }"
    )
  })

  it('animates the unread gift icon while respecting reduced motion', () => {
    expect(componentSource).toContain('red-packet-unread-badge-icon')
    expect(componentSource).toContain('@keyframes red-packet-gift-wiggle')
    expect(componentSource).toContain('@media (prefers-reduced-motion: reduce)')
    expect(componentSource).toContain('animation: none;')
  })

  it('keeps the admin account management item icon visible', () => {
    expect(componentSource).toContain(
      "{ path: '/admin/accounts', label: t('nav.accounts'), icon: GlobeIcon }"
    )
    expect(componentSource).toContain('const GlobeIcon = {')
  })

  it('provides localized accessible text for the unread status', () => {
    expect(zhLocaleSource).toContain("newRedPacketActivity: '有新的红包活动'")
    expect(enLocaleSource).toContain("newRedPacketActivity: 'New red packet activity'")
  })
})
