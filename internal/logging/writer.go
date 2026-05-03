package logging

import "io"

type Writer struct {
	logger *Logger
	level  string
}

func NewWriter(logger *Logger, level string) io.Writer {
	return &Writer{logger: logger, level: level}
}

func (w *Writer) Write(p []byte) (int, error) {
	if w.logger != nil {
		w.logger.write(w.level, "%s", string(p))
	}
	return len(p), nil
}
