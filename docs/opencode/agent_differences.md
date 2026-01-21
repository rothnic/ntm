# OpenCode Agent: Key Differences from Other Agents

This document explains the fundamental differences between OpenCode and other agents (Claude, Gemini, Codex) that drive the implementation approach.

## Architectural Differences

| Aspect | Traditional Agents (Claude, Gemini) | OpenCode |
|--------|-------------------------------------|----------|
| **Process Model** | Direct CLI invocation | Client/Server architecture |
| **State Location** | In-memory (terminal session) | Server-side (persists across connections) |
| **Session Identity** | Implicit (terminal = session) | Explicit (named sessions via SDK) |
| **Prompt Delivery** | Keypresses via tmux | SDK API (programmatic) |
| **Status Detection** | Screen scraping (regex) | SDK query (message history) |
| **Lifecycle Owner** | tmux (process dies with pane) | NTM (server persists beyond pane) |

## Why This Matters

### 1. Server Must Exist Before Agent Starts

**Traditional:**
```bash
tmux send-keys "claude" Enter   # Process starts immediately
```

**OpenCode:**
```bash
ntm opencode start              # Ensure server is running
opencode attach --session $ID   # Then attach client
```

**Impact:** NTM must manage server lifecycle separately from agent lifecycle.

### 2. Sessions Are Decoupled from Terminals

**Traditional:** Killing the tmux pane = agent is gone

**OpenCode:** Sessions persist on server. Must explicitly clean up.

**Impact:** Session ID must be stored (`@opencode_session_id` user option) for recovery and status detection.

### 3. Prompt Delivery Must Use SDK

**Traditional:**
```go
tmux.SendKeys(paneID, prompt)  // Works because agent reads stdin
```

**OpenCode:**
```go
client.Session.Prompt(ctx, sessionID, prompt)  // Must use SDK
```

**Impact:** Race conditions if multiple prompts sent simultaneously (fixed with mutex serialization).

### 4. Status Detection Uses Messages, Not Screen Content

**Traditional:**
```go
// Look for prompt pattern in terminal output
if regexp.MatchString(`claude>`, output) {
    return StateIdle
}
```

**OpenCode:**
```go
// Query SDK for last message
messages := client.Message.List(ctx, sessionID)
lastMsg := messages[len(messages)-1]
if lastMsg.Role == "assistant" {
    return StateIdle
}
```

**Impact:** More reliable but requires SDK connection. Falls back to activity detection if SDK unavailable.

---

## Implementation Consequences

### Server Lifecycle Management

Because OpenCode uses client/server:
- `Manager.Start()` must be idempotent
- `Manager.Stop()` must check for active connections
- Health checks needed before returning success

### Deterministic Session Binding

Because sessions decouple from terminals:
- Create session via SDK BEFORE launching agent
- Bind session ID to pane immediately
- Use `--session <ID>` flag to force specific session

### Serialized SDK Operations

Because of race conditions in prompt delivery:
- All SDK calls (session creation, prompts) use a mutex
- Prevents interleaved operations for multiple agents
- Solved the "wrong agent receives prompt" bug

### Activity-Based Detection

Because TUI rendering is unpredictable:
- Don't rely on exact prompt text matching
- Look for activity patterns: "Thinking", "Generating"
- Accept either prompt text OR activity as success

---

## Code Organization Implications

### Why Separate Package (`internal/opencode/`)

Unlike traditional agents where spawn.go handles everything, OpenCode needs:
- `manager.go` - Server lifecycle (Start, Stop, Status, etc.)
- `detector.go` - Status detection (implements RuntimeDetector)
- Future: `session.go` - Session management helpers

### Why Status Detector is Different

Traditional agents: Regex patterns in `internal/status/patterns.go`

OpenCode: SDK-based detection requires:
- Reading `@opencode_session_id` from pane
- Connecting to server via SDK
- Querying message history

This is why OpenCode has a dedicated `RuntimeDetector` implementation.

---

## Testing Implications

### E2E Test Differences

| Aspect | Traditional Agent Test | OpenCode Agent Test |
|--------|------------------------|---------------------|
| Verification | Check terminal output | Check pane content OR activity |
| Cleanup | Kill tmux session | Kill tmux + stop server |
| Setup | None | Ensure server running |
| Timeout | Short (agent starts fast) | Longer (server startup) |

### Unit Test Needs

Because of the complexity:
- `ProvisionSessions()` - Mock server, verify session creation
- `SendPrompt()` - Mock SDK, verify prompt format
- `isServerHealthy()` - Already pure function, testable

---

## Summary: The Three Key Insights

1. **Server-first architecture** → Manage lifecycle before agents
2. **Explicit sessions** → Store and retrieve session IDs
3. **SDK-based interaction** → Serialize operations, use message API

These differences drove every major design decision in the OpenCode integration.
