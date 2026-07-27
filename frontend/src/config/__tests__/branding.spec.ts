import { describe, expect, it } from 'vitest'

import {
  DEFAULT_SITE_LOGO,
  DEFAULT_SITE_NAME,
  resolveSiteLogo,
  resolveSiteName
} from '@/config/branding'

describe('branding defaults', () => {
  it('uses the RecurDream brand when the configured name is blank or legacy', () => {
    expect(resolveSiteName()).toBe(DEFAULT_SITE_NAME)
    expect(resolveSiteName('   ')).toBe(DEFAULT_SITE_NAME)
    expect(resolveSiteName('Sub2API')).toBe(DEFAULT_SITE_NAME)
  })

  it('preserves a non-default custom site name', () => {
    expect(resolveSiteName('My API')).toBe('My API')
  })

  it('uses the new logo for blank and legacy logo settings', () => {
    expect(resolveSiteLogo()).toBe(DEFAULT_SITE_LOGO)
    expect(resolveSiteLogo('/logo.png')).toBe(DEFAULT_SITE_LOGO)
  })

  it('preserves a custom logo URL', () => {
    expect(resolveSiteLogo('https://example.com/custom.svg')).toBe(
      'https://example.com/custom.svg'
    )
  })
})
