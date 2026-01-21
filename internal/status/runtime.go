package status

import "context"

// RuntimeDetector defines the interface for agent-specific status detection
type RuntimeDetector interface {
	// CanHandle returns true if this detector supports the given agent type
	CanHandle(agentType string) bool

	// Detect returns status. If it returns StateUnknown, the system may fall back to heuristics.
	Detect(ctx context.Context, paneID string, output string) (AgentState, error)
}
