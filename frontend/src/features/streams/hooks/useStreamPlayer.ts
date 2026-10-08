import { useCallback, useEffect, useRef, useState, type RefObject } from 'react'
import { streamSocketUrl } from '../../../api/streams'
import { MsePlayer, type PlayerState } from '../../../player/MsePlayer'

// How long a tile may stay scrolled out of view before its connection is
// dropped. Short scrolls past a tile shouldn't cause a reconnect.
const OFFSCREEN_GRACE_MS = 5000

export function useStreamPlayer(
  videoRef: RefObject<HTMLVideoElement | null>,
  streamId: string,
  { visible }: { visible: boolean },
) {
  const [state, setState] = useState<PlayerState>({ kind: 'idle' })
  const [playing, setPlaying] = useState(true)
  const playerRef = useRef<MsePlayer | null>(null)

  useEffect(() => {
    const video = videoRef.current
    if (!video) return

    const player = new MsePlayer(video, streamSocketUrl(streamId), setState)
    playerRef.current = player
    return () => {
      player.destroy()
      playerRef.current = null
    }
  }, [videoRef, streamId])

  useEffect(() => {
    const player = playerRef.current
    if (!player) return

    if (!playing) {
      player.pause()
      return
    }
    if (visible) {
      player.play()
      return
    }
    const timer = setTimeout(() => player.suspend(), OFFSCREEN_GRACE_MS)
    return () => clearTimeout(timer)
  }, [playing, visible, streamId])

  const toggle = useCallback(() => setPlaying((p) => !p), [])

  const retry = useCallback(() => {
    setPlaying(true)
    playerRef.current?.retry()
  }, [])

  return { state, playing, toggle, retry }
}
