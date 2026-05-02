package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

/*
logger provides JSON-formatted structured logging with four severity levels.
Supports Info, Error, Warn for standard logging and Fatal for terminating with exit code 1.
All functions accept message string and optional fields map for structured data output.
*/
func Init() {
	log.SetOutput(os.Stdout)
	log.SetFlags(0)
	Info("logger initialized", nil)
}

func Info(msg string, fields map[string]any) {
	logJSON("INFO", msg, fields)
}

func Error(msg string, fields map[string]any) {
	logJSON("ERROR", msg, fields)
}

func Fatal(msg string, fields map[string]any) {
	logJSON("FATAL", msg, fields)
	os.Exit(1)
}

func Warn(msg string, fields map[string]any) {
	logJSON("WARN", msg, fields)
}

func logJSON(level, msg string, fields map[string]any) {
	if fields == nil {
		fields = map[string]any{}
	}

	entry := map[string]any{
		"ts":     time.Now().UTC().Format(time.RFC3339Nano),
		"level":  level,
		"msg":    msg,
		"fields": fields,
	}

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
