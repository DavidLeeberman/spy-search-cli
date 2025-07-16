package models

// here we first focusing on ollama at first version
type LLM struct {
	Model    string
	apiKey   string
	provider string
}

// The completion internface should be provides as an abstruction to every model
// This is userful to check
type CompletionInterface interface {
	Completion(p string) string // not sure if here should be a string
}
