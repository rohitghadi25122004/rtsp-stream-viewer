export type StreamState = 'idle' | 'connecting' | 'live' | 'reconnecting' | 'error' | 'stopped'

export type Stream = {
  id: string
  name: string
  url: string
  createdAt: string
  status: { state: StreamState; viewers: number }
}

export type CreateStreamInput = {
  url: string
  name?: string
}
