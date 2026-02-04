package status

import (
	"context"

	"github.com/Dicklesworthstone/ntm/internal/opencode"
)

// openCodeRuntimeAdapter adapts opencode.Detector to status.RuntimeDetector.
// This adapter bridges the opencode package's self-contained types to the
// status package's interface, enabling clean package separation.
type openCodeRuntimeAdapter struct {
	detector *opencode.Detector
}

// NewOpenCodeRuntimeAdapter creates a new adapter for the OpenCode detector.
func NewOpenCodeRuntimeAdapter(mgr *opencode.Manager) RuntimeDetector {
	return &openCodeRuntimeAdapter{
		detector: opencode.NewDetector(mgr),
	}
}

func (a *openCodeRuntimeAdapter) CanHandle(agentType string) bool {
	return a.detector.CanHandle(agentType)
}

func (a *openCodeRuntimeAdapter) Detect(ctx context.Context, paneID string, output string) (AgentState, error) {
	state, err := a.detector.Detect(ctx, paneID, output)
	// Map opencode.AgentState to status.AgentState
	switch state {
	case opencode.StateIdle:
		return StateIdle, err
	case opencode.StateWorking:
		return StateWorking, err
	default:
		return StateUnknown, err
	}
}
