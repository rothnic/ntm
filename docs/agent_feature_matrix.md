# Agent Feature Integration Matrix

This document tracks feature implementation status across all supported agent types. It includes workarounds and alternative approaches for each feature.

## Quick Stats

| Metric | Count |
|--------|-------|
| Total agent-aware files | 234 |
| Files with all 4 agents | 14 |
| Files missing OpenCode | 220 |

---

## Legend

- ✅ Native support
- 🔧 Workaround available
- ❌ Not implemented in NTM
- N/A Not applicable to this agent

---

## P0 - Core Agent Infrastructure

### Agent Type Recognition
| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Profile constant | ✅ | ✅ | ✅ | ✅ | `internal/agents/profiles.go` |
| Tmux agent type | ✅ | ✅ | ✅ | ✅ | `internal/tmux/session.go` |

### Spawn/Launch
| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| CLI spawn | ✅ | ✅ | ✅ | ✅ | `internal/cli/spawn.go` |
| Add to session | ✅ | ✅ | ✅ | ⚠️ | `internal/cli/add.go` - needs verification |

### Status Detection
| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Pattern matching | ✅ | ✅ | ✅ | ✅ | `internal/status/patterns.go` |
| Runtime detector | N/A | N/A | N/A | ✅ | `internal/opencode/detector.go` - uses SDK |

---

## P0 - Context Management

### Compaction

**Purpose:** Reduce context window usage to avoid rotation/session restart.

| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Native command | ✅ `/compact` | ❌ None | ✅ `/clear` | ✅ `/compact` | [1] |
| Auto-compact | ❌ | ❌ | ❌ | ✅ 95% trigger | [1] |
| NTM integration | ✅ | ❌ | ✅ | ❌ | `internal/context/compact.go` |

**OpenCode Compaction Options:**
- `/compact` command via TUI, SDK (`Session.Command()`), or CLI
- **Auto-compact** triggers automatically at 95% context usage (`"autoCompact": true` in config)
- Experimental hook: `experimental.session.compacting` for custom logic [1]

### Context Rotation

**Purpose:** Rotate agent to fresh context when exhausted.

| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Rotation provider | ✅ | ✅ | ✅ | 🔧 | Session reset via SDK |
| NTM integration | ✅ | ✅ | ✅ | ❌ | `internal/rotation/provider.go` L16 |

**OpenCode Workaround:** Create new session via `Session.New()` SDK call, then `Session.Attach()` to tmux pane.

### Context Monitoring

**Purpose:** Track token usage to predict rotation needs.

| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Token estimation | 🔧 | 🔧 | 🔧 | 🔧 API | All use heuristics |
| NTM integration | ✅ | ✅ | ✅ | ⚠️ | `internal/context/monitor.go` |

**OpenCode Approach:** Query `Session.Messages()` to get message history, estimate tokens from content length. Can configure limits in `opencode.json` [2].

---

## P1 - Session Management

### Session Capture/Restore

**Purpose:** Save and restore session state for checkpoints.

| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Capture | ✅ | ✅ | ✅ | 🔧 SDK | `internal/session/capture.go` L79 |
| Restore | ✅ | ✅ | ✅ | 🔧 SDK | `internal/session/restore.go` L153 |

**OpenCode Workaround:** Use `Session.Messages()` to capture conversation history, `Session.Prompt()` to replay context on restore.

### Persona/Profile Assignment

**Purpose:** Assign specialized behavior profiles to agents.

| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Agent type check | ✅ | ✅ | ✅ | ❌ | `internal/persona/persona.go` L61, 86 |
| Custom profiles | 🔧 | 🔧 | 🔧 | ✅ Native | Via `.opencode/agents/` [2] |

**OpenCode Approach:** OpenCode has **native agent profiles** via `.opencode/agents/` directory:
- Custom system prompts in Markdown with YAML front matter
- Per-agent model, temperature, max_steps configuration
- Tool access control (Read, Write, Bash, Task)
- Primary vs subagent modes

See https://opencode.ai/docs/agents for full reference.

---

## P2 - Resource Management

### Quota/Rate Limiting

**Purpose:** Track API usage and enforce limits.

| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Provider file | ✅ `claude.go` | ✅ `codex.go` | ✅ `gemini.go` | ❌ Missing | Need `opencode.go` |
| Fetcher switch | ✅ | ✅ | ✅ | ❌ | `internal/quota/fetcher.go` L196, 210 |

**OpenCode Approach:** OpenCode uses underlying model providers (Anthropic, OpenAI, etc.). Quota tracking should query the actual provider used, configurable via `opencode.json` provider settings [3].

### Token Tracking

**Purpose:** Track actual token usage across sessions.

| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Token counter | ✅ | ✅ | ✅ | 🔧 | Model-dependent |

**OpenCode Approach:** Token counting depends on configured model. Default to Claude/OpenAI estimation based on provider in use.

---

## P2 - UI/UX

### TUI Icons

**Purpose:** Visual agent identification in dashboard.

| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| NerdFont icon | ✅ | ✅ | ✅ | ❌ | `internal/tui/icons/icons.go` |
| Unicode fallback | ✅ | ✅ | ✅ | ❌ | Need icon mapping |

**Suggested OpenCode Icons:**
- NerdFont: `󰘦` (code icon) or `󰅪` (open source)
- Unicode: `◇` (diamond) or `⬡` (hexagon)
- ASCII: `[OC]`

### Command Palette

**Purpose:** Quick command interface for agents.

| Feature | Claude | Codex | Gemini | OpenCode | Notes |
|---------|--------|-------|--------|----------|-------|
| Target type | ✅ | ✅ | ✅ | ❌ | `internal/palette/model.go` L364, 738, 884 |

**OpenCode Approach:** Add OpenCode as target type with SDK-based prompt delivery.

---

## Work Items

For OpenCode-specific integration tasks, see [opencode/work_items.md](opencode/work_items.md).

---

## Official Documentation & References

### OpenCode

| Resource | URL | Notes |
|----------|-----|-------|
| **Official Docs** | https://opencode.ai/docs | Configuration, providers, models |
| **SDK Docs** | https://opencode.ai/docs/sdk | TypeScript/JavaScript SDK reference |
| **Go SDK** | https://github.com/sst/opencode-sdk-go | Go API library (Go 1.22+) |
| **GitHub (Main)** | https://github.com/sst/opencode | Main repository |
| **OpenAPI Spec** | `http://localhost:4096/doc` | Auto-generated API docs |

### Key API Endpoints [4]

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/session` | POST | Create new session |
| `/session/:id/messages` | GET | List session messages |
| `/session/:id/message` | POST | Send message (async) |
| `/session/:id/attach` | GET | Attach to session (WebSocket) |

### Configuration Options

Schema: https://opencode.ai/config.json

```json
{
  "$schema": "https://opencode.ai/config.json",
  "model": "anthropic/claude-sonnet-4-20250514",
  "small_model": "anthropic/claude-haiku",
  "autoCompact": true
}
```

Key options:
- `"$schema"` - Enables validation and autocomplete in editors
- `"model"` - Format: `provider_id/model_id`
- `"small_model"` - Used for lightweight tasks (titles, etc.)
- `"autoCompact"` - Auto-summarize at 95% context (default: true)

See https://opencode.ai/docs/configuration for full reference.

---

## Agent-Specific Notes

### Claude
- Native `/compact` and `/clear` commands
- Direct terminal I/O
- Rate limits via Anthropic API

### Codex
- No native compaction (use summarization prompt)
- Direct terminal I/O
- Rate limits via OpenAI API

### Gemini
- Native `/clear` command (no `/compact`)
- Requires setup script (`internal/gemini/setup.go`)
- Rate limits via Google AI API

### OpenCode
- **Client/Server architecture** - Not direct CLI
- **SDK-based operations** - Use SDK, not terminal I/O
- **Auto-compact** - Built-in at 95% context usage
- **Session binding** - Session ID in tmux user option
- **Multi-provider** - Can use any supported LLM provider

See [opencode/agent_differences.md](opencode/agent_differences.md) for detailed comparison.

---

## Adding New Agents

See [adding_agent_runtime.md](adding_agent_runtime.md) for the complete integration checklist.

---

## References

1. OpenCode Compaction - https://opencode.ai/docs/configuration
2. OpenCode Configuration - https://opencode.ai/docs/getting-started
3. OpenCode Providers - https://opencode.ai/docs/providers
4. OpenCode SDK Reference - https://opencode.ai/docs/sdk
