# OpenCode Integration Work Items

Tracked work items for completing OpenCode integration. See [../agent_feature_matrix.md](../agent_feature_matrix.md) for current feature status.

## Status Legend
- [ ] Not started
- [/] In progress
- [x] Completed

---

## P0 - Critical

### Compaction Integration
**File:** `internal/context/compact.go`
- [ ] Add OpenCode case to `GetAgentCapabilities()` returning:
  - `SupportsBuiltinCompact: true`
  - `BuiltinCompactCommand: "/compact"`
  - `SupportsHistoryClear: true` (via session reset)

**Note:** OpenCode supports `/compact` natively + auto-compact at 95%.

### Agent-Based Model Configuration (Future)

OpenCode supports custom agents via `.opencode/agents/` with per-agent models. Consider:
- [ ] Define role-based agent profiles in `opencode.json`
- [ ] Map NTM personas to OpenCode agents with appropriate models
- [ ] Example: `architect` → `proxypal/claude-opus-4-20250514`, `quick` → `google/gemini-2.5-flash`

**Valid model format:** `provider/model` (run `opencode models` to list available)

Example models (prefer antigravity-routed):
- `google/antigravity-claude-sonnet-4-5` - Default coding
- `google/antigravity-claude-opus-4-5-thinking` - Deep analysis
- `google/antigravity-gemini-3-pro` - Fast tasks

---

## P1 - High Priority

### Session Management
**Files:** `internal/session/capture.go`, `internal/session/restore.go`
- [ ] Add OpenCode case to session capture (use SDK `Session.Messages()`)
- [ ] Add OpenCode case to session restore (use SDK `Session.Prompt()`)

### Context Rotation
**File:** `internal/rotation/provider.go`
- [ ] Add OpenCode rotation provider using `Session.New()` SDK call

### Persona Assignment
**File:** `internal/persona/persona.go`
- [ ] Add `"opencode"` case to agent type normalization (lines 61, 86)

### Handoff Integration
**Files:** `internal/handoff/generator.go`, `internal/handoff/constants.go`
- [ ] Add OpenCode-specific output patterns to `accomplishmentPatterns`
- [ ] Add OpenCode-specific task patterns
- [ ] Test `ntm resume` with OpenCode sessions
- [ ] Verify handoff generation from OpenCode transcript/output

**Note:** `GenerateFromTranscript` reads Claude's `session.jsonl`. OpenCode may need SDK-based message extraction via `Session.Messages()` instead.

---

## P2 - Medium Priority

### Quota/Token Tracking
**Files:** `internal/quota/`
- [ ] Create `internal/quota/opencode.go` (delegate to underlying provider)
- [ ] Add OpenCode case to `internal/quota/fetcher.go`

### TUI Icons
**File:** `internal/tui/icons/icons.go`
- [ ] Add OpenCode icon mappings:
  - NerdFont: `󰘦` (code icon)
  - Unicode: `◇`
  - ASCII: `[OC]`

### Command Palette
**File:** `internal/palette/model.go`
- [ ] Add `TargetOpenCode` constant
- [ ] Add OpenCode cases at lines 364, 738, 884

---

## Workflow Testing

### Test OpenCode with WORKFLOW_EXAMPLES.md patterns

| Workflow | Status | Notes |
|----------|--------|-------|
| Design-Implement-Test | [ ] | Replace agent with `opencode` |
| Implement-Review-Revise | [ ] | Test multi-agent with OpenCode |
| Error Handling with Retry | [ ] | Test retry behavior |
| For-Each Loop | [ ] | Test loop iteration |
| While Loop | [ ] | Test conditional loops |
| Parallel Execution | [ ] | Test parallel OpenCode agents |

### Agent Mail Integration

Agent Mail enables multi-agent coordination via messaging. Test OpenCode with:

| Feature | Status | Notes |
|---------|--------|-------|
| Agent registration | [ ] | OpenCode agent registers with Agent Mail |
| Send messages | [ ] | OpenCode sends mail to other agents |
| Receive messages | [ ] | OpenCode receives and processes mail |
| File reservations | [ ] | OpenCode reserves files before editing |
| Thread summarization | [ ] | Test thread participant lists |
| Mixed fleet mail | [ ] | Claude ↔ OpenCode messaging |

**Key Agent Mail Test Cases:**
- [ ] `ntm mail send` from OpenCode pane
- [ ] `ntm mail inbox` check from OpenCode
- [ ] File reservation before multi-agent edit
- [ ] Cross-agent task handoff via mail

### Robot Mode Testing

| Feature | Status | Notes |
|---------|--------|-------|
| `--robot-context` | [ ] | Get OpenCode context |
| `--robot-send` | [ ] | Send prompt to OpenCode |
| `--robot-ack` | [ ] | Acknowledge OpenCode response |
| `--robot-status` | [ ] | Check OpenCode status |

### CASS (Context-Aware Session Summary) Integration

CASS injects context summaries into agent sessions. Test OpenCode with:

| Feature | Status | Notes |
|---------|--------|-------|
| Context injection | [ ] | CASS injects summary into OpenCode |
| Summary extraction | [ ] | Extract summary from OpenCode session |
| Session handoff | [ ] | CASS-assisted handoff to/from OpenCode |

**Key CASS Test Cases:**
- [ ] `ntm cass inject` to OpenCode pane
- [ ] `ntm cass summary` from OpenCode session
- [ ] CASS-guided rotation with OpenCode

**Key Tests:**
- [ ] `ntm pipeline run` with OpenCode as primary agent
- [ ] Mixed fleet: Claude + OpenCode in same workflow
- [ ] Checkpoint save/restore with OpenCode sessions

### Events Integration

Verify OpenCode emits and handles all NTM events (from SKILL.md):

**Notification Events:**
| Event | Status | Notes |
|-------|--------|-------|
| `agent.error` | [ ] | OpenCode error detection |
| `agent.crashed` | [ ] | OpenCode crash detection |
| `agent.rate_limit` | [ ] | Rate limit from underlying provider |
| `rotation.needed` | [ ] | Context exhaustion trigger |
| `session.created` | [ ] | Emitted on spawn |
| `session.killed` | [ ] | Emitted on kill |
| `health.degraded` | [ ] | Health check failure |

**Command Hook Events:**
| Event | Status | Notes |
|-------|--------|-------|
| `pre-spawn` / `post-spawn` | [ ] | OpenCode spawn hooks |
| `pre-send` / `post-send` | [ ] | SDK prompt delivery hooks |
| `pre-add` / `post-add` | [ ] | Adding agents to session |
| `pre-shutdown` / `post-shutdown` | [ ] | Server shutdown hooks |

---

## References

- [Agent Feature Matrix](../agent_feature_matrix.md) - Feature comparison
- [Adding Agent Runtime Guide](../adding_agent_runtime.md) - Integration checklist
- [OpenCode Agent Differences](agent_differences.md) - Architecture notes
- [OpenCode SDK](https://github.com/sst/opencode-sdk-go) - Go SDK
- [OpenCode Docs](https://opencode.ai/docs) - Official documentation
