// Package agui implements AG-UI protocol SSE streaming for workflow state.
package agui

import "time"

// EventType identifies an AG-UI event.
type EventType string

const (
	EventRunStarted    EventType = "RUN_STARTED"
	EventRunFinished   EventType = "RUN_FINISHED"
	EventRunError      EventType = "RUN_ERROR"
	EventStepStarted   EventType = "STEP_STARTED"
	EventStepFinished  EventType = "STEP_FINISHED"
	EventStateSnapshot EventType = "STATE_SNAPSHOT"
	EventStateDelta    EventType = "STATE_DELTA"
	EventCustom        EventType = "CUSTOM"
)

// Event is a single SSE event emitted to the client.
type Event struct {
	Type       EventType `json:"type"`
	Timestamp  time.Time `json:"timestamp"`
	WorkflowID string    `json:"workflow_id"`
	Data       any       `json:"data,omitempty"`
}

// StateSnapshotData carries full state + UI schema in a STATE_SNAPSHOT event.
type StateSnapshotData struct {
	Phase    string `json:"phase"`
	State    any    `json:"state"`
	UISchema any    `json:"ui_schema"`
}

// StateDeltaData carries field-level deltas in a STATE_DELTA event.
type StateDeltaData struct {
	Phase    string  `json:"phase"`
	Patches  []Patch `json:"patches"`
	UISchema any     `json:"ui_schema"`
}

// Patch is an RFC 6902-style JSON Patch operation.
type Patch struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value,omitempty"`
}

// StepData carries phase transition info.
type StepData struct {
	Phase string `json:"phase"`
}

// ErrorData carries error info for RUN_ERROR events.
type ErrorData struct {
	Message string `json:"message"`
}
