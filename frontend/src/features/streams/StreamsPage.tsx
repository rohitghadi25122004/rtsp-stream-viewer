import { useCallback, useState } from 'react'
import { Button } from '../../components/Button'
import { CameraIcon } from '../../components/icons'
import { usePersistentState } from '../../lib/usePersistentState'
import type { PlayerState } from '../../player/MsePlayer'
import { AddStreamButton, AddStreamForm } from './components/AddStreamForm'
import { EmptyState } from './components/EmptyState'
import { LayoutPicker, type Columns } from './components/LayoutPicker'
import { StatusSummary } from './components/StatusSummary'
import { StreamGrid, StreamGridSkeleton } from './components/StreamGrid'
import { useStreams } from './hooks/useStreams'
import styles from './StreamsPage.module.css'

type Kind = PlayerState['kind']

export function StreamsPage() {
  const { streams, loading, error, add, remove, reload } = useStreams()
  const [columns, setColumns] = usePersistentState<Columns>('rtsp-viewer:columns', 'auto')
  const [states, setStates] = useState<Record<string, Kind>>({})

  const handleState = useCallback((id: string, kind: Kind) => {
    setStates((prev) => (prev[id] === kind ? prev : { ...prev, [id]: kind }))
  }, [])

  const hasStreams = !loading && streams.length > 0

  return (
    <div className={styles.page}>
      <header className={styles.topbar}>
        <div className={styles.brand}>
          <span className={styles.mark}>
            <CameraIcon width={16} height={16} />
          </span>
          <span className={styles.product}>RTSP Viewer</span>
        </div>
        <AddStreamButton />
      </header>

      <AddStreamForm onAdd={add} />

      <main className={styles.main}>
        {hasStreams && (
          <div className={styles.heading}>
            <div className={styles.titleGroup}>
              <h1 className={styles.title}>
                Streams <span className={styles.total}>{streams.length}</span>
              </h1>
              <StatusSummary states={streams.map((s) => states[s.id] ?? 'idle')} />
            </div>
            <LayoutPicker value={columns} onChange={setColumns} />
          </div>
        )}

        {error && (
          <div className={styles.banner} role="alert">
            <span>{error}</span>
            <Button onClick={reload}>Retry</Button>
          </div>
        )}

        {loading ? (
          <StreamGridSkeleton />
        ) : streams.length === 0 ? (
          !error && <EmptyState />
        ) : (
          <StreamGrid streams={streams} columns={columns} onRemove={remove} onStateChange={handleState} />
        )}
      </main>
    </div>
  )
}
