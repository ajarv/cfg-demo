package log

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// Level is the logging level
type Level int8

const (
	// DEBUG level for developer information
	DEBUG Level = iota - 1
	// INFO level for state and status
	INFO
	// WARN level for possible issues
	WARN
	// ERROR level for errors
	ERROR
	// PANIC level for unrecoverable errors that stop the goroutine
	PANIC
	// FATAL level for unrecoverable errors that stop the process
	FATAL
)

// String returns an upper case string representation of the log level
// nolint:goconst
func (l Level) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case PANIC:
		return "PANIC"
	case FATAL:
		return "FATAL"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

// PaddedString returns a five character upper case representation of the log level
// nolint:goconst
func (l Level) PaddedString() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO "
	case WARN:
		return "WARN "
	case ERROR:
		return "ERROR"
	case PANIC:
		return "PANIC"
	case FATAL:
		return "FATAL"
	default:
		return fmt.Sprintf("Level(%d)", l)
	}
}

// UnmarshalText converts an slice of characters to a Level
// nolint:goconst
func (l *Level) UnmarshalText(text []byte) bool {
	switch string(bytes.ToUpper(text)) {
	case "DEBUG":
		*l = DEBUG
	case "INFO", "":
		*l = INFO
	case "WARN":
		*l = WARN
	case "ERROR":
		*l = ERROR
	case "PANIC":
		*l = PANIC
	case "FATAL":
		*l = FATAL
	default:
		return false
	}
	return true
}

var defaultLogger *CoreLogger

// GetDefaultLogger returns the default logger implementation
func GetDefaultLogger() *CoreLogger {
	if defaultLogger == nil {
		defaultLogger = New()
	}
	return defaultLogger
}

// Configurator has methods to fetch the server configuration values
type Configurator interface {
	LogLevel() string
	Output() io.Writer
	TimestampFormat() string
	CallerFormat() string
}

// Setup is optionally called to configure the logging implementation. If
// it is not called, the default implementation will log at INFO level to
// standard output.
func Setup(config Configurator) {
	if defaultLogger == nil {
		defaultLogger = New()
	}
	defaultLogger.logLevel.UnmarshalText([]byte(config.LogLevel()))
	configOutfile := config.Output()
	if configOutfile != nil {
		defaultLogger.outfile = configOutfile
	}
	configTimestampFormat := config.TimestampFormat()
	if configTimestampFormat != "" {
		defaultLogger.timestampFormat = configTimestampFormat
	}
	configCallerFormat := config.CallerFormat()
	if configCallerFormat != "" {
		defaultLogger.callerFormat = configCallerFormat
	}
}

// CoreLogger is implements logging
type CoreLogger struct {
	logLevel        Level
	outfile         io.Writer
	timestampFormat string
	callerFormat    string
}

// New creates a new CoreLogger
func New() *CoreLogger {
	logger := CoreLogger{}
	logger.logLevel = INFO
	logger.outfile = os.Stdout
	logger.timestampFormat = "01-02 15:04:05.000 "
	logger.callerFormat = " %20.20s:%03d - "
	return &logger
}

// GetLevel gets the current logging level
func (l *CoreLogger) GetLevel() Level {
	return l.logLevel
}

// SetLevel sets a filter on the minimum level of messages that will be logged. For
// example if the level is WARN then no DEBUG or INFO messages will be logged.
func (l *CoreLogger) SetLevel(level Level) {
	l.logLevel = level
}

// Fatal logs a message at FATAL level and then calls os.Exit(1)
func (l *CoreLogger) Fatal(v ...interface{}) {
	l.log(FATAL, "", v, nil)
	os.Exit(1)
}

// Fatalf logs a formatted message at FATAL level and then calls os.Exit(1)
func (l *CoreLogger) Fatalf(format string, v ...interface{}) {
	l.log(FATAL, format, v, nil)
	os.Exit(1)
}

// Fatalln logs a message at FATAL level and then calls os.Exit(1)
func (l *CoreLogger) Fatalln(v ...interface{}) {
	l.log(FATAL, "", v, nil)
	os.Exit(1)
}

// Flags is not implemented
func (l *CoreLogger) Flags() int {
	return 0
}

// Output writes the output for a logging event. The string s contains
// the message to log. Calldepth is ignored.
func (l *CoreLogger) Output(calldepth int, s string) error {
	l.log(INFO, "", []interface{}{s}, nil)
	return nil
}

// Panic logs a message at PANIC level and then calls panic().
func (l *CoreLogger) Panic(v ...interface{}) {
	l.log(PANIC, "", v, nil)
	panic(fmt.Sprint(v...))
}

// Panicf logs a formatted message at PANIC level and then calls panic().
func (l *CoreLogger) Panicf(format string, v ...interface{}) {
	l.log(PANIC, format, v, nil)
	panic(fmt.Sprintf(format, v...))
}

// Panicln logs a message and at PANIC level then calls panic().
func (l *CoreLogger) Panicln(v ...interface{}) {
	l.log(PANIC, "", v, nil)
	panic(fmt.Sprint(v...))
}

// Prefix is not implemented.
func (l *CoreLogger) Prefix() string {
	return ""
}

// Print logs a message at INFO level.
func (l *CoreLogger) Print(v ...interface{}) {
	l.log(INFO, "", v, nil)
}

// Printf logs a formatted message at INFO level.
func (l *CoreLogger) Printf(format string, v ...interface{}) {
	l.log(INFO, format, v, nil)
}

// Println logs a message at INFO level.
func (l *CoreLogger) Println(v ...interface{}) {
	l.log(INFO, "", v, nil)
}

// SetFlags is not implemented.
func (l *CoreLogger) SetFlags(flag int) {
	// not implemented
}

// SetOutput sets the io.Writer to which all future log messages will be written.
func (l *CoreLogger) SetOutput(w io.Writer) {
	l.outfile = w
}

// SetPrefix is not implemented.
func (l *CoreLogger) SetPrefix(prefix string) {
	// not implemented
}

// extensions to standard go library

// Debugf logs a formatted message at DEBUG level.
func (l *CoreLogger) Debugf(format string, args ...interface{}) {
	l.log(DEBUG, format, args, nil)
}

// Infof logs a formatted message at INFO level.
func (l *CoreLogger) Infof(format string, args ...interface{}) {
	l.log(INFO, format, args, nil)
}

// Warnf logs a formatted message at WARN level.
func (l *CoreLogger) Warnf(format string, args ...interface{}) {
	l.log(WARN, format, args, nil)
}

// Errorf logs a formatted message at ERROR level.
func (l *CoreLogger) Errorf(format string, args ...interface{}) {
	l.log(ERROR, format, args, nil)
}

func (l *CoreLogger) log(level Level, format string, args []interface{}, context []interface{}) {
	if level < l.logLevel {
		return
	}
	_, file, line, ok := runtime.Caller(3)
	if !ok {
		file = "???"
		line = 0
	} else {
		file = filepath.Base(file)
	}
	var msg string
	if format == "" {
		msg = fmt.Sprint(args...)
	} else {
		msg = fmt.Sprintf(format, args...)
	}

	msg = sanitize(msg)

	var b strings.Builder
	b.WriteString(time.Now().Format(l.timestampFormat))
	b.WriteString(level.PaddedString())
	_, _ = fmt.Fprintf(&b, l.callerFormat, file, line)
	b.WriteString(msg)
	b.WriteString("\n")
	_, _ = l.outfile.Write([]byte(b.String()))
}

var matchers = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(password"\s*:?\s*")(.*?)(")`),
	regexp.MustCompile(`(?i)(search_pass"\s*:?\s*")(.*?)(")`),
	regexp.MustCompile(`(?i)(token\w*"\s*:?\s*")([a-zA-Z0-9_\.\-]+)`),
	regexp.MustCompile(`(?i)(token\s*=\s*)([a-zA-Z0-9_\.\-]+)`),
	regexp.MustCompile(`(?i)(token\s+)([a-zA-Z0-9_\.\-]+)`),
	regexp.MustCompile(`(?i)(key"\s*:?\s*")([^:'"]*)(")`),
}

const replacement = "${1}******${3}"

// sanitize removes passwords and tokens from logged messages
func sanitize(msg string) string {
	for _, regex := range matchers {
		msg = regex.ReplaceAllString(msg, replacement)
	}
	return msg
}

func log(level Level, format string, args []interface{}, context []interface{}) {
	if defaultLogger == nil {
		defaultLogger = New()
	}
	defaultLogger.log(level, format, args, context)
}

// golang log package compatibility functions

// Fatal logs a message at FATAL level and then calls os.Exit(1)
func Fatal(v ...interface{}) {
	log(FATAL, "", v, nil)
	os.Exit(1)
}

// Fatalf logs a formatted message at FATAL level and then calls os.Exit(1)
func Fatalf(format string, v ...interface{}) {
	log(FATAL, format, v, nil)
	os.Exit(1)
}

// Fatalln logs a message at FATAL level and then calls os.Exit(1)
func Fatalln(v ...interface{}) {
	log(FATAL, "", v, nil)
	os.Exit(1)
}

// Flags is not implemented
func Flags() int {
	return 0
}

// Output writes the output for a logging event. The string s contains
// the message to log. Calldepth is ignored.
func Output(calldepth int, s string) error {
	log(INFO, "", []interface{}{s}, nil)
	return nil
}

// Panic logs a message at PANIC level and then calls panic().
func Panic(v ...interface{}) {
	log(PANIC, "", v, nil)
	panic(fmt.Sprint(v...))
}

// Panicf logs a formatted message at PANIC level and then calls panic().
func Panicf(format string, v ...interface{}) {
	log(PANIC, format, v, nil)
	panic(fmt.Sprintf(format, v...))
}

// Panicln logs a message and at PANIC level then calls panic().
func Panicln(v ...interface{}) {
	log(PANIC, "", v, nil)
	panic(fmt.Sprint(v...))
}

// Prefix is not implemented.
func Prefix() string {
	return ""
}

// Print logs a message at INFO level.
func Print(v ...interface{}) {
	log(INFO, "", v, nil)
}

// Printf logs a formatted message at INFO level.
func Printf(format string, v ...interface{}) {
	log(INFO, format, v, nil)
}

// Println logs a message at INFO level.
func Println(v ...interface{}) {
	log(INFO, "", v, nil)
}

// SetFlags is not implemented.
func SetFlags(flag int) {
	// not implemented
}

// SetOutput sets the io.Writer to which all future log messages will be written.
func SetOutput(w io.Writer) {
	if defaultLogger == nil {
		defaultLogger = New()
	}
	defaultLogger.outfile = w
}

// SetPrefix is not implemented.
func SetPrefix(prefix string) {
	// not implemented
}

// extensions to standard go library

// SetLevel sets a filter on the minimum level of messages that will be logged. For
// example if the level is WARN then no DEBUG or INFO messages will be logged.
func SetLevel(level Level) {
	if defaultLogger == nil {
		defaultLogger = New()
	}
	defaultLogger.logLevel = level
}

// Debugf logs a formatted message at DEBUG level.
func Debugf(format string, args ...interface{}) {
	log(DEBUG, format, args, nil)
}

// Infof logs a formatted message at INFO level.
func Infof(format string, args ...interface{}) {
	log(INFO, format, args, nil)
}

// Warnf logs a formatted message at WARN level.
func Warnf(format string, args ...interface{}) {
	log(WARN, format, args, nil)
}

// Errorf logs a formatted message at ERROR level.
func Errorf(format string, args ...interface{}) {
	log(ERROR, format, args, nil)
}
