package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"kigrepair/internal/safety"
)

type Level string

const (
	LevelDebug   Level = "debug"
	LevelInfo    Level = "info"
	LevelWarning Level = "warning"
	LevelError   Level = "error"
)

type Options struct {
	Quiet    bool
	Level    string
	RunID    string
	Workflow string
}

type Logger struct {
	quiet    bool
	file     *os.File
	out      io.Writer
	level    Level
	runID    string
	workflow string
	mu       sync.Mutex
}

type Event struct {
	TS              time.Time      `json:"ts"`
	Level           Level          `json:"level"`
	RunID           string         `json:"run_id,omitempty"`
	Workflow        string         `json:"workflow,omitempty"`
	OperationID     string         `json:"operation_id,omitempty"`
	Event           string         `json:"event"`
	Message         string         `json:"message,omitempty"`
	Category        string         `json:"category,omitempty"`
	Status          string         `json:"status,omitempty"`
	DurationMS      int64          `json:"duration_ms,omitempty"`
	FailureCategory string         `json:"failure_category,omitempty"`
	Fields          map[string]any `json:"fields,omitempty"`
}

func New(path string, quiet bool) (*Logger, error) {
	return NewWithOptions(path, Options{Quiet: quiet, Level: string(LevelInfo)})
}

func NewWithOptions(path string, opts Options) (*Logger, error) {
	level, err := ParseLevel(opts.Level)
	if err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{
		quiet:    opts.Quiet,
		file:     file,
		out:      file,
		level:    level,
		runID:    opts.RunID,
		workflow: opts.Workflow,
	}, nil
}

func ParseLevel(value string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", string(LevelInfo):
		return LevelInfo, nil
	case string(LevelDebug):
		return LevelDebug, nil
	case string(LevelWarning), "warn":
		return LevelWarning, nil
	case string(LevelError):
		return LevelError, nil
	default:
		return "", fmt.Errorf("invalid log level %q: must be debug, info, warning, or error", value)
	}
}

func (l *Logger) Debug(format string, args ...any) {
	l.write(LevelDebug, "message", format, args...)
}

func (l *Logger) Info(format string, args ...any) {
	l.write(LevelInfo, "message", format, args...)
}

func (l *Logger) Warn(format string, args ...any) {
	l.write(LevelWarning, "message", format, args...)
}

func (l *Logger) Error(format string, args ...any) {
	l.write(LevelError, "message", format, args...)
}

func (l *Logger) Event(event Event) {
	if l == nil || !l.enabled(event.Level) {
		return
	}
	if event.TS.IsZero() {
		event.TS = time.Now().UTC()
	}
	if event.RunID == "" {
		event.RunID = l.runID
	}
	if event.Workflow == "" {
		event.Workflow = l.workflow
	}
	event.Message = safety.RedactString(event.Message)
	event.Fields = redactFields(event.Fields)
	l.writeEvent(event)
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) write(level Level, event string, format string, args ...any) {
	if l == nil || !l.enabled(level) {
		return
	}
	message := safety.RedactString(fmt.Sprintf(format, args...))
	l.writeEvent(Event{
		TS:       time.Now().UTC(),
		Level:    level,
		RunID:    l.runID,
		Workflow: l.workflow,
		Event:    event,
		Message:  message,
	})
}

func (l *Logger) writeEvent(event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		data = []byte(fmt.Sprintf(`{"ts":%q,"level":"error","event":"log_encode_failed","message":%q}`, time.Now().UTC().Format(time.RFC3339Nano), safety.RedactString(err.Error())))
	}
	line := safety.RedactString(string(data))
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.out != nil {
		_, _ = fmt.Fprintln(l.out, line)
	}
	if !l.quiet {
		_, _ = fmt.Fprintln(os.Stdout, line)
	}
}

func (l *Logger) enabled(level Level) bool {
	if l == nil {
		return false
	}
	return levelRank(level) >= levelRank(l.level)
}

func levelRank(level Level) int {
	switch level {
	case LevelDebug:
		return 0
	case LevelInfo:
		return 1
	case LevelWarning:
		return 2
	case LevelError:
		return 3
	default:
		return 1
	}
}

func redactFields(fields map[string]any) map[string]any {
	if len(fields) == 0 {
		return nil
	}
	redacted := make(map[string]any, len(fields))
	for key, value := range fields {
		redacted[key] = redactAny(value)
	}
	return redacted
}

func redactAny(value any) any {
	switch v := value.(type) {
	case string:
		return safety.RedactString(v)
	case []string:
		out := make([]string, len(v))
		for i := range v {
			out[i] = safety.RedactString(v[i])
		}
		return out
	default:
		return v
	}
}

func Discard() *Logger {
	return &Logger{
		quiet: true,
		out:   io.Discard,
		level: LevelError,
	}
}
