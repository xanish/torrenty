package logger

import (
	"log"
	"os"
)

// LogLevel type
type LogLevel int

const (
	Debug LogLevel = iota
	Info
	Warning
	Error
	Fatal
)

// logger variable
var logger *log.Logger = log.Default()
var fatalFunc = os.Exit

// SetLogger allows setting a custom logger for testing
func setLogger(l *log.Logger) {
	logger = l
}

// SetFatalFunc allows setting a custom fatal function for testing
func setFatalFunc(f func(int)) {
	fatalFunc = f
}

// Log helper function
func Log(level LogLevel, message string, args ...interface{}) {
	var prefix string
	switch level {
	case Debug:
		prefix = "[DEBUG] "
	case Info:
		prefix = "[INFO] "
	case Warning:
		prefix = "[WARN] "
	case Error:
		prefix = "[ERROR] "
	case Fatal:
		prefix = "[FATAL] "
	}

	logger.Printf(prefix+message, args...)
	if prefix == "[FATAL] " {
		fatalFunc(1)
	}
}
