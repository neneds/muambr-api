package utils

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger defines the interface for application logging
type Logger interface {
	// Info logs an informational message
	Info(msg string, fields ...Field)
	
	// Warn logs a warning message
	Warn(msg string, fields ...Field)
	
	// Error logs an error message
	Error(msg string, fields ...Field)
	
	// Debug logs a debug message
	Debug(msg string, fields ...Field)
	
	// Fatal logs a fatal message and exits the application
	Fatal(msg string, fields ...Field)
	
	// With creates a child logger with additional fields
	With(fields ...Field) Logger
	
	// Sync flushes any buffered log entries
	Sync() error
}

// Field represents a structured logging field
type Field struct {
	Key   string
	Value interface{}
}

// String creates a string field
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error creates an error field
func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any creates a field with any value
func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}

// ZapLogger implements the Logger interface using Uber's Zap library
type ZapLogger struct {
	logger *zap.Logger
}

// NewZapLogger creates a new ZapLogger instance
func NewZapLogger() (*ZapLogger, error) {
	return NewZapLoggerWithConfig(zap.NewProductionConfig())
}

// NewZapLoggerWithConfig creates a new ZapLogger with custom configuration
func NewZapLoggerWithConfig(config zap.Config) (*ZapLogger, error) {
	logger, err := config.Build()
	if err != nil {
		return nil, err
	}
	
	return &ZapLogger{logger: logger}, nil
}

// NewDevelopmentZapLogger creates a new ZapLogger for development
func NewDevelopmentZapLogger() (*ZapLogger, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.TimeKey = "timestamp"
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	
	return NewZapLoggerWithConfig(config)
}

// fieldsToZap converts our Field slice to zap.Field slice
func (z *ZapLogger) fieldsToZap(fields []Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		switch v := field.Value.(type) {
		case string:
			zapFields[i] = zap.String(field.Key, v)
		case int:
			zapFields[i] = zap.Int(field.Key, v)
		case int64:
			zapFields[i] = zap.Int64(field.Key, v)
		case float64:
			zapFields[i] = zap.Float64(field.Key, v)
		case bool:
			zapFields[i] = zap.Bool(field.Key, v)
		case error:
			zapFields[i] = zap.Error(v)
		default:
			zapFields[i] = zap.Any(field.Key, v)
		}
	}
	return zapFields
}

// Info logs an informational message
func (z *ZapLogger) Info(msg string, fields ...Field) {
	z.logger.Info(msg, z.fieldsToZap(fields)...)
}

// Warn logs a warning message
func (z *ZapLogger) Warn(msg string, fields ...Field) {
	z.logger.Warn(msg, z.fieldsToZap(fields)...)
}

// Error logs an error message
func (z *ZapLogger) Error(msg string, fields ...Field) {
	z.logger.Error(msg, z.fieldsToZap(fields)...)
}

// Debug logs a debug message
func (z *ZapLogger) Debug(msg string, fields ...Field) {
	z.logger.Debug(msg, z.fieldsToZap(fields)...)
}

// Fatal logs a fatal message and exits the application
func (z *ZapLogger) Fatal(msg string, fields ...Field) {
	z.logger.Fatal(msg, z.fieldsToZap(fields)...)
}

// With creates a child logger with additional fields
func (z *ZapLogger) With(fields ...Field) Logger {
	childLogger := z.logger.With(z.fieldsToZap(fields)...)
	return &ZapLogger{logger: childLogger}
}

// Sync flushes any buffered log entries
func (z *ZapLogger) Sync() error {
	return z.logger.Sync()
}

// Global logger instance
var globalLogger Logger

// InitLogger initializes the global logger
func InitLogger() error {
	logger, err := NewZapLogger()
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// InitDevelopmentLogger initializes the global logger for development
func InitDevelopmentLogger() error {
	logger, err := NewDevelopmentZapLogger()
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetLogger returns the global logger instance
func GetLogger() Logger {
	if globalLogger == nil {
		// Fallback to a basic logger if not initialized
		logger, _ := NewZapLogger()
		globalLogger = logger
	}
	return globalLogger
}

// Convenience functions for the global logger
func Info(msg string, fields ...Field) {
	GetLogger().Info(msg, fields...)
}

func Warn(msg string, fields ...Field) {
	GetLogger().Warn(msg, fields...)
}

func LogError(msg string, fields ...Field) {
	GetLogger().Error(msg, fields...)
}

func Debug(msg string, fields ...Field) {
	GetLogger().Debug(msg, fields...)
}

func Fatal(msg string, fields ...Field) {
	GetLogger().Fatal(msg, fields...)
}
