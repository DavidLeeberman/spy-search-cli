package tools

import (
	"encoding/json"
)

// modify the code
// modifier is a modify tool that modify strings (replace, insert, view)

// Args for the modifier tool
// operation: "replace", "insert", "view"
// input: the string to operate on
// target: the substring to replace (for replace)
// replacement: the replacement string (for replace/insert)
// index: the index at which to insert (for insert)
type modifyArgs struct {
	Operation   string `json:"operation"`
	Input       string `json:"input"`
	Target      string `json:"target"`
	Replacement string `json:"replacement"`
	Index       int    `json:"index"`
}

type ModifyTool struct {
	Tool
}

var modifierPrompt = `Modify a string by replacing, inserting, or viewing content.
- operation: "replace", "insert", or "view"
- input: the string to operate on
- target: the substring to replace (for replace)
- replacement: the replacement string (for replace/insert)
- index: the index at which to insert (for insert)`

func NewModifierTool() ModifyTool {
	modifierProperties := map[string]ToolProperty{}

	modifierProperties["operation"] = ToolProperty{
		Type:        "string",
		Description: "Operation to perform: replace, insert, or view",
	}
	modifierProperties["input"] = ToolProperty{
		Type:        "string",
		Description: "The string to operate on",
	}
	modifierProperties["target"] = ToolProperty{
		Type:        "string",
		Description: "The substring to replace (for replace)",
	}
	modifierProperties["replacement"] = ToolProperty{
		Type:        "string",
		Description: "The replacement string (for replace/insert)",
	}
	modifierProperties["index"] = ToolProperty{
		Type:        "integer",
		Description: "The index at which to insert (for insert)",
	}

	modifierParameter := ToolParameter{
		Type:       "object",
		Properties: modifierProperties,
		Required:   []string{"operation", "input"},
	}

	modifierFunction := ToolFunction{
		Name:        "modifier",
		Description: modifierPrompt,
		Parameters:  modifierParameter,
	}

	return ModifyTool{
		Tool{
			Type:         "function",
			ToolFunction: modifierFunction,
			Execute:      modifierExecutor,
		},
	}
}

func modifierExecutor(args map[string]any) (ToolExecutionResult, error) {
	var margs modifyArgs
	data, err := json.Marshal(args)
	if err != nil {
		return ToolExecutionResult{Result: "Error: " + err.Error(), Error: err, ErrorCode: -1}, err
	}
	json.Unmarshal(data, &margs)

	switch margs.Operation {
	case "replace":
		if margs.Target == "" {
			return ToolExecutionResult{Result: "Error: target required for replace", ErrorCode: 1}, nil
		}
		result := margs.Input
		if margs.Replacement == "" {
			result = replaceAll(margs.Input, margs.Target, "")
		} else {
			result = replaceAll(margs.Input, margs.Target, margs.Replacement)
		}
		return ToolExecutionResult{Result: result, ErrorCode: 0}, nil
	case "insert":
		if margs.Index < 0 || margs.Index > len(margs.Input) {
			return ToolExecutionResult{Result: "Error: invalid index", ErrorCode: 2}, nil
		}
		result := margs.Input[:margs.Index] + margs.Replacement + margs.Input[margs.Index:]
		return ToolExecutionResult{Result: result, ErrorCode: 0}, nil
	case "view":
		return ToolExecutionResult{Result: margs.Input, ErrorCode: 0}, nil
	default:
		return ToolExecutionResult{Result: "Error: unknown operation", ErrorCode: 3}, nil
	}
}

// Helper for replace all (strings.ReplaceAll, but no import)
func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	res := ""
	start := 0
	for {
		idx := indexOf(s, old, start)
		if idx == -1 {
			res += s[start:]
			break
		}
		res += s[start:idx] + new
		start = idx + len(old)
	}
	return res
}

// Helper for indexOf (like strings.Index, but with offset)
func indexOf(s, substr string, start int) int {
	if start >= len(s) {
		return -1
	}
	for i := start; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
