package executor

import (
	"fmt"
	"time"

	"github.com/finops-claw-gang/finops-go/internal/domain"
	"github.com/finops-claw-gang/finops-go/internal/policy"
)

// Executor performs deterministic action execution. It takes pre/post
// snapshots and calls the policy safety gate before any action.
type Executor struct {
	tags TagFetcher
}

// NewExecutor creates an Executor backed by the given TagFetcher.
func NewExecutor(tags TagFetcher) *Executor {
	return &Executor{tags: tags}
}

// Snapshot captures the pre- or post-action state for the given action.
// If the action has a target resource, the snapshot includes its tags.
func (e *Executor) Snapshot(action domain.RecommendedAction) (map[string]any, error) {
	if action.TargetResource != "" {
		tags, err := e.tags.ResourceTags(action.TargetResource)
		if err != nil {
			return nil, fmt.Errorf("executor: snapshot tags for %s: %w", action.TargetResource, err)
		}
		return map[string]any{"tags": tags}, nil
	}
	return map[string]any{}, nil
}

// ExecuteActions runs each approved action sequentially, taking pre/post
// snapshots and enforcing the policy safety gate up front.
// Execution stops on the first failure.
func (e *Executor) ExecuteActions(
	approval domain.ApprovalStatus,
	actions []domain.RecommendedAction,
	resourceTagsByARN map[string]map[string]string,
) ([]domain.ExecutionResult, error) {
	if err := policy.EnforceExecutorSafety(approval, actions, resourceTagsByARN); err != nil {
		return nil, err
	}

	results := make([]domain.ExecutionResult, 0, len(actions))
	for _, a := range actions {
		pre, err := e.Snapshot(a)
		if err != nil {
			return nil, fmt.Errorf("executor: pre-snapshot: %w", err)
		}

		// TODO: production impl should deep-copy pre into post and capture real state
		results = append(results, domain.ExecutionResult{
			ActionID:           a.ActionID,
			ExecutedAt:         time.Now().UTC().Format(time.RFC3339),
			Success:            true,
			Details:            fmt.Sprintf("stub executed %s on %s", a.ActionType, a.TargetResource),
			RollbackAvailable:  true,
			PreActionSnapshot:  pre,
			PostActionSnapshot: pre,
		})
	}
	return results, nil
}
