package logger

import (
	"log"
	"os"

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
	log.Printf(`{"level":"INFO","msg":"logger initialized"}`)
}

func Info(msg string, fields map[string]any) {
	log.Printf(`{"level":"INFO","msg":"%s","fields":%v}`, msg, fields)
}

func Error(msg string, fields map[string]any) {
	log.Printf(`{"level":"ERROR","msg":"%s","fields":%v}`, msg, fields)
}

func Fatal(msg string, fields map[string]any) {
	log.Printf(`{"level":"FATAL","msg":"%s","fields":%v}`, msg, fields)
	os.Exit(1)
}

func Warn(msg string, fields map[string]any) {
	log.Printf(`{"level":"WARN","msg":"%s","fields":%v}`, msg, fields)
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
