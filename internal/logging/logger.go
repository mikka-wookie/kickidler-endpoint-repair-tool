package logging

import (
	"fmt"
	"io"
	"log"
	"os"
)

type Logger struct {
	quiet bool
	file  *os.File
	log   *log.Logger
}

func New(path string, quiet bool) (*Logger, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{
		quiet: quiet,
		file:  file,
		log:   log.New(file, "", log.LstdFlags),
	}, nil
}

func (l *Logger) Info(format string, args ...any) {
	l.write("INFO", format, args...)
}

func (l *Logger) Warn(format string, args ...any) {
	l.write("WARN", format, args...)
}

func (l *Logger) Error(format string, args ...any) {
	l.write("ERROR", format, args...)
}

func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

func (l *Logger) write(level string, format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	l.log.Printf("[%s] %s", level, message)
	if !l.quiet {
		fmt.Fprintf(os.Stdout, "[%s] %s\n", level, message)
	}
}

func Discard() *Logger {
	return &Logger{
		quiet: true,
		log:   log.New(io.Discard, "", log.LstdFlags),
	}
}
