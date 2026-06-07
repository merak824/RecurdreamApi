import type { ApiKey } from '@/types'

export type ImageStudioSize = '1024x1024' | '1024x1536' | '1536x1024' | 'auto'
export type ImageStudioQuality = 'auto' | 'low' | 'medium' | 'high'
export type ImageStudioBackground = 'auto' | 'transparent' | 'opaque'
export type ImageStudioOutputFormat = 'png' | 'jpeg' | 'webp'

export interface ImageStudioGenerateRequest {
  apiKey: string
  baseUrl: string
  model: string
  prompt: string
  size: ImageStudioSize
  n: number
  quality: ImageStudioQuality
  background: ImageStudioBackground
  outputFormat: ImageStudioOutputFormat
  style?: string
  images?: File[]
}

export interface ImageStudioResult {
  id: string
  url: string
  mimeType: string
  fileName: string
  revisedPrompt?: string
}

interface OpenAIImageResponseItem {
  b64_json?: string
  url?: string
  revised_prompt?: string
}

interface OpenAIImageResponse {
  data?: OpenAIImageResponseItem[]
  b64_json?: string
  url?: string
  error?: {
    message?: string
    type?: string
    code?: string
  }
}

function normalizeGatewayBaseUrl(baseUrl: string): string {
  const trimmed = baseUrl.trim().replace(/\/+$/, '')
  if (!trimmed) return '/v1'
  if (trimmed.endsWith('/images/generations') || trimmed.endsWith('/images/edits')) {
    return trimmed.replace(/\/images\/(?:generations|edits)$/, '')
  }
  return trimmed
}

function mimeTypeForOutputFormat(format: ImageStudioOutputFormat): string {
  if (format === 'jpeg') return 'image/jpeg'
  if (format === 'webp') return 'image/webp'
  return 'image/png'
}

function extensionForOutputFormat(format: ImageStudioOutputFormat): string {
  return format === 'jpeg' ? 'jpg' : format
}

async function parseImageResponse(response: Response): Promise<OpenAIImageResponse> {
  const text = await response.text()
  let parsed: OpenAIImageResponse
  try {
    parsed = text ? JSON.parse(text) : {}
  } catch {
    throw new Error(text || `请求失败：HTTP ${response.status}`)
  }

  if (!response.ok) {
    const message = parsed.error?.message || `请求失败：HTTP ${response.status}`
    throw new Error(message)
  }
  return parsed
}

function responseToResults(
  payload: OpenAIImageResponse,
  outputFormat: ImageStudioOutputFormat,
): ImageStudioResult[] {
  const mimeType = mimeTypeForOutputFormat(outputFormat)
  const extension = extensionForOutputFormat(outputFormat)
  const items = payload.data?.length ? payload.data : [{ b64_json: payload.b64_json, url: payload.url }]

  return items
    .map((item, index) => {
      const rawUrl = item.url?.trim()
      const url = item.b64_json
        ? `data:${mimeType};base64,${item.b64_json}`
        : rawUrl || ''
      if (!url) return null
      const result: ImageStudioResult = {
        id: `${Date.now()}-${index}`,
        url,
        mimeType,
        fileName: `image-studio-${new Date().toISOString().replace(/[:.]/g, '-')}-${index + 1}.${extension}`,
        revisedPrompt: item.revised_prompt,
      }
      return result
    })
    .filter((item): item is ImageStudioResult => item !== null)
}

export async function generateImage(request: ImageStudioGenerateRequest): Promise<ImageStudioResult[]> {
  const baseUrl = normalizeGatewayBaseUrl(request.baseUrl)
  const hasReferenceImages = (request.images?.length ?? 0) > 0
  const endpoint = `${baseUrl}${hasReferenceImages ? '/images/edits' : '/images/generations'}`
  const headers: HeadersInit = {
    Authorization: `Bearer ${request.apiKey}`,
  }

  let body: BodyInit
  if (hasReferenceImages) {
    const formData = new FormData()
    formData.append('model', request.model)
    formData.append('prompt', request.prompt)
    formData.append('size', request.size)
    formData.append('n', String(request.n))
    formData.append('quality', request.quality)
    formData.append('background', request.background)
    formData.append('output_format', request.outputFormat)
    formData.append('response_format', 'b64_json')
    if (request.style?.trim()) {
      formData.append('style', request.style.trim())
    }
    request.images?.forEach((file) => {
      formData.append('image', file, file.name)
    })
    body = formData
  } else {
    headers['Content-Type'] = 'application/json'
    const payload: Record<string, unknown> = {
      model: request.model,
      prompt: request.prompt,
      size: request.size,
      n: request.n,
      quality: request.quality,
      background: request.background,
      output_format: request.outputFormat,
      response_format: 'b64_json',
    }
    if (request.style?.trim()) {
      payload.style = request.style.trim()
    }
    body = JSON.stringify(payload)
  }

  const response = await fetch(endpoint, {
    method: 'POST',
    headers,
    body,
  })

  const payload = await parseImageResponse(response)
  const results = responseToResults(payload, request.outputFormat)
  if (results.length === 0) {
    throw new Error('接口没有返回图片结果')
  }
  return results
}

export function isUsableImageKey(key: ApiKey): boolean {
  if (key.status !== 'active') return false
  if (key.expires_at && new Date(key.expires_at).getTime() <= Date.now()) return false
  return key.group?.platform === 'openai' && key.group?.allow_image_generation === true
}
