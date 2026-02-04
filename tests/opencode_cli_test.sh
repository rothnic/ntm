#!/bin/bash
# Integration test for OpenCode CLI commands

set -e

echo "=== Testing OpenCode CLI Commands ==="
echo

# Build ntm
echo "Building ntm..."
go build -o /tmp/ntm ./cmd/ntm
NTM="/tmp/ntm"

echo "✓ Build successful"
echo

# Test help commands
echo "Testing help commands..."
$NTM opencode --help >/dev/null
$NTM opencode start --help >/dev/null
$NTM opencode stop --help >/dev/null
$NTM opencode status --help >/dev/null
$NTM opencode list --help >/dev/null
$NTM opencode logs --help >/dev/null
$NTM opencode url --help >/dev/null
$NTM opencode reap --help >/dev/null
echo "✓ All help commands work"
echo

# Test list command (should return empty list or existing servers)
echo "Testing list command..."
$NTM opencode list >/dev/null
$NTM opencode list --json >/dev/null
echo "✓ List command works"
echo

# Test status for non-existent server
echo "Testing status for non-running server..."
tmpdir=$(mktemp -d)
$NTM opencode status "$tmpdir" >/dev/null
echo "✓ Status command works for non-running server"
echo

# Test reap command
echo "Testing reap command..."
$NTM opencode reap >/dev/null
$NTM opencode reap --json >/dev/null
echo "✓ Reap command works"
echo

# Clean up
rm -rf "$tmpdir"

echo "=== All CLI tests passed! ==="
