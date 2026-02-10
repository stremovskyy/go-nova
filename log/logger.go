package log

import (
	"io"
	stdlog "log"
	"os"
)

// Logger is a minimal printf-style logger used by the SDK.
//
// Implement this interface if you want to plug in your own logging (zap/logrus/etc).
type Logger interface {
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// Level controls what gets written by StdLogger.
//
// The ordering is: Debug < Info < Warn < Error < Off.
// Any message below the configured level is ignored.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelOff
)

// StdLogger is a tiny default implementation of Logger using the standard library log package.
type StdLogger struct {
	l     *stdlog.Logger
	level Level
	tag   string
}

func NewStdLogger(w io.Writer, level Level) *StdLogger {
	if w == nil {
		w = os.Stderr
	}
	return &StdLogger{
		l:     stdlog.New(w, "", stdlog.LstdFlags),
		level: level,
		tag:   "NovaPay",
	}
}

func NewDefault() *StdLogger {
	return NewStdLogger(os.Stderr, LevelInfo)
}

func (s *StdLogger) SetLevel(level Level) {
	if s == nil {
		return
	}
	s.level = level
}

func (s *StdLogger) SetTag(tag string) {
	if s == nil {
		return
	}
	s.tag = tag
}

func (s *StdLogger) format(format string) string {
	if s == nil || s.tag == "" {
		return format
	}
	return s.tag + ": " + format
}

func (s *StdLogger) Debugf(format string, args ...any) {
	if s == nil || s.level > LevelDebug {
		return
	}
	s.l.Printf("DEBUG: "+s.format(format), args...)
}

func (s *StdLogger) Infof(format string, args ...any) {
	if s == nil || s.level > LevelInfo {
		return
	}
	s.l.Printf("INFO: "+s.format(format), args...)
}

func (s *StdLogger) Warnf(format string, args ...any) {
	if s == nil || s.level > LevelWarn {
		return
	}
	s.l.Printf("WARN: "+s.format(format), args...)
}

func (s *StdLogger) Errorf(format string, args ...any) {
	if s == nil || s.level > LevelError {
		return
	}
	s.l.Printf("ERROR: "+s.format(format), args...)
}

// NopLogger discards all logs.
type NopLogger struct{}

func (NopLogger) Debugf(string, ...any) {}
func (NopLogger) Infof(string, ...any)  {}
func (NopLogger) Warnf(string, ...any)  {}
func (NopLogger) Errorf(string, ...any) {}
