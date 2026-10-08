import { useEffect, useRef, useState, type FormEvent } from 'react'
import { ApiError } from '../../../api/client'
import type { CreateStreamInput } from '../../../api/types'
import { Button, IconButton } from '../../../components/Button'
import { CloseIcon, PlusIcon } from '../../../components/icons'
import { Spinner } from '../../../components/Spinner'
import styles from './AddStreamForm.module.css'

const RTSP_URL = /^rtsps?:\/\/\S+$/i
const POPOVER_ID = 'add-stream'

/** Opens the add-stream panel. Can be placed anywhere on the page. */
export function AddStreamButton({ className }: { className?: string }) {
  return (
    <Button variant="primary" popoverTarget={POPOVER_ID} className={className}>
      <PlusIcon />
      Add stream
    </Button>
  )
}

type Props = {
  onAdd: (input: CreateStreamInput) => Promise<void>
}

export function AddStreamForm({ onAdd }: Props) {
  const panelRef = useRef<HTMLDivElement>(null)
  const urlRef = useRef<HTMLInputElement>(null)
  const [url, setUrl] = useState('')
  const [name, setName] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    const panel = panelRef.current
    if (!panel) return
    const onToggle = (e: Event) => {
      if ((e as ToggleEvent).newState === 'open') urlRef.current?.focus()
    }
    panel.addEventListener('toggle', onToggle)
    return () => panel.removeEventListener('toggle', onToggle)
  }, [])

  async function handleSubmit(e: FormEvent) {
    e.preventDefault()
    const trimmed = url.trim()
    if (!RTSP_URL.test(trimmed)) {
      setError('Enter a URL that starts with rtsp:// or rtsps://')
      urlRef.current?.focus()
      return
    }

    setSubmitting(true)
    setError(null)
    try {
      await onAdd({ url: trimmed, name: name.trim() || undefined })
      setUrl('')
      setName('')
      panelRef.current?.hidePopover()
    } catch (err) {
      setError(messageFor(err))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div ref={panelRef} id={POPOVER_ID} popover="auto" className={styles.panel} aria-labelledby="add-stream-title">
      <form onSubmit={handleSubmit} noValidate>
        <div className={styles.head}>
          <h2 id="add-stream-title">Add stream</h2>
          <IconButton label="Close" popoverTarget={POPOVER_ID} popoverTargetAction="hide">
            <CloseIcon />
          </IconButton>
        </div>

        <div className={styles.body}>
          <label className={styles.field}>
            <span className={styles.label}>RTSP URL</span>
            <input
              ref={urlRef}
              className={styles.mono}
              type="text"
              inputMode="url"
              autoComplete="off"
              spellCheck={false}
              placeholder="rtsp://user:pass@192.168.1.20:554/stream1"
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              aria-invalid={error !== null}
              aria-describedby={error ? 'add-stream-error' : 'add-stream-hint'}
            />
            {error ? (
              <span id="add-stream-error" className={styles.error} role="alert">
                {error}
              </span>
            ) : (
              <span id="add-stream-hint" className={styles.hint}>
                Login details can go in the URL. The password is hidden once saved.
              </span>
            )}
          </label>

          <label className={styles.field}>
            <span className={styles.label}>
              Name <span className={styles.optional}>optional</span>
            </span>
            <input
              type="text"
              autoComplete="off"
              placeholder="Front door"
              maxLength={64}
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </label>
        </div>

        <div className={styles.actions}>
          <Button variant="ghost" popoverTarget={POPOVER_ID} popoverTargetAction="hide">
            Cancel
          </Button>
          <Button type="submit" variant="primary" disabled={submitting}>
            {submitting && <Spinner size={14} />}
            Add stream
          </Button>
        </div>
      </form>
    </div>
  )
}

function messageFor(err: unknown): string {
  if (!(err instanceof ApiError)) return 'Something went wrong. Try again.'
  switch (err.code) {
    case 'duplicate_stream':
      return 'This stream is already added.'
    case 'stream_limit':
      return 'Stream limit reached. Remove a stream to add another.'
    default:
      return err.message
  }
}
