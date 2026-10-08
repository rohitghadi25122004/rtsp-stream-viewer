import { LayoutAutoIcon, LayoutGridIcon } from '../../../components/icons'
import styles from './LayoutPicker.module.css'

export type Columns = 'auto' | 1 | 2 | 3 | 4

const options: { value: Columns; label: string }[] = [
  { value: 'auto', label: 'Fit all streams' },
  { value: 1, label: '1×1' },
  { value: 2, label: '2×2' },
  { value: 3, label: '3×3' },
  { value: 4, label: '4×4' },
]

type Props = {
  value: Columns
  onChange: (value: Columns) => void
}

export function LayoutPicker({ value, onChange }: Props) {
  return (
    <div className={styles.picker} role="radiogroup" aria-label="Grid layout">
      {options.map((opt) => (
        <button
          key={opt.label}
          type="button"
          role="radio"
          aria-checked={value === opt.value}
          aria-label={opt.label}
          title={opt.label}
          className={styles.option}
          onClick={() => onChange(opt.value)}
        >
          {opt.value === 'auto' ? <LayoutAutoIcon /> : <LayoutGridIcon n={opt.value} />}
        </button>
      ))}
    </div>
  )
}
