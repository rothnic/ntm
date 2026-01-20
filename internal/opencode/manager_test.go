package opencode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestProjectID(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Test that same path generates same ID
	path := "/tmp/test-project"
	id1 := mgr.ProjectID(path)
	id2 := mgr.ProjectID(path)

	if id1 != id2 {
		t.Errorf("ProjectID() not stable: %s != %s", id1, id2)
	}

	// Test that different paths generate different IDs
	path2 := "/tmp/test-project-2"
	id3 := mgr.ProjectID(path2)

	if id1 == id3 {
		t.Errorf("ProjectID() generated same ID for different paths")
	}

	// Test ID length
	if len(id1) != 12 {
		t.Errorf("ProjectID() wrong length: got %d, want 12", len(id1))
	}
}

func TestPortForProject(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Test that same path generates same port
	path := "/tmp/test-project"
	port1 := mgr.PortForProject(path)
	port2 := mgr.PortForProject(path)

	if port1 != port2 {
		t.Errorf("PortForProject() not stable: %d != %d", port1, port2)
	}

	// Test port range
	if port1 < 28000 || port1 >= 29000 {
		t.Errorf("PortForProject() out of range: %d", port1)
	}
}

func TestProjectStateDir(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	projectID := "test123"
	stateDir := mgr.ProjectStateDir(projectID)

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".opencode", "servers", projectID)

	if stateDir != expected {
		t.Errorf("ProjectStateDir() = %s, want %s", stateDir, expected)
	}
}

func TestStatusNotRunning(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	// Create a temporary project directory
	tmpDir := t.TempDir()

	info, err := mgr.Status(tmpDir)
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}

	if info.Running {
		t.Errorf("Status() reported Running=true for non-existent server")
	}

	if info.ProjectPath != tmpDir {
		t.Errorf("Status() ProjectPath = %s, want %s", info.ProjectPath, tmpDir)
	}
}

func TestList(t *testing.T) {
	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager() failed: %v", err)
	}

	servers, err := mgr.List()
	if err != nil {
		t.Fatalf("List() failed: %v", err)
	}

	// Should return an empty slice, not nil
	if servers == nil {
		t.Errorf("List() returned nil, expected empty slice")
	}
}
