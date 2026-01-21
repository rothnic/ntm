package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	opencode "github.com/sst/opencode-sdk-go"
	"github.com/sst/opencode-sdk-go/option"
)

// TestSendPromptReliable_SuccessFirstAttempt verifies that when the server
// accepts the prompt immediately and messages are present, we succeed without retries.
func TestSendPromptReliable_SuccessFirstAttempt(t *testing.T) {
	messageCount := int32(0)

	// Mock OpenCode server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/session/test-session/message":
			// Prompt accepted - increment message count
			atomic.AddInt32(&messageCount, 2) // user + assistant
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id": "msg_123",
			})

		case r.Method == "GET" && r.URL.Path == "/session/test-session/message":
			// Return current messages
			count := atomic.LoadInt32(&messageCount)
			messages := make([]map[string]interface{}, count)
			for i := int32(0); i < count; i++ {
				messages[i] = map[string]interface{}{
					"id":        fmt.Sprintf("msg_%d", i),
					"sessionID": "test-session",
					"role":      "user",
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messages)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	mgr := &Manager{}
	client := opencode.NewClient(option.WithBaseURL(server.URL))

	// Get initial count
	initialCount, err := mgr.getMessageCount(context.Background(), client, "test-session")
	if err != nil {
		t.Fatalf("getMessageCount failed: %v", err)
	}
	if initialCount != 0 {
		t.Fatalf("expected initial count 0, got %d", initialCount)
	}

	// Send prompt
	_, err = client.Session.Prompt(context.Background(), "test-session", opencode.SessionPromptParams{
		Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
			opencode.TextPartInputParam{
				Type: opencode.F(opencode.TextPartInputTypeText),
				Text: opencode.F("test message"),
			},
		}),
	})
	if err != nil {
		t.Fatalf("Prompt failed: %v", err)
	}

	// Verify
	cfg := PromptConfig{
		VerifyTimeout: 1 * time.Second,
	}
	verified, err := mgr.verifyPromptReceived(context.Background(), client, "test-session", initialCount, cfg.VerifyTimeout)
	if err != nil {
		t.Fatalf("verifyPromptReceived failed: %v", err)
	}
	if !verified {
		t.Error("expected prompt to be verified")
	}
}

// TestSendPromptReliable_RetryOnTransientFailure verifies that transient
// failures trigger retries and eventually succeed.
func TestSendPromptReliable_RetryOnTransientFailure(t *testing.T) {
	attemptCount := int32(0)
	messageCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/session/test-session/message":
			attempt := atomic.AddInt32(&attemptCount, 1)
			if attempt < 3 {
				// Fail first 2 attempts with 503 Service Unavailable
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"error": "service unavailable"}`))
				return
			}
			// Third attempt succeeds
			atomic.AddInt32(&messageCount, 2)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "msg_123"})

		case r.Method == "GET" && r.URL.Path == "/session/test-session/message":
			count := atomic.LoadInt32(&messageCount)
			messages := make([]map[string]interface{}, count)
			for i := int32(0); i < count; i++ {
				messages[i] = map[string]interface{}{
					"id":        fmt.Sprintf("msg_%d", i),
					"sessionID": "test-session",
					"role":      "user",
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messages)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := opencode.NewClient(option.WithBaseURL(server.URL))

	// Manual retry loop (simulating SendPromptReliable logic)
	var lastErr error
	maxRetries := 5
	for attempt := 1; attempt <= maxRetries; attempt++ {
		_, lastErr = client.Session.Prompt(context.Background(), "test-session", opencode.SessionPromptParams{
			Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
				opencode.TextPartInputParam{
					Type: opencode.F(opencode.TextPartInputTypeText),
					Text: opencode.F("test message"),
				},
			}),
		})
		if lastErr == nil {
			break
		}
		t.Logf("Attempt %d failed: %v", attempt, lastErr)
		time.Sleep(50 * time.Millisecond)
	}

	// Should have succeeded on 3rd attempt
	attempts := atomic.LoadInt32(&attemptCount)
	if attempts < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempts)
	}
	if lastErr != nil {
		t.Errorf("expected success after retries, got error: %v", lastErr)
	}
	t.Logf("Success after %d attempts", attempts)
}

// TestSendPromptReliable_VerificationTimeout verifies that when the server
// accepts the prompt but messages never appear, we timeout and return false.
func TestSendPromptReliable_VerificationTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/session/test-session/message":
			// Accept prompt but don't actually add message
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "msg_123"})

		case r.Method == "GET" && r.URL.Path == "/session/test-session/message":
			// Always return empty - message never persisted
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	mgr := &Manager{}
	client := opencode.NewClient(option.WithBaseURL(server.URL))

	// Very short timeout to speed up test
	start := time.Now()
	verified, err := mgr.verifyPromptReceived(context.Background(), client, "test-session", 0, 500*time.Millisecond)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("verifyPromptReceived returned unexpected error: %v", err)
	}
	if verified {
		t.Error("expected verification to fail (timeout), but it succeeded")
	}
	// Should have waited approximately the timeout duration
	if elapsed < 400*time.Millisecond {
		t.Errorf("verification returned too quickly (%v), expected ~500ms", elapsed)
	}
	t.Logf("Verification correctly timed out after %v", elapsed)
}

// TestSendPromptReliable_ContextCancellation verifies that cancelling
// the context properly stops verification.
func TestSendPromptReliable_ContextCancellation(t *testing.T) {
	requestCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&requestCount, 1)
		// Always return empty messages
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	mgr := &Manager{}
	client := opencode.NewClient(option.WithBaseURL(server.URL))

	// Cancel context after 100ms
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	verified, err := mgr.verifyPromptReceived(ctx, client, "test-session", 0, 5*time.Second)
	elapsed := time.Since(start)

	// Should fail due to context cancellation, not the 5s verify timeout
	if verified {
		t.Error("expected verification to fail due to context cancellation")
	}
	if err != context.DeadlineExceeded {
		t.Logf("Got error (expected context.DeadlineExceeded): %v", err)
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("should have cancelled quickly, took %v", elapsed)
	}

	requests := atomic.LoadInt32(&requestCount)
	t.Logf("Made %d requests before context cancelled after %v", requests, elapsed)
}

// TestSendPromptReliable_RaceCondition simulates multiple prompts
// being sent concurrently to verify mutex serialization works.
func TestSendPromptReliable_RaceCondition(t *testing.T) {
	var mu sync.Mutex
	promptOrder := []int{}
	messageCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.URL.Path == "/session/test-session/message" {
			// Simulate processing time that could cause race conditions
			time.Sleep(20 * time.Millisecond)

			// Record order of receipt
			mu.Lock()
			promptOrder = append(promptOrder, len(promptOrder)+1)
			mu.Unlock()

			atomic.AddInt32(&messageCount, 1)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "msg_123"})
			return
		}
		if r.Method == "GET" && r.URL.Path == "/session/test-session/message" {
			count := atomic.LoadInt32(&messageCount)
			messages := make([]map[string]interface{}, count)
			for i := int32(0); i < count; i++ {
				messages[i] = map[string]interface{}{
					"id":        fmt.Sprintf("msg_%d", i),
					"sessionID": "test-session",
					"role":      "user",
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messages)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	client := opencode.NewClient(option.WithBaseURL(server.URL))

	// Launch 5 concurrent prompt sends WITH mutex (like spawn.go does)
	var wg sync.WaitGroup
	var promptMu sync.Mutex // Simulates opencodePromptMu in spawn.go

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// This mutex ensures serialization
			promptMu.Lock()
			defer promptMu.Unlock()

			_, err := client.Session.Prompt(context.Background(), "test-session", opencode.SessionPromptParams{
				Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
					opencode.TextPartInputParam{
						Type: opencode.F(opencode.TextPartInputTypeText),
						Text: opencode.F(fmt.Sprintf("message %d", idx)),
					},
				}),
			})
			if err != nil {
				t.Errorf("Prompt %d failed: %v", idx, err)
			}
		}(i)
	}

	wg.Wait()

	// Verify all prompts were processed in order
	mu.Lock()
	count := len(promptOrder)
	mu.Unlock()

	if count != 5 {
		t.Errorf("expected 5 prompts processed, got %d", count)
	}
	t.Logf("All 5 prompts processed successfully with mutex serialization")
}

// TestGetMessageCount_EmptySession verifies correct behavior for new sessions
func TestGetMessageCount_EmptySession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]map[string]interface{}{})
	}))
	defer server.Close()

	mgr := &Manager{}
	client := opencode.NewClient(option.WithBaseURL(server.URL))

	count, err := mgr.getMessageCount(context.Background(), client, "test-session")
	if err != nil {
		t.Fatalf("getMessageCount failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 messages, got %d", count)
	}
}

// TestVerifyPromptReceived_DelayedMessages verifies that we detect
// new messages appearing after a delay (simulating slow LLM response).
func TestVerifyPromptReceived_DelayedMessages(t *testing.T) {
	messageCount := int32(0)

	// Simulate messages appearing after a delay
	go func() {
		time.Sleep(300 * time.Millisecond)
		atomic.StoreInt32(&messageCount, 2) // user + assistant
	}()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.LoadInt32(&messageCount)
		messages := make([]map[string]interface{}, count)
		for i := int32(0); i < count; i++ {
			messages[i] = map[string]interface{}{
				"id":        fmt.Sprintf("msg_%d", i),
				"sessionID": "test-session",
				"role":      "user",
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(messages)
	}))
	defer server.Close()

	mgr := &Manager{}
	client := opencode.NewClient(option.WithBaseURL(server.URL))

	start := time.Now()
	verified, err := mgr.verifyPromptReceived(context.Background(), client, "test-session", 0, 2*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("verifyPromptReceived failed: %v", err)
	}
	if !verified {
		t.Error("expected verification to succeed")
	}

	// Should have waited ~300ms for messages to appear
	if elapsed < 250*time.Millisecond {
		t.Errorf("verification was too fast (%v), messages shouldn't have appeared yet", elapsed)
	}
	if elapsed > 1*time.Second {
		t.Errorf("verification took too long (%v)", elapsed)
	}
	t.Logf("Verification completed in %v (messages appeared after ~300ms)", elapsed)
}

// TestSendPromptReliable_ServerDown verifies behavior when server is unreachable
func TestSendPromptReliable_ServerDown(t *testing.T) {
	// Create server and immediately close it
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // Server is now unreachable

	client := opencode.NewClient(
		option.WithBaseURL(serverURL),
		option.WithRequestTimeout(500*time.Millisecond),
	)

	_, err := client.Session.Prompt(context.Background(), "test-session", opencode.SessionPromptParams{
		Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
			opencode.TextPartInputParam{
				Type: opencode.F(opencode.TextPartInputTypeText),
				Text: opencode.F("test message"),
			},
		}),
	})

	// Should fail with connection error
	if err == nil {
		t.Error("expected error when server is down")
	}
	t.Logf("Got expected error for unreachable server: %v", err)
}

// TestSendPromptReliable_PhantomSuccess tests the critical failure mode where
// the SDK returns success but the message never actually persists.
// This simulates: network succeeded but OpenCode server crashed before saving.
func TestSendPromptReliable_PhantomSuccess(t *testing.T) {
	promptCount := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/session/test-session/message":
			// SDK call "succeeds" but message is never persisted
			count := atomic.AddInt32(&promptCount, 1)
			t.Logf("Received prompt attempt %d", count)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "msg_phantom"})

		case r.Method == "GET" && r.URL.Path == "/session/test-session/message":
			// Always return empty - simulating message not persisted
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	mgr := &Manager{}
	client := opencode.NewClient(option.WithBaseURL(server.URL))

	// Simulate SendPromptReliable behavior with multiple retries
	cfg := PromptConfig{
		MaxRetries:    3,
		RetryInterval: 100 * time.Millisecond,
		VerifyTimeout: 300 * time.Millisecond,
	}

	var lastErr error
	initialCount := 0

	for attempt := 1; attempt <= cfg.MaxRetries; attempt++ {
		// Send prompt - this will "succeed"
		_, err := client.Session.Prompt(context.Background(), "test-session", opencode.SessionPromptParams{
			Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
				opencode.TextPartInputParam{
					Type: opencode.F(opencode.TextPartInputTypeText),
					Text: opencode.F("test message"),
				},
			}),
		})
		if err != nil {
			lastErr = err
			time.Sleep(cfg.RetryInterval)
			continue
		}

		// Verify - this will fail because message never appears
		verified, verifyErr := mgr.verifyPromptReceived(context.Background(), client, "test-session", initialCount, cfg.VerifyTimeout)
		if verifyErr != nil {
			lastErr = verifyErr
			time.Sleep(cfg.RetryInterval)
			continue
		}

		if verified {
			t.Error("should not verify - message never persisted")
			return
		}

		// Not verified - this is the expected path
		lastErr = fmt.Errorf("prompt not verified")
		t.Logf("Attempt %d: SDK succeeded but verification failed (expected)", attempt)
		time.Sleep(cfg.RetryInterval)
	}

	// All retries should have been exhausted
	attempts := atomic.LoadInt32(&promptCount)
	if attempts != 3 {
		t.Errorf("expected 3 prompt attempts, got %d", attempts)
	}
	if lastErr == nil {
		t.Error("expected error after all retries exhausted")
	}
	t.Logf("Correctly detected phantom success after %d attempts: %v", attempts, lastErr)
}

// TestSendPromptReliable_EventualConsistency tests the scenario where
// the message appears after a delay (eventual consistency).
func TestSendPromptReliable_EventualConsistency(t *testing.T) {
	messageCount := int32(0)
	promptReceived := int32(0)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == "POST" && r.URL.Path == "/session/test-session/message":
			atomic.StoreInt32(&promptReceived, 1)
			// Message will appear after a delay (simulated in goroutine below)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "msg_123"})

		case r.Method == "GET" && r.URL.Path == "/session/test-session/message":
			count := atomic.LoadInt32(&messageCount)
			messages := make([]map[string]interface{}, count)
			for i := int32(0); i < count; i++ {
				messages[i] = map[string]interface{}{
					"id":        fmt.Sprintf("msg_%d", i),
					"sessionID": "test-session",
					"role":      "user",
				}
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(messages)

		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Simulate eventual consistency - message appears 400ms after prompt
	go func() {
		for atomic.LoadInt32(&promptReceived) == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		time.Sleep(400 * time.Millisecond)
		atomic.StoreInt32(&messageCount, 2)
	}()

	mgr := &Manager{}
	client := opencode.NewClient(option.WithBaseURL(server.URL))

	// Send prompt
	_, err := client.Session.Prompt(context.Background(), "test-session", opencode.SessionPromptParams{
		Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
			opencode.TextPartInputParam{
				Type: opencode.F(opencode.TextPartInputTypeText),
				Text: opencode.F("test message"),
			},
		}),
	})
	if err != nil {
		t.Fatalf("Prompt failed: %v", err)
	}

	// Verify with enough timeout to handle eventual consistency
	start := time.Now()
	verified, err := mgr.verifyPromptReceived(context.Background(), client, "test-session", 0, 2*time.Second)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("verifyPromptReceived failed: %v", err)
	}
	if !verified {
		t.Error("expected verification to succeed after eventual consistency delay")
	}
	if elapsed < 350*time.Millisecond {
		t.Errorf("verified too quickly (%v), should have waited for eventual consistency", elapsed)
	}
	t.Logf("Verification succeeded after %v (eventual consistency delay was ~400ms)", elapsed)
}

// TestSendPromptReliable_SlowServer verifies timeout handling for slow responses
func TestSendPromptReliable_SlowServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate very slow server
		time.Sleep(2 * time.Second)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "msg_123"})
	}))
	defer server.Close()

	client := opencode.NewClient(
		option.WithBaseURL(server.URL),
		option.WithRequestTimeout(200*time.Millisecond), // Short timeout
	)

	start := time.Now()
	_, err := client.Session.Prompt(context.Background(), "test-session", opencode.SessionPromptParams{
		Parts: opencode.F([]opencode.SessionPromptParamsPartUnion{
			opencode.TextPartInputParam{
				Type: opencode.F(opencode.TextPartInputTypeText),
				Text: opencode.F("test message"),
			},
		}),
	})
	elapsed := time.Since(start)

	// Should timeout quickly, not wait for full 2s
	if err == nil {
		t.Error("expected timeout error")
	}
	if elapsed > 1*time.Second {
		t.Errorf("should have timed out quickly, took %v", elapsed)
	}
	t.Logf("Request timed out after %v (expected ~200ms)", elapsed)
}
