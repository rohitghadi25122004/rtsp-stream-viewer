const messages: Record<string, string> = {
  unreachable: "Can't reach the camera.",
  unauthorized: 'The camera rejected the credentials.',
  not_found: 'Stream path not found.',
  stalled: 'No video received.',
  exited: 'Stream interrupted.',
  invalid_data: 'The camera sent data that could not be read.',
  no_video: 'This stream has no video track.',
  mse_unsupported: "Live playback needs Media Source Extensions, which this browser doesn't support.",
  codec_unsupported: "This browser can't decode the stream's video format.",
  playback_failed: 'Playback failed in this browser.',
}

export function describeError(code: string | undefined): string {
  return (code && messages[code]) || 'Something went wrong with this stream.'
}
