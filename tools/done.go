package tools

import "encoding/json"

type DoneTool struct {
	Tool
}

var donePrompt = `Signal that the task is complete. Use this tool when you are finished and want to stop the agent loop.`

func NewDoneTool() DoneTool {
	doneProperties := map[string]ToolProperty{}
	doneProperties["message"] = ToolProperty{
		Type:        "string",
		Description: "A message to indicate completion.",
	}

	doneParameter := ToolParameter{
		Type:       "object",
		Properties: doneProperties,
		Required:   []string{"message"},
	}

	doneFunction := ToolFunction{
		Name:        "done",
		Description: donePrompt,
		Parameters:  doneParameter,
	}

	return DoneTool{
		Tool{
			Type:         "function",
			ToolFunction: doneFunction,
			Execute:      doneExecutor,
		},
	}
}

func doneExecutor(args map[string]any) (ToolExecutionResult, error) {
	var msg struct{ Message string `json:"message"` }
	data, _ := json.Marshal(args)
	json.Unmarshal(data, &msg)
	return ToolExecutionResult{
		Result:    msg.Message,
		Error:     nil,
		ErrorCode: 0,
	}, nil
} 