<template>
  <AppLayout>
    <div class="mx-auto max-w-[1680px] space-y-5">
      <section class="card overflow-hidden">
        <div class="flex flex-col gap-5 p-5 md:flex-row md:items-center md:justify-between">
          <div class="flex items-start gap-4">
            <div class="flex h-14 w-14 flex-shrink-0 items-center justify-center rounded-xl bg-primary-500/10 text-primary-500 ring-1 ring-primary-500/20">
              <Icon name="image" size="xl" :stroke-width="1.8" />
            </div>
            <div class="min-w-0">
              <h1 class="text-2xl font-semibold tracking-normal text-gray-900 dark:text-white">图片工作台</h1>
              <p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-dark-300">
                直接选择已有 API Key，调用站内图片接口进行生图或改图。
              </p>
              <div class="mt-2 inline-flex max-w-full items-center rounded-md bg-gray-100 px-2 py-1 text-xs text-gray-600 dark:bg-dark-900 dark:text-dark-300">
                <span class="truncate">网关地址：{{ gatewayBaseUrl }}</span>
              </div>
            </div>
          </div>
          <button class="btn btn-secondary self-start md:self-auto" :disabled="keysLoading" @click="loadKeys">
            <Icon name="refresh" size="md" :class="keysLoading ? 'animate-spin' : ''" />
            刷新
          </button>
        </div>
      </section>

      <div class="grid gap-5 xl:grid-cols-[400px_minmax(0,1fr)]">
        <aside class="space-y-5">
          <section class="card">
            <div class="card-header">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">选择密钥</h2>
            </div>
            <div class="card-body space-y-4">
              <label class="block">
                <span class="input-label">API Key</span>
                <Select
                  :model-value="selectedKeyValue"
                  :options="keySelectOptions"
                  placeholder="请选择 API Key"
                  empty-text="暂无可用 API Key"
                  :searchable="false"
                  :disabled="keysLoading"
                  @update:model-value="selectedKeyValue = String($event || '')"
                />
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
            <div class="card-header flex items-center justify-between gap-3">
              <div>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">模型</h2>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-300">先点击获取模型，再从当前 Key 的模型列表中选择。</p>
              </div>
              <button class="btn btn-secondary btn-sm whitespace-nowrap" type="button" :disabled="modelsLoading || !selectedKeyValue" @click="loadModels">
                <Icon name="refresh" size="sm" :class="modelsLoading ? 'animate-spin' : ''" />
                获取模型
              </button>
            </div>
            <div class="card-body space-y-4">
              <label class="block">
                <span class="input-label">图片模型({{ fetchedModelCount }})</span>
                <Select
                  v-model="model"
                  :options="modelSelectOptions"
                  placeholder="请先获取模型"
                  search-placeholder="搜索模型..."
                  empty-text="暂无模型，请先点击获取模型"
                  :searchable="true"
                  :disabled="modelsLoading"
                />
              </label>
              <p v-if="modelsHint" class="text-xs text-gray-500 dark:text-dark-400">{{ modelsHint }}</p>
            </div>
          </section>

          <section class="card">
            <div class="card-header">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">参考图</h2>
            </div>
            <div class="card-body">
              <label
                class="flex min-h-32 cursor-pointer flex-col items-center justify-center rounded-xl border border-dashed px-4 py-6 text-center transition-colors"
                :class="referenceDropActive ? 'border-primary-400 bg-primary-50/70 dark:border-primary-400 dark:bg-primary-950/30' : 'border-gray-300 bg-gray-50 hover:border-primary-400 hover:bg-primary-50/50 dark:border-dark-600 dark:bg-dark-900/70 dark:hover:border-primary-500 dark:hover:bg-primary-950/20'"
                @dragenter.prevent="handleReferenceDragEnter"
                @dragover.prevent="handleReferenceDragOver"
                @dragleave.prevent="handleReferenceDragLeave"
                @drop.prevent="handleReferenceDrop"
              >
                <Icon name="upload" size="lg" class="text-gray-400" />
                <span class="mt-3 text-sm font-medium text-gray-700 dark:text-dark-200">拖拽或点击上传参考图</span>
                <span class="mt-1 text-xs text-gray-500 dark:text-dark-400">PNG/JPG/WEBP，最多 8 张，也可拖入右侧生成结果</span>
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

        <main class="space-y-5">
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
                  class="input min-h-44 resize-y leading-6"
                  placeholder="例如：高级商业摄影风格的橙色宇航员猫咪贴纸，干净背景，细节丰富，柔和光影。"
                ></textarea>
              </label>

              <div class="rounded-xl border border-gray-100 bg-gray-50/70 p-4 dark:border-dark-700 dark:bg-dark-900/60">
              <div class="grid gap-4 md:grid-cols-2 2xl:grid-cols-4">
                <label class="block">
                  <span class="input-label">尺寸</span>
                  <Select
                    :model-value="size"
                    :options="sizeOptions"
                    :searchable="false"
                    @update:model-value="size = $event as ImageStudioSize"
                  />
                </label>
                <label class="block">
                  <span class="input-label">张数</span>
                  <Select
                    :model-value="count"
                    :options="countOptions"
                    :searchable="false"
                    @update:model-value="count = Number($event || 1)"
                  />
                </label>
                <label class="block">
                  <span class="input-label">质量</span>
                  <Select
                    :model-value="quality"
                    :options="qualityOptions"
                    :searchable="false"
                    @update:model-value="quality = $event as ImageStudioQuality"
                  />
                </label>
                <label class="block">
                  <span class="input-label">输出格式</span>
                  <Select
                    :model-value="outputFormat"
                    :options="outputFormatOptions"
                    :searchable="false"
                    @update:model-value="outputFormat = $event as ImageStudioOutputFormat"
                  />
                </label>
              </div>

              <div class="mt-4 grid gap-4 md:grid-cols-2">
                <label class="block">
                  <span class="input-label">背景</span>
                  <Select
                    :model-value="background"
                    :options="backgroundOptions"
                    :searchable="false"
                    @update:model-value="background = $event as ImageStudioBackground"
                  />
                </label>
                <label class="block">
                  <span class="input-label">风格</span>
                  <Select
                    :model-value="style"
                    :options="styleOptions"
                    :searchable="false"
                    @update:model-value="style = String($event ?? '')"
                  />
                </label>
              </div>
              </div>

              <div class="flex flex-col gap-3 border-t border-gray-100 pt-5 dark:border-dark-700 sm:flex-row sm:items-center">
                <button class="btn btn-primary min-w-36 justify-center" :disabled="generating || !canGenerate" @click="submit">
                  <span
                    v-if="generating"
                    class="h-4 w-4 rounded-full border-2 border-white/40 border-t-white animate-spin"
                    aria-hidden="true"
                  ></span>
                  <Icon v-else name="sparkles" size="md" />
                  {{ generating ? `生成中 · ${generationElapsedText}` : '开始生图' }}
                </button>
                <button class="btn btn-secondary" :disabled="generating || results.length === 0" @click="clearResults">
                  清空结果
                </button>
                <p v-if="validationMessage" class="text-sm text-amber-600 dark:text-amber-300">{{ validationMessage }}</p>
              </div>
            </div>
          </section>

          <section class="card">
            <div class="card-header flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between">
              <div>
                <div class="flex flex-wrap items-center gap-3">
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">结果预览</h2>
                  <span v-if="results.length > 0 && lastGenerationElapsedSeconds !== null" class="rounded-full bg-primary-500/10 px-3 py-1 text-xs font-medium text-primary-600 dark:text-primary-300">
                    共用时 {{ lastGenerationElapsedSeconds }}s
                  </span>
                </div>
                <p class="mt-1 text-sm text-gray-500 dark:text-dark-300">优先使用 base64 结果直接展示，无需额外图片代理。</p>
              </div>
            </div>
            <div class="card-body">
              <div v-if="generating" class="flex min-h-80 flex-col items-center justify-center rounded-xl border border-dashed border-gray-200 bg-gray-50 text-gray-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-dark-300">
                <div class="flex h-16 w-16 items-center justify-center rounded-full bg-primary-500/10">
                  <span class="h-10 w-10 rounded-full border-4 border-primary-500/20 border-t-primary-500 animate-spin" aria-hidden="true"></span>
                </div>
                <p class="mt-4 text-sm font-medium text-gray-700 dark:text-dark-100">正在等待图片结果...</p>
              </div>
              <div v-else-if="results.length === 0" class="flex min-h-80 flex-col items-center justify-center rounded-xl border border-dashed border-gray-200 bg-gray-50 text-gray-500 dark:border-dark-700 dark:bg-dark-900/70 dark:text-dark-300">
                <Icon name="inbox" size="xl" class="text-gray-400" />
                <p class="mt-4 text-sm">还没有图片结果</p>
              </div>
              <div v-else class="grid gap-5 md:grid-cols-2 2xl:grid-cols-3">
                <article
                  v-for="result in results"
                  :key="result.id"
                  class="overflow-hidden rounded-xl border border-gray-100 bg-white dark:border-dark-700 dark:bg-dark-900"
                  draggable="true"
                  @dragstart="handleResultDragStart($event, result)"
                  @dragend="handleResultDragEnd"
                >
                  <img :src="result.url" alt="生成结果" class="aspect-square w-full bg-gray-100 object-contain dark:bg-dark-950" draggable="false" />
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
                      <button class="btn btn-primary btn-sm" type="button" @click="continueWithResult(result)">
                        <Icon name="sparkles" size="sm" />
                        继续优化
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
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import keysAPI from '@/api/keys'
import {
  generateImage,
  isUsableImageKey,
  listImageModels,
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
const modelsLoading = ref(false)
const remoteModels = ref<string[]>([])
const modelsHint = ref('')
const selectedKeyValue = ref('')
const model = ref('')
const prompt = ref('')
const size = ref<ImageStudioSize>('1024x1024')
const count = ref(1)
const quality = ref<ImageStudioQuality>('auto')
const background = ref<ImageStudioBackground>('auto')
const outputFormat = ref<ImageStudioOutputFormat>('png')
const style = ref('')
const generating = ref(false)
const generationStartedAt = ref<number | null>(null)
const generationElapsedSeconds = ref(0)
let generationTimer: number | null = null
const results = ref<ImageStudioResult[]>([])
const referencePreviews = ref<ReferencePreview[]>([])
const referenceDropActive = ref(false)
const lastGenerationElapsedSeconds = ref<number | null>(null)
let referenceDragDepth = 0
const openedResultUrls: string[] = []

const sizeOptions: Array<{ value: ImageStudioSize, label: string }> = [
  { value: 'auto', label: '自动' },
  { value: '1024x1024', label: '1K 方图 · 1024 x 1024' },
  { value: '1536x1024', label: '2K 横图 · 1536 x 1024' },
  { value: '1024x1536', label: '2K 竖图 · 1024 x 1536' },
  { value: '1792x1024', label: '宽屏横图 · 1792 x 1024' },
  { value: '1024x1792', label: '长幅竖图 · 1024 x 1792' },
  { value: '2048x2048', label: '2K 方图 · 2048 x 2048' },
  { value: '2560x1440', label: '2.5K 横图 · 2560 x 1440' },
  { value: '1440x2560', label: '2.5K 竖图 · 1440 x 2560' },
  { value: '3840x2160', label: '4K 横图 · 3840 x 2160' },
  { value: '2160x3840', label: '4K 竖图 · 2160 x 3840' },
]
const qualityOptions: Array<{ value: ImageStudioQuality, label: string }> = [
  { value: 'auto', label: '自动' },
  { value: 'low', label: '低' },
  { value: 'medium', label: '中' },
  { value: 'high', label: '高' },
  { value: 'standard', label: '标准' },
  { value: 'hd', label: 'HD' },
  { value: 'ultra', label: '超清' },
]
const countOptions = [
  { value: 1, label: '1' },
  { value: 2, label: '2' },
  { value: 3, label: '3' },
  { value: 4, label: '4' },
]
const outputFormatOptions: Array<{ value: ImageStudioOutputFormat, label: string }> = [
  { value: 'png', label: 'PNG' },
  { value: 'jpeg', label: 'JPEG' },
  { value: 'webp', label: 'WEBP' },
]
const backgroundOptions: Array<{ value: ImageStudioBackground, label: string }> = [
  { value: 'auto', label: '自动' },
  { value: 'transparent', label: '透明' },
  { value: 'opaque', label: '不透明' },
]
const styleOptions = [
  { value: '', label: '默认' },
  { value: 'vivid', label: '鲜明' },
  { value: 'natural', label: '自然' },
]

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
const keySelectOptions = computed(() => selectableKeys.value.map((key) => ({
  value: key.key,
  label: `${key.name} - ${maskKey(key.key)}`,
})))
const fetchedModelCount = computed(() => remoteModels.value.length)
const modelOptions = computed(() => remoteModels.value)
const modelSelectOptions = computed(() => modelOptions.value.map((item) => ({ value: item, label: item })))
const generationElapsedText = computed(() => formatElapsedTime(generationElapsedSeconds.value))

const validationMessage = computed(() => {
  if (!selectedKeyValue.value) return '请先选择 API Key'
  if (!model.value.trim()) return '请先获取并选择图片模型'
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

watch(selectedKeyValue, () => {
  remoteModels.value = []
  model.value = ''
  modelsHint.value = ''
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

async function loadModels() {
  if (!selectedKeyValue.value) {
    appStore.showWarning('请先选择 API Key')
    return
  }

  modelsLoading.value = true
  remoteModels.value = []
  model.value = ''
  modelsHint.value = ''
  try {
    const models = await listImageModels(selectedKeyValue.value, gatewayBaseUrl.value)
    remoteModels.value = models
    if (models.length > 0) {
      modelsHint.value = `已获取 ${models.length} 个模型`
      const preferred = models.find((item) => /gpt-image|image|dall-e/i.test(item))
      model.value = preferred || models[0]
      appStore.showSuccess('模型列表已更新')
    } else {
      model.value = ''
      modelsHint.value = '接口没有返回可用模型'
      appStore.showWarning('没有获取到模型列表')
    }
  } catch (err: unknown) {
    modelsHint.value = '获取失败，请检查当前 Key 或稍后重试'
    appStore.showError(extractApiErrorMessage(err, '获取模型失败'))
  } finally {
    modelsLoading.value = false
  }
}

function maskKey(value: string): string {
  if (!value) return ''
  if (value.length <= 14) return value
  return `${value.slice(0, 7)}...${value.slice(-5)}`
}

async function addReferenceFiles(files: File[]) {
  if (files.length === 0) return

  const remaining = Math.max(0, 8 - referencePreviews.value.length)
  const accepted = files
    .filter((file) => file.type.startsWith('image/'))
    .slice(0, remaining)

  if (accepted.length < files.length) {
    appStore.showWarning('最多上传 8 张参考图，非图片文件会被忽略')
  }
  if (accepted.length === 0) return

  const next = await Promise.all(accepted.map(async (file) => ({
    id: `${file.name}-${file.lastModified}-${Math.random().toString(36).slice(2)}`,
    file,
    url: await fileToDataUrl(file),
    name: file.name,
  })))
  referencePreviews.value = [...referencePreviews.value, ...next]
}

async function handleFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  const files = Array.from(input.files || [])
  input.value = ''
  await addReferenceFiles(files)
}

function removeReference(id: string) {
  referencePreviews.value = referencePreviews.value.filter((preview) => preview.id !== id)
}

function handleReferenceDragEnter() {
  referenceDragDepth += 1
  referenceDropActive.value = true
}

function handleReferenceDragOver(event: DragEvent) {
  event.dataTransfer!.dropEffect = 'copy'
  referenceDropActive.value = true
}

function handleReferenceDragLeave() {
  referenceDragDepth = Math.max(0, referenceDragDepth - 1)
  if (referenceDragDepth === 0) {
    referenceDropActive.value = false
  }
}

async function handleReferenceDrop(event: DragEvent) {
  referenceDragDepth = 0
  referenceDropActive.value = false

  const files = Array.from(event.dataTransfer?.files || [])
  if (files.length > 0) {
    await addReferenceFiles(files)
    return
  }

  const resultId = event.dataTransfer?.getData('application/x-image-studio-result') || ''
  const result = results.value.find((item) => item.id === resultId)
  if (result) {
    await continueWithResult(result)
  }
}

function handleResultDragStart(event: DragEvent, result: ImageStudioResult) {
  event.dataTransfer?.setData('application/x-image-studio-result', result.id)
  event.dataTransfer?.setData('text/plain', result.fileName)
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'copy'
  }
}

function handleResultDragEnd() {
  referenceDropActive.value = false
  referenceDragDepth = 0
}

function fileToDataUrl(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result || ''))
    reader.onerror = () => reject(reader.error || new Error('读取图片失败'))
    reader.readAsDataURL(file)
  })
}

function formatElapsedTime(totalSeconds: number): string {
  const minutes = Math.floor(totalSeconds / 60)
  const seconds = totalSeconds % 60
  if (minutes <= 0) return `${seconds} 秒`
  return `${minutes} 分 ${String(seconds).padStart(2, '0')} 秒`
}

function stopGenerationTimer() {
  if (generationTimer) {
    window.clearInterval(generationTimer)
    generationTimer = null
  }
}

function startGenerationTimer() {
  stopGenerationTimer()
  generationStartedAt.value = Date.now()
  generationElapsedSeconds.value = 0
  generationTimer = window.setInterval(() => {
    if (!generationStartedAt.value) return
    generationElapsedSeconds.value = Math.floor((Date.now() - generationStartedAt.value) / 1000)
  }, 1000)
}

async function submit() {
  if (!canGenerate.value) {
    appStore.showWarning(validationMessage.value)
    return
  }

  generating.value = true
  lastGenerationElapsedSeconds.value = null
  startGenerationTimer()
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
    generationElapsedSeconds.value = generationStartedAt.value
      ? Math.max(1, Math.ceil((Date.now() - generationStartedAt.value) / 1000))
      : generationElapsedSeconds.value
    lastGenerationElapsedSeconds.value = generationElapsedSeconds.value
    results.value = generated
    appStore.showSuccess('图片生成成功')
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, '图片生成失败'))
  } finally {
    generating.value = false
    stopGenerationTimer()
  }
}

function clearResults() {
  results.value = []
}

async function resultToBlob(result: ImageStudioResult): Promise<Blob> {
  if (result.url.startsWith('data:')) {
    return dataUrlToBlob(result.url, result.mimeType)
  }
  const response = await fetch(result.url)
  const blob = await response.blob()
  if (blob.type) return blob
  return blob.slice(0, blob.size, result.mimeType)
}

function dataUrlToBlob(url: string, fallbackMimeType: string): Blob {
  const [meta = '', data = ''] = url.split(',', 2)
  const mimeType = meta.match(/^data:([^;]+)/)?.[1] || fallbackMimeType
  const binary = atob(data)
  const bytes = new Uint8Array(binary.length)
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index)
  }
  return new Blob([bytes], { type: mimeType })
}

async function resultToFile(result: ImageStudioResult): Promise<File> {
  const blob = await resultToBlob(result)
  return new File([blob], result.fileName, {
    type: blob.type || result.mimeType,
    lastModified: Date.now(),
  })
}

async function downloadResult(result: ImageStudioResult) {
  try {
    const blob = await resultToBlob(result)
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = result.fileName
    document.body.appendChild(a)
    a.click()
    a.remove()
    window.setTimeout(() => URL.revokeObjectURL(url), 30000)
  } catch {
    const a = document.createElement('a')
    a.href = result.url
    a.download = result.fileName
    document.body.appendChild(a)
    a.click()
    a.remove()
  }
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

async function openResult(result: ImageStudioResult) {
  const viewer = window.open('', '_blank')
  if (!viewer) {
    appStore.showError('打开失败，请允许浏览器弹窗或下载后查看')
    return
  }
  viewer.opener = null
  viewer.document.title = result.fileName
  viewer.document.body.style.margin = '0'
  viewer.document.body.style.background = '#0f172a'
  viewer.document.body.innerHTML = '<div style="min-height:100vh;display:grid;place-items:center;color:#cbd5e1;font-family:sans-serif;">正在打开图片...</div>'
  try {
    const blob = await resultToBlob(result)
    const url = URL.createObjectURL(blob)
    openedResultUrls.push(url)
    viewer.location.href = url
  } catch {
    viewer.location.href = result.url
  }
}

async function continueWithResult(result: ImageStudioResult) {
  try {
    const file = await resultToFile(result)
    await addReferenceFiles([file])
    appStore.showSuccess('已加入参考图，可继续优化')
  } catch {
    appStore.showError('加入参考图失败，请下载后重新上传')
  }
}

onMounted(async () => {
  if (!appStore.publicSettingsLoaded) {
    await appStore.fetchPublicSettings()
  }
  await loadKeys()
})

onBeforeUnmount(() => {
  stopGenerationTimer()
  openedResultUrls.forEach((url) => URL.revokeObjectURL(url))
})
</script>
