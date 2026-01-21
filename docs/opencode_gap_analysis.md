# OpenCode Integration Gap Analysis

This document tracks all locations where OpenCode (opencode) integration is missing or needs updates to match the level of support provided for Claude Code, Codex, and Gemini.

## High Priority (Functional)

### Agent Profiles
**File:** `internal/agents/profiles.go`
- [ ] Add `AgentTypeOpenCode` constant.
- [ ] Add `AgentProfile` default configuration in `loadDefaults`.
    - Suggested Specializations: `SpecComplex`, `SpecRefactorLarge` (similar to Claude?).
    - Suggested Model: `opencode-7b` (or whatever the default is).
- [ ] Update `NormalizeAgentType` and `ParseAgentType` to handle "oc", "opencode".

### TUI Icons
**File:** `internal/tui/icons/icons.go`
- [ ] Add `OpenCode` string field to `IconSet` struct.
- [ ] Add NerdFont/Unicode/ASCII mappings for OpenCode.
    - Suggestion: `󰘦` (Code icon) or `󰅪` (Open Source icon).
- [ ] Update `AgentIcon` method to return the OpenCode icon for "oc"/"opencode".

## Medium Priority (UI/UX & Tests)

### VS Code Extension
**File:** `vscode/src/extension.ts`
- [ ] Update `quickPick` items to include "OpenCode (--oc)".

### Documentation
**File:** `AGENTS.md`
- [ ] Add OpenCode to the "Agent Types" table.
- [ ] Document specific `--oc` flags in CLI reference.
- [ ] Update examples to include OpenCode usage.

**File:** `SKILL.md`
- [ ] Update description to mention OpenCode support alongside Claude/Codex/Gemini.
- [ ] Add examples of `ntm spawn ... --oc=N`.

### E2E Testing
**File:** `scripts/e2e/test-status.sh`
- [ ] Update test assertions to check for OpenCode panes if relevant (currently tests user + 2 claude).
- [ ] Consider adding a specific "mixed fleet" test case including OpenCode.

## Low Priority (Future/Cleanup)

### Future Plans
**File:** `PLAN_TO_ADD_WEB_UI_AND_REST_AND_WEBSOCKET_API_LAYERS_TO_NTM__GPT_PRO.md`
- [ ] Update "Agent Types" list in API design to include `opencode`.
- [ ] Update pagination filtering examples (`?agent_type=opencode`).

## Completed Integrations (Reference)

- **Spawn Command:** `internal/cli/spawn.go` (Integrated)
- **Status Detection:** `internal/status/opencode_detector.go` (Implemented) & `internal/status/unified.go` (Wired)
- **Server Management:** `internal/opencode/manager.go` (Implemented)
- **CLI Templates:** `internal/config/templates.go` (Updated)
