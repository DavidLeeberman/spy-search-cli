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
var thinkingPrompt = `In order to hnadler complex tasks you would like to think deeply and figure out the root of the problem. Don't hesitate to think longer if you think it 
is necesssary. You are given the following problem statement. 
`

func NewThinkingTool() ThinkingTool {

	// here we need to create a tool function , a tool parameter , a tool property

	thinkingProperties := map[string]ToolProperty{}

	tkingstep := ToolProperty{
		Type:        "Interger",
		Description: "Number of step that you think it takes to solve this problem. Minimial would be 1 and Maximum would be 25. Don't hesitate to make a large number if you think this task is difficult",
	}

	rethink := ToolProperty{
		Type:        "boolean",
		Description: "Do you think rethink for this step is necessary ?",
	}

	content := ToolProperty{
		Type:        "string",
		Description: "Your thinking process should be placed inside here. For each thinkstep you mentioned. You should have one sentecne describe your thinking. Don't hesitate to geneerate more content",
	}

	summary := ToolProperty{
		Type:        "string",
		Description: "summarize the content you mentioned about to save into your memory",
	}

	thinkingProperties["steps"] = tkingstep
	thinkingProperties["rethink"] = rethink
	thinkingProperties["content"] = content
	thinkingProperties["summary"] = summary

	thinkingParameter := ToolParameter{
		Type:       "object",
		Properties: thinkingProperties,
		Required:   []string{},
	}

	thinkFunction := ToolFunction{
		Name:        "thinking",
		Description: thinkingPrompt,
		Parameters:  thinkingParameter,
	}

	return ThinkingTool{
		Tool{
			Type:         "function",
			ToolFunction: thinkFunction,
			Execute:      thinkingExecutor,
		},
	}
}

// here can we actually not using any
func thinkingExecutor(args map[string]any) (ToolExecutionResult, error) {

	data, err := parsethinkingArgs(args)

	if err != nil {
		return ToolExecutionResult{
			Result:    "Error: " + err.Error(),
			Error:     err,
			ErrorCode: 0,
		}, err
	}

	result := ToolExecutionResult{
		Result:    data.Content,
		Error:     nil,
		ErrorCode: 0,
	}

	return result, nil
}

// parse Arguments
func parsethinkingArgs(args map[string]any) (thinkingArgs, error) {
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
	Summary      string `json:"summary"`
}
