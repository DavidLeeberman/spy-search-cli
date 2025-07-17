package models_test

import (
	"spysearch/models"
	"spysearch/tools"
	"testing"
)

func TestOllamaCompletion(t *testing.T) {
	mock_properties := map[string]tools.ToolProperty{}

	mock_properties["location"] = tools.ToolProperty{
		Type:        "string",
		Description: "The location to get the weather for, e.g. San Francisco, CA",
	}

	mock_param := tools.ToolParameter{
		Type:       "object",
		Properties: mock_properties,
	}
	mock_func := tools.ToolFunction{
		Name:        "get_current_weather",
		Description: "Get the current weather for a location",
		Parameters:  mock_param,
	}

	mock_tool := tools.Tool{
		ToolFunction: mock_func,
		Type:         "function",
	}

	list_tool := append([]tools.Tool{}, mock_tool)

	models.OllamaClient{}.Completion("What is the weather today in Toronto? You are require to use tool", list_tool)
}

func TestThinkingTool(t *testing.T) {
	tk := tools.NewThinkingTool()

	list_tool := append([]tools.Tool{}, tk.Tool)
	models.OllamaClient{}.Completion("Why my merge sort not working", list_tool)

}
