import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
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

describe('AppSidebar custom external navigation', () => {
  it('adds external links for the manual and image studio navigation items', () => {
    expect(componentSource).toContain("const feishuDocsUrl = 'https://recurdream.feishu.cn/wiki/HoSLwRZI0ilf4hkhs27crlUVnNe'")
    expect(componentSource).toContain("const imageStudioUrl = 'https://image.recurdream.com'")
    expect(componentSource).toContain("{ path: '/docs/feishu', label: '使用手册', icon: DocumentIcon, externalUrl: feishuDocsUrl }")
    expect(componentSource).toContain("{ path: '/image-studio', label: '图片工作台', icon: ImageIcon, externalUrl: imageStudioUrl }")
  })

  it('opens external navigation items in a new tab', () => {
    expect(componentSource).toContain('<a')
    expect(componentSource).toContain('v-if="item.externalUrl"')
    expect(componentSource).toContain(':href="item.externalUrl"')
    expect(componentSource).toContain('target="_blank"')
    expect(componentSource).toContain('rel="noopener noreferrer"')
    expect(componentSource).not.toContain("window.open(externalItem.externalUrl, '_blank', 'noopener,noreferrer')")
  })
})
