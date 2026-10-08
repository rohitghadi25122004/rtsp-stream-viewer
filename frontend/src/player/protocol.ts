export type ServerState = 'connecting' | 'live' | 'reconnecting' | 'error' | 'stopped'

export type StatusMessage = {
  type: 'status'
  state: ServerState
  code?: string
  message?: string
  retryInMs?: number
}

export type InitMessage = {
  type: 'init'
  codec: string
}

export type ServerMessage = StatusMessage | InitMessage

export const CloseCode = {
  Normal: 1000,
  GoingAway: 1001,
  StreamRemoved: 4000,
  StreamNotFound: 4004,
  TooSlow: 4008,
} as const

const serverStates: readonly string[] = ['connecting', 'live', 'reconnecting', 'error', 'stopped']

export function parseServerMessage(raw: string): ServerMessage | null {
  let data: unknown
  try {
    data = JSON.parse(raw)
  } catch {
    return null
  }
  if (typeof data !== 'object' || data === null) return null

  const msg = data as Record<string, unknown>
  if (msg.type === 'init' && typeof msg.codec === 'string') {
    return { type: 'init', codec: msg.codec }
  }
  if (msg.type === 'status' && typeof msg.state === 'string' && serverStates.includes(msg.state)) {
    return {
      type: 'status',
      state: msg.state as ServerState,
      code: typeof msg.code === 'string' ? msg.code : undefined,
      message: typeof msg.message === 'string' ? msg.message : undefined,
      retryInMs: typeof msg.retryInMs === 'number' ? msg.retryInMs : undefined,
    }
  }
  return null
}
