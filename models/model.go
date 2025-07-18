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

// The completion interface should be provided as an abstraction to every model
// This is useful to check
type CompletionInterface interface {
	Completion(p string, tool []tools.Tool) (LLMMessage, error)
}

// Currently let's handle ollama and open router first
type OllamaClient struct {
	LLM
}

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
		Model:    o.Model,
		Messages: o.Messages,
		Stream:   false,
		Tools:    tool,
	})
	if err != nil {
		slog.Error("Marshal err")
		slog.Error(err.Error())
		return LLMMessage{}, err
	}

	r, err := http.NewRequest("POST", "http://localhost:11434/api/chat", bytes.NewBuffer(body))
	if err != nil {
		slog.Error("Request creation failed")
		slog.Error(err.Error())
		return LLMMessage{}, err
	}

	r.Header = map[string][]string{
		"Content-Type": {"application/json"},
	}

	c := http.Client{}
	res, err := c.Do(r)
	if err != nil {
		slog.Error("HTTP request failed")
		slog.Error(err.Error())
		return LLMMessage{}, err
	}
	defer res.Body.Close()
	responsebody, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("ReadAll failed")
		slog.Error(err.Error())
		return LLMMessage{}, err
	}

	var ollamaresponse OllamaResponse
	err = json.Unmarshal(responsebody, &ollamaresponse)
	if err != nil {
		slog.Error("Unmarshal failed")
		slog.Error(err.Error())
		return LLMMessage{}, err
	}

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
		Model:    o.Model,
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

// OpenAI and OpenRouter support

type OpenAIClient struct {
	LLM
}

type OpenAIRequest struct {
	Model    string       `json:"model"`
	Messages []LLMMessage `json:"messages"`
	Stream   bool         `json:"stream"`
	Tools    []tools.Tool `json:"tools"`
}

type OpenAIResponse struct {
	Choices []struct {
		Message LLMMessage `json:"message"`
	} `json:"choices"`
}

func (o *OpenAIClient) Completion(p string, tool []tools.Tool) (LLMMessage, error) {
	message := LLMMessage{
		Role:    "user",
		Content: p,
	}
	o.Messages = append(o.Messages, message)

	body, err := json.Marshal(OpenAIRequest{
		Model:    o.Model,
		Messages: o.Messages,
		Stream:   false,
		Tools:    tool,
	})
	if err != nil {
		slog.Error("Marshal err")
		slog.Error(err.Error())
		return LLMMessage{}, err
	}

	r, err := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(body))
	r.Header = map[string][]string{
		"Content-Type": {"application/json"},
		"Authorization": {"Bearer " + o.apiKey},
	}

	c := http.Client{}
	res, err := c.Do(r)
	if err != nil {
		slog.Error(err.Error())
		return LLMMessage{}, err
	}
	defer res.Body.Close()
	responsebody, err := io.ReadAll(res.Body)
	if err != nil {
		return LLMMessage{}, err
	}

	var openairesponse OpenAIResponse
	err = json.Unmarshal(responsebody, &openairesponse)
	if err != nil || len(openairesponse.Choices) == 0 {
		return LLMMessage{}, err
	}

	msg := openairesponse.Choices[0].Message
	o.Messages = append(o.Messages, msg)
	return msg, nil
}

type OpenRouterClient struct {
	LLM
}

func (o *OpenRouterClient) Completion(p string, tool []tools.Tool) (LLMMessage, error) {
	message := LLMMessage{
		Role:    "user",
		Content: p,
	}
	o.Messages = append(o.Messages, message)

	body, err := json.Marshal(OpenAIRequest{
		Model:    o.Model,
		Messages: o.Messages,
		Stream:   false,
		Tools:    tool,
	})
	if err != nil {
		slog.Error("Marshal err")
		slog.Error(err.Error())
		return LLMMessage{}, err
	}

	r, err := http.NewRequest("POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(body))
	r.Header = map[string][]string{
		"Content-Type": {"application/json"},
		"Authorization": {"Bearer " + o.apiKey},
	}

	c := http.Client{}
	res, err := c.Do(r)
	if err != nil {
		slog.Error(err.Error())
		return LLMMessage{}, err
	}
	defer res.Body.Close()
	responsebody, err := io.ReadAll(res.Body)
	if err != nil {
		return LLMMessage{}, err
	}

	var openairesponse OpenAIResponse
	err = json.Unmarshal(responsebody, &openairesponse)
	if err != nil || len(openairesponse.Choices) == 0 {
		return LLMMessage{}, err
	}

	msg := openairesponse.Choices[0].Message
	o.Messages = append(o.Messages, msg)
	return msg, nil
}

// Factory for LLM
func NewLLMFromConfig(model, apiKey, provider string) CompletionInterface {
	switch provider {
	case "openai":
		return &OpenAIClient{LLM: LLM{Model: model, apiKey: apiKey, provider: provider}}
	case "openrouter":
		return &OpenRouterClient{LLM: LLM{Model: model, apiKey: apiKey, provider: provider}}
	case "ollama":
		fallthrough
	default:
		return &OllamaClient{LLM: LLM{Model: model, apiKey: apiKey, provider: provider}}
	}
}
