# OpenCode Integration Verification Log

## Date: 2026-01-20

## Summary
Successfully integrated OpenCode agent support into NTM, enabling management of local OpenCode servers and spawning of persistent, interactive agent sessions via the TUI.

## Verified Features

### 1. Server Management
- [x] `ntm opencode start [path]`: Starts server on stable hash-based port (28000-29000).
- [x] `ntm opencode status`: Reports PID, port, connection count, and health.
- [x] `ntm opencode stop --force`: Cleanly shuts down server (verified with active connections).
- [x] `ntm opencode list`: Lists all managed servers correctly.

### 2. Agent Spawning (`ntm spawn`)
- [x] **Automatic Server Provisioning**: `ntm spawn --oc=2` auto-starts the server for the project if needed.
- [x] **Unique Sessions**: Each agent gets a unique, persistent session ID (e.g., `NTM Session [session-name] (Agent 1)`).
- [x] **SDK Prompt Injection**:
  - Prompts are injected via `opencode.Manager.SendPrompt` in a serialized (mutex-locked) manner.
  - Verified 10s timeout prevents race conditions during high load.
  - Fallback to key injection works if SDK is slow/unavailable.
- [x] **TUI Interaction**:
  - `--json` mode removed from default test flow to verify real user experience.
  - Agents strictly verified to receive prompts and enter "Thinking"/"Generating" states.
  - Tmux window resizing implemented to prevent TUI text wrapping issues.

### 3. Resilience & Ops
- [x] **Port Stability**: Same project always maps to the same port.
- [x] **Concurrency**: Mutex locking on prompt injection prevents SQLite contention on the server.
- [x] **E2E Testing**: `tests/e2e/opencode_spawn_test.go` passes consistently with multiple agents.

## Known Limitations / Future Work
- **Status Detection**: Currently relies on SDK implementation details; future versions should expose a dedicated status API.
- **Cleanup**: `ntm opencode reap` logic for idle servers is implemented but not yet automated via systemd/cron in the default install.

## Test Artifacts
- Source: `tests/e2e/opencode_spawn_test.go`
- Coverage: Spawning, Server Start/Stop, SDK Prompting, TUI State Verification.

## Manual Verification Notes

These scenarios were manually verified in the terminal:

1.  **Multiple Agent Spawn**:
    ```bash
    bin/ntm spawn testproject3 --oc=2
    ```
    - **Result**: Successfully spawned 2 agents.
    - **Observation**: Both agents received the "Hello" prompt. The TUI showed "Provisioning OpenCode environment..." followed by successful attachment.
    - **Fix Verified**: Initially saw "unknown agent type oc" with stale binary. Rebuilding `bin/ntm` resolved this, confirming accurate agent type detection.

2.  **SDK vs Key Injection Fallback**:
    - **Scenario**: Simulated high load to trigger timeout.
    - **Result**: System gracefully fell back to key injection as intended when the 10s SDK timeout was reached, ensuring the prompt was delivered.

3.  **Directory Auto-Creation**:
    - **Scenario**: Spawning a session for a new project (`testproject3`).
    - **Result**: NTM correctly identified the missing directory, prompted for creation (in interactive mode), or used `NTM_PROJECTS_BASE` (in test mode) to resolve the path without error.
