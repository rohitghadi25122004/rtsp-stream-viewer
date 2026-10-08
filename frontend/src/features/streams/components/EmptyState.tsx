import { CameraIcon } from '../../../components/icons'
import { AddStreamButton } from './AddStreamForm'
import styles from './EmptyState.module.css'

const examples = [
  { label: 'Basic', url: 'rtsp://192.168.1.20:554/stream1' },
  { label: 'With login', url: 'rtsp://admin:password@192.168.1.20:554/stream1' },
]

export function EmptyState() {
  return (
    <section className={styles.empty}>
      <span className={styles.icon}>
        <CameraIcon width={22} height={22} />
      </span>
      <h1 className={styles.title}>No streams yet</h1>
      <p className={styles.text}>Add a camera's RTSP URL to start watching it live. Each stream gets its own tile.</p>
      <AddStreamButton />
      <dl className={styles.examples}>
        {examples.map((ex) => (
          <div key={ex.label}>
            <dt>{ex.label}</dt>
            <dd>
              <code>{ex.url}</code>
            </dd>
          </div>
        ))}
      </dl>
    </section>
  )
}
