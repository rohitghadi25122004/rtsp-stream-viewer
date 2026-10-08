import { useEffect, useState } from 'react'
import { IconButton } from '../../../components/Button'
import { ExpandIcon, PauseIcon, PlayIcon, TrashIcon } from '../../../components/icons'
import styles from './StreamControls.module.css'

const CONFIRM_WINDOW_MS = 3000

type Props = {
  playing: boolean
  onToggle: () => void
  onFullscreen: () => void
  onRemove: () => void
}

export function StreamControls({ playing, onToggle, onFullscreen, onRemove }: Props) {
  const [confirming, setConfirming] = useState(false)

  useEffect(() => {
    if (!confirming) return
    const timer = setTimeout(() => setConfirming(false), CONFIRM_WINDOW_MS)
    return () => clearTimeout(timer)
  }, [confirming])

  return (
    <div className={styles.controls} data-confirming={confirming || undefined}>
      <IconButton label={playing ? 'Pause' : 'Play'} variant="overlay" onClick={onToggle}>
        {playing ? <PauseIcon width={14} height={14} /> : <PlayIcon width={14} height={14} />}
      </IconButton>
      <IconButton label="Fullscreen" variant="overlay" onClick={onFullscreen}>
        <ExpandIcon width={14} height={14} />
      </IconButton>
      {confirming ? (
        <IconButton label="Confirm remove" variant="danger" onClick={onRemove} autoFocus>
          Remove
        </IconButton>
      ) : (
        <IconButton label="Remove stream" variant="overlay" onClick={() => setConfirming(true)}>
          <TrashIcon width={14} height={14} />
        </IconButton>
      )}
    </div>
  )
}
