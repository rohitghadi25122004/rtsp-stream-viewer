import type { PlayerState } from '../../../player/MsePlayer'
import styles from './StatusBadge.module.css'

const labels: Record<PlayerState['kind'], string> = {
  idle: 'Connecting',
  connecting: 'Connecting',
  buffering: 'Connecting',
  live: 'Live',
  paused: 'Paused',
  reconnecting: 'Reconnecting',
  error: 'Error',
  removed: 'Removed',
}

export function StatusBadge({ state }: { state: PlayerState['kind'] }) {
  return (
    <span className={styles.badge} data-state={state} role="status">
      <span className={styles.dot} aria-hidden="true" />
      {labels[state]}
    </span>
  )
}
