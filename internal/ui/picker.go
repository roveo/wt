package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/roveo/wt/internal/db"
	"github.com/sahilm/fuzzy"
	"golang.org/x/term"
)

// PickerAction represents what action the user wants to take
type PickerAction int

const (
	ActionNone PickerAction = iota
	ActionSwitch
	ActionAdd
	ActionBack   // Return from add mode to worktree list
	ActionDelete // Delete the selected worktree
)

// PickerResult contains the result of the picker
type PickerResult struct {
	Action   PickerAction
	Worktree *db.Worktree
}

// renderer uses stderr to avoid polluting stdout with terminal escape sequences
// We use ANSI profile to avoid terminal queries for color support detection
var renderer *lipgloss.Renderer

// Styles using terminal theme colors (ANSI)
var (
	selectedStyle lipgloss.Style
	normalStyle   lipgloss.Style
	helpStyle     lipgloss.Style
	matchStyle    lipgloss.Style
	promptStyle   lipgloss.Style
)

func init() {
	// Set the default termenv output to stderr BEFORE any terminal queries happen
	// This prevents escape sequences from being written to stdout
	output := termenv.NewOutput(os.Stderr, termenv.WithProfile(termenv.ANSI256))
	termenv.SetDefaultOutput(output)

	// Create lipgloss renderer using stderr
	renderer = lipgloss.NewRenderer(os.Stderr, termenv.WithProfile(termenv.ANSI256))
	lipgloss.SetDefaultRenderer(renderer)

	// Initialize styles
	selectedStyle = renderer.NewStyle().Foreground(lipgloss.ANSIColor(6)).Bold(true)   // cyan
	normalStyle = renderer.NewStyle()                                                  // default
	helpStyle = renderer.NewStyle().Faint(true)                                        // dimmed
	matchStyle = renderer.NewStyle().Foreground(lipgloss.ANSIColor(6)).Underline(true) // cyan
	promptStyle = renderer.NewStyle().Foreground(lipgloss.ANSIColor(6))                // cyan
}

// pickerModel is a minimal fzf-like picker
type pickerModel struct {
	worktrees []*db.Worktree
	filtered  []int // indices into worktrees
	matches   []fuzzy.Match
	cursor    int
	input     textinput.Model
	action    PickerAction
	quitting  bool
	height    int
}

func newPickerModel(worktrees []*db.Worktree) pickerModel {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.PromptStyle = promptStyle
	ti.Focus()

	// Initialize with all items
	filtered := make([]int, len(worktrees))
	for i := range worktrees {
		filtered[i] = i
	}

	return pickerModel{
		worktrees: worktrees,
		filtered:  filtered,
		matches:   nil,
		cursor:    0,
		input:     ti,
		action:    ActionNone,
		height:    10,
	}
}

// worktreeStrings returns searchable strings for fuzzy matching
func (m *pickerModel) worktreeStrings() []string {
	strs := make([]string, len(m.worktrees))
	for i, wt := range m.worktrees {
		strs[i] = formatWorktreeLabel(wt) + " " + wt.Path
	}
	return strs
}

func (m *pickerModel) updateFilter() {
	query := m.input.Value()
	if query == "" {
		// Show all
		m.filtered = make([]int, len(m.worktrees))
		for i := range m.worktrees {
			m.filtered[i] = i
		}
		m.matches = nil
	} else {
		// Fuzzy filter
		m.matches = fuzzy.Find(query, m.worktreeStrings())
		m.filtered = make([]int, len(m.matches))
		for i, match := range m.matches {
			m.filtered[i] = match.Index
		}
	}
	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

func (m pickerModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.height = min(msg.Height-3, 20) // Leave room for input and help
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.action = ActionNone
			m.quitting = true
			return m, tea.Quit

		case tea.KeyTab:
			m.action = ActionAdd
			m.quitting = true
			return m, tea.Quit

		case tea.KeyCtrlD:
			// Delete selected worktree (if not main)
			if len(m.filtered) > 0 {
				idx := m.filtered[m.cursor]
				wt := m.worktrees[idx]
				if !wt.IsMain {
					m.action = ActionDelete
					m.quitting = true
					return m, tea.Quit
				}
			}
			return m, nil

		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				m.action = ActionSwitch
				m.quitting = true
				return m, tea.Quit
			}

		case tea.KeyUp, tea.KeyCtrlP:
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil

		case tea.KeyDown, tea.KeyCtrlN:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil
		}
	}

	// Update text input
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.updateFilter()

	return m, cmd
}

func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// Input line
	b.WriteString(m.input.View())
	b.WriteString("\n")

	// Items
	visible := min(len(m.filtered), m.height)
	start := 0
	if m.cursor >= visible {
		start = m.cursor - visible + 1
	}

	for i := start; i < start+visible && i < len(m.filtered); i++ {
		idx := m.filtered[i]
		wt := m.worktrees[idx]
		label := formatWorktreeLabel(wt)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render("> " + label))
		} else {
			// Apply match highlighting if we have matches
			if m.matches != nil && i < len(m.matches) {
				highlighted := highlightMatches(label, m.matches[i].MatchedIndexes)
				b.WriteString("  " + highlighted)
			} else {
				b.WriteString(normalStyle.Render("  " + label))
			}
		}
		b.WriteString("\n")
	}

	// Help line
	countInfo := fmt.Sprintf("%d/%d", len(m.filtered), len(m.worktrees))
	help := helpStyle.Render(countInfo + "  enter:select  tab:add  ctrl-d:delete  esc:quit")
	b.WriteString(help)

	return b.String()
}

func highlightMatches(s string, indices []int) string {
	if len(indices) == 0 {
		return normalStyle.Render(s)
	}

	// Build highlighted string
	var b strings.Builder
	matchSet := make(map[int]bool)
	for _, idx := range indices {
		matchSet[idx] = true
	}

	for i, r := range s {
		if matchSet[i] {
			b.WriteString(matchStyle.Render(string(r)))
		} else {
			b.WriteString(normalStyle.Render(string(r)))
		}
	}
	return b.String()
}

// PickWorktree shows an interactive picker for worktrees
// Returns the selected worktree and the action (switch or add)
func PickWorktree(worktrees []*db.Worktree) (*PickerResult, error) {
	if len(worktrees) == 0 {
		return &PickerResult{Action: ActionAdd}, nil
	}

	m := newPickerModel(worktrees)

	// Redirect stdout fd to stderr during TUI to prevent terminal escape sequences
	// from polluting stdout (which is used for the cd command)
	stdout := os.Stdout
	os.Stdout = os.Stderr
	defer func() { os.Stdout = stdout }()

	// Use stderr for TUI output so stdout is clean for cd command
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(pickerModel)

	// Return the selected worktree for actions that need it
	if (result.action == ActionSwitch || result.action == ActionDelete || result.action == ActionAdd) && len(result.filtered) > 0 {
		idx := result.filtered[result.cursor]
		return &PickerResult{
			Action:   result.action,
			Worktree: result.worktrees[idx],
		}, nil
	}

	return &PickerResult{
		Action:   result.action,
		Worktree: nil,
	}, nil
}

// formatWorktreeLabel formats a worktree for display in the picker
func formatWorktreeLabel(wt *db.Worktree) string {
	var sb strings.Builder

	// Format: repo/branch (path)
	sb.WriteString(wt.RepoName)
	sb.WriteString("/")
	sb.WriteString(wt.Branch)

	if wt.IsMain {
		sb.WriteString(" [main]")
	}

	return sb.String()
}

// PickWorktreeSimple shows a simple worktree picker without Tab functionality
// Used by remove command where we don't need the add workflow
func PickWorktreeSimple(worktrees []*db.Worktree) (*db.Worktree, error) {
	if len(worktrees) == 0 {
		return nil, nil
	}

	// Use the same fzf-like picker but without tab=add functionality
	m := newPickerModel(worktrees)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))

	finalModel, err := p.Run()
	if err != nil {
		return nil, err
	}

	result := finalModel.(pickerModel)

	if result.action == ActionSwitch && len(result.filtered) > 0 {
		idx := result.filtered[result.cursor]
		return result.worktrees[idx], nil
	}

	return nil, nil
}

// inputBranchModel is a simple text input with back support
type inputBranchModel struct {
	input        textinput.Model
	action       PickerAction
	quitting     bool
	sourceRepo   string
	sourceBranch string
}

func newInputBranchModel(placeholder, sourceRepo, sourceBranch string) inputBranchModel {
	ti := textinput.New()
	ti.Prompt = promptStyle.Render("> ")
	ti.Placeholder = placeholder
	ti.PlaceholderStyle = renderer.NewStyle().Faint(true)
	ti.TextStyle = renderer.NewStyle()
	ti.Cursor.Style = renderer.NewStyle().Foreground(lipgloss.ANSIColor(6))
	ti.Focus()
	ti.CharLimit = 100
	ti.Width = 50

	return inputBranchModel{
		input:        ti,
		action:       ActionNone,
		sourceRepo:   sourceRepo,
		sourceBranch: sourceBranch,
	}
}

func (m inputBranchModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m inputBranchModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.action = ActionNone
			m.quitting = true
			return m, tea.Quit

		case tea.KeyTab:
			m.action = ActionBack
			m.quitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			m.action = ActionSwitch
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m inputBranchModel) View() string {
	if m.quitting {
		return ""
	}

	source := helpStyle.Render(fmt.Sprintf("from %s/%s", m.sourceRepo, m.sourceBranch))
	title := selectedStyle.Render("New branch name:") + " " + source
	help := helpStyle.Render("enter:create  tab:back  esc:quit")
	return fmt.Sprintf("%s\n%s\n\n%s", title, m.input.View(), help)
}

// InputBranch prompts for a branch name
// sourceRepo and sourceBranch are displayed to show where the worktree will be created from
// Returns the branch name and action (ActionBack if user wants to go back)
func InputBranch(placeholder, sourceRepo, sourceBranch string) (string, PickerAction, error) {
	m := newInputBranchModel(placeholder, sourceRepo, sourceBranch)

	// Redirect stdout fd to stderr during TUI to prevent terminal escape sequences
	// from polluting stdout (which is used for the cd command)
	stdout := os.Stdout
	os.Stdout = os.Stderr
	defer func() { os.Stdout = stdout }()

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	finalModel, err := p.Run()
	if err != nil {
		return "", ActionNone, err
	}

	result := finalModel.(inputBranchModel)

	if result.action == ActionSwitch {
		return strings.TrimSpace(result.input.Value()), ActionNone, nil
	}

	return "", result.action, nil
}

// Confirm shows a simple confirmation prompt
// Press enter to confirm, any other key to cancel
func Confirm(message string) (bool, error) {
	fmt.Fprintf(os.Stderr, "%s [enter to confirm] ", message)

	// Set terminal to raw mode to read single keypress
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return false, err
	}
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	// Read single byte
	b := make([]byte, 1)
	_, err = os.Stdin.Read(b)
	fmt.Fprintln(os.Stderr) // newline after keypress

	if err != nil {
		return false, err
	}

	// Enter key (13 = CR, 10 = LF)
	return b[0] == 13 || b[0] == 10, nil
}
