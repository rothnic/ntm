package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/Dicklesworthstone/ntm/internal/opencode"
)

func newOpencodeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "opencode",
		Short: "Manage per-project opencode servers",
		Long: `Manage lifecycle of opencode servers for projects.

Each project gets an isolated opencode server on a stable port.
Servers persist in ~/.opencode/servers/<project-id>/`,
	}

	cmd.AddCommand(newOpencodeStartCmd())
	cmd.AddCommand(newOpencodeStopCmd())
	cmd.AddCommand(newOpencodeStatusCmd())
	cmd.AddCommand(newOpencodeListCmd())
	cmd.AddCommand(newOpencodeLogsCmd())
	cmd.AddCommand(newOpencodeURLCmd())
	cmd.AddCommand(newOpencodeReapCmd())

	return cmd
}

func newOpencodeStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start [project-path]",
		Short: "Start opencode server for a project",
		Long: `Start an opencode server for the given project directory.

If no path is provided, uses current directory.
Server starts on a stable, hash-based port (28000-29000 range).

Examples:
  ntm opencode start                 # Start for current directory
  ntm opencode start /path/to/proj   # Start for specific project
  ntm opencode start --json          # Output JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: runOpencodeStart,
	}

	return cmd
}

func runOpencodeStart(cmd *cobra.Command, args []string) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Get absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Verify directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return fmt.Errorf("project directory does not exist: %s", absPath)
	}

	// Create manager
	mgr, err := opencode.NewManager()
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Start server
	info, err := mgr.Start(absPath)
	if err != nil {
		if jsonOutput {
			return outputJSON(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}
		return err
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"success":      true,
			"project_path": info.ProjectPath,
			"project_id":   info.ProjectID,
			"pid":          info.PID,
			"port":         info.Port,
			"url":          fmt.Sprintf("http://127.0.0.1:%d", info.Port),
			"log_file":     info.LogFile,
		})
	}

	fmt.Printf("✓ OpenCode server started for project: %s\n", absPath)
	fmt.Printf("  PID:      %d\n", info.PID)
	fmt.Printf("  Port:     %d\n", info.Port)
	fmt.Printf("  URL:      http://127.0.0.1:%d\n", info.Port)
	fmt.Printf("  Logs:     %s\n", info.LogFile)
	fmt.Printf("\nAttach with: opencode attach http://127.0.0.1:%d\n", info.Port)

	return nil
}

func newOpencodeStopCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "stop [project-path]",
		Short: "Stop opencode server for a project",
		Long: `Stop the opencode server for the given project directory.

By default, refuses to stop if there are active connections.
Use --force to kill the server regardless of connections.

Examples:
  ntm opencode stop                  # Stop server for current directory
  ntm opencode stop /path/to/proj    # Stop server for specific project
  ntm opencode stop --force          # Force stop even with connections`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpencodeStop(cmd, args, force)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force stop even with active connections")

	return cmd
}

func runOpencodeStop(cmd *cobra.Command, args []string, force bool) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Get absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Create manager
	mgr, err := opencode.NewManager()
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Stop server
	if err := mgr.Stop(absPath, force); err != nil {
		if jsonOutput {
			return outputJSON(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}
		return err
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"success":      true,
			"project_path": absPath,
		})
	}

	fmt.Printf("✓ OpenCode server stopped for project: %s\n", absPath)

	return nil
}

func newOpencodeStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status [project-path]",
		Short: "Show status of opencode server for a project",
		Long: `Show detailed status of the opencode server for the given project.

Examples:
  ntm opencode status                # Status for current directory
  ntm opencode status /path/to/proj  # Status for specific project
  ntm opencode status --json         # Output JSON`,
		Args: cobra.MaximumNArgs(1),
		RunE: runOpencodeStatus,
	}

	return cmd
}

func runOpencodeStatus(cmd *cobra.Command, args []string) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Get absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Create manager
	mgr, err := opencode.NewManager()
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Get status
	info, err := mgr.Status(absPath)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"project_path": info.ProjectPath,
			"project_id":   info.ProjectID,
			"running":      info.Running,
			"pid":          info.PID,
			"port":         info.Port,
			"connections":  info.Connections,
			"log_file":     info.LogFile,
		})
	}

	fmt.Printf("Project: %s\n", absPath)
	fmt.Printf("ID:      %s\n", info.ProjectID)

	if info.Running {
		fmt.Printf("Status:  ✓ Running\n")
		fmt.Printf("PID:     %d\n", info.PID)
		fmt.Printf("Port:    %d\n", info.Port)
		fmt.Printf("URL:     http://127.0.0.1:%d\n", info.Port)
		fmt.Printf("Connections: %d\n", info.Connections)
		fmt.Printf("Logs:    %s\n", info.LogFile)
	} else {
		fmt.Printf("Status:  ✗ Not running\n")
	}

	return nil
}

func newOpencodeListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all managed opencode servers",
		Long: `List all opencode servers managed by ntm.

Shows both running and stopped servers.

Examples:
  ntm opencode list       # List all servers
  ntm opencode list --json # Output JSON`,
		Args: cobra.NoArgs,
		RunE: runOpencodeList,
	}

	return cmd
}

func runOpencodeList(cmd *cobra.Command, args []string) error {
	// Create manager
	mgr, err := opencode.NewManager()
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Get list
	servers, err := mgr.List()
	if err != nil {
		return fmt.Errorf("list servers: %w", err)
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"servers": servers,
			"count":   len(servers),
		})
	}

	if len(servers) == 0 {
		fmt.Println("No opencode servers found.")
		return nil
	}

	// Use tabwriter for aligned columns
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tSTATUS\tPORT\tCONNECTIONS\tPID")
	fmt.Fprintln(w, "-------\t------\t----\t-----------\t---")

	for _, info := range servers {
		status := "stopped"
		statusIcon := "✗"
		if info.Running {
			status = "running"
			statusIcon = "✓"
		}

		portStr := "-"
		if info.Port > 0 {
			portStr = fmt.Sprintf("%d", info.Port)
		}

		pidStr := "-"
		if info.PID > 0 {
			pidStr = fmt.Sprintf("%d", info.PID)
		}

		connStr := "-"
		if info.Running {
			connStr = fmt.Sprintf("%d", info.Connections)
		}

		// Shorten project path for display
		projectPath := info.ProjectPath
		if home, err := os.UserHomeDir(); err == nil {
			projectPath = strings.Replace(projectPath, home, "~", 1)
		}

		fmt.Fprintf(w, "%s\t%s %s\t%s\t%s\t%s\n",
			projectPath,
			statusIcon,
			status,
			portStr,
			connStr,
			pidStr)
	}

	w.Flush()
	fmt.Printf("\nTotal: %d servers\n", len(servers))

	return nil
}

func newOpencodeLogsCmd() *cobra.Command {
	var follow bool
	var lines int

	cmd := &cobra.Command{
		Use:   "logs [project-path]",
		Short: "Tail opencode server logs",
		Long: `Show logs from the opencode server for the given project.

Examples:
  ntm opencode logs                  # Show last 20 lines for current directory
  ntm opencode logs -f               # Follow logs
  ntm opencode logs -n 50            # Show last 50 lines
  ntm opencode logs /path/to/proj -f # Follow logs for specific project`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runOpencodeLogs(cmd, args, follow, lines)
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&lines, "lines", "n", 20, "Number of lines to show")

	return cmd
}

func runOpencodeLogs(cmd *cobra.Command, args []string, follow bool, lines int) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Get absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Create manager
	mgr, err := opencode.NewManager()
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Get status to find log file
	info, err := mgr.Status(absPath)
	if err != nil {
		return fmt.Errorf("get status: %w", err)
	}

	if info.LogFile == "" {
		return fmt.Errorf("no log file found for project: %s", absPath)
	}

	// Check if log file exists
	if _, err := os.Stat(info.LogFile); os.IsNotExist(err) {
		return fmt.Errorf("log file does not exist: %s", info.LogFile)
	}

	// Use tail command to show logs
	args2 := []string{fmt.Sprintf("-n%d", lines)}
	if follow {
		args2 = append(args2, "-f")
	}
	args2 = append(args2, info.LogFile)

	tailCmd := exec.Command("tail", args2...)
	tailCmd.Stdout = os.Stdout
	tailCmd.Stderr = os.Stderr

	return tailCmd.Run()
}

func newOpencodeURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "url [project-path]",
		Short: "Get attach URL for project's opencode server",
		Long: `Get the HTTP URL for attaching to the opencode server.

Examples:
  ntm opencode url                   # URL for current directory
  ntm opencode url /path/to/proj     # URL for specific project
  opencode attach $(ntm opencode url) # Attach using the URL`,
		Args: cobra.MaximumNArgs(1),
		RunE: runOpencodeURL,
	}

	return cmd
}

func runOpencodeURL(cmd *cobra.Command, args []string) error {
	projectPath := "."
	if len(args) > 0 {
		projectPath = args[0]
	}

	// Get absolute path
	absPath, err := filepath.Abs(projectPath)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	// Create manager
	mgr, err := opencode.NewManager()
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Get URL
	url, err := mgr.URL(absPath)
	if err != nil {
		if jsonOutput {
			return outputJSON(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}
		return err
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"success":      true,
			"project_path": absPath,
			"url":          url,
		})
	}

	// Just print the URL (for easy shell scripting)
	fmt.Println(url)

	return nil
}

func newOpencodeReapCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "reap",
		Short: "Stop all idle opencode servers",
		Long: `Stop all opencode servers with no active connections.

This command is designed to be run periodically (e.g., via cron or systemd timer)
to clean up idle servers and free resources.

Examples:
  ntm opencode reap           # Stop all idle servers
  ntm opencode reap --json    # Output JSON`,
		Args: cobra.NoArgs,
		RunE: runOpencodeReap,
	}

	return cmd
}

func runOpencodeReap(cmd *cobra.Command, args []string) error {
	// Create manager
	mgr, err := opencode.NewManager()
	if err != nil {
		return fmt.Errorf("create manager: %w", err)
	}

	// Reap idle servers
	stopped, err := mgr.Reap()
	if err != nil {
		if jsonOutput {
			return outputJSON(map[string]interface{}{
				"success": false,
				"error":   err.Error(),
			})
		}
		return err
	}

	if jsonOutput {
		return outputJSON(map[string]interface{}{
			"success": true,
			"stopped": stopped,
		})
	}

	if stopped == 0 {
		fmt.Println("No idle servers to stop.")
	} else {
		fmt.Printf("✓ Stopped %d idle server(s)\n", stopped)
	}

	return nil
}
