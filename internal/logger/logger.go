package logger

import (
	"fmt"
	"log"
)

const (
	Debug = iota
	Info
	Warning
	Error
)

func Log(level int, format string, params ...interface{}) {
	switch level {
	case Debug:
		log.Println(fmt.Sprintf("[DEBUG] "+format, params...))
	case Info:
		log.Println(fmt.Sprintf("[INFO] "+format, params...))
	case Warning:
		log.Println(fmt.Sprintf("[WARN] "+format, params...))
	case Error:
		log.Println(fmt.Sprintf("[ERROR] "+format, params...))
	}
}
