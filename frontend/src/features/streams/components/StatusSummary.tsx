import type { PlayerState } from '../../../player/MsePlayer'
import styles from './StatusSummary.module.css'

type Kind = PlayerState['kind']

const groups: { key: string; label: string; kinds: Kind[] }[] = [
  { key: 'live', label: 'live', kinds: ['live'] },
  { key: 'connecting', label: 'connecting', kinds: ['idle', 'connecting', 'buffering'] },
  { key: 'reconnecting', label: 'reconnecting', kinds: ['reconnecting'] },
  { key: 'error', label: 'failed', kinds: ['error'] },
  { key: 'paused', label: 'paused', kinds: ['paused'] },
]

export function StatusSummary({ states }: { states: Kind[] }) {
  return (
    <ul className={styles.summary} aria-label="Stream status">
      {groups.map((g) => {
        const count = states.filter((s) => g.kinds.includes(s)).length
        if (count === 0) return null
        return (
          <li key={g.key} data-group={g.key}>
            <span className={styles.dot} aria-hidden="true" />
            <span className={styles.count}>{count}</span> {g.label}
          </li>
        )
      })}
    </ul>
  )
}
