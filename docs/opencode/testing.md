# OpenCode Testing Requirements

This document defines the requirements for evaluating OpenCode functionality and the testing approach.

## Functional Requirements

### FR1: Server Lifecycle Management

| ID | Requirement | Verification |
|----|-------------|--------------|
| FR1.1 | Start server for any project directory | `ntm opencode start` succeeds |
| FR1.2 | Server runs on stable hash-based port (28000-29000) | Same project → same port |
| FR1.3 | Idempotent start (no-op if already running) | Second `start` returns existing info |
| FR1.4 | Stop server with graceful shutdown | SIGTERM → wait → SIGKILL if needed |
| FR1.5 | Safe stop refuses if connections > 0 | Error without `--force` |
| FR1.6 | Force stop kills regardless of connections | `--force` flag works |
| FR1.7 | List all managed servers | `ntm opencode list` shows all |
| FR1.8 | Reap stops idle servers (0 connections) | `ntm opencode reap` cleans up |

### FR2: Session Management

| ID | Requirement | Verification |
|----|-------------|--------------|
| FR2.1 | Create named session via SDK | Session title = "NTM Session [name]" |
| FR2.2 | Session ID persisted to tmux pane | `@opencode_session_id` user option set |
| FR2.3 | Agent attaches to specific session | `--session <ID>` flag used |
| FR2.4 | Multiple agents get unique sessions | Each pane has different session ID |

### FR3: Prompt Delivery

| ID | Requirement | Verification |
|----|-------------|--------------|
| FR3.1 | Prompt sent via SDK (not keypresses) | SDK `SendPrompt` API used |
| FR3.2 | Serialized delivery (no race conditions) | Mutex-gated SDK calls |
| FR3.3 | All agents receive prompt | E2E test verifies each agent |
| FR3.4 | Prompt visible in agent TUI | Pane content includes prompt text |

### FR4: Status Detection

| ID | Requirement | Verification |
|----|-------------|--------------|
| FR4.1 | Detect working state | Agent shows activity patterns |
| FR4.2 | Detect idle state | Agent waiting for input |
| FR4.3 | Query via SDK for high fidelity | Message history determines state |
| FR4.4 | Fallback to heuristics if SDK fails | Graceful degradation |

---

## Test Coverage Matrix

### Unit Tests

| Component | File | Coverage |
|-----------|------|----------|
| Project ID generation | `manager_test.go` | ✅ Stable hashing |
| Port allocation | `manager_test.go` | ✅ Range validation |
| State directory | `manager_test.go` | ✅ Path construction |
| Status (not running) | `manager_test.go` | ✅ Clean state handling |
| List servers | `manager_test.go` | ✅ Enumeration |

### Integration Tests

| Test | File | What It Verifies |
|------|------|------------------|
| Full spawn workflow | `opencode_spawn_test.go` | FR1, FR2, FR3 end-to-end |
| Prompt + scrollback | `verify_opencode_interaction.sh` | `ntm send` command, scrollback capture |
| CLI smoke test | `opencode_cli_test.sh` | Basic command execution |

### Manual Verification

For features requiring human judgment or external dependencies:

| Feature | Verification Steps |
|---------|-------------------|
| TUI rendering | Attach to session, verify prompt visible |
| Model response | Submit prompt, verify coherent response |
| Connection tracking | `ss` or `lsof` shows connections |

---

## E2E Test Specification

### Test: `TestOpencodeSpawnIntegration`

**Purpose:** Verify complete spawn workflow with OpenCode agents.

**Prerequisites:**
- `tmux` installed
- `opencode` CLI installed
- No conflicting sessions

**Steps:**

1. **Setup**
   - Build NTM binary
   - Create temp project directory
   - Generate unique session name

2. **Execute**
   ```bash
   ntm spawn <session> --oc=2 --prompt="Hello E2E Test"
   ```

3. **Verify Tmux Session**
   - Session exists
   - At least 3 panes (2 agents + 1 user)

4. **Resize Terminal** (Critical)
   ```bash
   tmux resize-window -t <session> -x 200 -y 60
   ```

5. **Verify Prompt Delivery**
   - For each agent pane:
     - Capture pane content
     - Check for "Hello E2E Test" OR
     - Check for activity ("Thinking", "Generating")

6. **Cleanup**
   - Stop OpenCode server (`--force`)
   - Kill tmux session

**Success Criteria:**
- All N agents show prompt text OR activity state
- No timeout (30 second limit)

**Failure Handling:**
- Log failed pane contents for debugging
- Report which agents failed

---

## Running Tests

### Unit Tests
```bash
# OpenCode manager tests
go test -v ./internal/opencode/...

# Status detector tests
go test -v ./internal/status/...
```

### Integration Tests
```bash
# E2E tests (requires tmux + opencode)
go test -tags=integration -v ./tests/e2e/...

# Specific test
go test -tags=integration -v -run TestOpencodeSpawnIntegration ./tests/e2e/

# With timeout
go test -tags=integration -v -timeout 2m ./tests/e2e/...
```

### CLI Smoke Tests
```bash
./tests/opencode_cli_test.sh
```

---

## Test Environment Requirements

| Dependency | Version | Purpose |
|------------|---------|---------|
| Go | 1.21+ | Build and run tests |
| tmux | 3.0+ | Session management |
| opencode | latest | Agent runtime |
| ss/lsof | any | Connection counting |

### CI Considerations

- Tests require real tmux (no mock)
- OpenCode server must be startable
- Sufficient terminal for 200x60 resize
- 2+ minute timeout for slow CI

---

## Regression Test Triggers

Run full E2E suite when modifying:

- `internal/opencode/manager.go` - Server or session logic
- `internal/cli/spawn.go` - Agent spawning
- `internal/status/opencode_detector.go` - Status detection
- `tests/e2e/opencode_spawn_test.go` - Test itself

---

## See Also

- [Technical Guide](./technical_guide.md) - Critical findings and patterns
- [README](./README.md) - Quick start and CLI usage
- [Gap Analysis](./gap_analysis.md) - Remaining work
