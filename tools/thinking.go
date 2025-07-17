// This files provides a thinking tool for the agent
package tools

import (
	"encoding/json"
	"log/slog"
)

// Thinking Tool
type ThinkingTool Tool

func NewThinkingTool() ThinkingTool {
	return ThinkingTool{
		Name:        "Thinking",
		Description: "THINKING DESCRIPTION",
		Parameters:  ToolParameter{},
		Execute:     NewThinkingTool().ExecuteTool,
	}
}

// here can we actually not using any
func (t ThinkingTool) ExecuteTool(args map[string]any) {
	_, err := t.ParseArgs(args)
	if err != nil {
		// parsing handling
	}
}

// parse Arguments
func (t ThinkingTool) ParseArgs(args map[string]any) (*thinkingArgs, error) {
	// we have to handle parsing the thinkign argument here

	var tkargs thinkingArgs
	// the idea is to first convert the args to json and then convert it to thinkingArgs type
	data, err := json.Marshal(args)
	if err != nil {
		// we need more data handling here
		slog.Error(err.Error())
	}

	json.Unmarshal(data, &tkargs)

	return &tkargs, err
}

// private thinking args
type thinkingArgs struct {
	Thinkingstep int    `json:"thinkingstep"`
	Rethink      bool   `json:"rethink"`
	Content      string `json:"content"`
}
