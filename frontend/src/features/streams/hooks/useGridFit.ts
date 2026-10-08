import { useLayoutEffect, useState, type RefObject } from 'react'
import type { Columns } from '../components/LayoutPicker'

export const GRID_GAP = 12
const MIN_TILE_WIDTH = 280
// Height a tile adds below its 16:9 video: the 44px name bar plus borders.
const TILE_CHROME = 46
const BOTTOM_SPACE = 24

export type GridFit = { cols: number; tileWidth: number }

/**
 * Sizes the tiles to the screen.
 *
 * - 'auto': picks the column count that shows every stream on one screen
 *   at the largest size.
 * - n: an n×n wall, like an NVR. n columns and n rows fill the screen and
 *   any further streams continue below.
 *
 * Tiles never go below MIN_TILE_WIDTH; past that they fill the width and
 * the page scrolls.
 */
export function fitGrid(count: number, width: number, height: number, columns: Columns): GridFit {
  const total = Math.max(count, 1)
  const widthFor = (cols: number) => (width - (cols - 1) * GRID_GAP) / cols
  const heightFor = (rows: number) => ((height - (rows - 1) * GRID_GAP) / rows - TILE_CHROME) * (16 / 9)
  const maxCols = Math.max(1, Math.floor((width + GRID_GAP) / (MIN_TILE_WIDTH + GRID_GAP)))

  if (columns !== 'auto') {
    const cols = Math.min(columns, maxCols)
    const rows = Math.min(cols, Math.ceil(total / cols))
    const tileWidth = Math.min(widthFor(cols), Math.max(heightFor(rows), MIN_TILE_WIDTH))
    return { cols, tileWidth: Math.floor(tileWidth) }
  }

  let best: GridFit = { cols: 1, tileWidth: 0 }
  for (let cols = 1; cols <= Math.min(total, maxCols); cols++) {
    const tileWidth = Math.min(widthFor(cols), heightFor(Math.ceil(total / cols)))
    if (tileWidth > best.tileWidth) best = { cols, tileWidth }
  }

  if (best.tileWidth < MIN_TILE_WIDTH) {
    const cols = Math.min(total, maxCols)
    return { cols, tileWidth: Math.floor(widthFor(cols)) }
  }
  return { cols: best.cols, tileWidth: Math.floor(best.tileWidth) }
}

/** Measures the space below the grid's top edge and refits on resize. */
export function useGridFit(ref: RefObject<HTMLElement | null>, count: number, columns: Columns): GridFit | null {
  const [fit, setFit] = useState<GridFit | null>(null)

  useLayoutEffect(() => {
    const el = ref.current
    if (!el) return

    const measure = () => {
      const rect = el.getBoundingClientRect()
      const top = rect.top + window.scrollY
      const next = fitGrid(count, rect.width, window.innerHeight - top - BOTTOM_SPACE, columns)
      setFit((prev) => (prev?.cols === next.cols && prev.tileWidth === next.tileWidth ? prev : next))
    }

    const observer = new ResizeObserver(measure)
    observer.observe(el)
    window.addEventListener('resize', measure)
    return () => {
      observer.disconnect()
      window.removeEventListener('resize', measure)
    }
  }, [ref, count, columns])

  return fit
}
