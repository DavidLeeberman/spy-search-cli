package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	VIEW_CHAT = iota
	VIEW_SETTINGS
	VIEW_CODE_DIFF
)

type codeBlock struct {
	before string
	after  string
	file   string
}

type model struct {
	width  int
	height int
	view   int

	// Chat components
	viewport viewport.Model
	textarea textarea.Model
	messages []string

	// Settings components
	modelInput    textinput.Model
	apiKeyInput   textinput.Model
	settingsFocus int

	// Code diff components
	codeDiff     codeBlock
	diffViewport viewport.Model

	// Agent state
	agentModel      string
	agentAPIKey     string
	waitingForAgent bool
}

var (
	// Color palette from the image
	primaryColor   = lipgloss.Color("#00D4FF") // Cyan
	secondaryColor = lipgloss.Color("#B47EFF") // Purple
	accentColor    = lipgloss.Color("#FF7ED4") // Pink
	textColor      = lipgloss.Color("#E0E0E0") // Light gray
	dimColor       = lipgloss.Color("#808080") // Gray
	bgColor        = lipgloss.Color("#1a1a1a") // Dark bg
	surfaceColor   = lipgloss.Color("#2a2a2a") // Surface

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Background(bgColor).
			Padding(0, 2).
			MarginBottom(1)

	chatContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Background(surfaceColor).
				Padding(1, 2)

	userMessageStyle = lipgloss.NewStyle().
				Foreground(primaryColor).
				Bold(true)

	agentMessageStyle = lipgloss.NewStyle().
				Foreground(secondaryColor).
				Bold(false)

	systemMessageStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Italic(true)

	errorMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF6B6B")).
				Bold(true)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Background(surfaceColor).
			Padding(0, 1)

	settingsContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(secondaryColor).
				Background(surfaceColor).
				Padding(2, 3)

	settingsFocusedStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor).
				Background(surfaceColor).
				Padding(0, 1)

	codeBlockStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#0d1117")).
			Foreground(textColor).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(dimColor)

	diffBeforeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2d1b1b")).
			Foreground(lipgloss.Color("#ffb3b3")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#cc6666"))

	diffAfterStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1b2d1b")).
			Foreground(lipgloss.Color("#b3ffb3")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#66cc66"))

	statusStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Italic(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(dimColor).
			Align(lipgloss.Center)
)

const spyLogo = `
    ███████╗██████╗ ██╗   ██╗    ███████╗███████╗ █████╗ ██████╗  ██████╗██╗  ██╗
    ██╔════╝██╔══██╗╚██╗ ██╔╝    ██╔════╝██╔════╝██╔══██╗██╔══██╗██╔════╝██║  ██║
    ███████╗██████╔╝ ╚████╔╝     ███████╗█████╗  ███████║██████╔╝██║     ███████║
    ╚════██║██╔═══╝   ╚██╔╝      ╚════██║██╔══╝  ██╔══██║██╔══██╗██║     ██╔══██║
    ███████║██║        ██║       ███████║███████╗██║  ██║██║  ██║╚██████╗██║  ██║
    ╚══════╝╚═╝        ╚═╝       ╚══════╝╚══════╝╚═╝  ╚═╝╚═╝  ╚═╝ ╚═════╝╚═╝  ╚═╝
    
    Advanced SWE Agent Terminal Interface v2.0
    [CLASSIFIED] - Authorized Personnel Only
`

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.Focus()
	ta.CharLimit = 2000
	ta.SetWidth(70)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false) // Disable newline on enter

	vp := viewport.New(70, 15)
	welcomeMsg := systemMessageStyle.Render("SYSTEM") + ": Welcome to SPY SEARCH - Advanced SWE Agent Terminal\n\n"
	welcomeMsg += "Available commands:\n"
	welcomeMsg += "• Type naturally to chat\n"
	welcomeMsg += "• Use \\agent {prompt} to call the agent\n"
	welcomeMsg += "• Press Tab for settings\n"
	welcomeMsg += "• Press Ctrl+C to quit\n\n"
	vp.SetContent(welcomeMsg)

	// Settings inputs
	modelInput := textinput.New()
	modelInput.Placeholder = "gpt-4"
	modelInput.CharLimit = 50
	modelInput.Width = 25

	apiKeyInput := textinput.New()
	apiKeyInput.Placeholder = "Enter your API key"
	apiKeyInput.CharLimit = 100
	apiKeyInput.Width = 40
	apiKeyInput.EchoMode = textinput.EchoPassword

	// Code diff viewport
	diffVp := viewport.New(70, 15)

	return model{
		view:          VIEW_CHAT,
		viewport:      vp,
		textarea:      ta,
		messages:      []string{},
		modelInput:    modelInput,
		apiKeyInput:   apiKeyInput,
		settingsFocus: 0,
		diffViewport:  diffVp,
		agentModel:    "gpt-4",
		agentAPIKey:   "",
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		taCmd tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate responsive dimensions
		if m.width > 80 {
			newWidth := m.width - 20
			m.viewport.Width = newWidth
			m.textarea.SetWidth(newWidth)
			m.diffViewport.Width = newWidth
		}

		if m.height > 25 {
			newHeight := m.height - 15
			m.viewport.Height = newHeight
			m.diffViewport.Height = newHeight
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			if m.view == VIEW_CHAT {
				m.view = VIEW_SETTINGS
				m.modelInput.Focus()
				m.textarea.Blur()
			} else if m.view == VIEW_SETTINGS {
				m.view = VIEW_CHAT
				m.textarea.Focus()
				m.modelInput.Blur()
				m.apiKeyInput.Blur()
			}
		case "esc":
			if m.view == VIEW_CODE_DIFF {
				m.view = VIEW_CHAT
				m.textarea.Focus()
			}
		}

		// Handle different views
		switch m.view {
		case VIEW_CHAT:
			return m.updateChat(msg)
		case VIEW_SETTINGS:
			return m.updateSettings(msg)
		case VIEW_CODE_DIFF:
			return m.updateCodeDiff(msg)
		}

	case agentResponseMsg:
		return m.handleAgentResponse(msg)
	}

	m.viewport, vpCmd = m.viewport.Update(msg)
	m.textarea, taCmd = m.textarea.Update(msg)
	m.diffViewport, _ = m.diffViewport.Update(msg)

	return m, tea.Batch(tiCmd, vpCmd, taCmd)
}

func (m model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "enter":
		if m.textarea.Value() != "" {
			userMsg := strings.TrimSpace(m.textarea.Value())
			m.messages = append(m.messages, userMessageStyle.Render("YOU")+": "+userMsg)
			m.updateChatViewport()
			m.textarea.Reset()

			// Check if it's an agent command
			if strings.HasPrefix(userMsg, "\\agent ") {
				prompt := strings.TrimSpace(strings.TrimPrefix(userMsg, "\\agent "))
				m.waitingForAgent = true
				m.messages = append(m.messages, statusStyle.Render("Agent is processing your request..."))
				m.updateChatViewport()
				return m, m.callAgent(prompt)
			} else {
				// Regular chat response
				// TODO messgae handling here
				m.messages = append(m.messages, systemMessageStyle.Render("SYSTEM")+": Message received. Use \\agent {prompt} to interact with the agent.")
				m.updateChatViewport()
			}
		}
	}

	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "enter", "up", "down":
		if m.settingsFocus == 0 {
			m.settingsFocus = 1
			m.modelInput.Blur()
			m.apiKeyInput.Focus()
		} else {
			m.settingsFocus = 0
			m.apiKeyInput.Blur()
			m.modelInput.Focus()
		}
	case "ctrl+s":
		m.agentModel = m.modelInput.Value()
		if m.agentModel == "" {
			m.agentModel = "gpt-4"
		}
		m.agentAPIKey = m.apiKeyInput.Value()
		m.messages = append(m.messages, systemMessageStyle.Render("SYSTEM")+": Configuration saved successfully")
		m.updateChatViewport()
		m.view = VIEW_CHAT
		m.textarea.Focus()
	}

	if m.settingsFocus == 0 {
		m.modelInput, cmd = m.modelInput.Update(msg)
	} else {
		m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	}

	return m, cmd
}

func (m model) updateCodeDiff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.messages = append(m.messages, agentMessageStyle.Render("AGENT")+": Code changes have been accepted and applied.")
		m.updateChatViewport()
		m.view = VIEW_CHAT
		m.textarea.Focus()
	case "e", "E":
		m.messages = append(m.messages, agentMessageStyle.Render("AGENT")+": Please provide your modifications or feedback.")
		m.updateChatViewport()
		m.view = VIEW_CHAT
		m.textarea.Focus()
	case "n", "N":
		m.messages = append(m.messages, agentMessageStyle.Render("AGENT")+": Code changes have been rejected.")
		m.updateChatViewport()
		m.view = VIEW_CHAT
		m.textarea.Focus()
	}

	var cmd tea.Cmd
	m.diffViewport, cmd = m.diffViewport.Update(msg)
	return m, cmd
}

func (m *model) updateChatViewport() {
	content := strings.Join(m.messages, "\n\n")
	m.viewport.SetContent(content)
	m.viewport.GotoBottom()
}

// Agent interaction simulation
type agentResponseMsg struct {
	message    string
	isCodeDiff bool
	codeDiff   codeBlock
}

func (m model) callAgent(prompt string) tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		// Simulate different types of agent responses
		if strings.Contains(strings.ToLower(prompt), "code") || strings.Contains(strings.ToLower(prompt), "fix") || strings.Contains(strings.ToLower(prompt), "change") {
			return agentResponseMsg{
				message:    "I've analyzed your request and found code that needs modification. Please review the changes below:",
				isCodeDiff: true,
				codeDiff: codeBlock{
					file:   "main.go",
					before: "func processData(data string) error {\n    if data == \"\" {\n        return nil\n    }\n    // Basic processing\n    return process(data)\n}",
					after:  "func processData(data string) error {\n    if data == \"\" {\n        return errors.New(\"empty data\")\n    }\n    // Enhanced processing with validation\n    if err := validate(data); err != nil {\n        return err\n    }\n    return process(data)\n}",
				},
			}
		}

		return agentResponseMsg{
			message:    fmt.Sprintf("I understand you want me to: %s\n\nI'm analyzing the requirements and will provide a comprehensive solution. This involves:\n\n1. Code analysis and pattern recognition\n2. Implementation planning\n3. Testing strategy\n4. Documentation updates\n\nWould you like me to proceed with the implementation?", prompt),
			isCodeDiff: false,
		}
	})
}

func (m model) handleAgentResponse(msg agentResponseMsg) (tea.Model, tea.Cmd) {
	m.waitingForAgent = false
	m.messages = append(m.messages, agentMessageStyle.Render("AGENT")+": "+msg.message)
	m.updateChatViewport()

	if msg.isCodeDiff {
		m.codeDiff = msg.codeDiff
		m.view = VIEW_CODE_DIFF
		m.updateDiffViewport()
	}

	return m, nil
}

func (m *model) updateDiffViewport() {
	fileStyle := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true)
	content := "File: " + fileStyle.Render(m.codeDiff.file) + "\n\n"
	content += "BEFORE:\n"
	content += diffBeforeStyle.Render(m.codeDiff.before)
	content += "\n\nAFTER:\n"
	content += diffAfterStyle.Render(m.codeDiff.after)
	content += "\n\n" + helpStyle.Render("Y = Accept | E = Edit | N = Reject | ESC = Cancel")

	m.diffViewport.SetContent(content)
}

func (m model) View() string {
	header := titleStyle.Render(spyLogo)

	switch m.view {
	case VIEW_CHAT:
		return m.chatView(header)
	case VIEW_SETTINGS:
		return m.settingsView(header)
	case VIEW_CODE_DIFF:
		return m.codeDiffView(header)
	}

	return header
}

func (m model) chatView(header string) string {
	status := statusStyle.Render(fmt.Sprintf("Model: %s | API Key: %s",
		m.agentModel,
		func() string {
			if m.agentAPIKey == "" {
				return "Not configured"
			}
			return "Configured"
		}()))

	if m.waitingForAgent {
		status = statusStyle.Render("Agent is processing...")
	}

	chatContent := chatContainerStyle.Render(m.viewport.View())
	inputArea := inputStyle.Render(m.textarea.View())

	help := helpStyle.Render("Tab: Settings | Ctrl+C: Quit | \\agent {prompt}: Call Agent")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		status,
		"",
		chatContent,
		"",
		inputArea,
		"",
		help,
	)
}

func (m model) settingsView(header string) string {
	modelSection := "Model Configuration\n"
	if m.settingsFocus == 0 {
		modelSection += settingsFocusedStyle.Render(m.modelInput.View())
	} else {
		modelSection += m.modelInput.View()
	}

	apiSection := "\nAPI Key Configuration\n"
	if m.settingsFocus == 1 {
		apiSection += settingsFocusedStyle.Render(m.apiKeyInput.View())
	} else {
		apiSection += m.apiKeyInput.View()
	}

	settings := modelSection + apiSection + "\n\n" + helpStyle.Render("Enter/Up/Down: Navigate | Ctrl+S: Save | Tab: Return to Chat")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		settingsContainerStyle.Render(settings),
	)
}

func (m model) codeDiffView(header string) string {
	titleStyle := lipgloss.NewStyle().Foreground(secondaryColor).Bold(true)
	title := titleStyle.Render("CODE REVIEW")

	return lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		title,
		"",
		m.diffViewport.View(),
	)
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
