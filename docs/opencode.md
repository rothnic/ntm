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

Server health is verified using `nc -z 127.0.0.1 <port>` to test TCP connectivity.

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

## Requirements

- `opencode` CLI must be installed and in PATH
- `nc` (netcat) for health checks
- `ss` for connection counting
- `tail` for log viewing

## See Also

- [Original agentic flywheel PR](https://github.com/rothnic/agentic_coding_flywheel_setup/pull/7)
- [OpenCode documentation](https://github.com/example/opencode)
- [Verification Log (Manual Tests)](../manual_tests/opencode_verification.md)
