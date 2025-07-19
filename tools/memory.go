package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MemoryTool wraps the Tool struct
type MemoryTool struct {
	Tool
	store      MemoryStore
	summarizer SummarizerFunc
}

// MemoryEntry represents a single prompt/response interaction
type MemoryEntry struct {
	ID        string `json:"id"`
	Prompt    string `json:"prompt"`
	Response  string `json:"response"`
	Timestamp string `json:"timestamp"` // ISO8601 format
}

func ensureMemoryEntryMetadata(e *MemoryEntry) {
	if strings.TrimSpace(e.ID) == "" {
		e.ID = uuid.NewString()
	}
	if strings.TrimSpace(e.Timestamp) == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
}

type MemoryStore interface {
	Save(entry MemoryEntry) error
	LoadAll() ([]MemoryEntry, error)
}

type InMemoryStore struct {
	data []MemoryEntry
}

func (s *InMemoryStore) Save(entry MemoryEntry) error {
	s.data = append(s.data, entry)
	return nil
}

func (s *InMemoryStore) LoadAll() ([]MemoryEntry, error) {
	return s.data, nil
}

type SummarizerFunc func(entries []MemoryEntry) (string, error)

func BasicSummarizer(entries []MemoryEntry) (string, error) {
	var sb strings.Builder
	sb.WriteString("🧠 Summarized Conversation:\n\n")
	for _, e := range entries {
		sb.WriteString(fmt.Sprintf("Prompt: %s\nResponse: %s\n\n", e.Prompt, e.Response))
	}
	return sb.String(), nil
}

type OpenAISummarizer struct {
	APIKey  string
	Model   string // "gpt-4", "gpt-3.5-turbo", etc.
	APIHost string // usually "https://api.openai.com/v1"
}

func (s *OpenAISummarizer) Summarize(entries []MemoryEntry) (string, error) {
	var input strings.Builder
	for _, e := range entries {
		input.WriteString(fmt.Sprintf("User: %s\nAssistant: %s\n\n", e.Prompt, e.Response))
	}

	reqBody := map[string]any{
		"model": s.Model,
		"messages": []map[string]string{
			{"role": "system", "content": "Summarize the following conversation:"},
			{"role": "user", "content": input.String()},
		},
		"temperature": 0.3,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	req, err := http.NewRequest("POST", s.APIHost+"/chat/completions", bytes.NewBuffer(bodyBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("OpenAI error: %s", string(raw))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from OpenAI")
	}

	return result.Choices[0].Message.Content, nil
}

// MemoryArgs is a slice of prompt-response pairs
type MemoryArgs struct {
	History []MemoryEntry `json:"history"`
}

// NewMemoryTool returns a memory summarization tool
func NewMemoryTool(store MemoryStore, summarizer SummarizerFunc) MemoryTool {
	properties := map[string]ToolProperty{
		"history": {
			Type:        "array",
			Description: "A list of prompt-response history entries",
		},
	}

	params := ToolParameter{
		Type:       "object",
		Properties: properties,
		Required:   []string{"history"},
	}

	memFunction := ToolFunction{
		Name:        "summarize_memory",
		Description: "Summarizes the conversation history with optional LLM and persistence.",
		Parameters:  params,
	}

	tool := MemoryTool{
		store:      store,
		summarizer: summarizer,
	}

	tool.Tool = Tool{
		Type:         "function",
		ToolFunction: memFunction,
		Execute:      tool.memoryExecutor, // bind receiver
	}
	return tool
}

// memoryExecutor summarizes the prompt/response history
func (m MemoryTool) memoryExecutor(args map[string]any) (ToolExecutionResult, error) {
	memArgs, result, err := memoryParseArgs(args)
	if err != nil {
		return result, err
	}

	// Store entries
	for i := range memArgs.History {
		ensureMemoryEntryMetadata(&memArgs.History[i])
		if err = m.store.Save(memArgs.History[i]); err != nil {
			return ToolExecutionResult{Error: err, ErrorCode: 4}, err
		}
	}

	// Load everything and summarize
	all, err := m.store.LoadAll()
	if err != nil {
		return ToolExecutionResult{Error: err, ErrorCode: 5}, err
	}

	summary, err := m.summarizer(all)
	if err != nil {
		return ToolExecutionResult{Error: err, ErrorCode: 6}, err
	}

	return ToolExecutionResult{
		Result:    summary,
		Error:     nil,
		ErrorCode: 0,
	}, nil
}

// TODO we need a better error handling system
func memoryParseArgs(args map[string]any) (MemoryArgs, ToolExecutionResult, error) {
	var memArgs MemoryArgs
	data, err := json.Marshal(args)
	if err != nil {
		return MemoryArgs{}, ToolExecutionResult{Error: err, ErrorCode: 1}, err
	}
	if err = json.Unmarshal(data, &memArgs); err != nil {
		return MemoryArgs{}, ToolExecutionResult{Error: err, ErrorCode: 2}, err
	}

	if err = validateMemoryArgs(memArgs); err != nil {
		return MemoryArgs{}, ToolExecutionResult{Error: err, ErrorCode: 3}, err
	}

	return memArgs, ToolExecutionResult{}, nil
}

// validateMemoryArgs ensures MemoryArgs is well-formed
func validateMemoryArgs(memArgs MemoryArgs) error {
	if len(memArgs.History) == 0 {
		return errors.New("no conversation history provided")
	}
	for i, entry := range memArgs.History {
		if strings.TrimSpace(entry.Prompt) == "" {
			return fmt.Errorf("prompt at index %d is empty", i)
		}
		if strings.TrimSpace(entry.Response) == "" {
			return fmt.Errorf("response at index %d is empty", i)
		}
	}
	return nil
}
