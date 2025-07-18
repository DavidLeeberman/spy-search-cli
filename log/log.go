package log

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

var logFile = "log.json"
var mu sync.Mutex

type LogEntry struct {
	Timestamp string      `json:"timestamp"`
	Event     string      `json:"event"`
	Data      interface{} `json:"data"`
}

func LogEvent(event string, data interface{}) {
	entry := LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Event:     event,
		Data:      data,
	}
	writeLog(entry)
}

func LogToolCall(toolName string, args interface{}, result interface{}) {
	LogEvent("tool_call", map[string]interface{}{
		"tool":   toolName,
		"args":   args,
		"result": result,
	})
}

func writeLog(entry LogEntry) {
	mu.Lock()
	defer mu.Unlock()
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	_ = enc.Encode(entry)
}
