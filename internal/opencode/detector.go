package opencode

import (
	"context"

	"github.com/Dicklesworthstone/ntm/internal/tmux"
	opencode_sdk "github.com/sst/opencode-sdk-go"
)

// AgentState represents the detected state of an agent
type AgentState int

const (
	StateUnknown AgentState = iota
	StateIdle
	StateWorking
)

// Detector implements status detection for OpenCode agents.
// It queries the OpenCode SDK to determine agent state from message history,
// providing higher fidelity than screen scraping.
type Detector struct {
	manager *Manager
}

// NewDetector creates a new OpenCode detector.
func NewDetector(mgr *Manager) *Detector {
	return &Detector{
		manager: mgr,
	}
}

// CanHandle returns true if this detector can handle the given agent type.
func (d *Detector) CanHandle(agentType string) bool {
	return agentType == "oc" || agentType == "opencode"
}

// Detect determines the state of an OpenCode agent by querying the SDK.
//
// Detection flow:
//  1. Read @opencode_session_id from tmux pane
//  2. Get project path from pane
//  3. Connect to OpenCode server via SDK
//  4. Query session message history
//  5. Last message role determines state:
//     - "user" → StateWorking (agent processing)
//     - "assistant" → StateIdle (waiting for input)
func (d *Detector) Detect(ctx context.Context, paneID string, output string) (AgentState, error) {
	// Need session ID from pane
	sessionID, err := tmux.DefaultClient.GetUserOption(paneID, "@opencode_session_id")
	if err != nil || sessionID == "" {
		return StateUnknown, nil
	}

	// Need project path to get client
	panePath, err := tmux.GetPanePath(paneID)
	if err != nil {
		return StateUnknown, nil
	}

	client, err := d.manager.Client(panePath)
	if err != nil {
		return StateUnknown, nil
	}

	// Fetch latest message
	messages, err := client.Session.Messages(ctx, sessionID, opencode_sdk.SessionMessagesParams{})
	if err != nil {
		return StateUnknown, nil
	}

	if messages == nil || len(*messages) == 0 {
		return StateIdle, nil
	}

	// Last message determines state
	msgs := *messages
	lastMsg := msgs[len(msgs)-1]

	if lastMsg.Info.Role == "user" {
		return StateWorking, nil
	}

	return StateIdle, nil
}
