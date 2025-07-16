package tools

// here we provides an abstraction type of tool

type Tool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  []ToolParameter `json:"parameters"`
}

type ExecuteInterface interface {
	Execute(args map[string]any)
	Concurrent() bool
}

// list of tool parameter
type ToolParameter struct {
	Type       string
	Properties map[string]any
	Required   []string
}

// a tool execute should be stateless
//
