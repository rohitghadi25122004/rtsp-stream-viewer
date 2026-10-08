import { useCallback, useEffect, useState } from 'react'
import { ApiError } from '../../../api/client'
import { createStream, deleteStream, listStreams } from '../../../api/streams'
import type { CreateStreamInput, Stream } from '../../../api/types'

export function useStreams() {
  const [streams, setStreams] = useState<Stream[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback((signal?: AbortSignal) => {
    return listStreams(signal).then(
      (list) => {
        setStreams(list)
        setError(null)
        setLoading(false)
      },
      (err: unknown) => {
        if (signal?.aborted) return
        setError(err instanceof ApiError ? err.message : 'Could not load streams.')
        setLoading(false)
      },
    )
  }, [])

  useEffect(() => {
    const controller = new AbortController()
    load(controller.signal)
    return () => controller.abort()
  }, [load])

  const add = useCallback(async (input: CreateStreamInput) => {
    const created = await createStream(input)
    setStreams((prev) => [...prev, created])
  }, [])

  const remove = useCallback(async (id: string) => {
    let removed: Stream | undefined
    let index = -1
    setStreams((prev) => {
      index = prev.findIndex((s) => s.id === id)
      removed = prev[index]
      return prev.filter((s) => s.id !== id)
    })

    try {
      await deleteStream(id)
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) return
      if (removed) {
        const restored = removed
        setStreams((prev) => [...prev.slice(0, index), restored, ...prev.slice(index)])
      }
      setError(err instanceof ApiError ? err.message : 'Could not remove the stream.')
    }
  }, [])

  const reload = useCallback(() => {
    setLoading(true)
    void load()
  }, [load])

  return { streams, loading, error, add, remove, reload }
}
