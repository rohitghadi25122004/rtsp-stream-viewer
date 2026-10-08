import { useEffect, useState } from 'react'
import { Button } from '../../../components/Button'
import { PlayIcon, RetryIcon } from '../../../components/icons'
import { Spinner } from '../../../components/Spinner'
import type { PlayerState } from '../../../player/MsePlayer'
import { describeError } from '../errorMessages'
import styles from './StreamOverlay.module.css'

type Props = {
  state: PlayerState
  onPlay: () => void
  onRetry: () => void
}

export function StreamOverlay({ state, onPlay, onRetry }: Props) {
  switch (state.kind) {
    case 'live':
      return null

    case 'paused':
      return (
        <div className={`${styles.overlay} ${styles.dim}`}>
          <button type="button" className={styles.play} onClick={onPlay} aria-label="Resume stream">
            <PlayIcon width={22} height={22} />
          </button>
        </div>
      )

    case 'reconnecting':
      return (
        <div className={styles.overlay}>
          <Spinner size={20} />
          <p className={styles.title}>{state.code ? describeError(state.code) : 'Connection lost.'}</p>
          <p className={styles.detail}>
            {state.retryInMs ? <Countdown key={state.retryInMs} ms={state.retryInMs} /> : 'Reconnecting'}
          </p>
          {state.message && <p className={styles.diagnostic}>{state.message}</p>}
        </div>
      )

    case 'error':
      return (
        <div className={styles.overlay}>
          <p className={styles.title}>{describeError(state.code)}</p>
          {state.message && <p className={styles.diagnostic}>{state.message}</p>}
          {state.retryable && (
            <Button onClick={onRetry} className={styles.retry}>
              <RetryIcon />
              Retry
            </Button>
          )}
        </div>
      )

    case 'removed':
      return (
        <div className={styles.overlay}>
          <p className={styles.title}>Stream removed.</p>
        </div>
      )

    default:
      return (
        <div className={styles.overlay}>
          <Spinner size={20} />
          <p className={styles.detail}>Connecting</p>
        </div>
      )
  }
}

function Countdown({ ms }: { ms: number }) {
  const [deadline] = useState(() => Date.now() + ms)
  const [remaining, setRemaining] = useState(Math.ceil(ms / 1000))

  useEffect(() => {
    const timer = setInterval(() => {
      setRemaining(Math.max(0, Math.ceil((deadline - Date.now()) / 1000)))
    }, 250)
    return () => clearInterval(timer)
  }, [deadline])

  return <>{remaining > 0 ? `Retrying in ${remaining}s` : 'Reconnecting'}</>
}
