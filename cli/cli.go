package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"spysearch/agent"
	"spysearch/models"
	"spysearch/tools"

	"encoding/json"
	"io/ioutil"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	VIEW_CHAT = iota
	VIEW_CODE_REVIEW
	VIEW_SETTINGS
)

type codeChange struct {
	filename string
	before   string
	after    string
}

type settings struct {
	model    string
	apiKey   string
	provider string
	workDir  string
}

type Model struct {
	width, height int
	view          int

	viewport viewport.Model
	textarea textarea.Model
	messages []string

	// Settings
	settings     settings
	waiting      bool
	settingsMode int // 0: model, 1: provider, 2: apikey

	// Code review state
	currentChange codeChange
	showingSteps  bool
	steps         []string
	currentStep   int

	editingSetting bool
	editBuffer     string
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
	purple = lipgloss.Color("#ff00ff")
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

	logoStyle = lipgloss.NewStyle().
			Foreground(purple).
			Bold(true)

	settingsStyle = lipgloss.NewStyle().
			Foreground(white).
			Background(lipgloss.Color("#1a1a1a")).
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(purple)

	selectedStyle = lipgloss.NewStyle().
			Foreground(black).
			Background(cyan).
			Padding(0, 1)
)

const spyLogo = `
  ███████╗██████╗ ██╗   ██╗
  ██╔════╝██╔══██╗╚██╗ ██╔╝
  ███████╗██████╔╝ ╚████╔╝ 
  ╚════██║██╔═══╝   ╚██╔╝  
  ███████║██║        ██║   
  ╚══════╝╚═╝        ╚═╝   
   A G E N T   S E A R C H
`

func loadConfig() settings {
	cfg := settings{
		model:    "gpt-4",
		apiKey:   os.Getenv("OPENAI_API_KEY"),
		provider: "openai",
		workDir:  "",
	}
	data, err := ioutil.ReadFile("config.json")
	if err == nil {
		_ = json.Unmarshal(data, &cfg)
	}
	return cfg
}

func saveConfig(cfg settings) {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	_ = ioutil.WriteFile("config.json", data, 0644)
}

func NewModel() Model {
	ta := textarea.New()
	ta.Placeholder = "Enter message or command..."
	ta.Focus()
	ta.CharLimit = 4000
	ta.SetWidth(80)
	ta.SetHeight(2)
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	vp := viewport.New(80, 20)

	welcome := logoStyle.Render(spyLogo) + "\n" +
		agentStyle.Render("AGENT") + ": Ready for chat or commands\n" +
		dimStyle.Render("Commands: \\spyagent {prompt} | \\settings | help | clear")

	defaultSettings := loadConfig()

	model := Model{
		view:         VIEW_CHAT,
		viewport:     vp,
		textarea:     ta,
		messages:     []string{welcome},
		settings:     defaultSettings,
		settingsMode: 0,
	}

	// Initialize viewport with welcome message
	model.updateViewport()

	return model
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
	case agentStepMsg:
		return m.handleAgentStep(msg)
	case agentCodeMsg:
		return m.handleAgentCode(msg)
	case agentResponseMsg:
		return m.handleAgentResponse(msg)
	case editorCompleteMsg:
		return m.handleEditorComplete(msg)
	case runSpyAgentMsg:
		m.waiting = false
		m.messages = append(m.messages, agentStyle.Render(msg.result))
		m.updateViewport()
		return m, nil
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
		if m.view == VIEW_CODE_REVIEW || m.view == VIEW_SETTINGS {
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
	case VIEW_SETTINGS:
		return m.handleSettingsKeys(msg)
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

func (m Model) handleSettingsKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if !m.editingSetting && m.settingsMode > 0 {
			m.settingsMode--
		}
	case "down", "j":
		if !m.editingSetting && m.settingsMode < 3 {
			m.settingsMode++
		}
	case "enter":
		if !m.editingSetting {
			// Start editing
			m.editingSetting = true
			fields := []string{"model", "provider", "apiKey", "workDir"}
			var val string
			switch m.settingsMode {
			case 0:
				val = m.settings.model
			case 1:
				val = m.settings.provider
			case 2:
				val = m.settings.apiKey
			case 3:
				val = m.settings.workDir
			}
			m.editBuffer = val
			m.messages = append(m.messages, dimStyle.Render("Editing: "+fields[m.settingsMode]+" (type and Ctrl+S to save, Esc to cancel)"))
			m.updateViewport()
			return m, nil
		}
	case "ctrl+s":
		if m.editingSetting {
			// Save edit
			switch m.settingsMode {
			case 0:
				m.settings.model = m.editBuffer
			case 1:
				m.settings.provider = m.editBuffer
			case 2:
				m.settings.apiKey = m.editBuffer
			case 3:
				m.settings.workDir = m.editBuffer
			}
			m.editingSetting = false
			m.editBuffer = ""
			saveConfig(m.settings)
			m.messages = append(m.messages, agentStyle.Render("SETTINGS updated and saved."))
			m.updateViewport()
			return m, nil
		}
	case "esc":
		if m.editingSetting {
			m.editingSetting = false
			m.editBuffer = ""
			m.messages = append(m.messages, dimStyle.Render("Edit cancelled."))
			m.updateViewport()
			return m, nil
		}
		m.view = VIEW_CHAT
		m.textarea.Focus()
		return m, nil
	default:
		if m.editingSetting {
			if msg.Type == tea.KeyRunes {
				m.editBuffer += msg.String()
			} else if msg.Type == tea.KeyBackspace && len(m.editBuffer) > 0 {
				m.editBuffer = m.editBuffer[:len(m.editBuffer)-1]
			}
			m.updateViewport()
			return m, nil
		}
	}
	return m, nil
}

func (m Model) processInput() (tea.Model, tea.Cmd) {
	input := strings.TrimSpace(m.textarea.Value())
	m.messages = append(m.messages, promptStyle.Render("YOU")+": "+input)
	m.updateViewport()
	m.textarea.Reset()

	// Check for commands
	if strings.HasPrefix(input, "\\") {
		return m.handleCommand(input)
	}

	// Normal chat - send to agent
	m.waiting = true
	m.messages = append(m.messages, agentStyle.Render("AGENT")+": Thinking...")
	m.updateViewport()
	return m, m.callAgentChat(input)
}

func (m Model) handleCommand(input string) (tea.Model, tea.Cmd) {
	parts := strings.SplitN(input, " ", 2)
	command := parts[0]

	switch command {
	case "\\spyagent":
		if len(parts) > 1 {
			prompt := strings.TrimSpace(parts[1])
			if prompt != "" {
				ag := &agent.SpyAgent{
					Tools: []tools.Tool{
						tools.NewDoneTool().Tool,
						tools.NewModifierTool().Tool,
						tools.NewBashTool().Tool,
						tools.NewThinkingTool().Tool,
					},
					Steps: 5,
					Mmeory: []string{},
					Model:  models.OllamaClient{},
					WorkDir: m.settings.workDir,
				}
				m.waiting = true
				m.messages = append(m.messages, agentStyle.Render("SPY AGENT")+": Starting autonomous reasoning...")
				m.updateViewport()
				return m, func() tea.Msg {
					return runSpyAgentMsg{result: runSpyAgentWithCallback(ag, prompt, func(msg string, review *agent.CodeReviewMsg) {
						if review != nil {
							// Show code diff and prompt user
							m.currentChange = codeChange{
								filename: "(modifier)",
								before:   review.Before,
								after:    review.After,
							}
							m.messages = append(m.messages, agentStyle.Render("AGENT")+": Modifier tool result. Accept (A), Edit (E), or Decline (D)?")
							m.view = VIEW_CODE_REVIEW
							m.updateViewport()
							return
						}
						if msg != "" {
							m.messages = append(m.messages, msg)
							m.updateViewport()
						}
					})}
				}
			}
		}
		m.messages = append(m.messages, errorStyle.Render("ERROR")+": Usage: \\spyagent {prompt}")
		m.updateViewport()
		return m, nil
	case "\\settings":
		m.view = VIEW_SETTINGS
		m.textarea.Blur()
		return m, nil
	case "\\clear":
		m.messages = []string{}
		m.updateViewport()
	case "\\help":
		help := `Commands:
  \\spyagent {prompt}   - Run the autonomous agent on your prompt
  \\settings           - Configure model, API, provider, and working directory
  \\clear              - Clear screen
  \\help               - Show this help

Settings:
  Up/Down - Navigate options
  Enter   - Save current value
  ESC     - Back to chat`
		m.messages = append(m.messages, agentStyle.Render("HELP")+": "+help)
		m.updateViewport()
	default:
		m.messages = append(m.messages, errorStyle.Render("ERROR")+": Unknown command: "+command)
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

type agentResponseMsg struct {
	response string
}

type editorCompleteMsg struct {
	success bool
	message string
}

func (m Model) callAgentChat(message string) tea.Cmd {
	return func() tea.Msg {
		ollama := &models.OllamaClient{}
		resp, err := ollama.Completion(message, []tools.Tool{})
		if err != nil {
			return agentResponseMsg{response: "[Error] " + err.Error()}
		}
		// Simulate streaming by word
		for _, word := range strings.Split(resp.Content, " ") {
			m.messages = append(m.messages, "[LLM] "+word)
			m.updateViewport()
		}
		return agentResponseMsg{response: resp.Content}
	}
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
			return agentResponseMsg{response: "Task completed: " + prompt}
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

func (m Model) handleAgentResponse(msg agentResponseMsg) (tea.Model, tea.Cmd) {
	m.waiting = false

	// Remove "Thinking..." message
	if len(m.messages) > 0 && strings.Contains(m.messages[len(m.messages)-1], "Thinking...") {
		m.messages = m.messages[:len(m.messages)-1]
	}

	m.messages = append(m.messages, agentStyle.Render("AGENT")+": "+msg.response)
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
	case VIEW_SETTINGS:
		return m.settingsView()
	}
	return ""
}

func (m Model) chatView() string {
	status := fmt.Sprintf("Model: %s | Provider: %s", m.settings.model, m.settings.provider)
	if m.settings.apiKey != "" {
		status += " | API: OK"
	} else {
		status += " | API: Not configured"
	}

	if m.waiting {
		status += " | Processing..."
	}

	w := m.width - 4
	h := m.height - 8
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m.viewport.Width = w
	m.viewport.Height = h
	m.viewport.SetContent(strings.Join(m.messages, "\n"))
	m.viewport.YOffset = 0
	m.viewport.GotoBottom()
	// Wrapping is handled by lipgloss, no SetWrap method

	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Width(m.width).Render("SPY AGENT SEARCH"),
		dimStyle.Render(status),
		"",
		m.viewport.View(),
		"",
		m.textarea.View(),
		"",
		dimStyle.Render("Chat normally or use: \\spyagent {prompt} | \\settings | \\help | Ctrl+C: quit"))
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

	w := m.width - 4
	h := m.height - 8
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	m.viewport.Width = w
	m.viewport.Height = h
	m.viewport.SetContent(diff)
	m.viewport.YOffset = 0
	m.viewport.GotoBottom()
	// Wrapping is handled by lipgloss, no SetWrap method

	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Width(m.width).Render("CODE REVIEW: "+m.currentChange.filename),
		"",
		diff,
		"",
		dimStyle.Render("A: Accept | E: Edit | D: Decline | ESC: Back"))
}

func (m Model) settingsView() string {
	options := []string{
		fmt.Sprintf("Model: %s", m.settings.model),
		fmt.Sprintf("Provider: %s", m.settings.provider),
		fmt.Sprintf("API Key: %s", func() string {
			if m.settings.apiKey == "" {
				return "Not set"
			}
			return "***" + m.settings.apiKey[len(m.settings.apiKey)-4:]
		}()),
		fmt.Sprintf("WorkDir: %s", m.settings.workDir),
	}

	var renderedOptions []string
	for i, option := range options {
		if i == m.settingsMode {
			if m.editingSetting {
				renderedOptions = append(renderedOptions, selectedStyle.Render("> "+option+" ["+m.editBuffer+"]"))
			} else {
				renderedOptions = append(renderedOptions, selectedStyle.Render("> "+option))
			}
		} else {
			renderedOptions = append(renderedOptions, "  "+option)
		}
	}

	settingsContent := strings.Join(renderedOptions, "\n")

	return lipgloss.JoinVertical(lipgloss.Left,
		headerStyle.Width(m.width).Render("SETTINGS"),
		"",
		settingsStyle.Width(m.width-4).Render(settingsContent),
		"",
		dimStyle.Render("Up/Down: Navigate | Enter: Edit | Ctrl+S: Save | ESC: Cancel/Back"))
}

func Run() error {
	p := tea.NewProgram(NewModel(), tea.WithAltScreen())
	_, err := p.Run()
	return err
}

// Message type for agent run completion

type runSpyAgentMsg struct {
	result string
}

// Helper to run the agent and call a callback for each step
func runSpyAgentWithCallback(ag *agent.SpyAgent, prompt string, onStep func(string, *agent.CodeReviewMsg)) string {
	done := make(chan struct{})
	go func() {
		ag.RunWithCallback(prompt, func(msg interface{}) {
			switch v := msg.(type) {
			case string:
				if strings.HasPrefix(v, "[USING TOOL]") {
					onStep(v, nil)
					return
				}
				onStep(v, nil)
			case agent.CodeReviewMsg:
				onStep("", &v)
			}
		})
		done <- struct{}{}
	}()
	<-done
	return "[SPY AGENT] Finished."
}
