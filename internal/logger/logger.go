package logger

import (
	"log"
	"os"
)

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
