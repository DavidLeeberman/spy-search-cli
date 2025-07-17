package tools

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

type ToolExecutionResult struct{}

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
*/
