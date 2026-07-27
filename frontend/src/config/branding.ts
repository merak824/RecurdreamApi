export const DEFAULT_SITE_NAME = '递归梦境API'
export const DEFAULT_SITE_LOGO = '/recurdream-logo.svg'

const LEGACY_DEFAULT_SITE_NAMES = new Set(['Sub2API'])
const LEGACY_DEFAULT_SITE_LOGOS = new Set(['/logo.png', 'logo.png'])

export function resolveSiteName(siteName?: string | null): string {
  const normalized = siteName?.trim() || ''
  return !normalized || LEGACY_DEFAULT_SITE_NAMES.has(normalized)
    ? DEFAULT_SITE_NAME
    : normalized
}

export function resolveSiteLogo(siteLogo?: string | null): string {
  const normalized = siteLogo?.trim() || ''
  return !normalized || LEGACY_DEFAULT_SITE_LOGOS.has(normalized)
    ? DEFAULT_SITE_LOGO
    : normalized
}
