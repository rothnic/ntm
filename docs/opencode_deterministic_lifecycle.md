# OpenCode Deterministic Lifecycle

We have implemented a deterministic lifecycle for OpenCode sessions in `ntm` to avoid the ambiguity of session discovery.

## Workflow

1.  **Spawn Request**: User requests `ntm spawn ... --oc=1`.
2.  **Server Check**: `ntm` ensures an OpenCode server is running for the project.
3.  **Session Creation (SDK)**:
    *   `ntm` uses the OpenCode Go SDK (`client.Session.New`) to create a new session.
    *   The session is titled `NTM Session [<ntm-session-name>]`.
    *   The SDK returns a unique `SessionID`.
4.  **Agent Launch (CLI)**:
    *   `ntm` constructs the pane command: `opencode attach --session <SessionID> <ServerURL>`.
    *   This forces the TUI to attach to the *specific* session we just created.
5.  **Persistence (Tmux)**:
    *   `ntm` sets a custom user option on the tmux pane: `@opencode_session_id`.
    *   This allows other `ntm` processes (like the status detector or dashboard) to recover the Session ID simply by inspecting the pane.

## Benefits

*   **No Guesswork**: We don't need to list sessions and guess which one is "new".
*   **Race Condition Free**: We know the ID *before* the agent even starts.
*   **Observability**: We can query the exact status of the specific session using the SDK.
*   **Multi-Agent Safety**: If we spawn multiple OpenCode agents, each gets its own distinct ID and pane binding.
