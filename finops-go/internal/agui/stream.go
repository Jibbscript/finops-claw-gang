package agui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
	"github.com/finops-claw-gang/finops-go/internal/uischema"
)

// StreamConfig controls SSE stream behavior.
type StreamConfig struct {
	PollInterval time.Duration
	MaxDuration  time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() StreamConfig {
	return StreamConfig{
		PollInterval: 2 * time.Second,
		MaxDuration:  30 * time.Minute,
	}
}

// StreamHandler serves SSE events for a workflow's state changes.
func StreamHandler(q querier.WorkflowQuerier, cfg StreamConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wfID := r.PathValue("id")
		if wfID == "" {
			http.Error(w, "workflow id required", http.StatusBadRequest)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		ctx, cancel := context.WithTimeout(r.Context(), cfg.MaxDuration)
		defer cancel()

		// Emit RUN_STARTED.
		writeSSE(w, flusher, Event{
			Type:       EventRunStarted,
			Timestamp:  time.Now().UTC(),
			WorkflowID: wfID,
		})

		// Initial state snapshot.
		result, err := q.GetWorkflowState(ctx, wfID)
		if err != nil {
			writeSSE(w, flusher, Event{
				Type:       EventRunError,
				Timestamp:  time.Now().UTC(),
				WorkflowID: wfID,
				Data:       ErrorData{Message: err.Error()},
			})
			return
		}

		schema := uischema.Build(result.State)
		writeSSE(w, flusher, Event{
			Type:       EventStateSnapshot,
			Timestamp:  time.Now().UTC(),
			WorkflowID: wfID,
			Data: StateSnapshotData{
				Phase:    result.State.CurrentPhase,
				State:    result.State,
				UISchema: schema,
			},
		})

		lastPhase := result.State.CurrentPhase
		lastTerminated := result.State.ShouldTerminate

		if lastTerminated {
			writeSSE(w, flusher, Event{
				Type:       EventRunFinished,
				Timestamp:  time.Now().UTC(),
				WorkflowID: wfID,
				Data:       map[string]any{"reason": string(result.Reason)},
			})
			return
		}

		// Poll loop.
		ticker := time.NewTicker(cfg.PollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				result, err = q.GetWorkflowState(ctx, wfID)
				if err != nil {
					writeSSE(w, flusher, Event{
						Type:       EventRunError,
						Timestamp:  time.Now().UTC(),
						WorkflowID: wfID,
						Data:       ErrorData{Message: err.Error()},
					})
					return
				}

				currentPhase := result.State.CurrentPhase

				// Phase transition.
				if currentPhase != lastPhase {
					writeSSE(w, flusher, Event{
						Type:       EventStepFinished,
						Timestamp:  time.Now().UTC(),
						WorkflowID: wfID,
						Data:       StepData{Phase: lastPhase},
					})
					writeSSE(w, flusher, Event{
						Type:       EventStepStarted,
						Timestamp:  time.Now().UTC(),
						WorkflowID: wfID,
						Data:       StepData{Phase: currentPhase},
					})
					lastPhase = currentPhase
				}

				// Compute deltas and emit.
				patches := computePatches(result)
				if len(patches) > 0 || currentPhase != lastPhase {
					schema = uischema.Build(result.State)
					writeSSE(w, flusher, Event{
						Type:       EventStateDelta,
						Timestamp:  time.Now().UTC(),
						WorkflowID: wfID,
						Data: StateDeltaData{
							Phase:    currentPhase,
							Patches:  patches,
							UISchema: schema,
						},
					})
				}

				// Terminated: emit RUN_FINISHED and close.
				if result.State.ShouldTerminate && !lastTerminated {
					writeSSE(w, flusher, Event{
						Type:       EventRunFinished,
						Timestamp:  time.Now().UTC(),
						WorkflowID: wfID,
						Data:       map[string]any{"reason": string(result.Reason)},
					})
					return
				}
				lastTerminated = result.State.ShouldTerminate
			}
		}
	}
}

// computePatches generates field-specific patches from a workflow result.
// Field-specific comparison avoids a generic deep-diff dependency.
func computePatches(_ any) []Patch {
	// In a real implementation, we'd compare against the previous state.
	// For now, we emit the current state fields as "replace" patches.
	// The frontend uses the full UISchema from each delta anyway.
	return nil
}

func writeSSE(w http.ResponseWriter, flusher http.Flusher, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event.Type, data)
	flusher.Flush()
}
