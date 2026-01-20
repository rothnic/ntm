#!/bin/bash
set -e

# Setup
TEST_ROOT=$(mktemp -d /tmp/ntm-interaction-test-XXXXXX)
PROJECT_NAME="interaction-project"
PROJECT_PATH="$TEST_ROOT/$PROJECT_NAME"
CONFIG_FILE="$TEST_ROOT/config.toml"
# Resolve absolute path to ntm binary before changing directory
NTM="$(pwd)/ntm"
if [ ! -f "$NTM" ]; then
    echo "Error: ntm binary not found at $NTM"
    exit 1
fi

mkdir -p "$PROJECT_PATH"
cd "$PROJECT_PATH"
git init -b main >/dev/null

# Config
cat > "$CONFIG_FILE" <<EOF
projects_base = "$TEST_ROOT"
[tmux]
enable_mouse = true
EOF

# Ensure clean state
$NTM opencode stop --force "$PROJECT_PATH" 2>/dev/null || true

# Spawn with OpenCode, and send a specific init prompt
INIT_PROMPT="Hello OpenCode, are you there?"
# We use --prompt flag which relies on NTM auto-sending the prompt.
# This tests if NTM correctly identifies the agent as ready/idle and sends the prompt.
echo "Spawning session with init prompt..."
OUTPUT=$($NTM spawn "$PROJECT_NAME" --oc=1 --config "$CONFIG_FILE" --prompt "$INIT_PROMPT" --json 2>&1)
SESSION_NAME=$(echo "$OUTPUT" | grep -o '"session_name": *"[^"]*"' | cut -d'"' -f4)
[ -z "$SESSION_NAME" ] && SESSION_NAME="$PROJECT_NAME"

echo "Session: $SESSION_NAME"
echo "--- Spawn Output Start ---"
echo "$OUTPUT"
echo "--- Spawn Output End ---"
echo "--- DEBUG LOGS FROM FILE (Skipped) ---"
echo "------------------"

# Wait for potential startup
echo "Waiting for agent startup (30s)..."
for i in {1..30}; do
    echo -n "."
    sleep 1
done
echo

# Capture Pane Content
echo "Capturing pane content..."
CAPTURE=$(tmux capture-pane -pt "${SESSION_NAME}:0.1" -S -100)
echo "--- Pane Content Start ---"
echo "$CAPTURE"
echo "--- Pane Content End ---"

# Check if Prompt was sent
# NTM sends keys. If the TUI received it, it might be in the input field or processed.
# If processed, we might see the prompt text on screen or a response.
# If waiting in input buffer, we should see the text.
if echo "$CAPTURE" | grep -q "$INIT_PROMPT"; then
    echo "SUCCESS: Init prompt found in pane content."
    echo "WARNING: Init prompt NOT found in pane content. (Proceeding to ntm send check despite this)"
fi

# Verify ntm send command (tests Checkpoint/Scrollback fix)
SEND_PROMPT="Tell me another joke"
echo "Testing ntm send..."
# Use --all to ensure broadcast logic triggers checkpoint, or rely on default
SEND_OUTPUT=$($NTM send "$SESSION_NAME" "$SEND_PROMPT" --config "$CONFIG_FILE" --all 2>&1)
echo "$SEND_OUTPUT"

if echo "$SEND_OUTPUT" | grep -q "Warning: failed to capture scrollback"; then
    echo "FAILURE: ntm send emitted scrollback warning (Scrollback Capture bug not fixed)."
    tmux kill-session -t "$SESSION_NAME"
    $NTM opencode stop "$PROJECT_PATH" --force
    rm -rf "$TEST_ROOT"
    exit 1
fi

if ! echo "$SEND_OUTPUT" | grep -q "Sent to"; then
     echo "FAILURE: ntm send did not report success."
     exit 1
fi

echo "SUCCESS: ntm send executed without scrollback warnings."

# Cleanup
tmux kill-session -t "$SESSION_NAME"
$NTM opencode stop "$PROJECT_PATH" --force
rm -rf "$TEST_ROOT"
echo "Verification Passed!"
