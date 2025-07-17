package cli

import (
	"fmt"
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
	VIEW_CODE_EDIT
)

type codeBlock struct {
	before string
	after  string
	file   string
}

type Model struct {
	width, height int
	view          int

	viewport viewport.Model
	textarea textarea.Model
	messages []string

	modelInput, apiKeyInput textinput.Model
	settingsFocus           int

	codeDiff         codeBlock
	diffViewport     viewport.Model
	editTextarea     textarea.Model
	horizontalScroll int

	agentModel, agentAPIKey string
	waitingForAgent         bool
}

// Color scheme
var (
	primary   = lipgloss.Color("#00D4FF")
	secondary = lipgloss.Color("#B47EFF")
	accent    = lipgloss.Color("#FF7ED4")
	text      = lipgloss.Color("#E0E0E0")
	dim       = lipgloss.Color("#808080")
	surface   = lipgloss.Color("#2a2a2a")
)

// Styles
var (
	baseStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(1, 2)

	logoStyle = lipgloss.NewStyle().
			Foreground(primary).
			Bold(true).
			Align(lipgloss.Center).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primary).
			Padding(1, 2).
			MarginBottom(1)

	chatStyle     = baseStyle.Copy().BorderForeground(primary).Background(surface)
	settingsStyle = baseStyle.Copy().BorderForeground(secondary).Background(surface)
	inputStyle    = baseStyle.Copy().BorderForeground(primary).Background(surface).Padding(0, 1)
	focusStyle    = baseStyle.Copy().BorderForeground(accent).Background(surface).Padding(0, 1)

	userStyle      = lipgloss.NewStyle().Foreground(primary).Bold(true)
	agentStyle     = lipgloss.NewStyle().Foreground(secondary)
	systemStyle    = lipgloss.NewStyle().Foreground(accent).Italic(true)
	statusStyle    = lipgloss.NewStyle().Foreground(dim).Italic(true)
	helpStyle      = lipgloss.NewStyle().Foreground(dim).Align(lipgloss.Center)
	secondaryStyle = lipgloss.NewStyle().Foreground(secondary)

	diffBeforeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#2d1b1b")).
			Foreground(lipgloss.Color("#ffb3b3")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#cc6666")).
			Width(40)

	diffAfterStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#1b2d1b")).
			Foreground(lipgloss.Color("#b3ffb3")).
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#66cc66")).
			Width(40)

	diffHeaderStyle = lipgloss.NewStyle().
			Foreground(secondary).
			Bold(true).
			Align(lipgloss.Center).
			Width(40)
)

const spyLogo = `
  ____  ____  _   _     ____  _____    _    ____   ____ _   _ 
 / ___||  _ \| \ | |   / ___|| ____|  / \  |  _ \ / ___| | | |
 \___ \| |_) |  \| |   \___ \|  _|   / _ \ | |_) | |   | |_| |
  ___) |  __/| |\  |    ___) | |___ / ___ \|  _ <| |___|  _  |
 |____/|_|   |_| \_|   |____/|_____/_/   \_\_| \_\\____|_| |_|
                                                              
        Advanced SWE Agent Terminal v2.0`

func NewModel() Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message here..."
	ta.Focus()
	ta.CharLimit = 2000
	ta.SetWidth(70)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(70, 8)
	vp.SetContent(systemStyle.Render("SYSTEM") + ": Welcome to SPY SEARCH\n\nCommands:\n• Type to chat\n• \\agent {prompt} to call agent\n• Tab for settings\n• Ctrl+C to quit")

	modelInput := textinput.New()
	modelInput.Placeholder = "Ollama"
	modelInput.CharLimit = 50
	modelInput.Width = 25

	apiKeyInput := textinput.New()
	apiKeyInput.Placeholder = "Enter API key"
	apiKeyInput.CharLimit = 100
	apiKeyInput.Width = 40
	apiKeyInput.EchoMode = textinput.EchoPassword

	editTA := textarea.New()
	editTA.SetWidth(70)
	editTA.SetHeight(15)
	editTA.ShowLineNumbers = true

	return Model{
		view:             VIEW_CHAT,
		viewport:         vp,
		textarea:         ta,
		modelInput:       modelInput,
		apiKeyInput:      apiKeyInput,
		diffViewport:     viewport.New(70, 8),
		editTextarea:     editTA,
		horizontalScroll: 0,
		agentModel:       "gpt-4",
	}
}

func (m Model) Init() tea.Cmd {
	return textarea.Blink
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizeComponents()
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	case agentResponseMsg:
		return m.handleAgentResponse(msg)
	}

	return m.updateComponents(msg)
}

func (m *Model) resizeComponents() {
	if m.width > 80 {
		w := m.width - 20
		m.viewport.Width = w
		m.textarea.SetWidth(w)
		m.diffViewport.Width = w
		m.editTextarea.SetWidth(w)
	}
	if m.height > 25 {
		h := m.height - 17
		m.viewport.Height = h
		m.diffViewport.Height = h
		m.editTextarea.SetHeight(h)
	}
}

func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "tab":
		return m.handleTabKey()
	case "esc":
		return m.handleEscKey()
	}

	switch m.view {
	case VIEW_CHAT:
		return m.updateChat(msg)
	case VIEW_SETTINGS:
		return m.updateSettings(msg)
	case VIEW_CODE_DIFF:
		return m.updateCodeDiff(msg)
	case VIEW_CODE_EDIT:
		return m.updateCodeEdit(msg)
	}

	return m, nil
}

func (m Model) handleTabKey() (tea.Model, tea.Cmd) {
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
	return m, nil
}

func (m Model) handleEscKey() (tea.Model, tea.Cmd) {
	if m.view == VIEW_CODE_DIFF || m.view == VIEW_CODE_EDIT {
		m.view = VIEW_CHAT
		m.textarea.Focus()
		m.editTextarea.Blur()
		m.horizontalScroll = 0
	}
	return m, nil
}

func (m Model) updateChat(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "enter" && m.textarea.Value() != "" {
		return m.processUserInput()
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m Model) processUserInput() (tea.Model, tea.Cmd) {
	userMsg := strings.TrimSpace(m.textarea.Value())
	m.messages = append(m.messages, userStyle.Render("YOU")+": "+userMsg)
	m.updateChatViewport()
	m.textarea.Reset()

	if after, ok := strings.CutPrefix(userMsg, "\\agent "); ok {
		m.waitingForAgent = true
		m.messages = append(m.messages, statusStyle.Render("Agent processing..."))
		m.updateChatViewport()
		return m, m.callAgent(strings.TrimSpace(after))
	}

	m.messages = append(m.messages, systemStyle.Render("SYSTEM")+": Use \\agent {prompt} to interact with agent.")
	m.updateChatViewport()
	return m, nil
}

func (m Model) updateSettings(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "up", "down":
		m.settingsFocus = 1 - m.settingsFocus
		if m.settingsFocus == 0 {
			m.apiKeyInput.Blur()
			m.modelInput.Focus()
		} else {
			m.modelInput.Blur()
			m.apiKeyInput.Focus()
		}
	case "ctrl+s":
		return m.saveSettings()
	}

	var cmd tea.Cmd
	if m.settingsFocus == 0 {
		m.modelInput, cmd = m.modelInput.Update(msg)
	} else {
		m.apiKeyInput, cmd = m.apiKeyInput.Update(msg)
	}
	return m, cmd
}

func (m Model) saveSettings() (tea.Model, tea.Cmd) {
	m.agentModel = m.modelInput.Value()
	if m.agentModel == "" {
		m.agentModel = "gpt-4"
	}
	m.agentAPIKey = m.apiKeyInput.Value()
	m.messages = append(m.messages, systemStyle.Render("SYSTEM")+": Configuration saved")
	m.updateChatViewport()
	m.view = VIEW_CHAT
	m.textarea.Focus()
	return m, nil
}

func (m Model) updateCodeDiff(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "left", "h":
		if m.horizontalScroll > 0 {
			m.horizontalScroll--
			m.updateDiffViewport()
		}
	case "right", "l":
		m.horizontalScroll++
		m.updateDiffViewport()
	case "y", "Y":
		m.messages = append(m.messages, agentStyle.Render("AGENT")+": Changes accepted and applied.")
		m.updateChatViewport()
		m.view = VIEW_CHAT
		m.textarea.Focus()
		m.horizontalScroll = 0
	case "e", "E":
		m.editTextarea.SetValue(m.codeDiff.after)
		m.editTextarea.Focus()
		m.view = VIEW_CODE_EDIT
	case "n", "N":
		m.messages = append(m.messages, agentStyle.Render("AGENT")+": Changes rejected.")
		m.updateChatViewport()
		m.view = VIEW_CHAT
		m.textarea.Focus()
		m.horizontalScroll = 0
	}

	var cmd tea.Cmd
	m.diffViewport, cmd = m.diffViewport.Update(msg)
	return m, cmd
}

func (m Model) updateCodeEdit(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		m.codeDiff.after = m.editTextarea.Value()
		m.updateDiffViewport()
		m.view = VIEW_CODE_DIFF
		m.editTextarea.Blur()
	case "j":
		if !m.editTextarea.Focused() {
			m.editTextarea.CursorDown()
			return m, nil
		}
	case "k":
		if !m.editTextarea.Focused() {
			m.editTextarea.CursorUp()
			return m, nil
		}
	case "h":
		if !m.editTextarea.Focused() {
			m.editTextarea.KeyMap.CharacterBackward.SetEnabled(true)
			m.editTextarea, _ = m.editTextarea.Update(tea.KeyMsg{Type: tea.KeyLeft})
			return m, nil
		}
	case "l":
		if !m.editTextarea.Focused() {
			m.editTextarea.KeyMap.CharacterForward.SetEnabled(true)
			m.editTextarea, _ = m.editTextarea.Update(tea.KeyMsg{Type: tea.KeyRight})
			return m, nil
		}
	case "i":
		if !m.editTextarea.Focused() {
			m.editTextarea.Focus()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.editTextarea, cmd = m.editTextarea.Update(msg)
	return m, cmd
}

func (m Model) updateComponents(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	viewport, cmd := m.viewport.Update(msg)
	m.viewport = viewport
	cmds = append(cmds, cmd)

	textarea, cmd := m.textarea.Update(msg)
	m.textarea = textarea
	cmds = append(cmds, cmd)

	diffViewport, cmd := m.diffViewport.Update(msg)
	m.diffViewport = diffViewport
	cmds = append(cmds, cmd)

	editTextarea, cmd := m.editTextarea.Update(msg)
	m.editTextarea = editTextarea
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m *Model) updateChatViewport() {
	m.viewport.SetContent(strings.Join(m.messages, "\n\n"))
	m.viewport.GotoBottom()
}

type agentResponseMsg struct {
	message    string
	isCodeDiff bool
	codeDiff   codeBlock
}

func (m Model) callAgent(prompt string) tea.Cmd {
	return tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
		if strings.Contains(strings.ToLower(prompt), "code") || strings.Contains(strings.ToLower(prompt), "fix") {
			return agentResponseMsg{
				message:    "Code analysis complete. Review changes:",
				isCodeDiff: true,
				codeDiff: codeBlock{
					file:   "main.go",
					before: "func processData(data string) error {\n    if data == \"\" {\n        return nil\n    }\n    return process(data)\n}",
					after:  "func processData(data string) error {\n    if data == \"\" {\n        return errors.New(\"empty data\")\n    }\n    if err := validate(data); err != nil {\n        return err\n    }\n    return process(data)\n}",
				},
			}
		}
		return agentResponseMsg{
			message: fmt.Sprintf("Processing: %s\n\nAnalysis complete. Ready for implementation.", prompt),
		}
	})
}

func (m Model) handleAgentResponse(msg agentResponseMsg) (tea.Model, tea.Cmd) {
	m.waitingForAgent = false
	m.messages = append(m.messages, agentStyle.Render("AGENT")+": "+msg.message)
	m.updateChatViewport()

	if msg.isCodeDiff {
		m.codeDiff = msg.codeDiff
		m.view = VIEW_CODE_DIFF
		m.horizontalScroll = 0
		m.updateDiffViewport()
	}

	return m, nil
}

func (m *Model) updateDiffViewport() {
	// Split code into lines for horizontal scrolling
	beforeLines := strings.Split(m.codeDiff.before, "\n")
	afterLines := strings.Split(m.codeDiff.after, "\n")

	// Calculate visible lines based on scroll position
	viewWidth := 35 // Adjust based on available space

	var visibleBefore, visibleAfter []string

	for _, line := range beforeLines {
		if len(line) > m.horizontalScroll {
			if len(line) > m.horizontalScroll+viewWidth {
				visibleBefore = append(visibleBefore, line[m.horizontalScroll:m.horizontalScroll+viewWidth])
			} else {
				visibleBefore = append(visibleBefore, line[m.horizontalScroll:])
			}
		} else {
			visibleBefore = append(visibleBefore, "")
		}
	}

	for _, line := range afterLines {
		if len(line) > m.horizontalScroll {
			if len(line) > m.horizontalScroll+viewWidth {
				visibleAfter = append(visibleAfter, line[m.horizontalScroll:m.horizontalScroll+viewWidth])
			} else {
				visibleAfter = append(visibleAfter, line[m.horizontalScroll:])
			}
		} else {
			visibleAfter = append(visibleAfter, "")
		}
	}

	beforeContent := strings.Join(visibleBefore, "\n")
	afterContent := strings.Join(visibleAfter, "\n")

	beforeSection := lipgloss.JoinVertical(lipgloss.Left,
		diffHeaderStyle.Render("BEFORE"),
		diffBeforeStyle.Render(beforeContent))

	afterSection := lipgloss.JoinVertical(lipgloss.Left,
		diffHeaderStyle.Render("AFTER"),
		diffAfterStyle.Render(afterContent))

	content := lipgloss.JoinVertical(lipgloss.Left,
		fmt.Sprintf("File: %s (scroll: %d)", m.codeDiff.file, m.horizontalScroll),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, beforeSection, "  ", afterSection),
		"",
		helpStyle.Render("← → / h l: Scroll | Y: Accept | E: Edit | N: Reject | ESC: Cancel"))

	m.diffViewport.SetContent(content)
}

func (m Model) View() string {
	// Calculate logo width based on terminal width
	logoWidth := min(m.width-6, 70)
	if logoWidth < 50 {
		logoWidth = 50
	}

	header := logoStyle.Width(logoWidth).Render(spyLogo)

	switch m.view {
	case VIEW_CHAT:
		return m.chatView(header)
	case VIEW_SETTINGS:
		return m.settingsView(header)
	case VIEW_CODE_DIFF:
		return m.codeDiffView(header)
	case VIEW_CODE_EDIT:
		return m.codeEditView(header)
	}

	return header
}

func (m Model) chatView(header string) string {
	apiStatus := "Not configured"
	if m.agentAPIKey != "" {
		apiStatus = "Configured"
	}

	status := statusStyle.Render(fmt.Sprintf("Model: %s | API: %s", m.agentModel, apiStatus))

	if m.waitingForAgent {
		status = statusStyle.Render("Agent processing...")
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		status,
		"",
		chatStyle.Width(m.width-6).Render(m.viewport.View()),
		"",
		inputStyle.Width(m.width-6).Render(m.textarea.View()),
		"",
		helpStyle.Render("Tab: Settings | Ctrl+C: Quit | \\agent {prompt}: Call Agent"))
}

func (m Model) settingsView(header string) string {
	modelSection := "Model:\n"
	if m.settingsFocus == 0 {
		modelSection += focusStyle.Render(m.modelInput.View())
	} else {
		modelSection += m.modelInput.View()
	}

	apiSection := "\nAPI Key:\n"
	if m.settingsFocus == 1 {
		apiSection += focusStyle.Render(m.apiKeyInput.View())
	} else {
		apiSection += m.apiKeyInput.View()
	}

	settings := modelSection + apiSection + "\n\n" + helpStyle.Render("Enter/Up/Down: Navigate | Ctrl+S: Save | Tab: Chat")

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		settingsStyle.Width(m.width-6).Render(settings))
}

func (m Model) codeDiffView(header string) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		secondaryStyle.Render("CODE REVIEW"),
		"",
		m.diffViewport.View())
}

func (m Model) codeEditView(header string) string {
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		secondaryStyle.Render("CODE EDITOR"),
		"",
		inputStyle.Width(m.width-6).Render(m.editTextarea.View()),
		"",
		helpStyle.Render("Ctrl+S: Save | ESC: Cancel | vim keys: hjkl | i: Insert mode"))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
