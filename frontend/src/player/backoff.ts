const BASE_MS = 500
const MAX_MS = 10_000

/** Exponential reconnect delay with up to 30% jitter. */
export function nextDelay(attempt: number, random = Math.random): number {
  const base = Math.min(MAX_MS, BASE_MS * 2 ** attempt)
  return Math.round(base * (1 + random() * 0.3))
}
