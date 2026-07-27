<template>
  <div class="auth-page" :class="{ 'dark-mode': isDark }">
    <div class="auth-background" aria-hidden="true"></div>

    <header class="auth-toolbar">
      <RouterLink to="/home" class="auth-back-link" :aria-label="t('auth.backHome')">
        <Icon name="arrowLeft" size="sm" :stroke-width="1.8" />
        <span>{{ t('auth.backHome') }}</span>
      </RouterLink>
      <button
        type="button"
        class="auth-theme-button"
        :aria-label="isDark ? t('home.switchToLight') : t('home.switchToDark')"
        :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
        @click="toggleTheme"
      >
        <Icon :name="isDark ? 'moon' : 'sun'" size="md" :stroke-width="1.8" />
      </button>
    </header>

    <main class="auth-main">
      <div class="auth-shell">
        <div class="auth-brand">
          <img
            :src="siteLogo"
            :alt="`${siteName} Logo`"
            class="auth-logo"
            width="72"
            height="72"
          />
          <h1>{{ siteName }}</h1>
          <p>{{ siteSubtitle }}</p>
        </div>

        <div class="auth-card">
          <slot />
        </div>

        <div v-if="$slots.footer" class="auth-footer">
          <slot name="footer" />
        </div>

        <p class="auth-copyright">&copy; {{ currentYear }} {{ siteName }}</p>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { DEFAULT_SITE_LOGO, resolveSiteLogo, resolveSiteName } from '@/config/branding'
import { useAppStore } from '@/stores'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()
const { t } = useI18n()

const siteName = computed(() => resolveSiteName(appStore.cachedPublicSettings?.site_name || appStore.siteName))
const siteLogo = computed(() => sanitizeUrl(
  resolveSiteLogo(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo),
  { allowRelative: true, allowDataUrl: true }
) || DEFAULT_SITE_LOGO)
const siteSubtitle = computed(() => {
  const subtitle = appStore.cachedPublicSettings?.site_subtitle?.trim()
  return !subtitle || subtitle === 'Subscription to API Conversion Platform'
    ? '统一连接 GPT 与 Claude'
    : subtitle
})
const isDark = ref(true)

const currentYear = computed(() => new Date().getFullYear())

function setTheme(dark: boolean): void {
  isDark.value = dark
  document.documentElement.classList.toggle('dark', dark)
}

function initTheme(): void {
  const savedTheme = localStorage.getItem('theme')
  setTheme(savedTheme ? savedTheme === 'dark' : true)
}

function toggleTheme(): void {
  setTheme(!isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

onMounted(() => {
  initTheme()
  appStore.fetchPublicSettings()
})
</script>

<style scoped>
.auth-page {
  --auth-page: #f8fafc;
  --auth-surface: rgba(255, 255, 255, 0.9);
  --auth-surface-strong: #ffffff;
  --auth-surface-hover: #f8fafc;
  --auth-text: #0f172a;
  --auth-text-muted: #64748b;
  --auth-line: rgba(203, 213, 225, 0.78);
  --auth-line-strong: rgba(148, 163, 184, 0.88);
  --auth-primary: #2563eb;
  --auth-primary-hover: #1d4ed8;
  --auth-orange: #f47b38;
  position: relative;
  min-width: 320px;
  min-height: 100dvh;
  overflow-x: clip;
  background: var(--auth-page);
  color: var(--auth-text);
  font-family: Inter, "SF Pro Display", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif;
  letter-spacing: 0;
}

.auth-page.dark-mode {
  --auth-page: #0b0b10;
  --auth-surface: rgba(17, 24, 39, 0.88);
  --auth-surface-strong: #111827;
  --auth-surface-hover: #182234;
  --auth-text: #f8fafc;
  --auth-text-muted: #94a3b8;
  --auth-line: rgba(255, 255, 255, 0.11);
  --auth-line-strong: rgba(255, 255, 255, 0.2);
}

.auth-background {
  position: fixed;
  inset: 0;
  pointer-events: none;
  background: var(--auth-page);
}

.auth-background::before {
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle 18rem at calc(25% + 12rem) calc(25% + 12rem), rgba(37, 99, 235, 0.3) 0, rgba(37, 99, 235, 0.24) 38%, rgba(37, 99, 235, 0.11) 68%, transparent 100%),
    radial-gradient(circle 16rem at calc(75% - 10rem) calc(75% - 10rem), rgba(249, 115, 22, 0.2) 0, rgba(249, 115, 22, 0.15) 38%, rgba(249, 115, 22, 0.07) 68%, transparent 100%),
    radial-gradient(80% 80% at 50% -20%, rgba(37, 99, 235, 0.3), transparent),
    radial-gradient(50% 50% at 80%, rgba(249, 115, 22, 0.15), transparent),
    radial-gradient(50% 50% at 20% 80%, rgba(59, 130, 246, 0.2), transparent);
  content: "";
  opacity: 0.6;
  transition: opacity 0.3s ease;
}

.auth-page.dark-mode .auth-background::before {
  opacity: 1;
}

.auth-toolbar {
  position: absolute;
  z-index: 10;
  top: calc(16px + env(safe-area-inset-top));
  right: 16px;
  left: 16px;
  display: flex;
  max-width: 1152px;
  align-items: center;
  justify-content: space-between;
  margin: 0 auto;
}

.auth-back-link,
.auth-theme-button {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  justify-content: center;
  border: 1px solid var(--auth-line);
  border-radius: 10px;
  background: var(--auth-surface);
  color: var(--auth-text-muted);
  box-shadow: 0 10px 28px rgba(15, 23, 42, 0.08);
  backdrop-filter: blur(18px);
  cursor: pointer;
  transition: border-color 0.2s ease, background-color 0.2s ease, color 0.2s ease, transform 0.2s ease;
}

.auth-back-link {
  gap: 8px;
  padding: 0 15px;
  font-size: 13px;
  font-weight: 650;
  text-decoration: none;
}

.auth-theme-button {
  width: 44px;
  padding: 0;
}

.auth-back-link:hover,
.auth-theme-button:hover {
  border-color: var(--auth-line-strong);
  background: var(--auth-surface-hover);
  color: var(--auth-text);
  transform: translateY(-1px);
}

.auth-page :focus-visible {
  outline: 3px solid rgba(59, 130, 246, 0.48);
  outline-offset: 3px;
}

.auth-main {
  position: relative;
  z-index: 1;
  display: grid;
  min-height: 100dvh;
  place-items: center;
  padding: calc(88px + env(safe-area-inset-top)) 16px calc(30px + env(safe-area-inset-bottom));
}

.auth-shell {
  width: min(100%, 448px);
  animation: auth-login-enter 260ms cubic-bezier(0.16, 1, 0.3, 1) both;
}

@keyframes auth-login-enter {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
}

.auth-brand {
  margin-bottom: 24px;
  text-align: center;
}

.auth-logo {
  width: 72px;
  height: 72px;
  margin: 0 auto 14px;
  object-fit: contain;
  filter: drop-shadow(0 12px 22px rgba(37, 99, 235, 0.2));
}

.auth-brand h1 {
  margin: 0;
  background: linear-gradient(92deg, #2878ff 4%, #5d87df 51%, #f07731 96%);
  background-clip: text;
  color: transparent;
  font-size: 31px;
  font-weight: 740;
  line-height: 1.15;
}

.auth-brand p {
  margin: 8px 0 0;
  color: var(--auth-text-muted);
  font-size: 14px;
  line-height: 1.5;
}

.auth-card {
  position: relative;
  overflow: hidden;
  border: 1px solid var(--auth-line);
  border-radius: 14px;
  background: var(--auth-surface);
  padding: 30px;
  box-shadow: 0 22px 55px rgba(49, 70, 112, 0.14);
  backdrop-filter: blur(24px);
}

.auth-page.dark-mode .auth-card {
  box-shadow: 0 26px 60px rgba(0, 0, 0, 0.32);
}

.auth-card::before {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
  background: linear-gradient(90deg, #2563eb, #4c8df5 52%, #f47b38);
  content: "";
}

.auth-footer {
  margin-top: 20px;
  color: var(--auth-text-muted);
  font-size: 14px;
  text-align: center;
}

.auth-copyright {
  margin: 24px 0 0;
  color: var(--auth-text-muted);
  font-size: 12px;
  opacity: 0.72;
  text-align: center;
}

.auth-card :deep(.input-label) {
  display: block;
  margin-bottom: 7px;
  color: var(--auth-text);
  font-size: 13px;
  font-weight: 650;
}

.auth-card :deep(.input) {
  min-height: 48px;
  border-color: var(--auth-line);
  border-radius: 10px;
  background: var(--auth-surface-strong);
  color: var(--auth-text);
  box-shadow: none;
}

.auth-card :deep(.input:hover:not(:disabled)) {
  border-color: var(--auth-line-strong);
}

.auth-card :deep(.input:focus) {
  border-color: var(--auth-primary);
  box-shadow: 0 0 0 3px rgba(37, 99, 235, 0.14);
}

.auth-card :deep(.input-error) {
  border-color: #dc2626;
}

.auth-card :deep(.btn) {
  min-height: 48px;
  border-radius: 10px;
  cursor: pointer;
  transition: background-color 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease, transform 0.2s ease;
}

.auth-card :deep(.btn-primary) {
  border: 1px solid transparent;
  background: linear-gradient(90deg, #2563eb, #3b82f6);
  color: #ffffff;
  box-shadow: 0 12px 26px rgba(37, 99, 235, 0.24);
}

.auth-card :deep(.btn-primary:hover:not(:disabled)) {
  background: linear-gradient(90deg, #1d4ed8, #3073f1);
  box-shadow: 0 14px 30px rgba(37, 99, 235, 0.3);
  transform: translateY(-1px);
}

.auth-card :deep(.btn-secondary) {
  border-color: var(--auth-line);
  background: var(--auth-surface-strong);
  color: var(--auth-text);
  box-shadow: none;
}

.auth-card :deep(.btn-secondary:hover:not(:disabled)) {
  border-color: var(--auth-line-strong);
  background: var(--auth-surface-hover);
}

.auth-card :deep(a.text-primary-600),
.auth-footer :deep(a.text-primary-600) {
  color: var(--auth-primary) !important;
}

.auth-card :deep(a.text-primary-600:hover),
.auth-footer :deep(a.text-primary-600:hover) {
  color: var(--auth-primary-hover) !important;
}

@media (max-width: 520px) {
  .auth-toolbar {
    top: calc(10px + env(safe-area-inset-top));
    right: 10px;
    left: 10px;
  }

  .auth-main {
    align-items: start;
    padding: calc(78px + env(safe-area-inset-top)) 12px calc(22px + env(safe-area-inset-bottom));
  }

  .auth-brand {
    margin-bottom: 18px;
  }

  .auth-logo {
    width: 62px;
    height: 62px;
    margin-bottom: 11px;
  }

  .auth-brand h1 {
    font-size: 27px;
  }

  .auth-card {
    border-radius: 12px;
    padding: 26px 20px;
  }
}

@media (max-height: 760px) and (min-width: 521px) {
  .auth-main {
    align-items: start;
  }

  .auth-brand {
    margin-bottom: 18px;
  }

  .auth-logo {
    width: 58px;
    height: 58px;
    margin-bottom: 10px;
  }
}

@media (prefers-reduced-motion: reduce) {
  .auth-page *,
  .auth-page *::before,
  .auth-page *::after {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: 0.01ms !important;
  }
}
</style>
