package tmux

import (
	"os/exec"
	"strings"
	"testing"
)

// setupTestSession creates a tmux session for testing and returns a cleanup function
func setupTestSession(t *testing.T, sessionName string) func() {
	t.Helper()

	// Check if tmux is available
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not available, skipping test")
	}

	// Create a test session
	cmd := exec.Command("tmux", "new-session", "-d", "-s", sessionName)
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create test session: %v", err)
	}

	return func() {
		exec.Command("tmux", "kill-session", "-t", sessionName).Run()
	}
}

// getWindowIndices returns all window indices in a session
func getWindowIndices(session string) ([]string, error) {
	cmd := exec.Command("tmux", "list-windows", "-t", session, "-F", "#{window_index}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var indices []string
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		if line != "" {
			indices = append(indices, line)
		}
	}
	return indices, nil
}

// selectWindow switches to a specific window index in a session
func selectWindow(session string, index string) error {
	cmd := exec.Command("tmux", "select-window", "-t", session+":"+index)
	return cmd.Run()
}

func TestCreateWindow_WithExistingWindowAtNextIndex(t *testing.T) {
	sessionName := "wt-test-session"
	cleanup := setupTestSession(t, sessionName)
	defer cleanup()

	// Initial state: session has window 0 (created by new-session)
	// Create windows at indices 1, 2, 3 to have no gaps (worst case scenario)
	cmd := exec.Command("tmux", "new-window", "-t", sessionName+":1", "-n", "window-1")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create window at index 1: %v", err)
	}

	cmd = exec.Command("tmux", "new-window", "-t", sessionName+":2", "-n", "window-2")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create window at index 2: %v", err)
	}

	cmd = exec.Command("tmux", "new-window", "-t", sessionName+":3", "-n", "window-3")
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create window at index 3: %v", err)
	}

	// Now we have windows at indices 0, 1, 2, 3 (no gaps)
	// Select window 2 (to simulate being in window 2)
	if err := selectWindow(sessionName, "2"); err != nil {
		t.Fatalf("failed to select window 2: %v", err)
	}

	// Verify current state
	indices, err := getWindowIndices(sessionName)
	if err != nil {
		t.Fatalf("failed to get window indices: %v", err)
	}
	t.Logf("Window indices before CreateWindow: %v", indices)

	// Try to create a new window using our CreateWindow function
	// This should NOT fail with "index in use" error
	// The fix uses "-t session:" which lets tmux find the next available index
	err = CreateWindow(sessionName, "test-worktree", "/tmp", "")
	if err != nil {
		t.Errorf("CreateWindow failed: %v", err)
		t.Log("Bug: CreateWindow fails when next index after current window is in use")
	}

	// Verify the window was created
	if !WindowExists(sessionName, "test-worktree") {
		t.Error("Window 'test-worktree' should exist after CreateWindow")
	}

	// Check final indices
	indices, err = getWindowIndices(sessionName)
	if err != nil {
		t.Fatalf("failed to get window indices: %v", err)
	}
	t.Logf("Window indices after CreateWindow: %v", indices)
}

func TestCreateWindow_Basic(t *testing.T) {
	sessionName := "wt-test-basic"
	cleanup := setupTestSession(t, sessionName)
	defer cleanup()

	err := CreateWindow(sessionName, "basic-window", "/tmp", "")
	if err != nil {
		t.Errorf("CreateWindow failed: %v", err)
	}

	if !WindowExists(sessionName, "basic-window") {
		t.Error("Window 'basic-window' should exist")
	}
}

func TestKillWindow(t *testing.T) {
	sessionName := "wt-test-kill"
	cleanup := setupTestSession(t, sessionName)
	defer cleanup()

	// Create a window to kill
	err := CreateWindow(sessionName, "to-kill", "/tmp", "")
	if err != nil {
		t.Fatalf("CreateWindow failed: %v", err)
	}

	if !WindowExists(sessionName, "to-kill") {
		t.Fatal("Window 'to-kill' should exist before killing")
	}

	// Kill the window
	err = KillWindow(sessionName, "to-kill")
	if err != nil {
		t.Errorf("KillWindow failed: %v", err)
	}

	if WindowExists(sessionName, "to-kill") {
		t.Error("Window 'to-kill' should not exist after killing")
	}
}
