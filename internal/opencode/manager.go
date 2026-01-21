package opencode

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

// Server lifecycle constants
const (
	// Port allocation
	PortRangeStart = 28000
	PortRangeSize  = 1000

	// Server startup
	ServerStartTimeout      = 5 * time.Second
	ServerStartPollInterval = 100 * time.Millisecond
	HealthCheckTimeout      = 500 * time.Millisecond

	// Graceful shutdown
	ShutdownGracePeriod   = 5 * time.Second
	ShutdownPollInterval  = 500 * time.Millisecond
)

// Manager handles lifecycle of per-project opencode servers
type Manager struct {
	stateDir string
}

// ServerInfo contains information about a running opencode server
type ServerInfo struct {
	ProjectID   string
	ProjectPath string
	PID         int
	Port        int
	LogFile     string
	Running     bool
	Connections int
}

// NewManager creates a new opencode server manager
func NewManager() (*Manager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home dir: %w", err)
	}

	stateDir := filepath.Join(home, ".opencode", "servers")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}

	return &Manager{
		stateDir: stateDir,
	}, nil
}

// ProjectID generates a stable identifier for a project path
func (m *Manager) ProjectID(projectPath string) string {
	// Get absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		absPath = projectPath
	}

	// Generate MD5 hash for stable ID
	hash := md5.Sum([]byte(absPath))
	return hex.EncodeToString(hash[:])[:12] // Use first 12 chars
}

// ProjectStateDir returns the state directory for a project
func (m *Manager) ProjectStateDir(projectID string) string {
	return filepath.Join(m.stateDir, projectID)
}

// PortForProject calculates a stable port number for a project
func (m *Manager) PortForProject(projectPath string) int {
	// Use MD5 hash to generate port in range PortRangeStart to PortRangeStart+PortRangeSize
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		absPath = projectPath
	}

	hash := md5.Sum([]byte(absPath))
	// Use first 4 bytes of hash to generate port offset
	offset := int(hash[0])<<8 | int(hash[1])
	port := PortRangeStart + (offset % PortRangeSize)

	return port
}

// Start starts an opencode server for the given project
func (m *Manager) Start(projectPath string) (*ServerInfo, error) {
	projectID := m.ProjectID(projectPath)
	stateDir := m.ProjectStateDir(projectID)

	// Create state directory
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("create state dir: %w", err)
	}

	// Check if already running
	if info, err := m.Status(projectPath); err == nil && info.Running {
		return info, nil // Already running
	}

	// Calculate port
	port := m.PortForProject(projectPath)

	// Save project path for reference
	projectPathFile := filepath.Join(stateDir, "project_path")
	if err := os.WriteFile(projectPathFile, []byte(projectPath), 0644); err != nil {
		return nil, fmt.Errorf("write project path: %w", err)
	}

	// Save port
	portFile := filepath.Join(stateDir, "port")
	if err := os.WriteFile(portFile, []byte(fmt.Sprintf("%d", port)), 0644); err != nil {
		return nil, fmt.Errorf("write port file: %w", err)
	}

	// Setup log file
	logFile := filepath.Join(stateDir, "server.log")
	logFd, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}
	defer logFd.Close()

	// Start opencode server
	cmd := exec.Command("opencode", "serve", "--hostname", "127.0.0.1", "--port", fmt.Sprintf("%d", port))
	cmd.Stdout = logFd
	cmd.Stderr = logFd
	cmd.Dir = projectPath

	// Detach from parent process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start opencode server: %w", err)
	}

	pid := cmd.Process.Pid

	// Save PID
	pidFile := filepath.Join(stateDir, "server.pid")
	if err := os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644); err != nil {
		// Try to kill the process we just started
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("write PID file: %w", err)
	}

	// Release the process so it runs independently
	_ = cmd.Process.Release()

	// Wait for server to start (using constants)
	serverStarted := false
	maxAttempts := int(ServerStartTimeout / ServerStartPollInterval)
	for i := 0; i < maxAttempts; i++ {
		if m.isServerHealthy(port) {
			serverStarted = true
			break
		}
		time.Sleep(ServerStartPollInterval)
	}

	// Verify server is responding
	if !serverStarted {
		return nil, fmt.Errorf("server started but not responding on port %d", port)
	}

	return &ServerInfo{
		ProjectID:   projectID,
		ProjectPath: projectPath,
		PID:         pid,
		Port:        port,
		LogFile:     logFile,
		Running:     true,
	}, nil
}

// Stop stops the opencode server for the given project
func (m *Manager) Stop(projectPath string, force bool) error {
	info, err := m.Status(projectPath)
	if err != nil {
		return err
	}

	if !info.Running {
		return fmt.Errorf("server not running for project: %s", projectPath)
	}

	// Check connections if not forcing
	if !force && info.Connections > 0 {
		return fmt.Errorf("server has %d active connections (use --force to kill anyway)", info.Connections)
	}

	// Find process
	process, err := os.FindProcess(info.PID)
	if err != nil {
		return fmt.Errorf("find process: %w", err)
	}

	// Try graceful shutdown first (SIGTERM)
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("send SIGTERM: %w", err)
	}

	// Wait for graceful shutdown (using constants)
	maxAttempts := int(ShutdownGracePeriod / ShutdownPollInterval)
	for i := 0; i < maxAttempts; i++ {
		time.Sleep(ShutdownPollInterval)
		if !m.isProcessRunning(info.PID) {
			break
		}
	}

	// Force kill if still running
	if m.isProcessRunning(info.PID) {
		if err := process.Signal(syscall.SIGKILL); err != nil {
			return fmt.Errorf("send SIGKILL: %w", err)
		}
		time.Sleep(ShutdownPollInterval)
	}

	// Clean up state files
	stateDir := m.ProjectStateDir(info.ProjectID)
	pidFile := filepath.Join(stateDir, "server.pid")
	_ = os.Remove(pidFile)

	return nil
}

// Status returns the status of the opencode server for the given project
func (m *Manager) Status(projectPath string) (*ServerInfo, error) {
	projectID := m.ProjectID(projectPath)
	stateDir := m.ProjectStateDir(projectID)

	info := &ServerInfo{
		ProjectID:   projectID,
		ProjectPath: projectPath,
		Running:     false,
	}

	// Read PID
	pidFile := filepath.Join(stateDir, "server.pid")
	pidData, err := os.ReadFile(pidFile)
	if err != nil {
		return info, nil // Not running
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(pidData)))
	if err != nil {
		return info, nil // Invalid PID
	}
	info.PID = pid

	// Read port
	portFile := filepath.Join(stateDir, "port")
	portData, err := os.ReadFile(portFile)
	if err != nil {
		return info, nil // No port info
	}

	port, err := strconv.Atoi(strings.TrimSpace(string(portData)))
	if err != nil {
		return info, nil // Invalid port
	}
	info.Port = port

	// Read log file path
	info.LogFile = filepath.Join(stateDir, "server.log")

	// Check if process is actually running
	if !m.isProcessRunning(pid) {
		// Stale PID file, clean it up
		_ = os.Remove(pidFile)
		return info, nil
	}

	// Verify server is responding on port
	if !m.isServerHealthy(port) {
		return info, nil // Process running but server not responding
	}

	info.Running = true
	info.Connections = m.countConnections(port)

	return info, nil
}

// List returns all managed opencode servers
func (m *Manager) List() ([]*ServerInfo, error) {
	entries, err := os.ReadDir(m.stateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ServerInfo{}, nil
		}
		return nil, fmt.Errorf("read state dir: %w", err)
	}

	var servers []*ServerInfo

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectPathFile := filepath.Join(m.stateDir, entry.Name(), "project_path")
		pathData, err := os.ReadFile(projectPathFile)
		if err != nil {
			continue // Skip if can't read project path
		}

		projectPath := strings.TrimSpace(string(pathData))
		info, err := m.Status(projectPath)
		if err != nil {
			continue
		}

		servers = append(servers, info)
	}

	// Return empty slice instead of nil if no servers
	if servers == nil {
		servers = []*ServerInfo{}
	}

	return servers, nil
}

// Reap stops all servers with no active connections
func (m *Manager) Reap() (int, error) {
	servers, err := m.List()
	if err != nil {
		return 0, fmt.Errorf("list servers: %w", err)
	}

	stopped := 0
	for _, server := range servers {
		// Skip servers that aren't running
		if !server.Running {
			// Clean up stale state
			stateDir := m.ProjectStateDir(server.ProjectID)
			_ = os.RemoveAll(stateDir)
			continue
		}

		// Stop servers with no connections
		if server.Connections == 0 {
			if err := m.Stop(server.ProjectPath, true); err == nil {
				stopped++
			}
		}
	}

	return stopped, nil
}

// URL returns the attach URL for a project's server
func (m *Manager) URL(projectPath string) (string, error) {
	info, err := m.Status(projectPath)
	if err != nil {
		return "", err
	}

	if !info.Running {
		return "", fmt.Errorf("server not running for project: %s", projectPath)
	}

	return fmt.Sprintf("http://127.0.0.1:%d", info.Port), nil
}

// Client returns a configured OpenCode SDK client for the given project
func (m *Manager) Client(projectPath string) (*opencode.Client, error) {
	url, err := m.URL(projectPath)
	if err != nil {
		return nil, err
	}
	return opencode.NewClient(
		option.WithBaseURL(url),
	), nil
}

// CreateSession creates a new persistent session for the project
func (m *Manager) CreateSession(projectPath string, title string) (string, error) {
	client, err := m.Client(projectPath)
	if err != nil {
		return "", err
	}

	// Create session
	// Using opencode.String helper to avoid importing internal/param
	session, err := client.Session.New(context.Background(), opencode.SessionNewParams{
		Title:     opencode.String(title),
		Directory: opencode.String(projectPath),
	})
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return session.ID, nil
}

// isProcessRunning checks if a process with the given PID is running
func (m *Manager) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// isServerHealthy checks if the server is responding on the given port.
// Uses native Go net.DialTimeout for cross-platform compatibility.
func (m *Manager) isServerHealthy(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), HealthCheckTimeout)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// countConnections counts active connections to the server port
func (m *Manager) countConnections(port int) int {
	// Try ss command first (Linux standard)
	if path, err := exec.LookPath("ss"); err == nil {
		cmd := exec.Command(path, "-Htan", fmt.Sprintf("sport = :%d", port), "state", "established")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(lines) == 1 && lines[0] == "" {
				return 0
			}
			return len(lines)
		}
	}

	// Fallback to lsof (macOS/Unix)
	if path, err := exec.LookPath("lsof"); err == nil {
		// -i :port -sTCP:ESTABLISHED -t (terse, just PIDs)
		cmd := exec.Command(path, "-i", fmt.Sprintf(":%d", port), "-sTCP:ESTABLISHED", "-t")
		output, err := cmd.Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(output)), "\n")
			if len(lines) == 1 && lines[0] == "" {
				return 0
			}
			return len(lines)
		}
	}

	return 0
}

// ProvisionSessions ensures the server is running and creates unique sessions for the specified number of agents.
// It returns the server info and a list of session IDs in order.
func (m *Manager) ProvisionSessions(ctx context.Context, projectPath, ntmSessionName string, count int) (*ServerInfo, []string, error) {
	// 1. Ensure Server is Running
	info, err := m.Start(projectPath)
	if err != nil {
		return nil, nil, fmt.Errorf("start server: %w", err)
	}

	// 2. Get Client
	client, err := m.Client(projectPath)
	if err != nil {
		return info, nil, fmt.Errorf("get client: %w", err)
	}

	// 3. Create Sessions
	var sessionIDs []string
	for i := 1; i <= count; i++ {
		title := fmt.Sprintf("NTM Session [%s] Agent %d", ntmSessionName, i)
		session, err := client.Session.New(ctx, opencode.SessionNewParams{
			Title:     opencode.String(title),
			Directory: opencode.String(projectPath),
		})
		if err != nil {
			return info, sessionIDs, fmt.Errorf("create session %d: %w", i, err)
		}
		sessionIDs = append(sessionIDs, session.ID)
	}

	return info, sessionIDs, nil
}

// SendPrompt sends a prompt to the specified session via SDK
func (m *Manager) SendPrompt(ctx context.Context, projectPath, sessionID, prompt string) error {
	client, err := m.Client(projectPath)
	if err != nil {
		return err
	}

	_, err = client.Session.Prompt(ctx, sessionID, opencode.SessionPromptParams{
		Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
			opencode.TextPartInputParam{
				Type: opencode.F(opencode.TextPartInputTypeText),
				Text: opencode.F(prompt),
			},
		}),
	})
	return err
}
