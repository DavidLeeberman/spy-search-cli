package tools

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// here we provides an abstraction type of tool
type Tool struct {
	ToolFunction ToolFunction                                           `json:"function"`
	Type         string                                                 `json:"type"`
	Execute      func(args map[string]any) (ToolExecutionResult, error) `json:"-"` // maybe an interface is not a good option
}

type ToolFunction struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Parameters  ToolParameter `json:"parameters"`
}

// list of tool parameter
type ToolParameter struct {
	Type       string                  `json:"type"`
	Properties map[string]ToolProperty `json:"properties"`
	Required   []string                `json:"required"`
}

type ToolProperty struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type ToolExecutionResult struct {
	Result    string
	Error     error
	ErrorCode int
}

type ToolResponse struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}

// ExtractRawJSON extracts the raw JSON string from the response
func ExtractRawJSON(res string) (string, error) {
	pattern := "(?s)```json\\s*(.*?)\\s*```"
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex: %v", err)
	}
	matches := re.FindStringSubmatch(res)
	if len(matches) < 2 {
		return "", fmt.Errorf("no JSON content found")
	}
	return matches[1], nil
}

// ExtractResponse extracts and parses the JSON response into a ToolResponse struct
func ExtractResponse(res string) (*ToolResponse, error) {
	jsonStr, err := ExtractRawJSON(res)
	if err != nil {
		return nil, err
	}

	var toolResponse ToolResponse
	if err := json.Unmarshal([]byte(jsonStr), &toolResponse); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %v", err)
	}

	return &toolResponse, nil
}

// a tool execute should be stateless
//

/*
	A tool needs three things:
	1. an execute function --> which will be call when the agent think it is necessary
	2. description
	3. a function that can parse the description

	A tool should have
		- type
		- name
		- description
		- a list of parameters
			- there are many properties in the parameters
			- each properties should have type description
		- a list of requrie properties
	One agent should have multiple set of tools

	The execution function should be called after getting the tool call response
*/
