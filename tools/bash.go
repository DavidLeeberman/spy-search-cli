package tools

import (
	"encoding/json"
	"log/slog"
	"os/exec"
)

// this tool allows the agent to perform any kind of tool that can be run with bash
// for example create new files it should use echo >> " " or use cat to peek the content

// cite: ByteDance/TraeAgent
var bashprompt = `
"Run commands in a bash shell
* When invoking this tool, the contents of the "command" parameter does NOT need to be XML-escaped.
* You have access to a mirror of common linux and python packages via apt and pip.
* State is persistent across command calls and discussions with the user.
* To inspect a particular line range of a file, e.g. lines 10-25, try 'sed -n 10,25p /path/to/the/file'.
* Please avoid commands that may produce a very large amount of output.
`

type BashTool struct {
	Tool
}

func NewBashTool() BashTool {

	bashProperties := map[string]ToolProperty{}

	bashCommand := ToolProperty{
		Type:        "string",
		Description: "bash command to run",
	}

	restartCommand := ToolProperty{
		Type:        "boolean",
		Description: "restart the terminal",
	}

	bashProperties["command"] = bashCommand
	bashProperties["restart"] = restartCommand

	bashParameter := ToolParameter{
		Type:       "object",
		Properties: bashProperties,
		Required:   []string{"command", "restart"},
	}

	bashFunction := ToolFunction{
		Name:        "bash",
		Description: bashprompt,
		Parameters:  bashParameter,
	}

	return BashTool{
		Tool{
			Type:         "function",
			ToolFunction: bashFunction,
			Execute:      bashExecutor,
		},
	}
}

func bashExecutor(args map[string]any) (ToolExecutionResult, error) {
	bashArgs, err := bashParseArgs(args)
	if err != nil {
		return ToolExecutionResult{}, nil
	}

	cmd := exec.Command(bashArgs.Command)
	if err := cmd.Run(); err != nil {
		return ToolExecutionResult{
			Result:    "Error " + err.Error(),
			Error:     err,
			ErrorCode: -1,
		}, err
	}

	return ToolExecutionResult{
		Result:    "Running Command " + bashArgs.Command + " successfully",
		Error:     nil,
		ErrorCode: 0,
	}, nil
}

// TODO we need a better error handling system
func bashParseArgs(args map[string]any) (BashArgs, error) {
	var bashargs BashArgs
	data, err := json.Marshal(args)
	if err != nil {
		slog.Error(err.Error())
	}
	json.Unmarshal(data, &bashargs)
	return bashargs, err
}

type BashArgs struct {
	Command string `json:"command"`
	Restart bool   `json:"restart"`
}
