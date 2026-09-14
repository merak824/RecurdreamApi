<template>
  <div class="app-layout-shell">
    <div class="app-ambient-background" aria-hidden="true"></div>

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="app-layout-workspace relative min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[72px]' : 'lg:ml-64']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="p-4 md:p-6 lg:p-8">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const isAdmin = computed(() => authStore.user?.role === 'admin')

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>

<style scoped>
.app-layout-shell {
  --app-page: #f8fafc;
  position: relative;
  min-width: 320px;
  min-height: 100dvh;
  overflow-x: clip;
  background: var(--app-page);
}

:global(html.dark .app-layout-shell) {
  --app-page: #0b0b10;
}

.app-ambient-background {
  position: fixed;
  z-index: 0;
  inset: 0;
  pointer-events: none;
  background: var(--app-page);
}

.app-ambient-background::before {
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
}

:global(html.dark .app-ambient-background::before) {
  opacity: 1;
}

.app-layout-workspace {
  z-index: 1;
}
</style>
