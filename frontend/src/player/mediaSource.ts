type MediaSourceConstructor = {
  new (): MediaSource
  isTypeSupported(type: string): boolean
}

// iOS Safari 17.1+ only exposes ManagedMediaSource, which is not yet in the
// TypeScript DOM library.
type WindowWithMms = Window & { ManagedMediaSource?: MediaSourceConstructor }

export function getMediaSourceCtor(): MediaSourceConstructor | undefined {
  if (typeof window === 'undefined') return undefined
  return window.MediaSource ?? (window as WindowWithMms).ManagedMediaSource
}

export function mimeFor(codec: string): string {
  return `video/mp4; codecs="${codec}"`
}

export function canPlay(codec: string): boolean {
  return getMediaSourceCtor()?.isTypeSupported(mimeFor(codec)) ?? false
}
