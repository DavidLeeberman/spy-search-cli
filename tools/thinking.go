// This files provides a thinking tool for the agent
package tools

import (
	"encoding/json"
	"log/slog"
)

// Thinking Tool
type ThinkingTool struct {
	Tool
}

// convert this to string template
var ThinkingPrompt = `You are a software engineer agent. In order to hnadler complex tasks you would
like to think deeply and figure out the root of the problem. Don't hesitate to think longer if you think it 
is necesssary. You are given the following problem statement
`

func NewThinkingTool() ThinkingTool {
	return ThinkingTool{
		Tool{
			ToolFunction: ToolFunction{},
			Execute:      NewThinkingTool().ExecuteTool,
		},
	}
}

// here can we actually not using any
func (t ThinkingTool) ExecuteTool(args map[string]any) (ToolExecutionResult, error) {

	_, err := t.parseArgs(args)

	if err != nil {
		slog.Error(err.Error())
	}

	return ToolExecutionResult{}, nil
}

// parse Arguments
func (t ThinkingTool) parseArgs(args map[string]any) (thinkingArgs, error) {
	// we have to handle parsing the thinkign argument here

	var tkargs thinkingArgs
	// the idea is to first convert the args to json and then convert it to thinkingArgs type
	data, err := json.Marshal(args)
	if err != nil {
		// we need more data handling here
		slog.Error(err.Error())
	}

	json.Unmarshal(data, &tkargs)

	return tkargs, err
}

// private thinking args
type thinkingArgs struct {
	Thinkingstep int    `json:"thinkingstep"`
	Rethink      bool   `json:"rethink"`
	Content      string `json:"content"` // the content here is the thinking process and shoudl save the the summary
}
