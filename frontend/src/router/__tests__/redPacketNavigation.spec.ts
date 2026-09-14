import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

const currentDir = dirname(fileURLToPath(import.meta.url))
const routerSource = readFileSync(resolve(currentDir, '../index.ts'), 'utf8')
const sidebarSource = readFileSync(resolve(currentDir, '../../components/layout/AppSidebar.vue'), 'utf8')

describe('red packet route and navigation contract', () => {
  it('registers distinct user and admin routes with the correct guards', () => {
    expect(routerSource).toMatch(/path:\s*['"]\/red-packets['"][\s\S]*?requiresAdmin:\s*false/)
    expect(routerSource).toMatch(/path:\s*['"]\/admin\/red-packets['"][\s\S]*?requiresAdmin:\s*true/)
    expect(routerSource.match(/path:\s*['"]\/red-packets['"]/g)).toHaveLength(1)
    expect(routerSource.match(/path:\s*['"]\/admin\/red-packets['"]/g)).toHaveLength(1)
  })

  it('keeps the admin management entry out of the shared self-navigation builder', () => {
    const selfNav = sidebarSource.slice(
      sidebarSource.indexOf('function buildSelfNavItems'),
      sidebarSource.indexOf('function finalizeNav')
    )
    const adminNav = sidebarSource.slice(sidebarSource.indexOf('const adminNavItems'))

    expect(selfNav).toContain("path: '/red-packets'")
    expect(selfNav).not.toContain("path: '/admin/red-packets'")
    expect(adminNav).toContain("path: '/admin/red-packets'")
  })
})
