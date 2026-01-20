package main

import (
	"context"
	"fmt"
	"os"

	"github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

// usage: go run tests/verify_opencode_sdk.go <server-url>
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run verify_opencode_sdk.go <server-url>")
		os.Exit(1)
	}
	url := os.Args[1]

	client := opencode.NewClient(
		option.WithBaseURL(url),
	)

	fmt.Printf("Connecting to OpenCode SDK at %s...\n", url)

	sessions, err := client.Session.List(context.Background(), opencode.SessionListParams{})
	if err != nil {
		fmt.Printf("Error listing sessions: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Found %d sessions\n", len(*sessions))
	if len(*sessions) > 0 {
		s := (*sessions)[0]
		fmt.Printf("Sample Session: ID=%s, Project=%s, Directory=%s\n", s.ID, s.ProjectID, s.Directory)
	}
}

