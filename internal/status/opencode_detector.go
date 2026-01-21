package status

import (
	"context"

	"github.com/Dicklesworthstone/ntm/internal/opencode"
	"github.com/Dicklesworthstone/ntm/internal/tmux"
	opencode_sdk "github.com/sst/opencode-sdk-go"
)

// OpenCodeDetector implements RuntimeDetector for OpenCode agents
type OpenCodeDetector struct {
	manager *opencode.Manager
}

// NewOpenCodeDetector creates a new OpenCode detector
func NewOpenCodeDetector(mgr *opencode.Manager) *OpenCodeDetector {
	return &OpenCodeDetector{
		manager: mgr,
	}
}

func (d *OpenCodeDetector) CanHandle(agentType string) bool {
	return agentType == "oc" || agentType == "opencode"
}

func (d *OpenCodeDetector) Detect(ctx context.Context, paneID string, output string) (AgentState, error) {
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
