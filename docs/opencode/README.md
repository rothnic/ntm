# OpenCode Server Integration

## Overview

NTM provides first-class support for managing per-project OpenCode servers. Each project gets its own isolated OpenCode server running on a stable, hash-based port in the range 28000-29000.

## Quick Start

```bash
# Start OpenCode server for current directory
ntm opencode start

# Show server status
ntm opencode status

# Get attach URL
ntm opencode url

# Attach using opencode CLI
opencode attach $(ntm opencode url)

# Stop server when done
ntm opencode stop
```

## Architecture

### State Management

All server state is stored in `~/.opencode/servers/<project-id>/`:
- `server.pid` - Process ID of the running server
- `port` - Port number the server is listening on
- `project_path` - Absolute path to the project directory
- `server.log` - Server output logs

### Port Assignment

Each project is assigned a stable port based on its absolute path:
- Hash the absolute project path using MD5
- Map to port range 28000-29000
- Same project always gets the same port

### Connection Tracking

Servers track active connections using `ss` command:
- Count established TCP connections on server port
- Used by `reap` command to identify idle servers
- Safe stop refuses to kill servers with active connections

## Commands

### `ntm opencode start [project-path]`

Start an OpenCode server for the given project.

```bash
# Start for current directory
ntm opencode start

# Start for specific project
ntm opencode start /path/to/my/project

# JSON output
ntm opencode start --json
```

**Behavior:**
- If server already running, returns existing server info
- Creates state directory for project
- Spawns detached `opencode serve` process
- Waits for server to become healthy before returning
- Stores PID, port, and project path

### `ntm opencode stop [project-path]`

Stop the OpenCode server for the given project.

```bash
# Stop server for current directory (safe mode)
ntm opencode stop

# Force stop even with active connections
ntm opencode stop --force

# Stop specific project
ntm opencode stop /path/to/my/project --force
```

**Behavior:**
- By default, refuses to stop if connections > 0
- With `--force`, kills server regardless of connections
- Sends SIGTERM first, waits 5 seconds
- Sends SIGKILL if still running
- Cleans up PID file

### `ntm opencode status [project-path]`

Show detailed status of the OpenCode server.

```bash
# Status for current directory
ntm opencode status

# Status with JSON output
ntm opencode status --json
```

**Output includes:**
- Project path and ID
- Running status
- Process ID
- Port number
- Attach URL
- Active connection count
- Log file location

### `ntm opencode list`

List all managed OpenCode servers.

```bash
# List all servers
ntm opencode list

# JSON output
ntm opencode list --json
```

Shows:
- Project paths
- Running status
- Port numbers
- Connection counts
- Process IDs

### `ntm opencode logs [project-path]`

Tail server logs.

```bash
# Show last 20 lines
ntm opencode logs

# Show last 50 lines
ntm opencode logs -n 50

# Follow logs in real-time
ntm opencode logs -f

# Follow logs for specific project
ntm opencode logs /path/to/my/project -f
```

Uses standard `tail` command under the hood.

### `ntm opencode url [project-path]`

Get the HTTP URL for attaching to the server.

```bash
# Get URL for current directory
ntm opencode url

# Use with opencode attach
opencode attach $(ntm opencode url)
```

Returns: `http://127.0.0.1:<port>`

### `ntm opencode reap`

Stop all idle OpenCode servers (servers with no active connections).

```bash
# Reap idle servers
ntm opencode reap

# See what was stopped
ntm opencode reap --json
```

**Behavior:**
- Iterates all managed servers
- Checks connection count for each
- Stops servers with 0 connections
- Cleans up stale state directories
- Returns count of stopped servers

**Use case:** Run periodically (e.g., cron, systemd timer) to free resources.

## Integration with ntm spawn

`ntm spawn` has full integration with OpenCode servers.

```bash
# Spawn session with OpenCode agents
ntm spawn myproject --oc=2
```

**Capabilities:**
1.  **Automatic Server Management:** Starts the `opencode` server for the project if not running.
2.  **Deterministic Sessions:** Uses the OpenCode SDK to create a persistent session with a deterministic name (`NTM Session [<ntm-session-name>]`).
3.  **Automatic Attachment:** Creates tmux panes and attaches the OpenCode agent to the specific session ID.
4.  **Metadata Binding:** Binds the Session ID to the tmux pane via user option `@opencode_session_id`, enabling robust status monitoring.

## Status Detection (Unified Detector)

NTM V2 uses a generic `RuntimeDetector` interface for status detection, with a specialized `OpenCodeDetector` implementation.

**Mechanism:**
1.  **Pane Discovery:** Detector reads `@opencode_session_id` from the tmux pane.
2.  **SDK Query:** Connects to the project's OpenCode server via Go SDK.
3.  **Message Analysis:** Fetches recent session messages to determine state:
    -   Last message from `user` -> **Working**
    -   Last message from `assistant` -> **Idle**
4.  **High Fidelity:** This eliminates screen-scraping heuristics for OpenCode agents, providing 100% accuracy on agent state.

## Troubleshooting

### Server won't start

**Problem:** `ntm opencode start` fails

**Solutions:**
1. Check if `opencode` CLI is installed: `which opencode`
2. Check if port is already in use: `ss -ltn | grep <port>`
3. Check server logs: `ntm opencode logs`
4. Try different project to test port collision

### Server not responding

**Problem:** Server process running but not responding

**Solutions:**
1. Check server logs: `ntm opencode logs`
2. Test port connectivity: `nc -zv 127.0.0.1 <port>`
3. Force restart: `ntm opencode stop --force && ntm opencode start`

### Can't stop server

**Problem:** `ntm opencode stop` says server has active connections

**Solutions:**
1. Check connections: `ntm opencode status`
2. Find connected clients: `ss -tn | grep :<port>`
3. Close client connections first
4. Force stop if needed: `ntm opencode stop --force`

### Port collision

**Problem:** Two projects hash to same port

**Solutions:**
1. Check which projects are conflicting: `ntm opencode list`
2. Stop one server: `ntm opencode stop /path/to/project1`
3. *Future:* Automatic port collision detection and remapping

### Stale PID files

**Problem:** Server shows as running but process doesn't exist

**Solutions:**
1. Run reap to clean up: `ntm opencode reap`
2. Manually remove: `rm ~/.opencode/servers/<project-id>/server.pid`
3. Verify with status: `ntm opencode status`

## Periodic Cleanup

To automatically clean up idle servers, set up a systemd timer:

### systemd User Timer

Create `~/.config/systemd/user/opencode-reaper.service`:

```ini
[Unit]
Description=OpenCode Server Reaper
Documentation=https://github.com/rothnic/ntm

[Service]
Type=oneshot
ExecStart=/usr/local/bin/ntm opencode reap
```

Create `~/.config/systemd/user/opencode-reaper.timer`:

```ini
[Unit]
Description=OpenCode Server Reaper Timer
Documentation=https://github.com/rothnic/ntm

[Timer]
OnBootSec=15min
OnUnitActiveSec=15min

[Install]
WantedBy=timers.target
```

Enable and start:

```bash
systemctl --user daemon-reload
systemctl --user enable opencode-reaper.timer
systemctl --user start opencode-reaper.timer

# Check status
systemctl --user status opencode-reaper.timer
```

### Cron Alternative

Add to crontab (`crontab -e`):

```cron
# Reap idle OpenCode servers every 15 minutes
*/15 * * * * /usr/local/bin/ntm opencode reap
```

## Implementation Details

### Process Spawning

Servers are spawned with:
```bash
opencode serve --host 127.0.0.1 --port <port> <project-path>
```

Process is detached using `Setsid: true` in `SysProcAttr` (Unix).

### Health Checking

Server health is verified using native Go TCP connection (no external commands):

```go
conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), HealthCheckTimeout)
```

### Connection Counting

Active connections are counted using:
```bash
ss -Htan sport = :<port> state established
```

Each line of output represents one established connection.

### Process Checking

Process liveness is checked using `os.FindProcess()` and sending signal 0:
```go
process, _ := os.FindProcess(pid)
err := process.Signal(syscall.Signal(0))
// err == nil means process exists
```

## Testing

### E2E Integration Test

**File:** `tests/e2e/opencode_spawn_test.go`

The integration test verifies the complete OpenCode spawn workflow, from server management through prompt delivery. This test validates the same workflow a human would execute manually.

#### What It Tests

| Step | Description | Verification Method |
|------|-------------|---------------------|
| 1. Build | Compiles NTM binary | Build succeeds without error |
| 2. Environment | Creates isolated test project directory | Directory exists |
| 3. Spawn | Runs `ntm spawn <session> --oc=2 --prompt="Hello E2E Test"` | Command exits successfully |
| 4. Tmux Session | Verifies session was created with correct pane layout | At least 3 panes (2 agents + 1 user) |
| 5. Prompt Delivery | Each OpenCode agent receives and processes the prompt | Pane content contains prompt text OR agent shows activity |
| 6. Cleanup | Stops OpenCode server and kills tmux session | Resources freed |

#### Verification Logic

The test uses **activity-based detection** rather than exact text matching, which is robust against:
- TUI rendering differences
- Timing variations in agent startup
- Model response variations

```go
// Primary: Direct text match
if strings.Contains(content, "Hello E2E Test") {
    // Prompt text visible in pane
}

// Secondary: Activity detection (agent is processing)
if strings.Contains(content, "Thinking") || 
   strings.Contains(content, "Generating") {
    // Agent is actively working on the prompt
}
```

#### Terminal Geometry

The test resizes the tmux window to 200x60 characters before verification:

```go
resizeCmd := exec.Command("tmux", "resize-window", "-t", sessionName, 
    "-x", "200", "-y", "60")
```

**Why:** Default 80x24 terminal causes text wrapping in the TUI, making prompt text detection unreliable. Larger geometry ensures content is displayed cleanly.

#### Running the Test

```bash
# From repository root
go test -tags=integration -v ./tests/e2e/...

# Run specific test
go test -tags=integration -v -run TestOpencodeSpawnIntegration ./tests/e2e/

# With timeout (default 10m, but test should complete in ~60s)
go test -tags=integration -v -timeout 2m ./tests/e2e/...
```

#### Prerequisites

- `tmux` installed and in PATH
- `opencode` CLI installed and in PATH
- No conflicting tmux sessions with `test-opencode-*` names

#### Expected Output

```
=== RUN   TestOpencodeSpawnIntegration
    opencode_spawn_test.go:26: Building NTM binary...
    opencode_spawn_test.go:65: Spawning session test-opencode-1705791234 with 2 OpenCode agents...
    opencode_spawn_test.go:90: Verifying tmux session...
    opencode_spawn_test.go:110: Polling OpenCode panes for prompt reception or activity (strict check for 2 agents)...
    opencode_spawn_test.go:137: ✓ Pane 1 received prompt (text match)
    opencode_spawn_test.go:140: ✓ Pane 2 received prompt (agent active)
    opencode_spawn_test.go:162: SUCCESS: All 2 OpenCode agents responded to the prompt.
--- PASS: TestOpencodeSpawnIntegration (45.23s)
```

### CLI Smoke Test

**File:** `tests/opencode_cli_test.sh`

A lightweight shell script for basic CLI command verification:

```bash
./tests/opencode_cli_test.sh
```

Tests:
- `ntm opencode --help` exits cleanly
- `ntm opencode list` works with no servers
- `ntm opencode status` handles missing server gracefully
- `ntm opencode reap` works with nothing to reap

### Manual Verification Checklist

For features not covered by automated tests:

1. **Server Lifecycle**
   ```bash
   ntm opencode start
   ntm opencode status   # Verify running, has PID
   ntm opencode stop
   ntm opencode status   # Verify stopped
   ```

2. **Spawn with Prompt**
   ```bash
   ntm spawn test-manual --oc=1 --prompt="Summarize this project"
   tmux attach -t test-manual
   # Verify: OpenCode agent received and is processing prompt
   ```

3. **Connection Tracking**
   ```bash
   ntm opencode start
   opencode attach $(ntm opencode url) &  # Start client
   ntm opencode status  # Verify connections > 0
   ntm opencode stop    # Should refuse (has connections)
   ntm opencode stop --force  # Should succeed
   ```

4. **Reap Idle Servers**
   ```bash
   ntm opencode start /tmp/proj1
   ntm opencode start /tmp/proj2
   ntm opencode list    # 2 servers
   ntm opencode reap    # Should stop both (0 connections)
   ntm opencode list    # 0 servers
   ```

## Design Decisions

### Why Central State Directory?

We store state in `~/.opencode/servers/` instead of per-project `.opencode/`:

- **No pollution**: Project directories stay clean
- **Easy discovery**: `ntm opencode list` works globally
- **Survives moves**: State persists even if project moves (useful for debugging)
- **Separation**: Clear boundary between NTM state and OpenCode config

### Why Hash-Based Ports?

Using MD5 hash of project path for port assignment:

- **Stable**: Same project always gets same port
- **Collision-resistant**: Reduces port conflicts in multi-project scenarios
- **Predictable**: Easy to debug (port is deterministic)

> ⚠️ Port collisions are possible but rare. The 1000-port range handles typical workloads.

### Why Connection Counting?

Tracking active connections via `ss`/`lsof`:

- **Safe stop**: Prevents killing servers with active clients
- **Reap support**: Enables automatic cleanup of idle servers
- **Visibility**: `ntm opencode status` shows connection count

> ⚠️ Requires `ss` (Linux) or `lsof` (macOS). Returns 0 on unsupported platforms.

### Why Separate Reap Command?

Instead of automatic background cleanup:

- **User control**: Cleanup happens when you want it
- **Easy integration**: Works with cron, systemd timers, etc.
- **Predictable**: No surprise terminations
- **Testable**: Can verify behavior without side effects

## Requirements

- `opencode` CLI must be installed and in PATH
- `ss` or `lsof` for connection counting (optional, gracefully degrades)
- `tail` for log viewing

## See Also

- [Technical Guide](./technical_guide.md) - Critical findings and patterns
- [Testing Requirements](./testing.md) - Functional requirements and test coverage
- [Gap Analysis](./gap_analysis.md) - Remaining integration work
- [Original agentic flywheel PR](https://github.com/rothnic/agentic_coding_flywheel_setup/pull/7)

