import { nextDelay } from './backoff'
import { bufferedRange, liveEdgeSeek, trimRange } from './buffer'
import { canPlay, getMediaSourceCtor, mimeFor } from './mediaSource'
import { CloseCode, parseServerMessage, type StatusMessage } from './protocol'

export type PlayerState =
  | { kind: 'idle' }
  | { kind: 'connecting' }
  | { kind: 'buffering' }
  | { kind: 'live' }
  | { kind: 'paused' }
  | { kind: 'reconnecting'; code?: string; message?: string; retryInMs?: number }
  | { kind: 'error'; code: string; message?: string; retryable: boolean }
  | { kind: 'removed' }

// A backlog this large means appends have fallen far behind the socket
// (usually a backgrounded tab). Reconnecting is faster than catching up.
const MAX_QUEUE = 120
const MAX_PLAYBACK_FAILURES = 3

/**
 * Plays one stream: reads fMP4 segments from the backend WebSocket and
 * feeds them to a MediaSource attached to the video element.
 */
export class MsePlayer {
  private readonly video: HTMLVideoElement
  private readonly url: string
  private readonly onState: (state: PlayerState) => void

  private socket: WebSocket | null = null
  private mediaSource: MediaSource | null = null
  private sourceBuffer: SourceBuffer | null = null
  private queue: ArrayBuffer[] = []
  private started = false
  private wantPlaying = false
  private attempt = 0
  private failures = 0
  private reconnectTimer: ReturnType<typeof setTimeout> | undefined
  private state: PlayerState = { kind: 'idle' }

  constructor(video: HTMLVideoElement, url: string, onState: (state: PlayerState) => void) {
    this.video = video
    this.url = url
    this.onState = onState

    video.muted = true
    video.playsInline = true
    video.disableRemotePlayback = true
    video.addEventListener('playing', this.handlePlaying)
    video.addEventListener('waiting', this.handleWaiting)
    video.addEventListener('error', this.handlePlaybackError)
    document.addEventListener('visibilitychange', this.handleVisibility)
  }

  play(): void {
    this.wantPlaying = true
    if (this.socket || this.reconnectTimer) return
    this.connect()
  }

  pause(): void {
    this.stop()
    this.video.pause()
    this.setState({ kind: 'paused' })
  }

  /** Disconnects without reporting a user pause, e.g. for off-screen tiles. */
  suspend(): void {
    this.stop()
    this.setState({ kind: 'idle' })
  }

  retry(): void {
    this.stop()
    this.attempt = 0
    this.failures = 0
    this.play()
  }

  destroy(): void {
    this.stop()
    this.teardownMedia()
    this.video.removeEventListener('playing', this.handlePlaying)
    this.video.removeEventListener('waiting', this.handleWaiting)
    this.video.removeEventListener('error', this.handlePlaybackError)
    document.removeEventListener('visibilitychange', this.handleVisibility)
  }

  private stop(): void {
    this.wantPlaying = false
    clearTimeout(this.reconnectTimer)
    this.reconnectTimer = undefined
    this.closeSocket()
  }

  private connect(): void {
    this.reconnectTimer = undefined
    if (!getMediaSourceCtor()) {
      this.setState({ kind: 'error', code: 'mse_unsupported', retryable: false })
      return
    }

    const socket = new WebSocket(this.url)
    socket.binaryType = 'arraybuffer'
    socket.onmessage = (e: MessageEvent<string | ArrayBuffer>) => {
      if (socket === this.socket) this.handleMessage(e.data)
    }
    socket.onclose = (e) => {
      if (socket === this.socket) this.handleClose(e.code)
    }
    this.socket = socket

    if (this.state.kind !== 'reconnecting') this.setState({ kind: 'connecting' })
  }

  private closeSocket(): void {
    const socket = this.socket
    this.socket = null
    if (!socket) return
    // Closing a socket that is still connecting logs a browser warning, so
    // let it open first. Its handlers already ignore it.
    if (socket.readyState === WebSocket.CONNECTING) socket.onopen = () => socket.close(CloseCode.Normal)
    else if (socket.readyState === WebSocket.OPEN) socket.close(CloseCode.Normal)
  }

  private handleMessage(data: string | ArrayBuffer): void {
    if (typeof data !== 'string') {
      this.queue.push(data)
      if (this.queue.length > MAX_QUEUE) {
        this.restart()
        return
      }
      this.pump()
      return
    }

    const msg = parseServerMessage(data)
    if (msg?.type === 'init') this.handleInit(msg.codec)
    else if (msg?.type === 'status') this.handleStatus(msg)
  }

  private handleStatus(msg: StatusMessage): void {
    switch (msg.state) {
      case 'connecting':
        if (this.state.kind !== 'reconnecting') this.setState({ kind: 'connecting' })
        break
      case 'reconnecting':
        this.setState({ kind: 'reconnecting', code: msg.code, message: msg.message, retryInMs: msg.retryInMs })
        break
      case 'error':
        this.setState({ kind: 'error', code: msg.code ?? 'exited', message: msg.message, retryable: true })
        break
      // 'live' is reported from the video element's playing event, and
      // 'stopped' is always followed by a close frame.
    }
  }

  private handleInit(codec: string): void {
    if (!canPlay(codec)) {
      this.stop()
      this.setState({ kind: 'error', code: 'codec_unsupported', message: codec, retryable: false })
      return
    }

    // A new init segment means the backend restarted the pipeline:
    // timestamps start over and the codec may differ, so start fresh.
    this.teardownMedia()
    const Ctor = getMediaSourceCtor()!
    const mediaSource = new Ctor()
    this.mediaSource = mediaSource
    const objectUrl = URL.createObjectURL(mediaSource)
    this.video.src = objectUrl

    mediaSource.addEventListener(
      'sourceopen',
      () => {
        URL.revokeObjectURL(objectUrl)
        if (mediaSource !== this.mediaSource) return
        try {
          const sb = mediaSource.addSourceBuffer(mimeFor(codec))
          sb.addEventListener('updateend', this.handleUpdateEnd)
          sb.addEventListener('error', this.handlePlaybackError)
          this.sourceBuffer = sb
          this.pump()
        } catch {
          this.handlePlaybackError()
        }
      },
      { once: true },
    )

    if (this.state.kind !== 'reconnecting') this.setState({ kind: 'buffering' })
  }

  private pump(): void {
    const sb = this.sourceBuffer
    if (!sb || sb.updating || this.mediaSource?.readyState !== 'open') return

    const next = this.queue.shift()
    if (!next) return
    try {
      sb.appendBuffer(next)
    } catch {
      // Usually QuotaExceededError. Starting over at the live edge is
      // simpler than trying to make room in the buffer.
      this.restart()
    }
  }

  private readonly handleUpdateEnd = () => {
    const sb = this.sourceBuffer
    if (!sb) return

    if (!this.started && sb.buffered.length > 0) {
      this.started = true
      this.video.play().catch(() => {})
    }

    const range = bufferedRange(sb.buffered)
    this.seekToLive()

    const trim = trimRange(range, this.video.currentTime)
    if (trim) {
      sb.remove(trim[0], trim[1])
      return
    }
    this.pump()
  }

  private seekToLive(): void {
    const sb = this.sourceBuffer
    if (!sb) return
    const target = liveEdgeSeek(bufferedRange(sb.buffered), this.video.currentTime)
    if (target !== null) this.video.currentTime = target
  }

  private readonly handlePlaying = () => {
    this.attempt = 0
    this.failures = 0
    this.setState({ kind: 'live' })
  }

  private readonly handleWaiting = () => {
    this.seekToLive()
  }

  private readonly handleVisibility = () => {
    if (document.visibilityState === 'visible') this.seekToLive()
  }

  private readonly handlePlaybackError = () => {
    if (!this.mediaSource) return
    this.failures++
    if (this.failures >= MAX_PLAYBACK_FAILURES) {
      this.stop()
      this.teardownMedia()
      this.setState({ kind: 'error', code: 'playback_failed', retryable: true })
      return
    }
    this.restart()
  }

  private handleClose(code: number): void {
    this.socket = null
    if (!this.wantPlaying) return

    switch (code) {
      case CloseCode.StreamRemoved:
      case CloseCode.StreamNotFound:
        this.wantPlaying = false
        this.setState({ kind: 'removed' })
        return
      case CloseCode.TooSlow:
        this.scheduleReconnect(0)
        return
      default:
        this.scheduleReconnect(this.attempt++)
    }
  }

  /** Drops the connection and media, then reconnects shortly. */
  private restart(): void {
    this.closeSocket()
    this.teardownMedia()
    this.scheduleReconnect(0)
  }

  private scheduleReconnect(attempt: number): void {
    const delay = nextDelay(attempt)
    this.wantPlaying = true
    clearTimeout(this.reconnectTimer)
    this.reconnectTimer = setTimeout(() => this.connect(), delay)
    this.setState({ kind: 'reconnecting', retryInMs: delay })
  }

  private teardownMedia(): void {
    const sb = this.sourceBuffer
    if (sb) {
      sb.removeEventListener('updateend', this.handleUpdateEnd)
      sb.removeEventListener('error', this.handlePlaybackError)
    }
    this.sourceBuffer = null
    this.mediaSource = null
    this.queue = []
    this.started = false
    if (this.video.getAttribute('src')) {
      this.video.removeAttribute('src')
      this.video.load()
    }
  }

  private setState(state: PlayerState): void {
    this.state = state
    this.onState(state)
  }
}
