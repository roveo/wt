package tmux

import (
	"os"
	"os/exec"
	"strings"
)

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
	cmd := exec.Command("tmux", "new-session", "-d", "-s", name)
	return cmd.Run()
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
	cmd := exec.Command("tmux", "select-window", "-t", target)
	return cmd.Run()
}

// CreateWindow creates a new window in the given session
// If onEnter is provided, it will be executed as the initial command
func CreateWindow(session, windowName, path, onEnter string) error {
	args := []string{"new-window", "-t", session, "-n", windowName, "-c", path}
	if onEnter != "" {
		args = append(args, onEnter)
	}
	cmd := exec.Command("tmux", args...)
	return cmd.Run()
}

// SwitchClient switches the tmux client to a different session
func SwitchClient(session string) error {
	cmd := exec.Command("tmux", "switch-client", "-t", session)
	return cmd.Run()
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
	cmd := exec.Command("tmux", "kill-window", "-t", target)
	return cmd.Run()
}
