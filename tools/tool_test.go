package tools_test

import (
	"log/slog"
	"os"
	"spysearch/tools"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tk := tools.ThinkingTool{}

	mock_data := map[string]any{}

	mock_data["thinkginstep"] = 10
	mock_data["rethink"] = false
	mock_data["content"] = "this is a test content"

	_, err := tk.Execute(mock_data)

	if err != nil {
		slog.Error(err.Error())
	}
}

func TestMemoryTool(t *testing.T) {
	// Test BasicSummarizer
	store := &tools.InMemoryStore{}
	tool := tools.NewMemoryTool(store, tools.BasicSummarizer)
	tool.Execute(map[string]any{
		"history": []tools.MemoryEntry{
			{
				ID:        "1",
				Prompt:    "What is a goroutine?",
				Response:  "It's a lightweight thread in Go.",
				Timestamp: "2025-07-19T15:00:00Z",
			},
		},
	})

	// Test OpenAISummarizer
	store = &tools.InMemoryStore{}
	summarizer := &tools.OpenAISummarizer{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   "gpt-4",
		APIHost: "https://api.openai.com/v1",
	}
	tool = tools.NewMemoryTool(store, summarizer.Summarize)
	tool.Execute(map[string]any{
		"history": []tools.MemoryEntry{
			{
				ID:        "1",
				Prompt:    "What is a goroutine?",
				Response:  "It's a lightweight thread in Go.",
				Timestamp: "2025-07-19T15:00:00Z",
			},
		},
	})
}
