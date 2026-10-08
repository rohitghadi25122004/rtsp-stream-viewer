import { useRef, type CSSProperties } from 'react'
import type { Stream } from '../../../api/types'
import type { PlayerState } from '../../../player/MsePlayer'
import { GRID_GAP, useGridFit } from '../hooks/useGridFit'
import type { Columns } from './LayoutPicker'
import { StreamTile } from './StreamTile'
import styles from './StreamGrid.module.css'

type Props = {
  streams: Stream[]
  columns: Columns
  onRemove: (id: string) => void
  onStateChange: (id: string, kind: PlayerState['kind']) => void
}

export function StreamGrid({ streams, columns, onRemove, onStateChange }: Props) {
  const ref = useRef<HTMLElement>(null)
  const fit = useGridFit(ref, streams.length, columns)

  const style = fit && {
    '--cols': Math.min(fit.cols, streams.length),
    '--tile-width': `${fit.tileWidth}px`,
    '--gap': `${GRID_GAP}px`,
  }

  return (
    <section
      ref={ref}
      className={styles.grid}
      style={(style || undefined) as CSSProperties | undefined}
      data-fitted={fit ? '' : undefined}
      aria-label="Streams"
    >
      {streams.map((s) => (
        <StreamTile key={s.id} id={s.id} name={s.name} url={s.url} onRemove={onRemove} onStateChange={onStateChange} />
      ))}
    </section>
  )
}

export function StreamGridSkeleton() {
  return (
    <div className={styles.grid} aria-busy="true" aria-label="Loading streams">
      {[0, 1, 2].map((i) => (
        <div key={i} className={styles.skeleton}>
          <div className={styles.skeletonMedia} />
          <div className={styles.skeletonBar}>
            <span />
          </div>
        </div>
      ))}
    </div>
  )
}
