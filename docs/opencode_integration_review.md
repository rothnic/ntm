
# OpenCode Integration: Initial Working Version Review

## Summary of Changes
This release establishes a stable foundation for OpenCode agent integration within `ntm`. The primary focus was resolving reliability issues with agent spawning, status monitoring, and command interaction (`ntm send`).

### Key Implementation Details

1.  **Robust Spawning Interaction:**
    *   **What Worked:** Implementing a polling mechanism in `spawn.go` that waits for the OpenCode TUI to report an "idle" state before injecting the initial prompt.
    *   **Why:** The OpenCode TUI takes several seconds to initialize. The previous "blind sleep" approach was flaky; polling ensures the target is actually listening, preventing prompt loss.
    *   **Status:** ✅ Successfully verified with `verify_opencode_interaction.sh`.

2.  **Dashboard Status Fidelity:**
    *   **What Worked:** Updating `internal/status/patterns.go` with OpenCode-specific TUI patterns (`Ask anything...`, `ctrl+t variants`).
    *   **Why:** `ntm` uses screen scraping to determine agent state. Without these patterns, OpenCode agents appeared as "unknown" or "working" indefinitely.
    *   **Status:** ✅ Dashboard now correctly reports OpenCode agents as "IDLE".

3.  **Ambiguous Pane Targeting Fix:**
    *   **What Failed (Initially):** `ntm send ... --all` triggered auto-checkpoints that failed to capture scrollback (`can't find window: 1`).
    *   **Why:** The checkpoint system used relative pane indexes (e.g., `1`), which tmux interprets as *window* indexes in some contexts (`session:1`).
    *   **Solution:** Switched to using unique tmux Pane IDs (e.g., `%5`), which are absolute references.
    *   **Status:** ✅ Verified fix; warnings eliminated.

## Future Considerations

### 1. OpenCode Go SDK Integration
*   **Recommendation:** Migrate from screen scraping (regex on `capture-pane`) to direct API queries using the OpenCode Go SDK.
*   **Benefit:** Screen scraping is fragile to UI changes (e.g., if the TUI prompt text changes). The SDK would provide deterministic status (Idle vs Generating) and allow for richer interactions (e.g., structured input).

### 2. Custom Plugin Architecture
*   **Idea:** Inject a standard `.opencode/plugin.js` that reports status to a known file or socket.
*   **Benefit:** Would allow `ntm` to discover and monitor *any* OpenCode instance in a project, even those not spawned by `ntm` (e.g., manually started by the user).

### 3. Test Hardening
*   **Observation:** The `verify_opencode_interaction.sh` test occasionally warns about missing prompts despite correct delivery.
*   **Action:** Further harden the TUI verification, possibly by looking for state changes (e.g., the "Thinking" indicator) rather than just input echo.
