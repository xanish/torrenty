package logger

import (
	"bytes"
	"log"
	"testing"
)

func TestLog(t *testing.T) {
	tests := []struct {
		level       LogLevel
		message     string
		args        []interface{}
		expected    string
		shouldPanic bool
	}{
		{
			Debug,
			"should log a debug message: %s",
			[]interface{}{"test"},
			"[DEBUG] should log a debug message: test\n",
			false,
		},
		{
			Info,
			"should log an info message: %d",
			[]interface{}{42},
			"[INFO] should log an info message: 42\n",
			false,
		},
		{
			Warning,
			"should log a warning message",
			nil,
			"[WARN] should log a warning message\n",
			false,
		},
		{
			Error,
			"should log an error message",
			nil,
			"[ERROR] should log an error message\n",
			false,
		},
		{
			Fatal,
			"should log a fatal message",
			nil,
			"[FATAL] should log a fatal message\n",
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			var buf bytes.Buffer
			mockLogger := log.New(&buf, "", 0)
			setLogger(mockLogger)

			// Set a custom fatal function that triggers a panic
			setFatalFunc(func(int) {
				panic("Fatal log triggered")
			})

			log.SetOutput(&buf)

			// restore default logger after test
			defer setLogger(log.Default())

			if tt.shouldPanic {
				defer func() {
					if r := recover(); r != nil {
						if !tt.shouldPanic {
							t.Errorf("did not expect panic but got: %v", r)
						}
					} else {
						if tt.shouldPanic {
							t.Errorf("expected panic but did not get one")
						}
					}
				}()
			}

			if tt.args != nil {
				Log(tt.level, tt.message, tt.args...)
			} else {
				Log(tt.level, tt.message)
			}

			if buf.String() != tt.expected {
				t.Errorf("expected %q but got %q", tt.expected, buf.String())
			}
		})
	}
}
