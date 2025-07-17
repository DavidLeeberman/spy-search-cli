package models

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"spysearch/tools"
)

// here we first focusing on ollama at first version
type LLM struct {
	Model    string
	apiKey   string
	provider string
}

// LLMMessage (maybe we shall split into seperate folder)
type LLMMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// list of LLM Messages
type LLMMessages []LLMMessage

// The completion internface should be provides as an abstruction to every model
// This is userful to check
type CompletionInterface interface {
	Completion(p string) string // not sure if here should be a string
}

// Currently let's handle ollama and open router first
type OllamaClient LLM

type OllamaRequest struct {
	Model    string       `json:"model"`
	Messages []LLMMessage `json:"messages"`
	Stream   bool         `json:"stream"`
	Tools    []tools.Tool `json:"tools"`
}

type OllamaResponse struct {
	Model   string     `json:"model"`
	Create  string     `json:"created_at"`
	Message LLMMessage `json:"message"`
	Done    bool       `json:"done"`
}

// ollama completion logic the completion should be a tool call
func (o OllamaClient) Completion(p string, tool []tools.Tool) (LLMMessage, error) {

	message := LLMMessage{
		Role:    "user",
		Content: p,
	}

	messages := append([]LLMMessage{}, message)

	body, err := json.Marshal(OllamaRequest{
		Model:    "qwen2.5-coder:1.5b",
		Messages: messages,
		Stream:   false,
		Tools:    tool,
	})
	if err != nil {
		slog.Error("Marshal err")
		slog.Error(err.Error())
		panic("json handling failclea")
	}

	r, err := http.NewRequest("POST", "http://localhost:11434/api/chat", bytes.NewBuffer(body))

	header := map[string][]string{
		"Content-Type": {"application/json"},
	}
	r.Header = header

	c := http.Client{}
	res, err := c.Do(r)
	if err != nil {
		slog.Error(err.Error())
	}

	defer res.Body.Close()
	responsebody, err := io.ReadAll(res.Body)

	var ollamaresponse OllamaResponse
	err = json.Unmarshal([]byte(responsebody), &ollamaresponse)

	return ollamaresponse.Message, nil
}

// convert to Tool
func (o OllamaClient) ToolHandling() {}
