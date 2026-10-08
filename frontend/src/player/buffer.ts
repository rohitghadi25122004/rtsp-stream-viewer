export type Range = { start: number; end: number }

// Playback drifts behind the live edge when the tab is throttled or the
// network stalls. Past this many seconds we jump forward instead of
// playing stale video.
const MAX_LATENCY = 1.5
// Seek slightly behind the newest frame so the decoder has data to play.
const LIVE_EDGE_OFFSET = 0.3
// Keep this much history behind the playhead; older data is removed so the
// SourceBuffer never hits its quota during long sessions.
const KEEP_BEHIND = 10
const TRIM_AFTER = 30

export function bufferedRange(buffered: TimeRanges): Range | null {
  if (buffered.length === 0) return null
  return { start: buffered.start(0), end: buffered.end(buffered.length - 1) }
}

/** Returns the time to seek to, or null when playback is close enough to live. */
export function liveEdgeSeek(range: Range | null, currentTime: number): number | null {
  if (!range) return null
  const target = Math.max(range.start, range.end - LIVE_EDGE_OFFSET)
  if (currentTime < range.start || range.end - currentTime > MAX_LATENCY) return target
  return null
}

/** Returns the [start, end) span to remove from the buffer, if any. */
export function trimRange(range: Range | null, currentTime: number): [number, number] | null {
  if (!range || currentTime - range.start <= TRIM_AFTER) return null
  return [range.start, currentTime - KEEP_BEHIND]
}
