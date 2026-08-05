<template>
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <div v-else v-html="homeContent"></div>
  </div>

  <div
    v-else-if="compactHomeEnabled"
    data-testid="compact-home"
    class="flex min-h-screen flex-col bg-gray-50 text-gray-900 dark:bg-dark-950 dark:text-white"
  >
    <header class="border-b border-gray-200 px-4 py-4 sm:px-6 dark:border-dark-800">
      <nav class="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3 sm:gap-4">
        <div class="flex min-w-0 flex-1 items-center gap-3">
          <img
            :src="siteLogo || '/logo.svg'"
            alt="Logo"
            class="h-9 w-9 shrink-0 rounded-lg object-contain"
          />
          <span class="min-w-0 truncate text-base font-semibold">{{ siteName }}</span>
        </div>
        <div class="flex max-w-full shrink-0 flex-wrap items-center justify-end gap-2">
          <LocaleSwitcher />
          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>
          <button
            class="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg text-gray-500 hover:bg-gray-100 dark:text-dark-400 dark:hover:bg-dark-800"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>
          <router-link
            :to="isAuthenticated ? dashboardPath : '/login'"
            class="inline-flex min-h-10 shrink-0 items-center justify-center rounded-lg bg-gray-900 px-4 py-2 text-sm font-medium text-white hover:bg-gray-800 dark:bg-white dark:text-gray-900 dark:hover:bg-gray-200"
          >
            {{ isAuthenticated ? t('home.dashboard') : t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="flex min-w-0 flex-1 items-center justify-center px-4 py-16 sm:px-6">
      <div class="min-w-0 max-w-2xl text-center">
        <img
          :src="siteLogo || '/logo.svg'"
          alt="Logo"
          class="mx-auto mb-6 h-20 w-20 rounded-2xl object-contain"
        />
        <h1 class="[overflow-wrap:anywhere] text-3xl font-bold md:text-4xl">{{ siteName }}</h1>
        <p class="mt-4 whitespace-pre-wrap [overflow-wrap:anywhere] text-base text-gray-600 dark:text-dark-300">{{ siteSubtitle }}</p>
        <router-link
          :to="isAuthenticated ? dashboardPath : '/login'"
          class="mt-8 inline-flex min-h-10 items-center justify-center rounded-lg bg-primary-600 px-5 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          {{ isAuthenticated ? t('home.goToDashboard') : t('home.login') }}
        </router-link>
      </div>
    </main>

    <footer class="min-w-0 border-t border-gray-200 px-4 py-5 text-center text-sm text-gray-500 [overflow-wrap:anywhere] sm:px-6 dark:border-dark-800">
      &copy; {{ currentYear }} {{ siteName }}
    </footer>
  </div>

  <div
    v-else
    class="home-page"
    :class="{
      'motion-ready': motionReady,
      'dark-mode': isDark,
      'leaving-for-login': loginTransitionInProgress
    }"
  >
    <svg class="brand-symbols" aria-hidden="true" focusable="false">
      <symbol id="home-openai-mark" viewBox="0 0 24 24">
        <path d="M22.282 9.821a5.985 5.985 0 00-.516-4.911 6.046 6.046 0 00-6.51-2.9A6.065 6.065 0 004.981 4.182a5.985 5.985 0 00-3.998 2.9 6.046 6.046 0 00.743 7.096 5.98 5.98 0 00.511 4.911 6.051 6.051 0 006.515 2.9A5.985 5.985 0 0013.26 24a6.056 6.056 0 005.772-4.206 5.989 5.989 0 003.997-2.9 6.056 6.056 0 00-.747-7.073zm-9.022 12.608a4.476 4.476 0 01-2.877-1.041l.142-.08 4.778-2.758a.795.795 0 00.393-.682v-6.737l2.02 1.169a.071.071 0 01.038.052v5.583a4.504 4.504 0 01-4.494 4.494zm-9.661-4.125a4.471 4.471 0 01-.535-3.014l.142.085 4.783 2.758a.771.771 0 00.781 0l5.843-3.369v2.333a.08.08 0 01-.033.061L9.74 19.95a4.499 4.499 0 01-6.141-1.646zM2.341 7.896a4.485 4.485 0 012.365-1.973V11.6a.766.766 0 00.388.677l5.814 3.354-2.02 1.169a.076.076 0 01-.071 0l-4.83-2.787a4.504 4.504 0 01-1.646-6.141zm16.596 3.856l-5.833-3.388 2.015-1.164a.076.076 0 01.071 0l4.83 2.791a4.494 4.494 0 01-.676 8.104v-5.677a.79.79 0 00-.407-.666zm2.011-3.024l-.142-.085-4.774-2.782a.776.776 0 00-.785 0L9.409 9.23V6.897a.066.066 0 01.028-.061l4.831-2.787a4.499 4.499 0 016.68 4.66zM8.307 12.863l-2.02-1.164a.08.08 0 01-.039-.056V6.074a4.499 4.499 0 017.376-3.453l-.142.08-4.778 2.758a.795.795 0 00-.393.681zm1.097-2.365l2.602-1.5 2.607 1.5v2.999l-2.597 1.5-2.607-1.5z" />
      </symbol>
      <symbol id="home-claude-mark" viewBox="0 0 24 24">
        <path d="m4.714 15.956 4.718-2.648.079-.23-.079-.128h-.231l-.789-.048-2.696-.073-2.337-.097-2.265-.122-.57-.121-.535-.704.055-.352.48-.322.685.06 1.518.104 2.277.158 1.651.097 2.447.255h.389l.054-.158-.133-.097-.104-.097-2.331-1.615-2.55-1.688-1.336-.971-.722-.492-.365-.461-.157-1.008.655-.722.881.06.224.061.893.686 1.906 1.475 2.489 1.834.365.303.145-.103.019-.073-.164-.273-1.354-2.447-1.445-2.489-.643-1.032-.17-.62c-.061-.255-.104-.467-.104-.728L6.287.134 6.7 0l.996.134.419.364.619 1.415 1.002 2.228 1.554 3.03.456.898.243.832.091.255h.158v-.146l.127-1.706.237-2.095.231-2.695.079-.759.376-.91.747-.492.583.279.479.686-.067.443-.285 1.852-.559 2.902-.364 1.943h.212l.243-.243.984-1.305 1.651-2.064.729-.82.85-.904.546-.431h1.032l.759 1.13-.34 1.165-1.063 1.348-.88 1.141-1.263 1.7-.789 1.36.073.109.188-.018 2.854-.607 1.542-.28 1.84-.315.831.388.091.395-.328.807-1.967.486-2.307.461-3.436.814-.043.03.049.061 1.548.146.662.036h1.621l3.018.225.789.522.473.637-.079.486-1.214.619-1.639-.389-3.825-.91-1.312-.328h-.182v.109l1.093 1.069 2.003 1.809 2.508 2.331.127.577-.321.455-.34-.048-2.204-1.658-.85-.747-1.925-1.621h-.127v.17l.443.65 2.344 3.521.121 1.081-.17.352-.607.212-.668-.121-1.372-1.925-2.671-4.098-1.141-1.943-.14.079-.674 7.255-.316.371-.728.279-.607-.461-.322-.747.322-1.475.388-1.925.316-1.53.285-1.9.17-.631-.012-.043-.14.019-1.432 1.967-2.18 2.945-1.724 1.845-.413.164-.716-.37.067-.662.4-.589 2.386-3.036 1.439-1.882.929-1.087-.006-.158h-.055l-6.338 4.117-1.13.145-.485-.455.061-.747.23-.243 1.907-1.311z" />
      </symbol>
    </svg>

    <header class="navbar">
      <a class="brand" href="#top" aria-label="返回首页顶部">
        <img :src="siteLogo" :alt="`${siteName} Logo`" width="31" height="31" />
        <span>{{ siteName }}</span>
      </a>

      <nav class="nav-links" aria-label="主导航">
        <a href="#status">渠道状态</a>
        <a href="#models">支持模型</a>
        <a href="https://image.recurdream.com" target="_blank" rel="noopener noreferrer">生图工作站</a>
        <a :href="docUrl" target="_blank" rel="noopener noreferrer">接入文档</a>
      </nav>

      <div class="nav-actions">
        <button
          class="theme-button"
          type="button"
          :aria-label="isDark ? '切换到浅色主题' : '切换到深色主题'"
          :title="isDark ? '切换到浅色主题' : '切换到深色主题'"
          @click="toggleTheme"
        >
          <Icon :name="isDark ? 'moon' : 'sun'" size="md" :stroke-width="1.8" />
        </button>
        <span class="service-pill" :class="`is-${channelStatusSummary.tone}`">
          {{ channelStatusSummary.label }}
        </span>
        <router-link
          class="nav-cta"
          :to="isAuthenticated ? dashboardPath : '/login'"
          @pointerenter="prefetchLoginPage"
          @focus="prefetchLoginPage"
          @click.capture="handlePrimaryNavigation"
        >
          {{ isAuthenticated ? '控制台' : '登录' }}
        </router-link>
      </div>
    </header>

    <main id="top" class="home-hero">
      <div class="model-row" aria-label="支持的模型">
        <span class="model-chip"><span class="model-logo" style="--chip-color:#67e8d2"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-openai-mark" /></svg></span>GPT-5.6 Sol</span>
        <span class="model-chip"><span class="model-logo round" style="--chip-color:#55b8ff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-openai-mark" /></svg></span>GPT-5.6 Terra</span>
        <span class="model-chip"><span class="model-logo" style="--chip-color:#8a9cff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-openai-mark" /></svg></span>GPT-5.6 Luna</span>
        <span class="model-chip"><span class="model-logo round" style="--chip-color:#a879f4;--chip-ink:#fff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-openai-mark" /></svg></span>GPT-image-2</span>
        <span class="model-chip"><span class="model-logo" style="--chip-color:#ef906d;--chip-ink:#fff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-claude-mark" /></svg></span>Claude Fable 5</span>
        <span class="model-chip"><span class="model-logo round" style="--chip-color:#f29b76;--chip-ink:#fff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-claude-mark" /></svg></span>Claude Opus 4.8</span>
        <span class="model-chip"><span class="model-logo" style="--chip-color:#f7b18f;--chip-ink:#fff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-claude-mark" /></svg></span>Claude Sonnet 5</span>
      </div>

      <p class="brand-title">{{ siteName }}</p>
      <h1>双模型智能中转站</h1>
      <p class="description">
        一个 API Key，稳定连接 GPT 与 Claude。<br />
        统一接口、透明计费，调用过程简单清晰。<br />
        为个人开发者和团队提供可靠的 AI 基础服务。
      </p>

      <div class="terminal" aria-label="API 接入地址">
        <span class="terminal-prompt">$</span>
        <span class="terminal-runtime">{{ typewriterText }}</span>
        <span class="cursor" aria-hidden="true"></span>
      </div>

      <div class="actions">
        <router-link
          class="button button-primary"
          :to="isAuthenticated ? dashboardPath : '/login'"
          @pointerenter="prefetchLoginPage"
          @focus="prefetchLoginPage"
          @click.capture="handlePrimaryNavigation"
        >
          开始使用
        </router-link>
        <a class="button button-secondary" :href="docUrl" target="_blank" rel="noopener noreferrer">接入文档</a>
      </div>

      <section class="metrics" aria-label="平台能力">
        <article class="metric metric-uptime">
          <span class="uptime-title">本站已稳定运行</span>
          <strong class="metric-value"><span>{{ runtime.days }}</span><small>天</small></strong>
          <span class="runtime-compact" aria-label="当前运行时长">
            <span><b>{{ runtime.hours }}</b>H</span>
            <span><b>{{ runtime.minutes }}</b>M</span>
            <span><b>{{ runtime.seconds }}</b>S</span>
          </span>
        </article>
        <article class="metric"><Icon name="cube" size="lg" class="metric-icon" /><strong class="metric-value">02</strong><span class="metric-label">模型提供商</span></article>
        <article class="metric"><Icon name="key" size="lg" class="metric-icon" /><strong class="metric-value">01</strong><span class="metric-label">统一 API Key</span></article>
        <article class="metric"><Icon name="chart" size="lg" class="metric-icon" /><strong class="metric-value">24</strong><span class="metric-label">小时持续服务</span></article>
        <article class="metric"><Icon name="sync" size="lg" class="metric-icon" /><strong class="metric-value">02</strong><span class="metric-label">兼容协议</span></article>
        <article class="metric"><Icon name="cloud" size="lg" class="metric-icon" /><strong class="metric-value compact">SSE</strong><span class="metric-label">流式响应</span></article>
      </section>
    </main>

    <section id="status" class="content-section status-section" aria-labelledby="status-title">
      <div class="section-inner">
        <header class="section-header reveal">
          <p class="section-kicker">CHANNEL HEALTH</p>
          <h2 id="status-title" class="section-title">渠道 <span>状态</span></h2>
          <p class="section-description">实时同步后台渠道监控，清晰展示模型可用率、响应延迟与最近检测记录。</p>

          <div class="status-toolbar" aria-live="polite">
            <span class="status-overall" :class="`is-${channelStatusSummary.tone}`">
              {{ channelStatusSummary.label }}
            </span>
            <span v-if="channelStatusLastUpdated" class="status-updated">
              更新于 {{ channelStatusLastUpdated }}
            </span>
            <button
              type="button"
              class="status-refresh"
              :disabled="channelStatusLoading"
              aria-label="刷新渠道状态"
              title="刷新渠道状态"
              @click="refreshChannelStatus(false)"
            >
              <Icon name="refresh" size="sm" :class="{ 'is-spinning': channelStatusLoading }" />
            </button>
          </div>
        </header>

        <div
          v-if="channelStatusLoading && channelMonitors.length === 0"
          class="status-grid"
          aria-label="正在加载渠道状态"
          aria-busy="true"
        >
          <article v-for="index in 3" :key="index" class="status-card status-card-skeleton">
            <div class="skeleton-line skeleton-heading"></div>
            <div class="skeleton-metrics"><span></span><span></span></div>
            <div class="skeleton-line skeleton-availability"></div>
            <div class="skeleton-timeline"></div>
          </article>
        </div>

        <div v-else-if="channelStatusError && channelMonitors.length === 0" class="status-state" role="status">
          <span class="status-state-icon"><Icon name="cloud" size="lg" /></span>
          <h3>暂时无法获取渠道状态</h3>
          <p>监控服务可能正在更新，请稍后重试。</p>
          <button type="button" class="status-retry" @click="refreshChannelStatus(false)">
            <Icon name="refresh" size="sm" />
            重新加载
          </button>
        </div>

        <div v-else-if="channelMonitors.length === 0" class="status-state" role="status">
          <span class="status-state-icon"><Icon name="chart" size="lg" /></span>
          <h3>暂无公开监控渠道</h3>
          <p>渠道配置完成后，实时状态会自动显示在这里。</p>
        </div>

        <template v-else>
          <p v-if="channelStatusError" class="status-stale-note" role="status">
            本次刷新失败，当前显示上一次获取的数据。
          </p>

          <div class="status-grid" aria-label="渠道服务实时状态">
            <article
              v-for="(monitor, index) in channelMonitors"
              :key="`${monitor.provider}-${monitor.name}-${monitor.primary_model}`"
              class="status-card"
              :style="{ '--reveal-index': index }"
            >
              <div class="status-card-head">
                <div class="status-identity">
                  <span class="status-provider-icon" :class="`provider-${monitor.provider}`">
                    <ProviderIcon :provider="monitor.provider" :size="21" />
                  </span>
                  <div class="status-identity-copy">
                    <h3 :title="monitor.name">{{ monitor.name }}</h3>
                    <div class="status-model-line">
                      <span class="status-provider-tag" :class="`provider-${monitor.provider}`">
                        {{ providerLabel(monitor.provider) }}
                      </span>
                      <code :title="monitor.primary_model || '未配置模型'">
                        {{ monitor.primary_model || '未配置模型' }}
                      </code>
                    </div>
                  </div>
                </div>
                <span class="status-badge" :class="`is-${statusTone(monitor.primary_status)}`">
                  {{ statusLabel(monitor.primary_status) }}
                </span>
              </div>

              <dl class="status-metrics">
                <div>
                  <dt><Icon name="bolt" size="xs" /> 对话延迟</dt>
                  <dd>{{ formatLatency(monitor.primary_latency_ms) }}<small v-if="monitor.primary_latency_ms != null">ms</small></dd>
                </div>
                <div>
                  <dt><Icon name="globe" size="xs" /> 端点 PING</dt>
                  <dd>{{ formatLatency(monitor.primary_ping_latency_ms) }}<small v-if="monitor.primary_ping_latency_ms != null">ms</small></dd>
                </div>
              </dl>

              <div class="status-availability">
                <span>可用性 · 7 天</span>
                <strong :class="`is-${availabilityTone(monitor.availability_7d, monitor.primary_status)}`">
                  {{ formatAvailability(monitor.availability_7d, monitor.primary_status) }}
                </strong>
              </div>

              <div class="status-history" :aria-label="timelineAriaLabel(monitor)">
                <div class="status-history-meta">
                  <span>最近 {{ monitor.timeline.length }} 次检测</span>
                  <span>过去 / 现在</span>
                </div>
                <div class="status-bars" aria-hidden="true">
                  <span
                    v-for="(bar, barIndex) in timelineBars(monitor)"
                    :key="barIndex"
                    class="status-bar"
                    :class="`is-${bar.tone}`"
                    :style="{ height: `${bar.height}%` }"
                    :title="bar.title"
                  ></span>
                </div>
              </div>
            </article>
          </div>
        </template>
      </div>
    </section>

    <section id="models" class="content-section models-section" aria-labelledby="models-title">
      <div class="section-inner">
        <header class="section-header reveal">
          <p class="section-kicker">SUPPORTED MODELS</p>
          <h2 id="models-title" class="section-title">支持 <span>模型</span></h2>
          <p class="section-description">当前仅接入 GPT 与 Claude 两类模型，统一使用同一套 API Key 和调用入口。</p>
        </header>

        <div class="supported-models-grid" aria-label="已支持的模型">
          <article class="supported-model-card reveal-card" style="--reveal-index:0"><span class="model-logo" style="--chip-color:#67e8d2"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-openai-mark" /></svg></span><h3>GPT-5.6 Sol</h3><p>OpenAI</p><span class="model-available">可用</span></article>
          <article class="supported-model-card reveal-card" style="--reveal-index:1"><span class="model-logo round" style="--chip-color:#55b8ff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-openai-mark" /></svg></span><h3>GPT-5.6 Terra</h3><p>OpenAI</p><span class="model-available">可用</span></article>
          <article class="supported-model-card reveal-card" style="--reveal-index:2"><span class="model-logo" style="--chip-color:#8a9cff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-openai-mark" /></svg></span><h3>GPT-5.6 Luna</h3><p>OpenAI</p><span class="model-available">可用</span></article>
          <article class="supported-model-card reveal-card" style="--reveal-index:3"><span class="model-logo round" style="--chip-color:#a879f4;--chip-ink:#fff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-openai-mark" /></svg></span><h3>GPT-image-2</h3><p>OpenAI</p><span class="model-available">可用</span></article>
          <article class="supported-model-card reveal-card" style="--reveal-index:4"><span class="model-logo" style="--chip-color:#ef906d;--chip-ink:#fff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-claude-mark" /></svg></span><h3>Claude Fable 5</h3><p>Anthropic</p><span class="model-available">可用</span></article>
          <article class="supported-model-card reveal-card" style="--reveal-index:5"><span class="model-logo round" style="--chip-color:#f29b76;--chip-ink:#fff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-claude-mark" /></svg></span><h3>Claude Opus 4.8</h3><p>Anthropic</p><span class="model-available">可用</span></article>
          <article class="supported-model-card reveal-card" style="--reveal-index:6"><span class="model-logo" style="--chip-color:#f7b18f;--chip-ink:#fff"><svg viewBox="0 0 24 24" aria-hidden="true"><use href="#home-claude-mark" /></svg></span><h3>Claude Sonnet 5</h3><p>Anthropic</p><span class="model-available">可用</span></article>
        </div>
      </div>
    </section>

    <a class="support" :href="docUrl" target="_blank" rel="noopener noreferrer" aria-label="在线支持" title="在线支持"><Icon name="chat" size="lg" /></a>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import ProviderIcon from '@/components/user/monitor/ProviderIcon.vue'
import {
  listPublic as listPublicChannelMonitors,
  type MonitorStatus,
  type PublicMonitorView,
} from '@/api/channelMonitor'
import { DEFAULT_SITE_LOGO, resolveSiteLogo, resolveSiteName } from '@/config/branding'
import { useAppStore, useAuthStore } from '@/stores'
import { summarizeChannelStatus } from '@/utils/channelStatusSummary'
import { sanitizeUrl } from '@/utils/url'

const authStore = useAuthStore()
const appStore = useAppStore()
const router = useRouter()
const { t } = useI18n()

const siteName = computed(() => resolveSiteName(appStore.cachedPublicSettings?.site_name || appStore.siteName))
const siteLogo = computed(() => sanitizeUrl(
  resolveSiteLogo(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo),
  { allowRelative: true, allowDataUrl: true }
) || DEFAULT_SITE_LOGO)
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || 'https://docs.recurdream.com'))
const homeContent = computed(() => {
  const content = appStore.cachedPublicSettings?.home_content
  return typeof content === 'string' && content.trim() ? content : ''
})
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const compactHomeEnabled = computed(() => appStore.cachedPublicSettings?.compact_home_enabled === true)
const currentYear = computed(() => new Date().getFullYear())
const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const isDark = ref(true)
const motionReady = ref(false)
const typewriterText = ref('')
const runtime = reactive({ days: 0, hours: 0, minutes: 0, seconds: 0 })
const channelMonitors = ref<PublicMonitorView[]>([])
const channelStatusLoading = ref(true)
const channelStatusError = ref(false)
const channelStatusLastUpdated = ref('')

type ChannelStatusTone = 'operational' | 'degraded' | 'failed' | 'neutral'

interface StatusTimelineBar {
  tone: ChannelStatusTone | 'empty'
  height: number
  title: string
}

const channelStatusSummary = computed<{ label: string; tone: ChannelStatusTone }>(() => {
  if (channelStatusLoading.value && channelMonitors.value.length === 0) {
    return { label: '正在检测', tone: 'neutral' }
  }
  if (channelStatusError.value && channelMonitors.value.length === 0) {
    return { label: '状态未知', tone: 'neutral' }
  }
  if (channelMonitors.value.length === 0) {
    return { label: '暂无监控', tone: 'neutral' }
  }
  return summarizeChannelStatus(channelMonitors.value.map(item => item.primary_status))
})

const launchedAt = new Date('2025-12-18T02:26:18Z').getTime()
const command = 'https://api.recurdream.com'
const channelStatusRefreshMs = 60_000
const channelTimelineLength = 60
const reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches
let runtimeInterval: number | undefined
let typewriterTimer: number | undefined
let channelStatusInterval: number | undefined
let channelStatusAbortController: AbortController | undefined
let revealObserver: IntersectionObserver | undefined
let commandIndex = 0
let deleting = false
let previousScrollBehavior = ''
let loginPagePrefetch: Promise<unknown> | undefined
const loginTransitionInProgress = ref(false)

function setTheme(dark: boolean) {
  isDark.value = dark
  document.documentElement.classList.toggle('dark', dark)
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (!savedTheme) setTheme(true)
  else setTheme(savedTheme === 'dark')
}

function toggleTheme() {
  setTheme(!isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

async function refreshChannelStatus(silent = false) {
  channelStatusAbortController?.abort()
  const controller = new AbortController()
  channelStatusAbortController = controller
  if (!silent) {
    channelStatusLoading.value = true
    channelStatusError.value = false
  }

  try {
    const response = await listPublicChannelMonitors({ signal: controller.signal })
    if (controller.signal.aborted || channelStatusAbortController !== controller) return
    channelMonitors.value = response.items ?? []
    channelStatusError.value = false
    channelStatusLastUpdated.value = new Date().toLocaleTimeString('zh-CN', {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      hour12: false,
    })
  } catch (error: unknown) {
    const requestError = error as { name?: string; code?: string }
    if (requestError.name === 'AbortError' || requestError.code === 'ERR_CANCELED') return
    channelStatusError.value = true
  } finally {
    if (channelStatusAbortController === controller) {
      channelStatusLoading.value = false
      channelStatusAbortController = undefined
    }
  }
}

function providerLabel(provider: string): string {
  const labels: Record<string, string> = {
    openai: 'OpenAI',
    anthropic: 'Anthropic',
    gemini: 'Gemini',
    grok: 'Grok',
  }
  return labels[provider] ?? provider
}

function statusTone(status: MonitorStatus | ''): ChannelStatusTone {
  if (status === 'operational') return 'operational'
  if (status === 'degraded') return 'degraded'
  if (status === 'failed' || status === 'error') return 'failed'
  return 'neutral'
}

function statusLabel(status: MonitorStatus | ''): string {
  const labels: Record<MonitorStatus, string> = {
    operational: '正常',
    degraded: '降级',
    failed: '故障',
    error: '异常',
  }
  return status ? labels[status] : '待检测'
}

function formatLatency(value: number | null): string {
  return value == null ? '--' : String(Math.round(value))
}

function availabilityTone(value: number, status: MonitorStatus | ''): ChannelStatusTone {
  if (!status) return 'neutral'
  if (value >= 75) return 'operational'
  if (value >= 50) return 'degraded'
  return 'failed'
}

function formatAvailability(value: number, status: MonitorStatus | ''): string {
  if (!status || Number.isNaN(value)) return '--'
  return `${value.toFixed(2)}%`
}

function timelineBars(monitor: PublicMonitorView): StatusTimelineBar[] {
  const points = [...(monitor.timeline ?? [])].slice(0, channelTimelineLength).reverse()
  const bars: StatusTimelineBar[] = Array.from(
    { length: Math.max(0, channelTimelineLength - points.length) },
    () => ({ tone: 'empty', height: 16, title: '' })
  )

  for (const point of points) {
    const tone = statusTone(point.status)
    const checkedAt = Number.isNaN(Date.parse(point.checked_at))
      ? '未知时间'
      : new Date(point.checked_at).toLocaleString('zh-CN', { hour12: false })
    bars.push({
      tone,
      height: tone === 'operational' ? 100 : tone === 'degraded' ? 66 : tone === 'neutral' ? 16 : 36,
      title: `${checkedAt} · ${statusLabel(point.status)} · ${formatLatency(point.latency_ms)}ms`,
    })
  }
  return bars
}

function timelineAriaLabel(monitor: PublicMonitorView): string {
  if (monitor.timeline.length === 0) return `${monitor.name} 暂无检测记录`
  const operational = monitor.timeline.filter(point => point.status === 'operational').length
  const degraded = monitor.timeline.filter(point => point.status === 'degraded').length
  const failed = monitor.timeline.length - operational - degraded
  return `${monitor.name} 最近 ${monitor.timeline.length} 次检测：正常 ${operational} 次，降级 ${degraded} 次，异常 ${failed} 次`
}

function prefetchLoginPage(): void {
  if (!isAuthenticated.value && !loginPagePrefetch) {
    loginPagePrefetch = import('@/views/auth/LoginView.vue')
  }
}

async function handlePrimaryNavigation(event: MouseEvent): Promise<void> {
  if (
    isAuthenticated.value ||
    event.button !== 0 ||
    event.ctrlKey ||
    event.metaKey ||
    event.shiftKey ||
    event.altKey
  ) {
    return
  }

  event.preventDefault()
  if (loginTransitionInProgress.value) return

  prefetchLoginPage()
  loginTransitionInProgress.value = true
  try {
    if (!reducedMotion) {
      await new Promise<void>((resolve) => {
        window.setTimeout(resolve, 180)
      })
    }
    await router.push('/login')
  } finally {
    loginTransitionInProgress.value = false
  }
}

function updateRuntime() {
  const elapsed = Math.max(0, Math.floor((Date.now() - launchedAt) / 1000))
  runtime.days = Math.floor(elapsed / 86400)
  runtime.hours = Math.floor((elapsed % 86400) / 3600)
  runtime.minutes = Math.floor((elapsed % 3600) / 60)
  runtime.seconds = elapsed % 60
}

function typeCommand() {
  if (reducedMotion) {
    typewriterText.value = command
    return
  }

  commandIndex += deleting ? -1 : 1
  typewriterText.value = command.slice(0, commandIndex)

  let delay = deleting ? 28 : 58
  if (!deleting && commandIndex === command.length) {
    deleting = true
    delay = 1800
  } else if (deleting && commandIndex === 0) {
    deleting = false
    delay = 560
  }
  typewriterTimer = window.setTimeout(typeCommand, delay)
}

function initRevealMotion() {
  const revealElements = document.querySelectorAll<HTMLElement>('.home-page .reveal, .home-page .reveal-card')
  if (reducedMotion || !('IntersectionObserver' in window)) {
    revealElements.forEach((element) => element.classList.add('is-visible'))
    return
  }

  motionReady.value = true
  revealObserver = new IntersectionObserver((entries) => {
    entries.forEach((entry) => {
      if (!entry.isIntersecting) return
      entry.target.classList.add('is-visible')
      revealObserver?.unobserve(entry.target)
    })
  }, { threshold: 0.14, rootMargin: '0px 0px -8% 0px' })
  revealElements.forEach((element) => revealObserver?.observe(element))
}

initTheme()

onMounted(() => {
  authStore.checkAuth()
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings()

  previousScrollBehavior = document.documentElement.style.scrollBehavior
  document.documentElement.style.scrollBehavior = reducedMotion ? 'auto' : 'smooth'
  updateRuntime()
  runtimeInterval = window.setInterval(updateRuntime, 1000)
  void refreshChannelStatus(false)
  channelStatusInterval = window.setInterval(() => {
    if (!document.hidden) void refreshChannelStatus(true)
  }, channelStatusRefreshMs)
  typeCommand()
  requestAnimationFrame(initRevealMotion)
})

onBeforeUnmount(() => {
  if (runtimeInterval !== undefined) window.clearInterval(runtimeInterval)
  if (typewriterTimer !== undefined) window.clearTimeout(typewriterTimer)
  if (channelStatusInterval !== undefined) window.clearInterval(channelStatusInterval)
  channelStatusAbortController?.abort()
  revealObserver?.disconnect()
  document.documentElement.style.scrollBehavior = previousScrollBehavior
})
</script>

<style scoped>
.home-page {
  --page: #f8fafc;
  --nav: rgba(255, 255, 255, 0.86);
  --surface: rgba(255, 255, 255, 0.84);
  --surface-hover: rgba(255, 255, 255, 0.96);
  --text: #0f172a;
  --text-2: #475569;
  --text-3: #7d8aa0;
  --line: rgba(203, 213, 225, 0.68);
  --line-strong: rgba(148, 163, 184, 0.8);
  --primary: #3073f1;
  --primary-2: #3c7ff7;
  --orange: #f47b38;
  --success: #22c982;
  --warning: #f59e0b;
  --danger: #ef5350;
  --shadow: 0 18px 44px rgba(62, 80, 125, 0.1);
  position: relative;
  width: 100%;
  min-width: 320px;
  min-height: 100dvh;
  overflow-x: clip;
  background: var(--page);
  color: var(--text);
  font-family: Inter, "SF Pro Display", "PingFang SC", "Microsoft YaHei", system-ui, sans-serif;
  letter-spacing: 0;
  transition: background-color 0.28s ease, color 0.28s ease, opacity 180ms ease-in;
}

.home-page.dark-mode {
  --page: #0b0b10;
  --nav: rgba(17, 24, 39, 0.86);
  --surface: rgba(255, 255, 255, 0.078);
  --surface-hover: rgba(255, 255, 255, 0.122);
  --text: #f8fafc;
  --text-2: #94a3b8;
  --text-3: #748096;
  --line: rgba(255, 255, 255, 0.1);
  --line-strong: rgba(255, 255, 255, 0.2);
  --shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
}

.home-page::before {
  position: absolute;
  z-index: 0;
  top: 0;
  right: 0;
  left: 0;
  height: max(100dvh, 760px);
  pointer-events: none;
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
.home-page.dark-mode::before { opacity: 1; }
.home-page.leaving-for-login { opacity: 0; }

.home-page a { color: inherit; text-decoration: none; }
.home-page button { color: inherit; font: inherit; }
.home-page a,
.home-page button { touch-action: manipulation; }
.home-page :focus-visible { outline: 3px solid rgba(74, 130, 255, 0.48); outline-offset: 3px; }
.brand-symbols { position: absolute; width: 0; height: 0; overflow: hidden; }

.navbar {
  position: fixed;
  z-index: 20;
  top: 16px;
  right: 16px;
  left: 16px;
  display: flex;
  width: auto;
  max-width: 1152px;
  height: 66px;
  align-items: center;
  justify-content: space-between;
  margin: 0 auto;
  border: 1px solid var(--line);
  border-radius: 16px;
  background: var(--nav);
  padding: 0 31px;
  box-shadow: var(--shadow);
  backdrop-filter: blur(24px);
}
.brand { display: inline-flex; width: fit-content; align-items: center; gap: 11px; font-size: 16px; font-weight: 720; white-space: nowrap; }
.brand img { width: 31px; height: 31px; border-radius: 8px; object-fit: contain; }
.nav-links { display: flex; align-items: center; gap: 24px; color: var(--text-2); font-size: 13px; font-weight: 530; }
.nav-links a { padding: 14px 0; transition: color 0.2s ease, transform 0.2s ease; }
.nav-links a:hover { color: var(--text); transform: translateY(-1px); }
.nav-actions { display: flex; align-items: center; justify-content: flex-end; gap: 10px; }
.theme-button { display: inline-grid; width: 44px; height: 44px; place-items: center; border: 0; border-radius: 8px; background: transparent; cursor: pointer; transition: background-color 0.2s ease, color 0.2s ease; }
.theme-button:hover { background: rgba(127, 127, 127, 0.1); }
.service-pill { display: inline-flex; width: 88px; min-height: 34px; align-items: center; justify-content: center; gap: 7px; border: 1px solid var(--line); border-radius: 999px; background: var(--surface); color: var(--text-2); font-size: 11px; font-weight: 620; }
.service-pill::before { width: 7px; height: 7px; flex: 0 0 7px; border-radius: 50%; background: currentColor; content: ""; box-shadow: 0 0 0 4px rgba(148, 163, 184, 0.1); }
.service-pill.is-operational { color: #169c68; }
.service-pill.is-degraded { color: #c67a08; }
.service-pill.is-failed { color: #dc4242; }
.home-page.dark-mode .service-pill.is-operational { color: #35dc99; }
.home-page.dark-mode .service-pill.is-degraded { color: #f9b83f; }
.home-page.dark-mode .service-pill.is-failed { color: #ff7777; }
.nav-cta { display: inline-flex; width: 82px; height: 34px; min-height: 34px; align-items: center; justify-content: center; border-radius: 10px; background: #2563eb; color: #fff !important; font-size: 13px; font-weight: 680; box-shadow: 0 8px 20px rgba(52, 120, 246, 0.24); transition: transform 0.2s ease, background-color 0.2s ease; }
.nav-cta:hover { background: rgba(37, 99, 235, 0.9); transform: translateY(-2px); }

.home-hero {
  position: relative;
  z-index: 1;
  display: flex;
  width: min(calc(100% - 40px), 1112px);
  height: 100dvh;
  min-height: 760px;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  margin: 0 auto;
  padding: 96px 0 64px;
  scroll-margin-top: 82px;
  text-align: center;
}
.model-row { display: flex; min-height: 35px; align-items: center; justify-content: center; gap: 8px; animation: enter 0.55s ease-out both; }
.model-chip { display: inline-flex; height: 35px; align-items: center; gap: 7px; border: 1px solid var(--line); border-radius: 999px; background: rgba(255, 255, 255, 0.86); padding: 0 12px; color: var(--text); font-size: 11px; font-weight: 640; box-shadow: 0 8px 20px rgba(0, 0, 0, 0.11); }
.home-page.dark-mode .model-chip { background: rgba(17, 24, 39, 0.88); color: #e9eef9; }
.model-logo { display: inline-grid; width: 20px; height: 20px; flex: 0 0 20px; place-items: center; border-radius: 6px; background: var(--chip-color); color: var(--chip-ink, #07111c); }
.model-logo.round { border-radius: 50%; }
.model-logo svg { width: 12px; height: 12px; fill: currentColor; }
.brand-title { margin: 27px 0 0; background: linear-gradient(92deg, #2878ff 4%, #5d87df 51%, #f07731 96%); background-clip: text; color: transparent; font-size: 68px; font-weight: 740; line-height: 1; animation: enter 0.62s 0.05s ease-out both; }
.home-hero h1 { margin: 16px 0 0; font-size: 54px; font-weight: 740; line-height: 1.08; animation: enter 0.64s 0.09s ease-out both; }
.description { max-width: 630px; margin: 24px 0 0; color: var(--text-2); font-size: 18px; line-height: 1.55; animation: enter 0.64s 0.13s ease-out both; }
.terminal { display: flex; width: 385px; height: 44px; align-items: center; justify-content: center; margin-top: 30px; border: 1px solid rgba(80, 122, 199, 0.13); border-radius: 8px; background: #111827; color: #edf4ff; font-family: "SFMono-Regular", Consolas, monospace; font-size: 12px; font-weight: 650; animation: enter 0.64s 0.17s ease-out both; }
.terminal-prompt { margin-right: 8px; color: var(--orange); }
.terminal-runtime { min-width: 0; font-variant-numeric: tabular-nums; }
.cursor { width: 7px; height: 15px; background: #f5f7fb; animation: blink 0.9s steps(1) infinite; }
.actions { display: flex; align-items: center; gap: 16px; margin-top: 40px; animation: enter 0.64s 0.21s ease-out both; }
.button { display: inline-flex; min-width: 175px; min-height: 62px; align-items: center; justify-content: center; border: 1px solid transparent; border-radius: 12px; font-size: 16px; font-weight: 670; transition: transform 0.2s ease, background-color 0.2s ease, border-color 0.2s ease; }
.button:hover { transform: translateY(-3px); }
.button-primary { background: linear-gradient(90deg, #2563eb, #3b82f6); color: #fff !important; box-shadow: 0 12px 28px rgba(37, 99, 235, 0.22); }
.button-secondary { border-color: var(--line); background: var(--surface); color: var(--text); }
.button-secondary:hover { border-color: var(--line-strong); background: var(--surface-hover); }
.metrics { display: grid; width: 1072px; max-width: 100%; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 16px; margin-top: 64px; animation: enter 0.72s 0.25s ease-out both; }
.metric { display: flex; height: 139px; flex-direction: column; align-items: center; justify-content: center; border: 1px solid var(--line); border-radius: 14px; background: var(--surface); box-shadow: 0 12px 28px rgba(2, 6, 23, 0.1); transition: transform 0.22s ease, background-color 0.22s ease, border-color 0.22s ease; }
.metric:hover { border-color: var(--line-strong); background: var(--surface-hover); transform: translateY(-5px); }
.metric-icon { margin-bottom: 7px; color: var(--primary); }
.metric-value { display: flex; min-height: 36px; align-items: baseline; justify-content: center; gap: 3px; color: var(--primary); font-size: 30px; font-variant-numeric: tabular-nums; font-weight: 720; line-height: 1; }
.metric-value.compact { font-size: 24px; }
.metric-value small { color: var(--text-2); font-size: 10px; font-weight: 620; }
.metric-label { margin-top: 2px; color: var(--text-2); font-size: 12px; font-weight: 540; white-space: nowrap; }
.metric-uptime { padding: 12px 8px 10px; }
.uptime-title { display: inline-flex; align-items: center; gap: 7px; margin-bottom: 7px; color: var(--text-2); font-size: 11px; font-weight: 620; white-space: nowrap; }
.uptime-title::before { width: 7px; height: 7px; flex: 0 0 7px; border-radius: 50%; background: var(--success); content: ""; box-shadow: 0 0 0 4px rgba(71, 220, 148, 0.1); }
.metric-uptime .metric-value { min-height: 35px; font-size: 34px; }
.metric-uptime .metric-value small { font-size: 11px; }
.runtime-compact { display: inline-flex; align-items: baseline; gap: 5px; margin-top: 5px; color: var(--text-2); font-family: "SFMono-Regular", Consolas, monospace; font-size: 9px; font-variant-numeric: tabular-nums; white-space: nowrap; }
.runtime-compact span { display: inline-flex; align-items: baseline; gap: 1px; }
.runtime-compact b { color: var(--orange); font-size: 11px; font-weight: 800; }

.content-section { position: relative; z-index: 1; display: flex; min-height: 100dvh; align-items: center; scroll-margin-top: 82px; border-top: 1px solid var(--line); padding: 122px 24px 96px; }
.status-section { background: rgba(255, 255, 255, 0.32); }
.models-section { background: rgba(241, 245, 249, 0.44); }
.home-page.dark-mode .status-section { background: rgba(6, 8, 14, 0.42); }
.home-page.dark-mode .models-section { background: rgba(15, 20, 32, 0.34); }
.section-inner { width: min(100%, 1280px); margin: 0 auto; }
.section-header { max-width: 760px; margin: 0 auto 56px; text-align: center; }
.section-kicker { margin: 0 0 12px; color: var(--primary-2); font-size: 13px; font-weight: 700; }
.section-title { margin: 0; font-size: 46px; font-weight: 740; line-height: 1.12; }
.section-title span { background: linear-gradient(92deg, #2878ff 4%, #5d87df 51%, #f07731 96%); background-clip: text; color: transparent; }
.section-description { max-width: 680px; margin: 18px auto 0; color: var(--text-2); font-size: 17px; line-height: 1.65; }
.status-toolbar { display: flex; min-height: 44px; align-items: center; justify-content: center; gap: 10px; margin-top: 24px; }
.status-overall { display: inline-flex; min-height: 32px; align-items: center; gap: 7px; border: 1px solid var(--line); border-radius: 999px; background: var(--surface); padding: 0 12px; color: var(--text-2); font-size: 12px; font-weight: 680; }
.status-overall::before { width: 7px; height: 7px; flex: 0 0 7px; border-radius: 50%; background: currentColor; content: ""; }
.status-overall.is-operational { border-color: rgba(34, 201, 130, 0.24); background: rgba(34, 201, 130, 0.08); color: #148358; }
.status-overall.is-degraded { border-color: rgba(245, 158, 11, 0.28); background: rgba(245, 158, 11, 0.09); color: #ad6906; }
.status-overall.is-failed { border-color: rgba(239, 83, 80, 0.28); background: rgba(239, 83, 80, 0.09); color: #c43b3b; }
.home-page.dark-mode .status-overall.is-operational { color: #35dc99; }
.home-page.dark-mode .status-overall.is-degraded { color: #f9b83f; }
.home-page.dark-mode .status-overall.is-failed { color: #ff7777; }
.status-updated { color: var(--text-3); font-family: "SFMono-Regular", Consolas, monospace; font-size: 11px; font-variant-numeric: tabular-nums; }
.status-refresh { display: inline-grid; width: 44px; height: 44px; place-items: center; border: 1px solid var(--line); border-radius: 9px; background: var(--surface); color: var(--text-2); cursor: pointer; transition: border-color 0.2s ease, background-color 0.2s ease, color 0.2s ease; }
.status-refresh:hover:not(:disabled) { border-color: var(--line-strong); background: var(--surface-hover); color: var(--primary-2); }
.status-refresh:disabled { cursor: wait; opacity: 0.62; }
.status-refresh .is-spinning { animation: spin 0.8s linear infinite; }
.status-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 24px; }
.status-card,
.supported-model-card { border: 1px solid var(--line); background: var(--surface); box-shadow: 0 12px 28px rgba(2, 6, 23, 0.08); transition: transform 0.24s ease, border-color 0.24s ease, background-color 0.24s ease; }
.supported-model-card:hover { border-color: var(--line-strong); background: var(--surface-hover); transform: translateY(-4px); }
.status-card { min-width: 0; min-height: 336px; border-radius: 14px; padding: 24px; animation: status-card-enter 0.34s ease-out both; animation-delay: calc(var(--reveal-index, 0) * 45ms); }
.status-card:hover:not(.status-card-skeleton) { border-color: var(--line-strong); background: var(--surface-hover); transform: translateY(-3px); }
.status-card-head { display: flex; min-width: 0; align-items: flex-start; justify-content: space-between; gap: 14px; }
.status-identity { display: flex; min-width: 0; align-items: flex-start; gap: 12px; }
.status-provider-icon { display: grid; width: 42px; height: 42px; flex: 0 0 42px; place-items: center; border: 1px solid rgba(48, 115, 241, 0.2); border-radius: 10px; background: rgba(48, 115, 241, 0.09); color: var(--primary-2); }
.status-provider-icon.provider-openai { border-color: rgba(16, 185, 129, 0.22); background: rgba(16, 185, 129, 0.1); color: #13a879; }
.status-provider-icon.provider-anthropic { border-color: rgba(249, 115, 22, 0.24); background: rgba(249, 115, 22, 0.1); color: #e76f20; }
.status-provider-icon.provider-gemini { border-color: rgba(14, 165, 233, 0.24); background: rgba(14, 165, 233, 0.1); color: #168ac3; }
.status-provider-icon.provider-grok { border-color: rgba(100, 116, 139, 0.3); background: rgba(100, 116, 139, 0.11); color: var(--text-2); }
.status-identity-copy { min-width: 0; padding-top: 1px; }
.status-card h3 { max-width: 100%; overflow: hidden; margin: 0; color: var(--text); font-size: 17px; font-weight: 720; line-height: 1.35; text-overflow: ellipsis; white-space: nowrap; }
.status-model-line { display: flex; min-width: 0; align-items: center; gap: 7px; margin-top: 6px; }
.status-provider-tag { display: inline-flex; min-height: 22px; flex: 0 0 auto; align-items: center; border-radius: 6px; padding: 0 7px; background: rgba(48, 115, 241, 0.1); color: var(--primary-2); font-size: 11px; font-weight: 680; }
.status-provider-tag.provider-openai { background: rgba(16, 185, 129, 0.11); color: #13875f; }
.status-provider-tag.provider-anthropic { background: rgba(249, 115, 22, 0.11); color: #bd5817; }
.status-provider-tag.provider-gemini { background: rgba(14, 165, 233, 0.11); color: #167eae; }
.status-provider-tag.provider-grok { background: rgba(100, 116, 139, 0.12); color: var(--text-2); }
.home-page.dark-mode .status-provider-tag.provider-openai { color: #55ddb0; }
.home-page.dark-mode .status-provider-tag.provider-anthropic { color: #ffac70; }
.home-page.dark-mode .status-provider-tag.provider-gemini { color: #67cbf4; }
.status-model-line code { min-width: 0; overflow: hidden; color: var(--text-3); font-family: "SFMono-Regular", Consolas, monospace; font-size: 11px; text-overflow: ellipsis; white-space: nowrap; }
.status-badge { display: inline-flex; min-height: 28px; flex: 0 0 auto; align-items: center; gap: 7px; border: 1px solid var(--line); border-radius: 999px; padding: 0 10px; color: var(--text-2); font-size: 11px; font-weight: 680; }
.status-badge::before { width: 7px; height: 7px; flex: 0 0 7px; border-radius: 50%; background: currentColor; content: ""; }
.status-badge.is-operational { border-color: rgba(34, 201, 130, 0.22); background: rgba(34, 201, 130, 0.09); color: #148358; }
.status-badge.is-degraded { border-color: rgba(245, 158, 11, 0.25); background: rgba(245, 158, 11, 0.09); color: #ad6906; }
.status-badge.is-failed { border-color: rgba(239, 83, 80, 0.25); background: rgba(239, 83, 80, 0.09); color: #c43b3b; }
.home-page.dark-mode .status-badge.is-operational { color: #35dc99; }
.home-page.dark-mode .status-badge.is-degraded { color: #f9b83f; }
.home-page.dark-mode .status-badge.is-failed { color: #ff7777; }
.status-metrics { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); margin: 24px 0 0; border-top: 1px solid var(--line); border-bottom: 1px solid var(--line); padding: 18px 0; }
.status-metrics > div { min-width: 0; padding: 0 14px; }
.status-metrics > div:first-child { padding-left: 0; }
.status-metrics > div + div { border-left: 1px solid var(--line); padding-right: 0; }
.status-metrics dt { display: flex; align-items: center; gap: 6px; color: var(--text-3); font-size: 11px; font-weight: 650; }
.status-metrics dd { overflow: hidden; margin: 8px 0 0; color: var(--text); font-family: "SFMono-Regular", Consolas, monospace; font-size: 21px; font-variant-numeric: tabular-nums; font-weight: 720; line-height: 1; text-overflow: ellipsis; }
.status-metrics dd small { margin-left: 3px; color: var(--text-3); font-size: 11px; font-weight: 550; }
.status-availability { display: flex; min-height: 50px; align-items: flex-end; justify-content: space-between; gap: 16px; padding-top: 16px; }
.status-availability > span { color: var(--text-3); font-size: 11px; font-weight: 620; }
.status-availability strong { font-size: 29px; font-variant-numeric: tabular-nums; font-weight: 760; line-height: 1; }
.status-availability strong.is-operational { color: #16a56f; }
.status-availability strong.is-degraded { color: #c67a08; }
.status-availability strong.is-failed { color: #dc4242; }
.status-availability strong.is-neutral { color: var(--text-3); }
.home-page.dark-mode .status-availability strong.is-operational { color: #35d485; }
.home-page.dark-mode .status-availability strong.is-degraded { color: #f0b62f; }
.home-page.dark-mode .status-availability strong.is-failed { color: #ff6969; }
.status-history { margin-top: 17px; }
.status-history-meta { display: flex; align-items: center; justify-content: space-between; gap: 12px; margin-bottom: 9px; color: var(--text-3); font-family: "SFMono-Regular", Consolas, monospace; font-size: 11px; }
.status-bars { display: grid; height: 22px; grid-template-columns: repeat(60, minmax(2px, 1fr)); align-items: end; gap: 2px; }
.status-bar { min-width: 2px; border-radius: 2px; background: var(--text-3); opacity: 0.35; }
.status-bar.is-operational { background: #20bd7d; opacity: 1; }
.status-bar.is-degraded { background: #f0a51c; opacity: 1; }
.status-bar.is-failed { background: #ee5151; opacity: 1; }
.status-bar.is-neutral,
.status-bar.is-empty { background: var(--text-3); opacity: 0.25; }
.status-stale-note { width: fit-content; margin: -24px auto 18px; color: #b56f09; font-size: 12px; text-align: center; }
.home-page.dark-mode .status-stale-note { color: #f1b84a; }
.status-state { display: flex; min-height: 300px; flex-direction: column; align-items: center; justify-content: center; padding: 32px 20px; text-align: center; }
.status-state-icon { display: grid; width: 52px; height: 52px; place-items: center; border: 1px solid rgba(48, 115, 241, 0.22); border-radius: 12px; background: rgba(48, 115, 241, 0.09); color: var(--primary-2); }
.status-state h3 { margin: 18px 0 0; font-size: 18px; font-weight: 720; }
.status-state p { margin: 8px 0 0; color: var(--text-2); font-size: 14px; }
.status-retry { display: inline-flex; min-height: 44px; align-items: center; gap: 8px; margin-top: 20px; border: 1px solid var(--line); border-radius: 9px; background: var(--surface); padding: 0 16px; color: var(--text); cursor: pointer; font-size: 13px; font-weight: 680; }
.status-retry:hover { border-color: var(--line-strong); background: var(--surface-hover); }
.status-card-skeleton { animation: none; }
.skeleton-line,
.skeleton-metrics span,
.skeleton-timeline { display: block; border-radius: 6px; background: rgba(148, 163, 184, 0.18); animation: skeleton-pulse 1.4s ease-in-out infinite; }
.skeleton-heading { width: 62%; height: 42px; }
.skeleton-metrics { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 18px; margin-top: 24px; }
.skeleton-metrics span { height: 66px; }
.skeleton-availability { width: 100%; height: 46px; margin-top: 18px; }
.skeleton-timeline { width: 100%; height: 22px; margin-top: 22px; }
.supported-models-grid { display: flex; flex-wrap: wrap; justify-content: center; gap: 24px; }
.supported-model-card { display: flex; width: calc((100% - 72px) / 4); min-height: 210px; flex-direction: column; align-items: center; justify-content: center; border-radius: 14px; padding: 28px 20px; text-align: center; }
.supported-model-card .model-logo { width: 48px; height: 48px; flex-basis: 48px; border-radius: 13px; }
.supported-model-card .model-logo.round { border-radius: 50%; }
.supported-model-card .model-logo svg { width: 27px; height: 27px; }
.supported-model-card h3 { margin: 20px 0 0; font-size: 18px; font-weight: 700; }
.supported-model-card p { margin: 7px 0 0; color: var(--text-2); font-size: 13px; }
.model-available { display: inline-flex; min-height: 28px; align-items: center; gap: 6px; margin-top: 18px; border: 1px solid var(--line); border-radius: 999px; padding: 0 10px; color: var(--text-2); font-size: 11px; font-weight: 620; }
.model-available::before { width: 6px; height: 6px; border-radius: 50%; background: var(--success); content: ""; }
.motion-ready .reveal,
.motion-ready .reveal-card { opacity: 0; transform: translateY(20px); }
.motion-ready .reveal.is-visible,
.motion-ready .reveal-card.is-visible { opacity: 1; transform: translateY(0); transition: opacity 0.42s ease-out, transform 0.42s ease-out; }
.motion-ready .reveal-card.is-visible { transition-delay: calc(var(--reveal-index, 0) * 45ms); }
.support { position: fixed; z-index: 30; right: 22px; bottom: 20px; display: grid; width: 48px; height: 48px; place-items: center; border: 1px solid rgba(93, 151, 255, 0.42); border-radius: 50%; background: #1769e8; color: #fff !important; box-shadow: 0 12px 30px rgba(18, 77, 180, 0.35); transition: transform 0.2s ease; }
.support:hover { transform: translateY(-3px) scale(1.03); }

@keyframes enter { from { opacity: 0; transform: translateY(12px); } to { opacity: 1; transform: translateY(0); } }
@keyframes status-card-enter { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
@keyframes skeleton-pulse { 0%, 100% { opacity: 0.5; } 50% { opacity: 1; } }
@keyframes spin { to { transform: rotate(360deg); } }
@keyframes blink { 0%, 46% { opacity: 1; } 47%, 100% { opacity: 0; } }

@media (min-width: 1600px) {
  .brand-title { margin-top: 39px; }
}

@media (max-width: 1050px) {
  .navbar { padding: 0 20px; }
  .nav-links { gap: 18px; }
  .service-pill { display: none; }
  .home-hero { width: min(calc(100% - 32px), 940px); }
  .brand-title { font-size: 61px; }
  .home-hero h1 { font-size: 48px; }
  .metrics { grid-template-columns: repeat(6, minmax(120px, 1fr)); gap: 10px; }
  .metric { height: 128px; }
  .status-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .supported-model-card { width: calc((100% - 48px) / 3); }
}

@media (max-width: 720px) {
  .home-page::before { height: max(100dvh, 700px); }
  .navbar { top: 10px; right: 10px; left: 10px; display: grid; width: auto; height: 58px; grid-template-columns: 1fr auto; margin: 0; border-radius: 12px; padding: 0 12px; }
  .brand { font-size: 14px; }
  .nav-links,
  .service-pill { display: none; }
  .nav-actions { gap: 6px; }
  .theme-button { width: 42px; height: 42px; }
  .nav-cta { width: 82px; min-height: 38px; font-size: 12px; }
  .home-hero { width: calc(100% - 24px); height: 100dvh; min-height: 700px; padding: 80px 0 48px; }
  .model-row { min-height: 64px; flex-wrap: wrap; gap: 6px; }
  .model-chip { height: 29px; padding: 0 9px; font-size: 9px; }
  .model-logo { width: 17px; height: 17px; flex-basis: 17px; }
  .model-logo svg { width: 10px; height: 10px; }
  .brand-title { margin-top: 14px; font-size: 43px; }
  .home-hero h1 { margin-top: 10px; font-size: 34px; }
  .description { max-width: 355px; margin-top: 14px; font-size: 14px; }
  .terminal { width: min(100%, 350px); height: 42px; margin-top: 18px; font-size: 10px; }
  .actions { width: min(100%, 350px); gap: 10px; margin-top: 18px; }
  .button { min-width: 0; min-height: 46px; flex: 1; border-radius: 9px; font-size: 13px; }
  .metrics { width: min(100%, 360px); grid-template-columns: repeat(3, 1fr); gap: 8px; margin-top: 18px; }
  .metric { height: 90px; border-radius: 9px; }
  .metric-icon { width: 17px; height: 17px; margin-bottom: 2px; }
  .metric-value { min-height: 25px; font-size: 21px; }
  .metric-value.compact { font-size: 17px; }
  .metric-label { margin-top: 0; font-size: 9px; }
  .metric-uptime { padding: 5px 2px; }
  .metric-uptime .uptime-title { gap: 4px; margin-bottom: 1px; font-size: 8px; }
  .metric-uptime .uptime-title::before { width: 5px; height: 5px; flex-basis: 5px; box-shadow: 0 0 0 2px rgba(71, 220, 148, 0.1); }
  .metric-uptime .metric-value { min-height: 25px; font-size: 23px; }
  .metric-uptime .metric-value small { font-size: 9px; }
  .runtime-compact { gap: 2px; margin-top: 0; font-size: 7px; }
  .runtime-compact b { font-size: 8px; }
  .content-section { min-height: 100dvh; scroll-margin-top: 68px; padding: 104px 16px 72px; }
  .section-header { margin-bottom: 38px; }
  .section-kicker { margin-bottom: 10px; font-size: 11px; }
  .section-title { font-size: 34px; }
  .section-description { margin-top: 14px; font-size: 14px; line-height: 1.6; }
  .status-toolbar { flex-wrap: wrap; gap: 8px; margin-top: 18px; }
  .status-updated { width: 100%; }
  .status-grid { grid-template-columns: 1fr; gap: 12px; }
  .status-card { min-height: 322px; border-radius: 10px; padding: 20px 18px; }
  .status-card-head { gap: 10px; }
  .status-identity { gap: 10px; }
  .status-provider-icon { width: 38px; height: 38px; flex-basis: 38px; border-radius: 9px; }
  .status-card h3 { margin: 0; font-size: 16px; }
  .status-provider-tag { min-height: 21px; padding: 0 6px; }
  .status-badge { min-height: 26px; padding: 0 8px; }
  .status-metrics { margin-top: 20px; padding: 16px 0; }
  .status-metrics > div { padding: 0 10px; }
  .status-metrics dd { font-size: 19px; }
  .status-availability { padding-top: 14px; }
  .status-availability strong { font-size: 27px; }
  .status-history { margin-top: 15px; }
  .status-history-meta { font-size: 10px; }
  .status-bars { grid-template-columns: repeat(60, minmax(2px, 1fr)); gap: 1px; }
  .status-stale-note { margin-top: -20px; }
  .supported-models-grid { gap: 12px; }
  .supported-model-card { width: calc((100% - 12px) / 2); min-height: 164px; border-radius: 10px; padding: 20px 10px; }
  .supported-model-card .model-logo { width: 40px; height: 40px; flex-basis: 40px; }
  .supported-model-card .model-logo svg { width: 22px; height: 22px; }
  .supported-model-card h3 { margin-top: 14px; font-size: 14px; }
  .supported-model-card p { margin-top: 5px; font-size: 11px; }
  .model-available { min-height: 25px; margin-top: 12px; font-size: 9px; }
  .support { right: 12px; bottom: 12px; width: 42px; height: 42px; }
}

@media (max-height: 820px) and (min-width: 721px) {
  .brand-title { margin-top: 19px; font-size: 60px; }
  .home-hero h1 { margin-top: 12px; font-size: 47px; }
  .description { margin-top: 18px; font-size: 16px; }
  .terminal { margin-top: 20px; }
  .actions { margin-top: 24px; }
  .button { min-height: 54px; }
  .metrics { margin-top: 34px; }
  .metric { height: 124px; }
}

@media (prefers-reduced-motion: reduce) {
  .home-page *,
  .home-page *::before,
  .home-page *::after { animation-duration: 0.01ms !important; animation-iteration-count: 1 !important; transition-duration: 0.01ms !important; }
  .motion-ready .reveal,
  .motion-ready .reveal-card { opacity: 1; transform: none; }
}
</style>
