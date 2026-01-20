# OpenCode SDK & Plugin Investigation

## SDK Investigation
We have verified that `github.com/sst/opencode-sdk-go` (v0.19.2) is the correct SDK for interacting with the OpenCode server.

### Capabilities
- **Session Management**: `client.Session.List`, `client.Session.Create` (implied), `client.Session.Prompt`.
- **TUI Integration**: `client.Tui.*` methods allow controlling the specific TUI view.
- **Projects**: `client.Project.List`.

### Integration Strategy for NTM
Instead of screen scraping `tmux capture-pane`, `ntm` should:
1.  Initialize an SDK client pointing to the project's OpenCode server port (which `ntm` manages).
2.  Use `client.Session.Get(id)` to poll for status (Idle/Thinking).
3.  Use `client.Session.Prompt` (or TUI methods) to inject prompts reliably without race conditions.

### Verification
A test script `tests/explore_opencode_sdk.go` was created and successfully connected to a running OpenCode server, listing 300+ sessions.

## Plugin Investigation
The OpenCode plugin system operates by loading a JS module from `.opencode/plugin.js` (or similar) in the project root.

### Plugin Structure
A standard plugin exports an `init` function that receives a `context`.
This context likely contains event emitters or hooks for:
- Session lifecycle
- Messages
- Tool execution

### Proposed Plugin
A fixture `tests/fixtures/opencode_plugin.js` has been created to log all available context keys and hook into observable events. This will serve as the "Universal Visibility" tool for `ntm`.
