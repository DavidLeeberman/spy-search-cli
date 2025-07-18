package models

import (
	"bufio"
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

	Messages []LLMMessage
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
func (o *OllamaClient) Completion(p string, tool []tools.Tool) (LLMMessage, error) {

	message := LLMMessage{
		Role:    "user",
		Content: p,
	}

	o.Messages = append(o.Messages, message)

	body, err := json.Marshal(OllamaRequest{
		Model:    "qwen2.5-coder:3b",
		Messages: o.Messages,
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

	o.Messages = append(o.Messages, ollamaresponse.Message)

	return ollamaresponse.Message, nil
}

// Streaming version of Completion
func (o *OllamaClient) CompletionStream(p string, tool []tools.Tool, onChunk func(content string, done bool, toolCall map[string]interface{})) error {
	message := LLMMessage{
		Role:    "user",
		Content: p,
	}
	o.Messages = append(o.Messages, message)

	body, err := json.Marshal(OllamaRequest{
		Model:    "qwen2.5-coder:3b",
		Messages: o.Messages,
		Stream:   true,
		Tools:    tool,
	})
	if err != nil {
		slog.Error("Marshal err")
		slog.Error(err.Error())
		return err
	}

	r, err := http.NewRequest("POST", "http://localhost:11434/api/chat", bytes.NewBuffer(body))
	r.Header = map[string][]string{
		"Content-Type": {"application/json"},
	}

	c := http.Client{}
	res, err := c.Do(r)
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var chunk map[string]interface{}
		if err := json.Unmarshal(line, &chunk); err != nil {
			continue
		}
		// Extract content and tool call
		msg, ok := chunk["message"].(map[string]interface{})
		if !ok {
			continue
		}
		content, _ := msg["content"].(string)
		toolCall := msg["tool_calls"]
		done, _ := chunk["done"].(bool)
		onChunk(content, done, toolCall.(map[string]interface{}))
		if done {
			break
		}
	}
	return nil
}

// convert to Tool
func (o OllamaClient) ToolHandling() {}
