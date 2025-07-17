package models_test

import (
	"spysearch/models"
	"testing"
)

func TestOllamaCompletion(t *testing.T) {
	models.OllamaClient{}.Completion("test")
}
