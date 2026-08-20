package workflow

import "errors"

type State string

const (
	StateInit     State = "INIT"
	StateChecking State = "CHECKING"
	StateReady    State = "READY"
	StateStarting State = "STARTING"
	StateRunning  State = "RUNNING"
	StateDegraded State = "DEGRADED"
	StateFailed   State = "FAILED"
	StateStopping State = "STOPPING"
)

var (
	ErrWorkflowFailed  = errors.New("WORKFLOW_FAILED")
	ErrWorkflowRunning = errors.New("WORKFLOW_RUNNING")
)

type Diagnostic struct {
	Module     string `json:"module"`
	Code       string `json:"code"`
	Reason     string `json:"reason"`
	Suggestion string `json:"suggestion"`
}

type Status struct {
	State      State       `json:"state"`
	Running    bool        `json:"running"`
	Diagnostic *Diagnostic `json:"diagnostic,omitempty"`
}

type StartOptions struct {
	WebDAVUsername string
	WebDAVPassword string
}
