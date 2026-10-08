package hub

type State string

const (
	StateIdle         State = "idle"
	StateConnecting   State = "connecting"
	StateLive         State = "live"
	StateReconnecting State = "reconnecting"
	StateError        State = "error"
	StateStopped      State = "stopped"
)

// Status is the pipeline state as reported to viewers. Code and Message
// explain the last failure while reconnecting or in error.
type Status struct {
	State     State  `json:"state"`
	Code      string `json:"code,omitempty"`
	Message   string `json:"message,omitempty"`
	RetryInMs int64  `json:"retryInMs,omitempty"`
}
