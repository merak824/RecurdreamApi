<template>
  <AppLayout>
    <div class="mx-auto max-w-[1760px] space-y-6">
      <section class="card overflow-hidden">
        <div class="flex flex-col gap-4 p-5 md:flex-row md:items-center md:justify-between">
          <div class="flex items-start gap-4">
            <div class="flex h-11 w-11 flex-shrink-0 items-center justify-center rounded-xl bg-primary-500/10 text-primary-500">
              <Icon name="sparkles" size="lg" />
            </div>
            <div>
              <h1 class="text-xl font-semibold text-gray-900 dark:text-white">图片工作台</h1>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
                直接选择已有 API Key，调用站内图片接口进行生图或改图。
              </p>
              <div class="mt-2 inline-flex max-w-full items-center rounded-md bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-900 dark:text-dark-300">
                <span class="truncate">网关地址：{{ gatewayBaseUrl }}</span>
              </div>
            </div>
          </div>
          <button class="btn btn-secondary" :disabled="keysLoading" @click="loadKeys">
            <Icon name="refresh" size="md" :class="keysLoading ? 'animate-spin' : ''" />
            刷新
          </button>
        </div>
      </section>

      <div class="grid gap-6 xl:grid-cols-[420px_minmax(0,1fr)]">
        <aside class="space-y-6">
          <section class="card">
            <div class="card-header">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">选择密钥</h2>
            </div>
            <div class="card-body space-y-4">
              <label class="block">
                <span class="input-label">API Key</span>
                <select v-model="selectedKeyValue" class="input">
                  <option value="" disabled>请选择 API Key</option>
                  <option v-for="key in selectableKeys" :key="key.id" :value="key.key">
                    {{ key.name }} - {{ maskKey(key.key) }}
                  </option>
                </select>
              </label>

              <div v-if="selectedKey" class="rounded-xl border border-gray-100 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-900/70">
                <div class="flex flex-wrap items-center gap-2">
                  <span class="rounded-full bg-emerald-500/10 px-2 py-1 text-xs font-medium text-emerald-600 dark:text-emerald-300">
                    {{ selectedKey.group?.name || '未绑定分组' }}
                  </span>
                  <span
                    class="rounded-full px-2 py-1 text-xs font-medium"
                    :class="selectedKey.group?.allow_image_generation ? 'bg-primary-500/10 text-primary-600 dark:text-primary-300' : 'bg-amber-500/10 text-amber-600 dark:text-amber-300'"
                  >
                    {{ selectedKey.group?.allow_image_generation ? '已启用生图' : '未启用生图' }}
                  </span>
                </div>
                <dl class="mt-4 grid grid-cols-[80px_minmax(0,1fr)] gap-2 text-sm">
                  <dt class="text-gray-500 dark:text-dark-400">API 密钥</dt>
                  <dd class="truncate font-mono text-gray-900 dark:text-white">{{ maskKey(selectedKey.key) }}</dd>
                  <dt class="text-gray-500 dark:text-dark-400">状态</dt>
                  <dd class="text-gray-900 dark:text-white">{{ selectedKey.status }}</dd>
                </dl>
              </div>

              <div v-if="!keysLoading && imageCapableKeys.length === 0" class="rounded-xl border border-amber-200 bg-amber-50 p-3 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200">
                当前账号下没有检测到“OpenAI + 已允许生图”的活跃 Key。
              </div>
            </div>
          </section>

          <section class="card">
            <div class="card-header">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">模型</h2>
            </div>
            <div class="card-body space-y-4">
              <label class="block">
                <span class="input-label">图片模型</span>
                <input v-model.trim="model" class="input font-mono" placeholder="gpt-image-2" />
              </label>
              <div class="flex flex-wrap gap-2">
                <button
                  v-for="preset in modelPresets"
                  :key="preset"
                  type="button"
                  class="rounded-full border px-3 py-1 text-xs transition-colors"
                  :class="model === preset ? 'border-primary-400 bg-primary-500/10 text-primary-600 dark:text-primary-300' : 'border-gray-200 text-gray-500 hover:border-primary-300 hover:text-primary-500 dark:border-dark-600 dark:text-dark-300'"
                  @click="model = preset"
                >
                  {{ preset }}
                </button>
              </div>
            </div>
          </section>

          <section class="card">
            <div class="card-header">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">参考图</h2>
            </div>
            <div class="card-body">
              <label
                class="flex min-h-32 cursor-pointer flex-col items-center justify-center rounded-xl border border-dashed border-gray-300 bg-gray-50 px-4 py-6 text-center transition-colors hover:border-primary-400 hover:bg-primary-50/50 dark:border-dark-600 dark:bg-dark-900/70 dark:hover:border-primary-500 dark:hover:bg-primary-950/20"
              >
                <Icon name="upload" size="lg" class="text-gray-400" />
                <span class="mt-3 text-sm font-medium text-gray-700 dark:text-dark-200">上传参考图</span>
                <span class="mt-1 text-xs text-gray-500 dark:text-dark-400">PNG/JPG/WEBP，最多 8 张</span>
                <input class="hidden" type="file" accept="image/png,image/jpeg,image/webp" multiple @change="handleFileChange" />
              </label>

              <div v-if="referencePreviews.length" class="mt-4 grid grid-cols-3 gap-3">
                <div v-for="preview in referencePreviews" :key="preview.id" class="group relative overflow-hidden rounded-lg border border-gray-100 bg-gray-100 dark:border-dark-700 dark:bg-dark-900">
                  <img :src="preview.url" :alt="preview.name" class="aspect-square w-full object-cover" />
                  <button
                    class="absolute right-1 top-1 rounded-md bg-black/60 p-1 text-white opacity-0 transition-opacity group-hover:opacity-100"
                    type="button"
                    @click="removeReference(preview.id)"
                  >
                    <Icon name="x" size="xs" />
                  </button>
                </div>
              </div>
            </div>
          </section>
        </aside>

        <main class="space-y-6">
          <section class="card">
            <div class="card-header flex items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">提示词与参数</h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">
                  未上传参考图时调用 /v1/images/generations，上传后调用 /v1/images/edits。
                </p>
              </div>
              <span class="rounded-full bg-primary-500/10 px-3 py-1 text-xs font-medium text-primary-600 dark:text-primary-300">
                {{ referenceFiles.length ? '改图' : '生图' }}
              </span>
            </div>
            <div class="card-body space-y-5">
              <label class="block">
                <span class="input-label">提示词</span>
                <textarea
                  v-model.trim="prompt"
                  class="input min-h-36 resize-y leading-6"
                  placeholder="例如：高级商业摄影风格的橙色宇航员猫咪贴纸，干净背景，细节丰富，柔和光影。"
                ></textarea>
              </label>

              <div class="grid gap-4 md:grid-cols-2 2xl:grid-cols-4">
                <label class="block">
                  <span class="input-label">尺寸</span>
                  <select v-model="size" class="input">
                    <option value="1024x1024">1K 方图 - 1024 x 1024</option>
                    <option value="1024x1536">竖图 - 1024 x 1536</option>
                    <option value="1536x1024">横图 - 1536 x 1024</option>
                    <option value="auto">自动</option>
                  </select>
                </label>
                <label class="block">
                  <span class="input-label">张数</span>
                  <select v-model.number="count" class="input">
                    <option :value="1">1</option>
                    <option :value="2">2</option>
                    <option :value="3">3</option>
                    <option :value="4">4</option>
                  </select>
                </label>
                <label class="block">
                  <span class="input-label">质量</span>
                  <select v-model="quality" class="input">
                    <option value="auto">自动</option>
                    <option value="low">低</option>
                    <option value="medium">中</option>
                    <option value="high">高</option>
                  </select>
                </label>
                <label class="block">
                  <span class="input-label">输出格式</span>
                  <select v-model="outputFormat" class="input">
                    <option value="png">PNG</option>
                    <option value="jpeg">JPEG</option>
                    <option value="webp">WEBP</option>
                  </select>
                </label>
              </div>

              <div class="grid gap-4 md:grid-cols-2">
                <label class="block">
                  <span class="input-label">背景</span>
                  <select v-model="background" class="input">
                    <option value="auto">自动</option>
                    <option value="transparent">透明</option>
                    <option value="opaque">不透明</option>
                  </select>
                </label>
                <label class="block">
                  <span class="input-label">风格</span>
                  <select v-model="style" class="input">
                    <option value="">默认</option>
                    <option value="vivid">鲜明</option>
                    <option value="natural">自然</option>
                  </select>
                </label>
              </div>

              <div class="flex flex-wrap items-center gap-3">
                <button class="btn btn-primary" :disabled="generating || !canGenerate" @click="submit">
                  <Icon name="sparkles" size="md" :class="generating ? 'animate-pulse' : ''" />
                  {{ generating ? '生成中...' : '开始生图' }}
                </button>
                <button class="btn btn-secondary" :disabled="generating || results.length === 0" @click="clearResults">
                  清空结果
                </button>
                <p v-if="validationMessage" class="text-sm text-amber-600 dark:text-amber-300">{{ validationMessage }}</p>
              </div>
            </div>
          </section>

          <section class="card">
            <div class="card-header">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">结果预览</h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">优先使用 base64 结果直接展示，无需额外图片代理。</p>
            </div>
            <div class="card-body">
              <div v-if="generating" class="flex min-h-80 flex-col items-center justify-center rounded-xl border border-dashed border-gray-200 bg-gray-50 text-gray-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-dark-300">
                <Icon name="sparkles" size="xl" class="animate-pulse text-primary-500" />
                <p class="mt-4 text-sm">正在等待图片结果...</p>
              </div>
              <div v-else-if="results.length === 0" class="flex min-h-80 flex-col items-center justify-center rounded-xl border border-dashed border-gray-200 bg-gray-50 text-gray-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-dark-300">
                <Icon name="inbox" size="xl" class="text-gray-400" />
                <p class="mt-4 text-sm">还没有图片结果</p>
              </div>
              <div v-else class="grid gap-5 md:grid-cols-2 2xl:grid-cols-3">
                <article v-for="result in results" :key="result.id" class="overflow-hidden rounded-xl border border-gray-100 bg-white dark:border-dark-700 dark:bg-dark-900">
                  <img :src="result.url" alt="生成结果" class="aspect-square w-full bg-gray-100 object-contain dark:bg-dark-950" />
                  <div class="space-y-3 p-3">
                    <p v-if="result.revisedPrompt" class="line-clamp-2 text-xs text-gray-500 dark:text-dark-300">
                      {{ result.revisedPrompt }}
                    </p>
                    <div class="flex flex-wrap gap-2">
                      <button class="btn btn-secondary btn-sm" type="button" @click="downloadResult(result)">
                        <Icon name="download" size="sm" />
                        下载
                      </button>
                      <button class="btn btn-secondary btn-sm" type="button" @click="copyResult(result)">
                        <Icon name="copy" size="sm" />
                        复制
                      </button>
                      <button class="btn btn-ghost btn-sm" type="button" @click="openResult(result)">
                        <Icon name="externalLink" size="sm" />
                        打开
                      </button>
                    </div>
                  </div>
                </article>
              </div>
            </div>
          </section>
        </main>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import keysAPI from '@/api/keys'
import {
  generateImage,
  isUsableImageKey,
  type ImageStudioBackground,
  type ImageStudioOutputFormat,
  type ImageStudioQuality,
  type ImageStudioResult,
  type ImageStudioSize,
} from '@/api/images'
import type { ApiKey } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'

interface ReferencePreview {
  id: string
  file: File
  url: string
  name: string
}

const appStore = useAppStore()

const keys = ref<ApiKey[]>([])
const keysLoading = ref(false)
const selectedKeyValue = ref('')
const model = ref('gpt-image-2')
const prompt = ref('')
const size = ref<ImageStudioSize>('1024x1024')
const count = ref(1)
const quality = ref<ImageStudioQuality>('auto')
const background = ref<ImageStudioBackground>('auto')
const outputFormat = ref<ImageStudioOutputFormat>('png')
const style = ref('')
const generating = ref(false)
const results = ref<ImageStudioResult[]>([])
const referencePreviews = ref<ReferencePreview[]>([])

const modelPresets = ['gpt-image-2', 'gpt-image-1']

const gatewayBaseUrl = computed(() => {
  const configured = appStore.apiBaseUrl || appStore.cachedPublicSettings?.api_base_url || ''
  if (configured.trim()) {
    return configured.trim().replace(/\/+$/, '')
  }
  return `${window.location.origin}/v1`
})

const activeKeys = computed(() => keys.value.filter((key) => key.status === 'active'))
const imageCapableKeys = computed(() => activeKeys.value.filter(isUsableImageKey))
const selectableKeys = computed(() => imageCapableKeys.value.length > 0 ? imageCapableKeys.value : activeKeys.value)
const selectedKey = computed(() => selectableKeys.value.find((key) => key.key === selectedKeyValue.value) || null)
const referenceFiles = computed(() => referencePreviews.value.map((preview) => preview.file))

const validationMessage = computed(() => {
  if (!selectedKeyValue.value) return '请先选择 API Key'
  if (!model.value.trim()) return '请填写图片模型'
  if (!prompt.value.trim()) return '请填写提示词'
  if (selectedKey.value && !isUsableImageKey(selectedKey.value)) return '当前 Key 所属分组未确认开启生图'
  return ''
})

const canGenerate = computed(() => validationMessage.value === '')

watch(selectableKeys, (items) => {
  if (!selectedKeyValue.value && items.length > 0) {
    selectedKeyValue.value = items[0].key
  }
})

async function loadKeys() {
  keysLoading.value = true
  try {
    const response = await keysAPI.list(1, 200, {
      status: 'active',
      sort_by: 'created_at',
      sort_order: 'desc',
    })
    keys.value = response.items
    if (!selectedKeyValue.value) {
      selectedKeyValue.value = selectableKeys.value[0]?.key || ''
    }
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, '加载 API Key 失败'))
  } finally {
    keysLoading.value = false
  }
}

function maskKey(value: string): string {
  if (!value) return ''
  if (value.length <= 14) return value
  return `${value.slice(0, 7)}...${value.slice(-5)}`
}

function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  if (files.length === 0) return

  const remaining = Math.max(0, 8 - referencePreviews.value.length)
  const accepted = files
    .filter((file) => file.type.startsWith('image/'))
    .slice(0, remaining)

  if (accepted.length < files.length) {
    appStore.showWarning('最多上传 8 张参考图，非图片文件会被忽略')
  }

  const next = accepted.map((file) => ({
    id: `${file.name}-${file.lastModified}-${Math.random().toString(36).slice(2)}`,
    file,
    url: URL.createObjectURL(file),
    name: file.name,
  }))
  referencePreviews.value = [...referencePreviews.value, ...next]
}

function removeReference(id: string) {
  const current = referencePreviews.value.find((preview) => preview.id === id)
  if (current) URL.revokeObjectURL(current.url)
  referencePreviews.value = referencePreviews.value.filter((preview) => preview.id !== id)
}

async function submit() {
  if (!canGenerate.value) {
    appStore.showWarning(validationMessage.value)
    return
  }

  generating.value = true
  try {
    const generated = await generateImage({
      apiKey: selectedKeyValue.value,
      baseUrl: gatewayBaseUrl.value,
      model: model.value.trim(),
      prompt: prompt.value.trim(),
      size: size.value,
      n: count.value,
      quality: quality.value,
      background: background.value,
      outputFormat: outputFormat.value,
      style: style.value,
      images: referenceFiles.value,
    })
    results.value = generated
    appStore.showSuccess('图片生成成功')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, '图片生成失败'))
  } finally {
    generating.value = false
  }
}

function clearResults() {
  results.value = []
}

async function resultToBlob(result: ImageStudioResult): Promise<Blob> {
  const response = await fetch(result.url)
  return response.blob()
}

async function downloadResult(result: ImageStudioResult) {
  const blob = await resultToBlob(result)
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = result.fileName
  document.body.appendChild(a)
  a.click()
  a.remove()
  URL.revokeObjectURL(url)
}

async function copyResult(result: ImageStudioResult) {
  try {
    const blob = await resultToBlob(result)
    if ('ClipboardItem' in window && navigator.clipboard?.write) {
      await navigator.clipboard.write([
        new ClipboardItem({
          [blob.type || result.mimeType]: blob,
        }),
      ])
      appStore.showSuccess('图片已复制')
      return
    }
    await navigator.clipboard.writeText(result.url)
    appStore.showSuccess('图片地址已复制')
  } catch {
    appStore.showError('复制失败，请下载图片')
  }
}

function openResult(result: ImageStudioResult) {
  window.open(result.url, '_blank', 'noopener,noreferrer')
}

onMounted(async () => {
  if (!appStore.publicSettingsLoaded) {
    await appStore.fetchPublicSettings()
  }
  await loadKeys()
})

onBeforeUnmount(() => {
  referencePreviews.value.forEach((preview) => URL.revokeObjectURL(preview.url))
})
</script>

