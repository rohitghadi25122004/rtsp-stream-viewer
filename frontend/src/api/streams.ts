import { API_URL, request } from './client'
import type { CreateStreamInput, Stream } from './types'

export async function listStreams(signal?: AbortSignal): Promise<Stream[]> {
  const { streams } = await request<{ streams: Stream[] }>('/api/streams', { signal })
  return streams
}

export function createStream(input: CreateStreamInput): Promise<Stream> {
  return request<Stream>('/api/streams', { method: 'POST', body: JSON.stringify(input) })
}

export function deleteStream(id: string): Promise<void> {
  return request<void>(`/api/streams/${encodeURIComponent(id)}`, { method: 'DELETE' })
}

export function streamSocketUrl(id: string): string {
  return `${API_URL.replace(/^http/, 'ws')}/ws/streams/${encodeURIComponent(id)}`
}
