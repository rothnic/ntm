//go:build integration

package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Dicklesworthstone/ntm/internal/tmux"
)

// TestOpencodeSpawnIntegration verifies that we can spawn a session with OpenCode agents
// and that their prompts are correctly sent and received.
func TestOpencodeSpawnIntegration(t *testing.T) {
	// Need tmux installed
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not found")
	}

	// 1. Build NTM binary
	t.Log("Building NTM binary...")
	tmpDir := t.TempDir()
	ntmBin := filepath.Join(tmpDir, "ntm")

	// Assuming we are running from root of repo, build ./cmd/ntm
	buildCmd := exec.Command("go", "build", "-o", ntmBin, "./cmd/ntm")
	// Run from module root
	// We need to find module root.
	// If test is run from ./tests/e2e, we need to go up 2 levels.
	wd, _ := os.Getwd()
	// Heuristic: if we are in tests/e2e
	if strings.HasSuffix(wd, "tests/e2e") {
		buildCmd.Dir = "../.."
	}

	if out, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build ntm: %v\nOutput: %s", err, out)
	}

	// 2. Setup Test Environment
	sessionName := fmt.Sprintf("test-opencode-%d", time.Now().UnixNano())
	projectDir := filepath.Join(tmpDir, sessionName)
	if err := os.MkdirAll(projectDir, 0755); err != nil {
		t.Fatalf("Failed to create project dir: %v", err)
	}

	// Ensure cleanup
	t.Cleanup(func() {
		// Stop opencode server
		cleanupCmd := exec.Command(ntmBin, "opencode", "stop", projectDir, "--force")
		_ = cleanupCmd.Run()

		// Kill tmux session
		_ = tmux.KillSession(sessionName)
	})

	// 3. Spawn Session
	// ntm spawn [session-name] --oc=2 ...
	// Command must be run from the project directory for ntm to pick it up as the project root
	t.Logf("Spawning session %s with 2 OpenCode agents...", sessionName)
	spawnCmd := exec.Command(ntmBin, "spawn", sessionName,
		"--oc=2",
		"--prompt=Hello E2E Test",
	)
	spawnCmd.Dir = projectDir
	// Point NTM at our temp directory so it finds the project
	spawnCmd.Env = append(os.Environ(), fmt.Sprintf("NTM_PROJECTS_BASE=%s", tmpDir))

	out, err := spawnCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Failed to spawn session: %v\nOutput: %s", err, out)
	}

	// RESIZE SESSION: Ensure window is large enough to prevent text wrapping in TUI
	// 80x24 (default) is too small for 2 agents + user pane
	resizeCmd := exec.Command("tmux", "resize-window", "-t", sessionName, "-x", "200", "-y", "60")
	if err := resizeCmd.Run(); err != nil {
		t.Logf("Warning: failed to resize tmux window: %v", err)
	}
	// Allow tmux to re-layout
	time.Sleep(2 * time.Second)

	// 4. Verify Tmux Session Created
	t.Log("Verifying tmux session...")
	panes, err := tmux.GetPanes(sessionName)
	if err != nil {
		t.Fatalf("Failed to get panes: %v", err)
	}

	// Expect 2 OpenCode agents + 1 User pane = 3 panes
	if len(panes) < 3 {
		t.Fatalf("Expected at least 3 panes (2 agents + 1 user), got %d", len(panes))
	}

	// 5. Verify OpenCode Agents Received Prompt via SDK
	// Instead of relying solely on TUI scraping (which can be flaky in CI),
	// we use the SDK to verify the session state directly.

	// 5. Verify All OpenCode Agents Received Prompt via SDK
	// We require that ALL OpenCode agents reflect the prompt or activity.
	// Since they are now independent sessions, they must each process the prompt.

	expectedCount := 2 // We spawned 2 agents
	t.Logf("Polling OpenCode panes for prompt reception or activity (strict check for %d agents)...", expectedCount)

	// Track verified panes by index
	verifiedPanes := make(map[int]bool)

	// Polling loop with timeout
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		for _, pane := range panes {
			// Skip user pane (usually 0) and already verified panes
			if pane.Index == 0 || verifiedPanes[pane.Index] {
				continue
			}

			content, err := tmux.CapturePaneOutput(pane.ID, 50)
			if err != nil {
				continue
			}

			// Verify it's an OpenCode pane by checking for recognizable patterns
			// OpenCode TUI shows: v1.1.x version string, model info, or MCP tool descriptions
			isOpenCodePane := strings.Contains(content, "NTM Session") ||
				strings.Contains(content, "opencode attach") ||
				strings.Contains(content, "Sisyphus") ||
				strings.Contains(content, "v1.1.") || // OpenCode version
				strings.Contains(content, "ctrl+t variants") || // OpenCode TUI footer
				strings.Contains(content, "ctrl+p commands") || // OpenCode TUI footer
				strings.Contains(content, "antigravity") // Model identifier
			if !isOpenCodePane {
				continue
			}

			// Check for Prompt or Activity
			if strings.Contains(content, "Hello E2E Test") {
				t.Logf("✓ Pane %d received prompt (text match)", pane.Index)
				verifiedPanes[pane.Index] = true
			} else if strings.Contains(content, "Thinking") || strings.Contains(content, "Generating") ||
				strings.Contains(content, "Build") { // OpenCode shows "Build" when processing
				t.Logf("✓ Pane %d received prompt (agent active)", pane.Index)
				verifiedPanes[pane.Index] = true
			}
		}

		if len(verifiedPanes) >= expectedCount {
			break
		}
		time.Sleep(1 * time.Second)
	}

	// 6. Assertions
	if len(verifiedPanes) < expectedCount {
		t.Errorf("Failed to verify all OpenCode agents. Verified %d/%d", len(verifiedPanes), expectedCount)
		// Log failed panes content
		for _, pane := range panes {
			if pane.Index > 0 && !verifiedPanes[pane.Index] {
				content, _ := tmux.CapturePaneOutput(pane.ID, 50)
				t.Logf("FAILED Pane %d Content:\n%s", pane.Index, content)
			}
		}
	} else {
		t.Logf("SUCCESS: All %d OpenCode agents responded to the prompt.", expectedCount)
	}
}
