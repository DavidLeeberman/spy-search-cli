package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	VIEW_CHAT = iota
	VIEW_CODE_REVIEW
)

type codeChange struct {
	filename string
	before   string
	after    string
}

type Model struct {
	width, height int
	view          int

	viewport viewport.Model
	textarea textarea.Model
	messages []string

	agentModel  string
	agentAPIKey string
	waiting     bool

	// Code review state
	currentChange codeChange
	showingSteps  bool
	steps         []string
	currentStep   int
}

// Clean color scheme
var (
	green  = lipgloss.Color("#00ff41")
	cyan   = lipgloss.Color("#00ffff")
	yellow = lipgloss.Color("#ffff00")
	red    = lipgloss.Color("#ff0040")
	gray   = lipgloss.Color("#666666")
	white  = lipgloss.Color("#ffffff")
	black  = lipgloss.Color("#000000")
)

// Clean styles
var (
	promptStyle = lipgloss.NewStyle().
			Foreground(green).
			Bold(true)

	agentStyle = lipgloss.NewStyle().
			Foreground(cyan)

	errorStyle = lipgloss.NewStyle().
			Foreground(red).
			Bold(true)

	stepStyle = lipgloss.NewStyle().
			Foreground(yellow)

	dimStyle = lipgloss.NewStyle().
			Foreground(gray)

	codeBeforeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#330000")).
			Foreground(white).
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(red)

	codeAfterStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#003300")).
			Foreground(white).
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(green)

	headerStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(gray).
			Padding(0, 1).
			Bold(true)
)

func NewModel() Model {
	ta := textarea.New()
	ta.Placeholder = "Enter command..."
	ta.Focus()
	ta.CharLimit = 4000
	ta.SetWidth(80)
	ta.SetHeight(2)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(80, 20)

	welcome := agentStyle.Render("AGENT") + ": Ready\n" +
		dimStyle.Render("Usage: \\agent {your prompt here}")

	return Model{
		view:        VIEW_CHAT,
		viewport:    vp,
		textarea:    ta,
		messages:    []string{welcome},
		agentModel:  "gpt-4",
		agentAPIKey: os.Getenv("OPENAI_API_KEY"),
	}
}

func (m Model) Init() tea.Cmd {
	m.updateViewport()
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizeComponents()
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case agentStepMsg:
		return m.handleAgentStep(msg)
	case agentCodeMsg:
		return m.handleAgentCode(msg)
	case editorCompleteMsg:
		return m.handleEditorComplete(msg)
	}

	var cmd tea.Cmd
	if m.view == VIEW_CHAT {
		m.textarea, cmd = m.textarea.Update(msg)
	}
	return m, cmd
}

func (m *Model) resizeComponents() {
	w := m.width - 4
	h := m.height - 6

	if w > 20 {
		m.viewport.Width = w
		m.textarea.SetWidth(w)
	}
	if h > 5 {
		m.viewport.Height = h
	}
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		if m.view == VIEW_CODE_REVIEW {
			m.view = VIEW_CHAT
			m.textarea.Focus()
		}
		return m, nil
	}

	switch m.view {
	case VIEW_CHAT:
		return m.handleChatKeys(msg)
	case VIEW_CODE_REVIEW:
		return m.handleCodeReviewKeys(msg)
	}

	return m, nil
}

func (m Model) handleChatKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if m.textarea.Value() != "" {
			return m.processInput()
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) handleCodeReviewKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "a", "A":
		// Accept changes
		m.messages = append(m.messages, agentStyle.Render("AGENT")+": Changes accepted")
		m.updateViewport()
		m.view = VIEW_CHAT
		m.textarea.Focus()
		return m, nil
	case "e", "E":
		// Edit with vim
		return m, m.openEditor()
	case "d", "D":
		// Decline changes
		m.messages = append(m.messages, agentStyle.Render("AGENT")+": Changes declined")
		m.updateViewport()
		m.view = VIEW_CHAT
		m.textarea.Focus()
		return m, nil
	}
	return m, nil
}

func (m Model) processInput() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	m.messages = append(m.messages, promptStyle.Render("YOU")+": "+input)
	m.updateViewport()
	m.textarea.Reset()

	// Check if it's an agent command
	if strings.HasPrefix(input, "\\agent ") {
		prompt := strings.TrimSpace(input[7:])
		if prompt != "" {
			m.waiting = true
			m.showingSteps = true
			m.steps = []string{}
			m.currentStep = 0

			m.messages = append(m.messages, agentStyle.Render("AGENT")+": Processing...")
			m.updateViewport()

			return m, m.callAgent(prompt)
		}
	}

	// Handle other commands
	switch input {
	case "clear":
		m.messages = []string{}
		m.updateViewport()
	case "help":
		help := `Commands:
  \\agent {prompt}  - Send prompt to AI agent
  clear           - Clear screen
  help            - Show this help
  
Code Review:
  A - Accept changes
  E - Edit with vim
  D - Decline changes
  ESC - Back to chat`
		m.messages = append(m.messages, agentStyle.Render("HELP")+": "+help)
		m.updateViewport()
	default:
		m.messages = append(m.messages, errorStyle.Render("ERROR")+": Unknown command. Use \\agent {prompt} or 'help'")
		m.updateViewport()
	}

	return m, nil
}

func (m *Model) updateViewport() {
	content := strings.Join(m.messages, "\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// Agent message types
type agentStepMsg struct {
	step string
}

type agentCodeMsg struct {
	filename string
	before   string
	after    string
}

type editorCompleteMsg struct {
	success bool
	message string
}

func (m Model) callAgent(prompt string) tea.Cmd {
	return tea.Sequence(
		// Step 1: Analysis
		tea.Tick(time.Millisecond*800, func(t time.Time) tea.Msg {
			return agentStepMsg{step: "1. Analyzing request..."}
		}),
		// Step 2: Planning
		tea.Tick(time.Millisecond*1600, func(t time.Time) tea.Msg {
			return agentStepMsg{step: "2. Planning solution..."}
		}),
		// Step 3: Implementation
		tea.Tick(time.Millisecond*2400, func(t time.Time) tea.Msg {
			return agentStepMsg{step: "3. Generating code..."}
		}),
		// Final result
		tea.Tick(time.Millisecond*3200, func(t time.Time) tea.Msg {
			// Mock code generation based on prompt
			if strings.Contains(strings.ToLower(prompt), "function") ||
				strings.Contains(strings.ToLower(prompt), "code") ||
				strings.Contains(strings.ToLower(prompt), "fix") {
				return agentCodeMsg{
					filename: "main.go",
					before: `func processData(data string) error {
    if data == "" {
        return nil
    }
    return process(data)
}`,
					after: `func processData(data string) error {
    if data == "" {
        return errors.New("empty data not allowed")
    }
    
    if err := validate(data); err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    return process(data)
}`,
				}
			}
			return agentStepMsg{step: "Complete: " + prompt}
		}),
	)
}

func (m Model) handleAgentStep(msg agentStepMsg) (tea.Model, tea.Cmd) {
	if m.showingSteps {
		// Replace the "Processing..." message with steps
		if len(m.messages) > 0 && strings.Contains(m.messages[len(m.messages)-1], "Processing...") {
			m.messages = m.messages[:len(m.messages)-1]
			m.showingSteps = false
		}
	}

	m.messages = append(m.messages, stepStyle.Render("STEP")+": "+msg.step)
	m.updateViewport()
	return m, nil
}

func (m Model) handleAgentCode(msg agentCodeMsg) (tea.Model, tea.Cmd) {
	m.waiting = false
	m.currentChange = codeChange{
		filename: msg.filename,
		before:   msg.before,
		after:    msg.after,
	}

	m.messages = append(m.messages, agentStyle.Render("AGENT")+": Code changes ready for review")
	m.updateViewport()

	m.view = VIEW_CODE_REVIEW
	m.textarea.Blur()

	return m, nil
}

func (m Model) openEditor() tea.Cmd {
	// Create temporary file with the "after" code
	tmpFile := "/tmp/agent_edit_" + fmt.Sprintf("%d", time.Now().Unix()) + ".go"

	return tea.Sequence(
		tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
			err := os.WriteFile(tmpFile, []byte(m.currentChange.after), 0644)
			if err != nil {
				return editorCompleteMsg{success: false, message: "Failed to create temp file"}
			}
			return nil
		}),
		tea.ExecProcess(exec.Command("vim", tmpFile), func(err error) tea.Msg {
			if err != nil {
				return editorCompleteMsg{success: false, message: "Editor failed: " + err.Error()}
			}

			// Read the modified content
			_, readErr := os.ReadFile(tmpFile)
			if readErr != nil {
				return editorCompleteMsg{success: false, message: "Failed to read edited file"}
			}

			// Clean up temp file
			os.Remove(tmpFile)

			return editorCompleteMsg{success: true, message: "File edited successfully"}
		}),
	)
}

func (m Model) handleEditorComplete(msg editorCompleteMsg) (tea.Model, tea.Cmd) {
	if msg.success {
		m.messages = append(m.messages, agentStyle.Render("AGENT")+": "+msg.message)
	} else {
		m.messages = append(m.messages, errorStyle.Render("ERROR")+": "+msg.message)
	}
	m.updateViewport()

	m.view = VIEW_CHAT
	m.textarea.Focus()
	return m, nil
}

func (m Model) View() string {
	switch m.view {
	case VIEW_CHAT:
		return m.chatView()
	case VIEW_CODE_REVIEW:
		return m.codeReviewView()
	}
	return ""
}

func (m Model) chatView() string {
	status := fmt.Sprintf("Model: %s", m.agentModel)
	if m.agentAPIKey != "" {
		status += " | API: OK"
	} else {
		status += " | API: Not configured"
	}

	if m.waiting {
		status += " | Processing..."
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Width(m.width).Render("SPY AGENT"),
		dimStyle.Render(status),
		"",
		m.viewport.View(),
		"",
		m.textarea.View(),
		"",
		dimStyle.Render("\\agent {prompt} | help | clear | Ctrl+C: quit"))
}

func (m Model) codeReviewView() string {
	beforeSection := lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Width(40).Render("BEFORE"),
		codeBeforeStyle.Width(40).Render(m.currentChange.before))

	afterSection := lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Width(40).Render("AFTER"),
		codeAfterStyle.Width(40).Render(m.currentChange.after))

	diff := lipgloss.JoinHorizontal(lipgloss.Top,
		beforeSection,
		"  ",
		afterSection)

	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Width(m.width).Render("CODE REVIEW: "+m.currentChange.filename),
		"",
		diff,
		"",
		dimStyle.Render("A: Accept | E: Edit | D: Decline | ESC: Back"))
}

func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
