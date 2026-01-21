# Adding a New Agent Runtime to NTM

This guide explains how to integrate a new AI agent runtime (like Claude, Gemini, OpenCode) into NTM.

## Overview

NTM supports multiple agent types through:
1. **Agent Profiles** (`internal/agents/profiles.go`) - Metadata and defaults
2. **Spawn Logic** (`internal/cli/spawn.go`) - How to launch agents
3. **Status Detection** (`internal/status/`) - How to monitor agent state

## Step-by-Step Integration

### Step 1: Define Agent Type

Add your agent to `internal/agents/profiles.go`:

```go
const (
    AgentTypeYourAgent AgentType = "youragent"
)

func init() {
    DefaultProfiles[AgentTypeYourAgent] = AgentProfile{
        Type:          AgentTypeYourAgent,
        DisplayName:   "Your Agent",
        DefaultModel:  "your-model-v1",
        ContextBudget: 200000,
        Specializations: []string{"general"},
    }
}
```

### Step 2: Add CLI Alias

In `internal/agents/profiles.go`, update normalization:

```go
func NormalizeAgentType(t string) AgentType {
    switch strings.ToLower(t) {
    case "ya", "youragent":
        return AgentTypeYourAgent
    // ...
    }
}
```

### Step 3: Add Spawn Flag

In `internal/cli/spawn.go`, add a flag for your agent count:

```go
cmd.Flags().IntVar(&yourAgentCount, "ya", 0, "Number of YourAgent agents")
```

### Step 4: Implement Launch Logic

Add launch function in spawn.go or create a dedicated file:

```go
func launchYourAgent(paneID, projectPath, prompt string) error {
    // Build command to start your agent
    cmd := fmt.Sprintf("youragent --project %s", projectPath)
    return tmux.SendKeys(paneID, cmd)
}
```

### Step 5: Add Status Detection (Optional)

For simple agents, the default heuristics may work. For complex agents:

#### Option A: Add Regex Patterns

In `internal/status/patterns.go`:

```go
var yourAgentPromptPatterns = []string{
    `youragent>`,
    `\[YA\]`,
}
```

#### Option B: Implement RuntimeDetector

For SDK-based agents, create a detector:

```go
// internal/youragent/detector.go
type Detector struct { ... }

func (d *Detector) CanHandle(agentType string) bool {
    return agentType == "youragent" || agentType == "ya"
}

func (d *Detector) Detect(ctx context.Context, paneID, output string) (AgentState, error) {
    // Query your agent's API/SDK for status
}
```

Register in `internal/status/unified.go`:

```go
runtimes: []RuntimeDetector{
    &openCodeRuntimeAdapter{...},
    &yourAgentAdapter{...},  // Add here
}
```

### Step 6: Add Tests

1. **Unit tests** for profile normalization
2. **E2E test** for spawn and prompt delivery

### Step 7: Document

Create `docs/youragent/` with:
- `README.md` - Quick start
- `technical_guide.md` - Implementation details (if complex)

---

## Agent Architecture Types

### Type A: Direct CLI (Simple)

Examples: Claude, Codex

```
tmux pane → agent process → stdin/stdout
```

- Spawn: `tmux send-keys "claude" Enter`
- Prompt: `tmux send-keys "your prompt" Enter`
- Status: Screen scraping (regex)

### Type B: Client/Server (Complex)

Examples: OpenCode

```
Server (persists) → SDK → Session
                          ↓
tmux pane ←──────── Client (attaches)
```

- Spawn: Start server → Create session → Attach client
- Prompt: SDK API call
- Status: SDK query

See [OpenCode Agent Differences](./opencode/agent_differences.md) for detailed comparison.

---

## Complete Feature Integration Checklist

When adding a new agent, consider integration with ALL of these NTM features:

### Core Integration (Required)

- [ ] **Agent Profiles** (`internal/agents/profiles.go`)
  - Agent type constant
  - Default profile configuration
  - Normalization aliases
- [ ] **Spawn Logic** (`internal/cli/spawn.go`)
  - CLI flag (`--youragent=N`)
  - Launch function
- [ ] **Status Detection** (`internal/status/`)
  - Pattern matching OR RuntimeDetector
  - Register in `unified.go`

### Context Management (Important)

- [ ] **Compaction** (`internal/context/compact.go`)
  - Add case to `GetAgentCapabilities()` (lines 89-118)
  - Document if agent supports `/compact`, `/clear`, or SDK-based compaction
- [ ] **Context Monitoring** (`internal/context/monitor.go`)
  - Verify context limits are configured
  - Test estimation accuracy
- [ ] **Token Tracking** (`internal/tokens/tokens.go`)
  - Add token model mappings if different from defaults

### Session Management

- [ ] **Checkpoints** (`internal/checkpoint/`)
  - Verify scrollback capture works
  - Test checkpoint restore
- [ ] **Recovery** (`internal/session/`, `internal/handoff/`)
  - Test session recovery scenarios
  - Verify handoff works correctly

### Orchestration

- [ ] **Supervisor** (`internal/supervisor/`)
  - Health check patterns
  - Restart capability
- [ ] **Alerts** (`internal/alerts/`)
  - Verify alerts fire for agent issues
  - Add agent-specific patterns if needed
- [ ] **Pipeline/Workflow** (`internal/pipeline/`, `internal/workflow/`)
  - Test with workflow recipes

### Robot Mode

- [ ] **Context Command** (`--robot-context`)
- [ ] **Send Command** (`--robot-send`)
- [ ] **Ack Command** (`--robot-ack`)
- [ ] **Status Command** (`--robot-status`)

### UI/UX

- [ ] **TUI Icons** (`internal/tui/icons/icons.go`)
  - NerdFont, Unicode, ASCII mappings
  - `AgentIcon` method

### Testing

- [ ] **Unit Tests**
  - Profile normalization
  - Compaction capabilities
- [ ] **E2E Tests**
  - Spawn and prompt delivery
  - Mixed fleet (multiple agent types)
- [ ] **Documentation**
  - Agent-specific README
  - Technical guide if complex

---

## Reference Implementations

| Agent | Architecture | Key Files | Complexity |
|-------|--------------|-----------|------------|
| Claude | Direct CLI | `spawn.go` (inline) | Low |
| Gemini | Direct CLI + Setup | `internal/gemini/` | Medium |
| OpenCode | Client/Server | `internal/opencode/` | High |

Study the OpenCode implementation for complex agents requiring:
- Server lifecycle management
- SDK-based session handling
- Custom status detection
- Session binding via tmux user options
