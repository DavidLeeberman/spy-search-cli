package tools_test

import (
	"fmt"
	"log/slog"
	"spysearch/tools"
	"testing"
)

func TestParseArgs(t *testing.T) {
	tk := tools.ThinkingTool{}

	mock_data := map[string]any{}

	mock_data["thinkginstep"] = 10
	mock_data["rethink"] = false
	mock_data["content"] = "this is a test content"

	d, err := tk.ParseArgs(mock_data)
	if err != nil {
		slog.Error(err.Error())
	}
	fmt.Println(d)
}
