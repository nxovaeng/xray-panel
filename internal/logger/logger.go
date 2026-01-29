package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"xray-panel/internal/config"
)

// Level represents log level
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarning
	LevelError
)

var (
	currentLevel Level
	debugLogger  *log.Logger
	infoLogger   *log.Logger
	warnLogger   *log.Logger
	errorLogger  *log.Logger
)

// Init initializes the logger with configuration
func Init(cfg *config.LogConfig) error {
	// Parse log level
	currentLevel = parseLevel(cfg.Level)

	// Setup output writers
	var writers []io.Writer

	// Always write to stdout
	writers = append(writers, os.Stdout)

	// Add file output if configured
	if cfg.File != "" {
		// Ensure directory exists
		dir := filepath.Dir(cfg.File)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		// Setup log rotation
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,    // MB
			MaxBackups: cfg.MaxBackups, // number of backups
			MaxAge:     cfg.MaxAge,     // days
			Compress:   cfg.Compress,   // compress rotated files
		}
		writers = append(writers, fileWriter)
	}

	// Create multi-writer
	multiWriter := io.MultiWriter(writers...)

	// Initialize loggers with different prefixes
	debugLogger = log.New(multiWriter, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
	infoLogger = log.New(multiWriter, "[INFO]  ", log.LstdFlags)
	warnLogger = log.New(multiWriter, "[WARN]  ", log.LstdFlags)
	errorLogger = log.New(multiWriter, "[ERROR] ", log.LstdFlags|log.Lshortfile)

	// Set default logger
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags)

	Info("Logger initialized with level: %s", cfg.Level)
	if cfg.File != "" {
		Info("Logging to file: %s", cfg.File)
	}

	return nil
}

// parseLevel parses log level string
func parseLevel(level string) Level {
	switch strings.ToLower(level) {
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "warning", "warn":
		return LevelWarning
	case "error":
		return LevelError
	default:
		return LevelInfo
	}
}

// Debug logs debug message
func Debug(format string, v ...interface{}) {
	if currentLevel <= LevelDebug {
		debugLogger.Printf(format, v...)
	}
}

// Info logs info message
func Info(format string, v ...interface{}) {
	if currentLevel <= LevelInfo {
		infoLogger.Printf(format, v...)
	}
}

// Warn logs warning message
func Warn(format string, v ...interface{}) {
	if currentLevel <= LevelWarning {
		warnLogger.Printf(format, v...)
	}
}

// Error logs error message
func Error(format string, v ...interface{}) {
	if currentLevel <= LevelError {
		errorLogger.Printf(format, v...)
	}
}

// Fatal logs fatal message and exits
func Fatal(format string, v ...interface{}) {
	errorLogger.Fatalf(format, v...)
}

// Debugln logs debug message with newline
func Debugln(v ...interface{}) {
	if currentLevel <= LevelDebug {
		debugLogger.Println(v...)
	}
}

// Infoln logs info message with newline
func Infoln(v ...interface{}) {
	if currentLevel <= LevelInfo {
		infoLogger.Println(v...)
	}
}

// Warnln logs warning message with newline
func Warnln(v ...interface{}) {
	if currentLevel <= LevelWarning {
		warnLogger.Println(v...)
	}
}

// Errorln logs error message with newline
func Errorln(v ...interface{}) {
	if currentLevel <= LevelError {
		errorLogger.Println(v...)
	}
}

// SetLevel sets the current log level
func SetLevel(level string) {
	currentLevel = parseLevel(level)
	Info("Log level changed to: %s", level)
}

// GetLevel returns the current log level as string
func GetLevel() string {
	switch currentLevel {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarning:
		return "warning"
	case LevelError:
		return "error"
	default:
		return "info"
	}
}
