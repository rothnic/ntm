# OpenCode Integration Status - Jan 2026

## Overview
We have transitioned the `ntm` OpenCode integration from a "fire-and-forget" CLI spawn to a robust, managed lifecycle using the official OpenCode Go SDK.

## Key Components

### 1. Server Management (`internal/opencode/manager.go`)
*   **`Manager` Struct**: Handles server lifecycle (Start/Stop/Status).
*   **`Client(projectPath)`**: Returns a configured `opencode.Client` for SDK interaction.
*   **`CreateSession(projectPath, title)`**: Deterministically creates a named persistent session.

### 2. Deterministic Spawn (`internal/cli/spawn.go`)
*   **Workflow**:
    1.  User runs `ntm spawn`.
    2.  `ntm` ensures server is running.
    3.  `ntm` creates a session via SDK: `NTM Session [<name>]`.
    4.  `ntm` launches the OpenCode agent using `opencode attach --session <ID>`.
    5.  The Session ID is stamped onto the tmux pane via `@opencode_session_id`.

### 3. Benefits
*   **Eliminates Ambiguity**: No more guessing which session belongs to which agent.
*   **Observability Ready**: `UnifiedDetector` and other components can now read the `@opencode_session_id` from the pane and query the SDK for precise status (Idle/Thinking), completely replacing screen scraping.
*   **Persistence**: Sessions are named and trackable.

## Next Steps
1.  **Update `UnifiedDetector`**: Refactor `internal/status/unified.go` to use the SDK for status checks instead of regex.
2.  **Plugin Integration**: Resume plugin work (Universal Visibility) once the core communication channel is stable.
