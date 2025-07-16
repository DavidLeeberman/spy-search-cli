package tools

// here we provides an abstraction type of tool

type Tool struct{}

type ExecuteInterface interface {
	Execute(args map[string]any)
}
