package tmux

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// runTmux executes a tmux command and returns a descriptive error if it fails
func runTmux(args ...string) error {
	cmd := exec.Command("tmux", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		errMsg := strings.TrimSpace(string(output))
		if errMsg != "" {
			return fmt.Errorf("%s", errMsg)
		}
		return err
	}
	return nil
}

// InTmux returns true if currently running inside a tmux session
func InTmux() bool {
	return os.Getenv("TMUX") != ""
}

// CurrentSession returns the name of the current tmux session
// Returns empty string if not in tmux
func CurrentSession() string {
	if !InTmux() {
		return ""
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#{session_name}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// SessionExists checks if a tmux session with the given name exists
func SessionExists(name string) bool {
	cmd := exec.Command("tmux", "has-session", "-t", name)
	return cmd.Run() == nil
}

// CreateSession creates a new detached tmux session
func CreateSession(name string) error {
	return runTmux("new-session", "-d", "-s", name)
}

// WindowExists checks if a window with the given name exists in the session
func WindowExists(session, windowName string) bool {
	cmd := exec.Command("tmux", "list-windows", "-t", session, "-F", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == windowName {
			return true
		}
	}
	return false
}

// SwitchToWindow switches to an existing window in the given session
func SwitchToWindow(session, windowName string) error {
	target := session + ":" + windowName
	return runTmux("select-window", "-t", target)
}

// CreateWindow creates a new window in the given session
// If onEnter is provided, it will be executed as the initial command
func CreateWindow(session, windowName, path, onEnter string) error {
	// Use "session:" (with trailing colon) to target the session without
	// specifying a window index. This lets tmux automatically find the next
	// available index, avoiding "index in use" errors when the index after
	// the current window is already taken.
	args := []string{"new-window", "-t", session + ":", "-n", windowName, "-c", path}
	if onEnter != "" {
		args = append(args, onEnter)
	}
	return runTmux(args...)
}

// SwitchClient switches the tmux client to a different session
func SwitchClient(session string) error {
	return runTmux("switch-client", "-t", session)
}

// CurrentWindow returns the name of the current tmux window
// Returns empty string if not in tmux or on error
func CurrentWindow() string {
	if !InTmux() {
		return ""
	}
	cmd := exec.Command("tmux", "display-message", "-p", "#{window_name}")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// KillWindow kills a window in the given session
func KillWindow(session, windowName string) error {
	target := session + ":" + windowName
	return runTmux("kill-window", "-t", target)
}
