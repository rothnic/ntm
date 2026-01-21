# OpenCode Integration: Technical Guide

This document captures the critical technical findings and proven approaches for integrating OpenCode with NTM.

## Critical Findings

These lessons were learned through iteration and are essential for reliable operation.

### 1. Serialized SDK Prompt Injection (Mutex-Gated)

**Problem:** Multiple agents spawned simultaneously created race conditions in prompt delivery.

**Solution:** All SDK calls for session creation and prompt injection are serialized using a mutex.

```go
// internal/cli/spawn.go
var opencodeSDKMutex sync.Mutex

func sendPromptToAgent(mgr *opencode.Manager, sessionID, prompt string) error {
    opencodeSDKMutex.Lock()
    defer opencodeSDKMutex.Unlock()
    
    return mgr.SendPrompt(ctx, sessionID, prompt)
}
```

**Why it matters:** Without serialization, SDK calls can interleave and cause:
- Wrong session receiving prompts
- SDK timeout errors
- Partial prompt delivery

---

### 2. Unique Session Isolation

**Problem:** Session discovery was ambiguous - multiple agents could claim the same session.

**Solution:** Deterministic session naming with NTM session prefix.

```go
// Session title format
sessionTitle := fmt.Sprintf("NTM Session [%s]", ntmSessionName)

// Create session via SDK BEFORE launching agent
sessionID, err := mgr.CreateSession(projectPath, sessionTitle)

// Pass session ID to agent
cmd := fmt.Sprintf("opencode attach --session %s %s", sessionID, serverURL)
```

**Session ID Persistence:** The session ID is stored on the tmux pane:
```go
tmux.DefaultClient.SetUserOption(paneID, "@opencode_session_id", sessionID)
```

This allows:
- Status detector to query the correct session
- Recovery after crashes
- Multi-agent disambiguation

---

### 3. Activity-Based State Detection

**Problem:** Screen scraping for prompt patterns was unreliable due to TUI rendering.

**Solution:** Detect activity states rather than exact text.

| State | Detection Method |
|-------|-----------------|
| Working | Agent shows "Thinking", "Generating", or similar |
| Idle | Last SDK message was from `assistant` |
| Unknown | No session ID or SDK error |

```go
// Check for activity patterns in pane content
if strings.Contains(content, "Thinking") || 
   strings.Contains(content, "Generating") {
    return StateWorking
}
```

**Best for E2E tests:** Use activity detection rather than checking for exact prompt text.

---

### 4. Terminal Geometry Requirements

**Problem:** Default 80x24 terminal causes TUI text wrapping, breaking detection.

**Solution:** Resize tmux window before verification.

```go
// Resize to 200x60 for reliable TUI rendering
resizeCmd := exec.Command("tmux", "resize-window", "-t", sessionName, 
    "-x", "200", "-y", "60")
resizeCmd.Run()

// Allow tmux to re-layout
time.Sleep(2 * time.Second)
```

**Minimum recommended geometry:** 160x40 for 2-3 agents

---

### 5. SDK Timeout Tuning

**Problem:** Default SDK timeouts were too short for slow model responses.

**Solution:** Use 10-second timeout for SDK operations.

```go
const SDKPromptTimeout = 10 * time.Second

ctx, cancel := context.WithTimeout(context.Background(), SDKPromptTimeout)
defer cancel()
```

---

## Proven Architecture Patterns

### Server Lifecycle Pattern

```
User Request → Check Server Status → Start if needed → Return URL
                     ↓
              Already Running? → Return existing info
```

### Spawn Integration Pattern

```
ntm spawn --oc=N
    ↓
1. Ensure server running (Manager.Start)
2. Create N sessions via SDK (Manager.CreateSession)  [SERIALIZED]
3. Create N tmux panes
4. Bind session IDs to panes (@opencode_session_id)
5. Launch agents with: opencode attach --session <ID> <URL>
6. Send prompts via SDK (Manager.SendPrompt)  [SERIALIZED]
7. Verify via activity detection
```

### Status Detection Pattern

```
Pane ID → Read @opencode_session_id → SDK Client → Query Messages
                                          ↓
                              Last message from user? → Working
                              Last message from assistant? → Idle
                              Error? → fall through to heuristics
```

---

## Common Pitfalls

### ❌ Fire-and-Forget Session Discovery
Don't: Start `opencode` and then try to find "the new session"
Do: Create session via SDK first, then attach to specific ID

### ❌ Parallel SDK Calls
Don't: Make concurrent SDK calls for multiple agents
Do: Serialize all SDK operations with a mutex

### ❌ Screen Scraping for State
Don't: Look for "gemini>" or specific prompt patterns
Do: Use activity detection or SDK message queries

### ❌ Assuming Fixed Terminal Size
Don't: Run tests with default 80x24 terminal
Do: Resize to 200x60 before verification

### ❌ Short Timeouts
Don't: Use 1-2 second timeouts for SDK calls
Do: Use 10+ seconds to handle slow model responses

---

## See Also

- [README](./README.md) - Quick start and CLI usage
- [Testing](./testing.md) - Test requirements and verification
- [Gap Analysis](./gap_analysis.md) - Remaining integration work
