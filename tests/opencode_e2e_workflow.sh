#!/bin/bash
# E2E Integrated Workflow Test for OpenCode in NTM
#
# Scenarios covered:
# 1. Project setup and OpenCode server lifecycle (Start)
# 2. Tmux session spawning (headless via --json)
# 3. Manual injection of OpenCode clients into panes (simulating spawn integration)
# 4. Sending prompts to agents
# 5. Instructions for user attachment

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[TEST]${NC} $1"
}

info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1"
    exit 1
}

# Dependencies check
command -v go >/dev/null || error "go not found"
command -v tmux >/dev/null || error "tmux not found"
command -v opencode >/dev/null || error "opencode CLI not found"

# 1. Build NTM
log "Building ntm..."
go build -o /tmp/ntm ./cmd/ntm || error "Build failed"
NTM="/tmp/ntm"

# 2. Setup Temporary Project
PROJECT_DIR=$(mktemp -d /tmp/ntm-opencode-demo-XXXXXX)
log "Created project at $PROJECT_DIR"
cd "$PROJECT_DIR"

# Initialize git to satisfy project detection if needed
git init -b main >/dev/null

# 3. Start OpenCode Server
log "Starting OpenCode server..."
# Ensure clean slate
$NTM opencode stop --force . 2>/dev/null || true

START_OUTPUT=$($NTM opencode start .)
SERVER_URL=$($NTM opencode url .)
log "Server running at: $SERVER_URL"

# 4. Create Headless Tmux Session
SESSION_NAME="opencode-demo-$(basename "$PROJECT_DIR")"
log "Creating tmux session: $SESSION_NAME"

# --json prevents auto-attach and auto-creates directory if missing
$NTM create "$SESSION_NAME" --panes=2 --json >/dev/null

if ! tmux has-session -t "$SESSION_NAME" 2>/dev/null; then
    error "Session failed to create"
fi

# 5. Inject OpenCode Clients
# We use tmux send-keys to simulate the user starting agents or ntm eventually doing it
MODEL="github-copilot/gpt-5-mini"

log "Injecting OpenCode clients into panes..."

# Pane 0: Agent 1
tmux send-keys -t "${SESSION_NAME}:0.0" "export OPENCODE_SERVER_URL=$SERVER_URL" C-m
tmux send-keys -t "${SESSION_NAME}:0.0" "clear" C-m
# Using --server explicitly if supported, or relying on env var if client supports it. 
# Only `ntm opencode start` guaranteed us a URL. 
# Rely on OPENCODE_SERVER_URL env var which we exported.
tmux send-keys -t "${SESSION_NAME}:0.0" "opencode --model $MODEL" C-m

# Pane 1: Agent 2
tmux send-keys -t "${SESSION_NAME}:0.1" "export OPENCODE_SERVER_URL=$SERVER_URL" C-m
tmux send-keys -t "${SESSION_NAME}:0.1" "clear" C-m
tmux send-keys -t "${SESSION_NAME}:0.1" "opencode --model $MODEL" C-m

# 6. Send Initial Prompts
log "Sending prompts to agents..."
sleep 2 # Wait for clients to initialize

tmux send-keys -t "${SESSION_NAME}:0.0" "Hello Agent 1, report status." C-m
tmux send-keys -t "${SESSION_NAME}:0.1" "Hello Agent 2, write a haiku about mulitplexing." C-m

echo
log "✅ E2E Setup Complete!"
info "Project Dir: $PROJECT_DIR"
info "Server URL:  $SERVER_URL"
echo
echo -e "${GREEN}To attach and interact with the agents, run:${NC}"
echo -e "  ntm attach $SESSION_NAME"
echo
echo -e "${BLUE}To cleanup manually later:${NC}"
echo -e "  ntm opencode stop $PROJECT_DIR"
echo -e "  tmux kill-session -t $SESSION_NAME"
echo
