package logger

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"
)

var logLevel = "INFO"

var levelPriority = map[string]int{
	"DEBUG": 0,
	"INFO":  1,
	"WARN":  2,
	"ERROR": 3,
	"FATAL": 4,
}

func Init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	logLevel = getEnv("LOG_LEVEL", "INFO")
	Info("logger initialized", map[string]any{"level": logLevel})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func Debug(msg string, fields map[string]any) {
	logJSON("DEBUG", msg, fields)
}

func Info(msg string, fields map[string]any) {
	logJSON("INFO", msg, fields)
}

func Fatal(msg string, fields map[string]any) {
	logJSON("FATAL", msg, fields)
	os.Exit(1)
}

func Warn(msg string, fields map[string]any) {
	logJSON("WARN", msg, fields)
}

func logJSON(level, msg string, fields map[string]any) {
	if levelPriority[level] < levelPriority[logLevel] {
		return
	}

	if fields == nil {
		fields = map[string]any{}
	}

	entry := map[string]any{
		"ts":    time.Now().UTC().Format(time.RFC3339Nano),
		"level": level,
		"msg":   msg,
	}

	if _, file, line, ok := runtime.Caller(2); ok {
		entry["caller"] = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	entry["fields"] = fields

	b, err := json.Marshal(entry)
	if err != nil {
		log.Printf(`{"level":"ERROR","msg":"logger marshal failed","fields":{"error":"%v"}}`, err)
		return
	}

	log.Print(string(b))
}

func WithRequestID(c *gin.Context, fields map[string]any) map[string]any {
	reqID := c.GetString("request_id")

	if reqID == "" {
		reqID = "unknown"
	}

	if fields == nil {
		fields = make(map[string]any)
	}

	fields["request_id"] = reqID

	return fields
}
