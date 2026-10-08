import { memo, useEffect, useRef, useState } from 'react'
import { useInView } from '../../../lib/useInView'
import type { PlayerState } from '../../../player/MsePlayer'
import { useStreamPlayer } from '../hooks/useStreamPlayer'
import { StatusBadge } from './StatusBadge'
import { StreamControls } from './StreamControls'
import { StreamOverlay } from './StreamOverlay'
import styles from './StreamTile.module.css'

type Props = {
  id: string
  name: string
  url: string
  onRemove: (id: string) => void
  onStateChange: (id: string, kind: PlayerState['kind']) => void
}

type FullscreenVideo = HTMLVideoElement & { webkitEnterFullscreen?: () => void }

export const StreamTile = memo(function StreamTile({ id, name, url, onRemove, onStateChange }: Props) {
  const tileRef = useRef<HTMLElement>(null)
  const videoRef = useRef<HTMLVideoElement>(null)
  const visible = useInView(tileRef, '200px')
  const { state, playing, toggle, retry } = useStreamPlayer(videoRef, id, { visible })
  const [resolution, setResolution] = useState<string | null>(null)

  useEffect(() => {
    onStateChange(id, state.kind)
  }, [id, state.kind, onStateChange])

  function updateResolution() {
    const video = videoRef.current
    setResolution(video && video.videoWidth > 0 ? `${video.videoWidth}×${video.videoHeight}` : null)
  }

  function enterFullscreen() {
    const tile = tileRef.current
    if (tile?.requestFullscreen) {
      tile.requestFullscreen().catch(() => {})
    } else {
      // iOS Safari only supports fullscreen on the video element itself.
      ;(videoRef.current as FullscreenVideo | null)?.webkitEnterFullscreen?.()
    }
  }

  return (
    <article ref={tileRef} className={styles.tile} data-state={state.kind} aria-label={name}>
      <div className={styles.media} onDoubleClick={enterFullscreen}>
        <video
          ref={videoRef}
          className={styles.video}
          muted
          playsInline
          autoPlay
          onLoadedMetadata={updateResolution}
          onResize={updateResolution}
        />
        <StreamOverlay state={state} onPlay={toggle} onRetry={retry} />
        <div className={styles.top}>
          <StatusBadge state={state.kind} />
          <StreamControls
            playing={playing}
            onToggle={toggle}
            onFullscreen={enterFullscreen}
            onRemove={() => onRemove(id)}
          />
        </div>
      </div>
      <footer className={styles.bar}>
        <h2 className={styles.name}>{name}</h2>
        <p className={styles.url} title={url}>
          {url}
        </p>
        {state.kind === 'live' && resolution && <span className={styles.resolution}>{resolution}</span>}
      </footer>
    </article>
  )
})
