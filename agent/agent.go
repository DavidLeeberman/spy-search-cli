package agent

import (
	"bytes"
	"fmt"
	"os/exec"
	"spysearch/log"
	"spysearch/models"
	"spysearch/tools"
)

// this is an agent package
type Agent struct {
	Tools  []tools.Tool // a list of tool
	Steps  int          // number of step allow the agent to run
	Mmeory []string     // save the memory
	Model  models.OllamaClient // Exported for CLI access
	WorkDir string      // Working directory for tool execution
}

// all agent need a run function
type AgentInterface interface {
	Run(p string)
}

// we call our agent spy agent
type SpyAgent Agent

type CodeReviewMsg struct {
	Before string
	After  string
	Desc   string
}

// Helper to get a tool by name
func (s *SpyAgent) getTool(name string) *tools.Tool {
	for i, tool := range s.Tools {
		if tool.ToolFunction.Name == name {
			return &s.Tools[i]
		}
	}
	return nil
}

// Helper to execute a tool with working directory support
func (s *SpyAgent) executeTool(tool *tools.Tool, args map[string]any) (result tools.ToolExecutionResult, err error) {
	// Pass workDir in args if tool supports it
	if s.WorkDir != "" {
		args["workDir"] = s.WorkDir
	}
	return tool.Execute(args)
}

// Enhanced RunWithCallback: always use all tools, think before each step, log tool usage, limit to 5 steps, stream LLM (simulate)
func (s *SpyAgent) RunWithCallback(p string, onStep func(interface{})) {
	if s.Mmeory == nil {
		s.Mmeory = []string{}
	}
	userMsg := p
	maxSteps := 5
	steps := 0
	for steps < maxSteps {
		steps++
		// 1. Think before acting
		thinkingTool := s.getTool("thinking")
		if thinkingTool != nil {
			thought, _ := thinkingTool.Execute(map[string]any{
				"thinkingstep": steps,
				"rethink":      false,
				"content":      fmt.Sprintf("Step %d: Considering next action for: %s", steps, userMsg),
				"summary":      "",
			})
			onStep("[THINKING] " + thought.Result)
			log.LogToolCall("thinking", map[string]any{
				"thinkingstep": steps,
				"rethink":      false,
				"content":      fmt.Sprintf("Step %d: Considering next action for: %s", steps, userMsg),
				"summary":      "",
			}, thought.Result)
		}

		// 2. Send to LLM (simulate streaming by splitting response)
		resp, err := s.Model.Completion(userMsg, s.Tools)
		if err != nil {
			onStep("[Agent Error] " + err.Error())
			log.LogEvent("agent_error", err.Error())
			return
		}
		s.Mmeory = append(s.Mmeory, "[LLM] "+resp.Content)
		log.LogEvent("llm_response", resp.Content)
		// Simulate streaming by word
		for _, word := range bytes.Split([]byte(resp.Content), []byte(" ")) {
			onStep("[LLM] " + string(word))
		}

		toolResp, err := tools.ExtractResponse(resp.Content)
		if err != nil || toolResp == nil {
			onStep("[Agent Final]: " + resp.Content)
			log.LogEvent("agent_final", resp.Content)
			return
		}

		tool := s.getTool(toolResp.Name)
		if tool == nil {
			onStep("[Agent] Tool not found: " + toolResp.Name)
			log.LogEvent("tool_not_found", toolResp.Name)
			return
		}
		// Always show which tool is being used
		onStep(fmt.Sprintf("[USING TOOL] %s", tool.ToolFunction.Name))
		log.LogEvent("using_tool", tool.ToolFunction.Name)

		// Special handling for bash: capture output and set working directory
		if tool.ToolFunction.Name == "bash" {
			cmdStr, _ := toolResp.Arguments["command"].(string)
			cmd := exec.Command("bash", "-c", cmdStr)
			if s.WorkDir != "" {
				cmd.Dir = s.WorkDir
			}
			var out bytes.Buffer
			cmd.Stdout = &out
			cmd.Stderr = &out
			err := cmd.Run()
			result := out.String()
			if err != nil {
				result += "\n[Error] " + err.Error()
			}
			onStep("[BASH OUTPUT] " + result)
			log.LogToolCall("bash", toolResp.Arguments, result)
			s.Mmeory = append(s.Mmeory, "[Tool bash]: "+result)
			userMsg = result
			continue
		}

		// Modifier tool: show diff and ask for approval
		if tool.ToolFunction.Name == "modifier" {
			before := ""
			if v, ok := toolResp.Arguments["input"].(string); ok {
				before = v
			}
			result, err := s.executeTool(tool, toolResp.Arguments)
			if err != nil {
				onStep("[Tool Error] " + err.Error())
				log.LogToolCall("modifier", toolResp.Arguments, err.Error())
				return
			}
			onStep(CodeReviewMsg{
				Before: before,
				After:  result.Result,
				Desc:   "Modifier tool result. Accept, edit, or decline?",
			})
			log.LogToolCall("modifier", toolResp.Arguments, result.Result)
			// Wait for user input (accept/edit/decline) - handled in CLI
			return
		}

		// Normal tool execution
		result, err := s.executeTool(tool, toolResp.Arguments)
		if err != nil {
			onStep("[Tool Error] " + err.Error())
			log.LogToolCall(tool.ToolFunction.Name, toolResp.Arguments, err.Error())
		}
		onStep(fmt.Sprintf("[TOOL %s RESULT] %s", tool.ToolFunction.Name, result.Result))
		log.LogToolCall(tool.ToolFunction.Name, toolResp.Arguments, result.Result)
		s.Mmeory = append(s.Mmeory, fmt.Sprintf("[Tool %s]: %s", tool.ToolFunction.Name, result.Result))

		if tool.ToolFunction.Name == "done" {
			onStep("[Agent Done]: " + result.Result)
			log.LogEvent("agent_done", result.Result)
			return
		}

		userMsg = result.Result
	}
	onStep("[Agent] Step limit reached.")
	log.LogEvent("step_limit_reached", nil)
}
